package gateway

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/ashishk1331/seline-agent/internal/config"
)

// ProcessFunc handles a coalesced batch of user text.
type ProcessFunc func(ctx context.Context, update *models.Update, text string)

// Debouncer coalesces rapid consecutive messages into a single agent run. Each
// new message cancels the pending dispatch and extends the delay. Port of
// debounce.py. It is global (not keyed per chat), matching the Python design.
type Debouncer struct {
	delay    time.Duration
	jitter   time.Duration
	maxDelay time.Duration
	parent   context.Context

	mu      sync.Mutex
	texts   []string
	counter int
	cancel  context.CancelFunc
}

// NewDebouncer constructs a Debouncer. parent is the base context; dispatched
// tasks are cancelled when it is cancelled (graceful shutdown).
func NewDebouncer(cfg *config.Config, parent context.Context) *Debouncer {
	secs := func(f float64) time.Duration { return time.Duration(f * float64(time.Second)) }
	return &Debouncer{
		delay:    secs(cfg.MessageDebounceDelay),
		jitter:   secs(cfg.MessageDebounceJitter),
		maxDelay: secs(cfg.MessageDebounceMaxDelay),
		parent:   parent,
	}
}

// Add registers a message and (re)schedules dispatch.
func (d *Debouncer) Add(update *models.Update, processor ProcessFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if update.Message != nil && update.Message.Text != "" {
		d.texts = append(d.texts, update.Message.Text)
	}

	if d.cancel != nil {
		d.cancel()
		d.counter++
	}

	ctx, cancel := context.WithCancel(d.parent)
	d.cancel = cancel
	go d.dispatch(ctx, update, processor)
}

func (d *Debouncer) calculateDelay() time.Duration {
	d.mu.Lock()
	delay := d.delay + time.Duration(d.counter)*d.jitter
	d.mu.Unlock()
	if delay > d.maxDelay {
		return d.maxDelay
	}
	return delay
}

func (d *Debouncer) dispatch(ctx context.Context, update *models.Update, processor ProcessFunc) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(d.calculateDelay()):
	}

	// Snapshot and reset the accumulated texts before processing so messages
	// arriving during the run are not lost.
	d.mu.Lock()
	combined := strings.Join(d.texts, "\n")
	d.texts = nil
	d.counter = 0
	d.cancel = nil
	d.mu.Unlock()

	processor(ctx, update, combined)
}

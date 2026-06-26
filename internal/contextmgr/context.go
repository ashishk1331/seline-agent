package contextmgr

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/logging"
	"github.com/ashishk1331/seline-agent/internal/prompts"
	"github.com/ashishk1331/seline-agent/internal/provider"
	"github.com/ashishk1331/seline-agent/internal/types"
)

// ContextManager owns the conversation context, token accounting and
// compaction. It embeds Session for persistence. A mutex guards all mutable
// state because the Telegram library dispatches handlers concurrently (Python
// relied on its single-threaded event loop and needed no lock).
type ContextManager struct {
	*Session

	cfg      *config.Config
	prompts  *prompts.PromptBase
	resolver *provider.LLMResolver

	mu                sync.Mutex
	context           []types.Message
	compactionContext []types.Message
	maxTokens         int
	currentTokens     int
}

// New constructs a ContextManager, loading any persisted session.
func New(cfg *config.Config, p *prompts.PromptBase, resolver *provider.LLMResolver) (*ContextManager, error) {
	session, err := NewSession(cfg, "")
	if err != nil {
		return nil, err
	}

	loaded, err := session.loadMessagesFromSession()
	if err != nil {
		return nil, err
	}

	c := &ContextManager{
		Session:  session,
		cfg:      cfg,
		prompts:  p,
		resolver: resolver,
		context: append(
			[]types.Message{{Role: "system", Content: p.GetSystemPrompt()}},
			loaded...,
		),
		compactionContext: []types.Message{
			{Role: "system", Content: p.GetCompactionPrompt()},
		},
		maxTokens:     cfg.MaxContextTokens,
		currentTokens: session.retrieveConsumption().CurrentTokens,
	}
	return c, nil
}

// Append adds a message to the context and persists it. When usage is provided
// the token count is updated and compaction may be triggered.
//
// Persistence order differs slightly from the Python original: the message is
// written to the session file before compaction runs. This avoids a latent
// Python bug where a compaction-triggering message was written twice (once by
// the compaction rewrite, once by the trailing append).
func (c *ContextManager) Append(ctx context.Context, msg types.Message, usage *types.Usage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.context = append(c.context, msg)
	if err := c.appendMessageToSession(msg); err != nil {
		logging.Log.Error("failed to persist message", "err", err)
	}

	if usage != nil {
		c.currentTokens = usage.TotalTokens
		c.detectAndCompactLocked(ctx)
	}

	if err := c.dumpConsumption(c.consumptionLocked()); err != nil {
		logging.Log.Error("failed to persist consumption", "err", err)
	}
}

// GetContext returns a copy of the current context slice (safe to iterate while
// other goroutines mutate the manager).
func (c *ContextManager) GetContext() []types.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]types.Message, len(c.context))
	copy(out, c.context)
	return out
}

// GetConsumption returns the current token-usage snapshot.
func (c *ContextManager) GetConsumption() Consumption {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consumptionLocked()
}

func (c *ContextManager) consumptionLocked() Consumption {
	remaining := c.maxTokens - c.currentTokens
	pct := 0.0
	if c.maxTokens > 0 {
		pct = math.Round(float64(c.currentTokens)/float64(c.maxTokens)*100*100) / 100
	}
	return Consumption{
		CurrentTokens:          c.currentTokens,
		CurrentTokensInWords:   inWords(c.currentTokens),
		MaxTokens:              c.maxTokens,
		MaxTokensInWords:       inWords(c.maxTokens),
		RemainingTokens:        remaining,
		RemainingTokensInWords: inWords(remaining),
		PercentageUsed:         pct,
	}
}

// Checkpoint returns the current context length (for rollback on cancellation).
func (c *ContextManager) Checkpoint() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.context)
}

// Rollback truncates the context back to a checkpoint and rewrites the session.
func (c *ContextManager) Rollback(checkpoint int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.context) > checkpoint {
		c.context = c.context[:checkpoint]
		if err := c.overwriteMessagesInSession(c.context[1:]); err != nil {
			logging.Log.Error("failed to rewrite session on rollback", "err", err)
		}
	}
}

// detectAndCompactLocked runs sliding-window compaction if the token threshold
// is exceeded. Caller must hold the mutex.
func (c *ContextManager) detectAndCompactLocked(ctx context.Context) {
	if float64(c.currentTokens) < float64(c.maxTokens)*c.cfg.CompactionThreshold {
		return
	}
	if len(c.context) <= c.cfg.CompactionRecentN+1 {
		logging.Log.Error("[CONTEXT] not enough messages to compact", "length", len(c.context))
		return
	}

	logging.Log.Info("[CONTEXT] Auto-compaction triggered.")

	n := c.cfg.CompactionRecentN
	recent := append([]types.Message(nil), c.context[len(c.context)-n:]...)
	previous := c.context[1 : len(c.context)-n]
	prevTokens := c.currentTokens

	compactionInput := append(
		append([]types.Message(nil), c.compactionContext...),
		types.Message{Role: "user", Content: c.messagesIron(previous)},
	)

	summary, usage, err := c.compaction(ctx, compactionInput)
	if err != nil || summary == "" {
		logging.Log.Error("[CONTEXT] Compaction failed. Keeping existing context.", "err", err)
		return
	}

	rebuilt := []types.Message{
		{Role: "system", Content: c.prompts.GetSystemPrompt()},
		{Role: "system", Content: fmt.Sprintf("[Compacted summary of earlier conversation: %s]", summary)},
	}
	rebuilt = append(rebuilt, recent...)
	c.context = rebuilt

	if usage != nil {
		c.currentTokens = usage.TotalTokens
	}

	if err := c.overwriteMessagesInSession(c.context[1:]); err != nil {
		logging.Log.Error("failed to rewrite session after compaction", "err", err)
	}
	if err := c.dumpConsumption(c.consumptionLocked()); err != nil {
		logging.Log.Error("failed to persist consumption after compaction", "err", err)
	}

	logging.Log.Info("[CONTEXT] Compaction completed.", "before", prevTokens, "after", c.currentTokens)
}

// compaction asks the model to summarize. Returns the summary, usage and error.
func (c *ContextManager) compaction(ctx context.Context, messages []types.Message) (string, *types.Usage, error) {
	data, err := c.resolver.Resolve(ctx, messages)
	if err != nil || data == nil {
		logging.Log.Error("No response from API during compaction.", "err", err)
		return "", nil, err
	}
	if len(data.Choices) == 0 {
		return "", nil, nil
	}
	return contentToString(data.Choices[0].Message.Content), &data.Usage, nil
}

// messagesIron flattens messages into a "[role] content" transcript.
func (c *ContextManager) messagesIron(messages []types.Message) string {
	lines := make([]string, 0, len(messages))
	for _, m := range messages {
		lines = append(lines, fmt.Sprintf("[%s] %s", m.Role, contentToString(m.Content)))
	}
	return strings.Join(lines, "\n")
}

// inWords renders a token count compactly: 1.2M / 3.4K / 567.
func inWords(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%gM", math.Round(float64(n)/1_000_000*10)/10)
	case n >= 1_000:
		return fmt.Sprintf("%gK", math.Round(float64(n)/1_000*10)/10)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// contentToString normalizes a message content (string or list of parts) into a
// plain string, mirroring the Python list-content handling in messages_iron.
func contentToString(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	case []types.ContentPart:
		parts := make([]string, 0, len(v))
		for _, p := range v {
			parts = append(parts, p.Text)
		}
		return strings.Join(parts, " ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

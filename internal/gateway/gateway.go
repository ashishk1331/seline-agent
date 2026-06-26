package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/contextmgr"
	"github.com/ashishk1331/seline-agent/internal/llm"
	"github.com/ashishk1331/seline-agent/internal/logging"
	"github.com/ashishk1331/seline-agent/internal/provider"
)

// command is a (name, description) pair for the bot command menu.
type command struct{ name, description string }

// commands are the registered bot commands. Port of agent.py::COMMANDS.
var commands = []command{
	{"start", "Start the bot"},
	{"consumption", "Check token consumption"},
}

// Gateway owns the Telegram bot and routes updates to the agent.
type Gateway struct {
	cfg      *config.Config
	agent    *llm.Agent
	ctxmgr   *contextmgr.ContextManager
	status   *Status
	resolver *provider.LLMResolver

	bot        *bot.Bot
	debouncer  *Debouncer
	richClient *http.Client
}

// New constructs a Gateway.
func New(
	cfg *config.Config,
	agent *llm.Agent,
	ctxmgr *contextmgr.ContextManager,
	status *Status,
	resolver *provider.LLMResolver,
) *Gateway {
	return &Gateway{
		cfg:        cfg,
		agent:      agent,
		ctxmgr:     ctxmgr,
		status:     status,
		resolver:   resolver,
		richClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Run creates the bot, registers handlers and starts long polling. It blocks
// until ctx is cancelled, then closes the resolver. Port of telegram_loop +
// post_init + post_shutdown.
func (g *Gateway) Run(ctx context.Context) error {
	g.debouncer = NewDebouncer(g.cfg, ctx)

	opts := []bot.Option{
		bot.WithDefaultHandler(g.handleMessage),
	}
	b, err := bot.New(g.cfg.TelegramBotToken, opts...)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}
	g.bot = b
	g.status.SetBot(b)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, g.startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/consumption", bot.MatchTypePrefix, g.consumptionHandler)

	g.postInit(ctx)

	logging.Log.Info("Seline is up.")
	b.Start(ctx) // blocks until ctx is cancelled

	g.resolver.Close()
	logging.Log.Info("LLM resolver http client closed.")
	return nil
}

// postInit registers the command menu and prints the command table.
func (g *Gateway) postInit(ctx context.Context) {
	tgCommands := make([]models.BotCommand, 0, len(commands))
	for _, c := range commands {
		tgCommands = append(tgCommands, models.BotCommand{Command: c.name, Description: c.description})
	}
	if _, err := g.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: tgCommands}); err != nil {
		logging.Log.Error("failed to set bot commands", "err", err)
	}

	var sb strings.Builder
	sb.WriteString("Bot Commands ->\n")
	for _, c := range commands {
		sb.WriteString(fmt.Sprintf("  /%s — %s\n", c.name, c.description))
	}
	fmt.Print(sb.String())
}

// isUserAllowed checks an @-prefixed username against the allowlist.
func (g *Gateway) isUserAllowed(name string) bool {
	for _, u := range g.cfg.TelegramAllowlistCleaned {
		if u == name {
			return true
		}
	}
	return false
}

// senderName returns the @-prefixed username (matching Python's user.name),
// falling back to the first name when no username is set.
func senderName(u *models.User) string {
	if u == nil {
		return ""
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.FirstName
}

// startHandler handles /start.
func (g *Gateway) startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	name := senderName(update.Message.From)
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Hi %s! I'm seline.", name),
	})
}

// consumptionHandler handles /consumption.
func (g *Gateway) consumptionHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	stats := g.ctxmgr.GetConsumption()
	msg := fmt.Sprintf(
		"Current token usage: %s / %s tokens (%g%% used, %s tokens remaining)",
		stats.CurrentTokensInWords, stats.MaxTokensInWords, stats.PercentageUsed, stats.RemainingTokensInWords,
	)
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})
}

// handleMessage is the default handler for plain text messages. Port of
// agent.py::handle_message.
func (g *Gateway) handleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	// Ignore commands (Python excludes them via filters.COMMAND).
	if strings.HasPrefix(update.Message.Text, "/") {
		return
	}

	if update.Message.From != nil {
		name := senderName(update.Message.From)
		logging.Log.Info("incoming message", "from", name, "text", update.Message.Text)
		if !g.isUserAllowed(name) {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "You're not in allow list.",
			})
			return
		}
	} else {
		logging.Log.Info("received message", "text", update.Message.Text)
	}

	g.debouncer.Add(update, g.process)
}

// process runs the agent for a coalesced message batch. Port of agent.py::_process.
func (g *Gateway) process(ctx context.Context, update *models.Update, text string) {
	g.status.SetUpdate(update)
	g.status.Start(ctx)

	response, err := g.agent.Complete(ctx, &text)

	// Cancellation: the debouncer cancelled this run (a newer message arrived,
	// or shutdown). Clear the bubble and bail.
	if errors.Is(err, context.Canceled) || ctx.Err() != nil {
		g.status.Stop(ctx)
		return
	}

	g.status.Stop(ctx)

	if err != nil {
		g.handleError(ctx, update, err)
		return
	}

	if response == "" || update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	if result, rerr := g.sendRichMessage(ctx, chatID, response, 0); rerr != nil || result == nil {
		// Fall back to a standard markdown message.
		if _, serr := g.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      response,
			ParseMode: models.ParseModeMarkdown,
		}); serr != nil {
			logging.Log.Error("failed to send response", "err", serr)
		}
	}
}

package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/ashishk1331/seline-agent/internal/logging"
	"github.com/ashishk1331/seline-agent/internal/provider"
)

// notify replies to the user with an error message. Port of error.py::_notify.
func (g *Gateway) notify(ctx context.Context, update *models.Update, message string) {
	if update.Message == nil {
		return
	}
	if _, err := g.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("[⚠️ Error] %s", message),
	}); err != nil {
		logging.Log.Error("Unable to notify user about the error", "message", message, "err", err)
	}
}

// handleError classifies an error and notifies the user. Port of
// error.py::handle_error, adapted to Go error types.
func (g *Gateway) handleError(ctx context.Context, update *models.Update, err error) {
	var httpErr *provider.HTTPStatusError
	switch {
	case errors.As(err, &httpErr) && httpErr.Code == 429:
		logging.Log.Warn("Rate limited by provider")
		g.notify(ctx, update, "Rate limited. Wait a moment and try again.")

	case errors.As(err, &httpErr):
		logging.Log.Error("HTTP error from provider", "status", httpErr.Code)
		g.notify(ctx, update, fmt.Sprintf("Provider returned an error (%d).", httpErr.Code))

	case isTimeout(err):
		logging.Log.Error("LLM timeout", "err", err)
		g.notify(ctx, update, "Timed out waiting on model. Try again")

	case isNetworkError(err):
		logging.Log.Error("Network error", "err", err)
		g.notify(ctx, update, "Network error. Please try again.")

	default:
		logging.Log.Error("Unhandled error in agent pipeline", "err", err)
		g.notify(ctx, update, "Something went wrong. Please try again.")
	}
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

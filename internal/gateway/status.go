// Package gateway wires the Telegram bot to the agent. Go port of the gateway/
// package (agent.py, status.py, debounce.py, error.py, messages.py).
package gateway

import (
	"context"
	"math/rand"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/ashishk1331/seline-agent/internal/constants"
	"github.com/ashishk1331/seline-agent/internal/logging"
)

// Status manages the single "thinking" status bubble. Port of status.py.
// A mutex guards its state since handlers run concurrently.
type Status struct {
	b *bot.Bot

	mu        sync.Mutex
	chatID    int64
	replyTo   int
	messageID int // 0 = no active status message
}

// NewStatus creates a Status. The bot is attached later via SetBot.
func NewStatus() *Status {
	return &Status{}
}

// SetBot attaches the bot instance (called once the bot is created).
func (s *Status) SetBot(b *bot.Bot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.b = b
}

// SetUpdate records the chat/message the next status bubble should reply to.
func (s *Status) SetUpdate(update *models.Update) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if update.Message != nil {
		s.chatID = update.Message.Chat.ID
		s.replyTo = update.Message.ID
	}
}

// Start clears any existing bubble and posts a new random "thinking" message.
func (s *Status) Start(ctx context.Context) {
	s.Stop(ctx)

	s.mu.Lock()
	b, chatID, replyTo := s.b, s.chatID, s.replyTo
	s.mu.Unlock()
	if b == nil || chatID == 0 {
		return
	}

	phrase := constants.ThinkingPhrases[rand.Intn(len(constants.ThinkingPhrases))]
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          chatID,
		Text:            phrase,
		ReplyParameters: &models.ReplyParameters{MessageID: replyTo},
	})
	if err != nil {
		logging.Log.Error("failed to post status message", "err", err)
		return
	}

	s.mu.Lock()
	s.messageID = msg.ID
	s.mu.Unlock()
}

// Stop deletes the current status bubble, if any.
func (s *Status) Stop(ctx context.Context) {
	s.mu.Lock()
	b, chatID, messageID := s.b, s.chatID, s.messageID
	s.messageID = 0
	s.mu.Unlock()
	if b == nil || messageID == 0 {
		return
	}

	if _, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	}); err != nil {
		logging.Log.Error("Message not found to delete.", "err", err)
	}
}

// Update edits the current status bubble's text. Satisfies tools.StatusUpdater.
func (s *Status) Update(ctx context.Context, message string) error {
	s.mu.Lock()
	b, chatID, messageID := s.b, s.chatID, s.messageID
	s.mu.Unlock()
	if b == nil || messageID == 0 {
		return nil
	}

	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      message,
	}); err != nil {
		logging.Log.Error("Failed to update status message", "err", err)
		return err
	}
	return nil
}

package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// sendRichMessage posts to Telegram's custom sendRichMessage endpoint with a
// markdown payload. Port of messages.py. Returns (nil, nil) on a non-200 so the
// caller can fall back to a standard sendMessage.
func (g *Gateway) sendRichMessage(ctx context.Context, chatID int64, markdown string, replyTo int) (map[string]any, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"rich_message": map[string]any{
			"markdown": markdown,
		},
	}
	if replyTo != 0 {
		payload["reply_parameters"] = map[string]any{"message_id": replyTo}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendRichMessage", g.cfg.TelegramBotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.richClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// Package provider implements the LLM resolver: the HTTP client that talks to
// the OpenAI-compatible chat-completions endpoint. Go port of
// provider/resolver.py.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/logging"
	"github.com/ashishk1331/seline-agent/internal/types"
)

// HTTPStatusError is returned when the provider responds with a non-2xx status.
// It lets the gateway classify rate-limit / HTTP errors (error.py).
type HTTPStatusError struct {
	Code int
	Body string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("provider returned status %d", e.Code)
}

// LLMResolver wraps the HTTP client to the provider.
type LLMResolver struct {
	cfg    *config.Config
	url    string
	apiKey string
	tools  []map[string]any
	client *http.Client
}

// NewResolver builds a resolver. toolsPayload is the registry's provider schema,
// injected to avoid importing the tools package (and a cycle).
func NewResolver(cfg *config.Config, toolsPayload []map[string]any) *LLMResolver {
	// Timeouts mirror httpx: connect=10s, read(response header)=60s, plus an
	// overall per-request deadline applied via context in Resolve.
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	return &LLMResolver{
		cfg:    cfg,
		url:    cfg.AIProviderLLMURL,
		apiKey: cfg.AIProviderAPIKey,
		tools:  toolsPayload,
		client: &http.Client{Transport: transport},
	}
}

// payload builds the chat-completions request body.
func (r *LLMResolver) payload(messages []types.Message) map[string]any {
	return map[string]any{
		"model":       r.cfg.ModelName,
		"max_tokens":  r.cfg.MaxTokens,
		"temperature": r.cfg.Temperature,
		"messages":    messages,
		"tools":       r.tools,
	}
}

// Resolve posts the messages and returns the decoded completion. On a non-200
// response it returns (nil, *HTTPStatusError); on success (resp, nil).
func (r *LLMResolver) Resolve(ctx context.Context, messages []types.Message) (*types.CompletionResponse, error) {
	body, err := json.Marshal(r.payload(messages))
	if err != nil {
		return nil, err
	}

	// Overall per-request deadline (read budget).
	rctx, cancel := context.WithTimeout(ctx, 70*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(rctx, http.MethodPost, r.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logging.Log.Error("API request failed", "status", resp.StatusCode, "body", string(respBody))
		return nil, &HTTPStatusError{Code: resp.StatusCode, Body: string(respBody)}
	}

	var data types.CompletionResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// Close releases idle connections.
func (r *LLMResolver) Close() {
	r.client.CloseIdleConnections()
}

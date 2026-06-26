// Package llm implements the recursive tool-calling agent loop. Go port of
// llm.py. The module-level globals (context, complete) become an Agent struct
// with injected dependencies.
package llm

import (
	"context"
	"encoding/json"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/contextmgr"
	"github.com/ashishk1331/seline-agent/internal/logging"
	"github.com/ashishk1331/seline-agent/internal/provider"
	"github.com/ashishk1331/seline-agent/internal/tools"
	"github.com/ashishk1331/seline-agent/internal/types"
)

// Agent runs the completion loop against the provider, dispatching tool calls.
type Agent struct {
	ctxmgr   *contextmgr.ContextManager
	resolver *provider.LLMResolver
	tools    *tools.Registry
	cfg      *config.Config
}

// NewAgent constructs an Agent.
func NewAgent(
	ctxmgr *contextmgr.ContextManager,
	resolver *provider.LLMResolver,
	registry *tools.Registry,
	cfg *config.Config,
) *Agent {
	return &Agent{ctxmgr: ctxmgr, resolver: resolver, tools: registry, cfg: cfg}
}

// Complete runs the agent loop for an incoming user message. message may be nil
// on recursive tool-resolution turns. Returns the final assistant text.
func (a *Agent) Complete(ctx context.Context, message *string) (string, error) {
	return a.complete(ctx, message, a.cfg.MaxToolCalls, -1)
}

func (a *Agent) complete(ctx context.Context, message *string, maxToolCalls, checkpoint int) (string, error) {
	if maxToolCalls <= 0 {
		logging.Log.Error("Maximum tool call limit reached.")
		return "", nil
	}

	// Take a checkpoint on first entry, for rollback on cancellation.
	if checkpoint < 0 {
		checkpoint = a.ctxmgr.Checkpoint()
	}

	// Cancellation -> roll back the conversation and surface the error.
	if err := ctx.Err(); err != nil {
		a.ctxmgr.Rollback(checkpoint)
		return "", err
	}

	if message != nil {
		a.ctxmgr.Append(ctx, types.Message{Role: "user", Content: *message}, nil)
	}

	data, err := a.resolver.Resolve(ctx, a.ctxmgr.GetContext())
	if err != nil {
		if ctx.Err() != nil {
			a.ctxmgr.Rollback(checkpoint)
		}
		return "", err
	}
	if data == nil || len(data.Choices) == 0 {
		logging.Log.Error("No response from API.")
		return "", nil
	}

	msg := data.Choices[0].Message
	usage := data.Usage

	if len(msg.ToolCalls) > 0 {
		// Persist the assistant message that requested the tools.
		a.ctxmgr.Append(ctx, msg, &usage)

		for _, tc := range msg.ToolCalls {
			if err := ctx.Err(); err != nil {
				a.ctxmgr.Rollback(checkpoint)
				return "", err
			}

			args := map[string]any{}
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					logging.Log.Error("failed to decode tool arguments", "tool", tc.Function.Name, "err", err)
				}
			}

			result := a.tools.Call(ctx, tc.Function.Name, args)

			a.ctxmgr.Append(ctx, types.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			}, nil)
		}

		return a.complete(ctx, nil, maxToolCalls-1, checkpoint)
	}

	content := contentToString(msg.Content)
	a.ctxmgr.Append(ctx, types.Message{Role: "assistant", Content: content}, &usage)
	logging.Log.Info("Seline reply", "content", content)
	return content, nil
}

// contentToString normalizes string-or-list content to a plain string.
func contentToString(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if content == nil {
		return ""
	}
	b, _ := json.Marshal(content)
	return string(b)
}

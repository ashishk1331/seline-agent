// Package types holds the shared domain types exchanged between the context
// manager, the provider/resolver and the agent loop. Keeping them in a
// dependency-free package breaks what would otherwise be an import cycle
// between contextmgr and provider.
package types

// Message is a single chat message. It models the heterogeneous dicts the
// Python code stored (user/assistant/tool/system, optionally with tool_calls).
// JSON tags use omitempty so the on-disk JSONL stays compatible with existing
// sessions written by the Python implementation.
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"` // string, []ContentPart, or nil
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ContentPart is one element of a structured (list) message content, used by
// some providers and handled by ContextManager.MessagesIron during compaction.
type ContentPart struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

// ToolCall is a function/tool invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type,omitempty"`
	Function ToolCallFunc `json:"function"`
}

// ToolCallFunc carries the tool name and its raw JSON-encoded arguments.
type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string; unmarshalled in the loop
}

// Usage mirrors the OpenAI-compatible usage block. Only TotalTokens is consumed
// by the agent; provider-specific extras are intentionally ignored.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionResponse is the decoded chat-completions response.
type CompletionResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is one completion choice.
type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

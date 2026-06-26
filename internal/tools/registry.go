// Package tools implements the agent's tool registry and the built-in tools.
// It is the Go port of the tools/ package. Python built tool schemas at runtime
// by parsing docstrings; Go has no such reflection, so each tool declares its
// schema explicitly (see Register / RegisterDefaults).
package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/logging"
)

// defaultMaxChars is the default truncation limit for tool results (Python's
// register_tool default of 1000). A value of -1 disables truncation.
const defaultMaxChars = 1000

// StatusUpdater is the minimal surface the registry needs to push a live status
// message while a tool runs. The gateway's Status satisfies it. Declared here
// (rather than importing the gateway) to avoid an import cycle.
type StatusUpdater interface {
	Update(ctx context.Context, message string) error
}

// ToolFunc is a tool's handler. args holds the decoded JSON arguments.
type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

// Param describes a single tool parameter for schema generation.
type Param struct {
	Name        string
	Type        string // "string" | "integer" | "number" | "boolean"
	Description string
}

// Tool is a registered tool: its schema metadata plus its handler.
type Tool struct {
	Name          string
	Description   string
	Params        []Param
	Handler       ToolFunc
	MaxChars      int    // -1 = no truncation
	StatusMessage string // e.g. `Searching for "$query"`; "" = no status update
}

// Registry holds the registered tools and the provider-facing tool schema.
type Registry struct {
	cfg     *config.Config
	status  StatusUpdater
	byName  map[string]Tool
	payload []map[string]any
}

// NewRegistry constructs a registry and registers the built-in tools.
func NewRegistry(cfg *config.Config, status StatusUpdater) *Registry {
	r := &Registry{
		cfg:    cfg,
		status: status,
		byName: make(map[string]Tool),
	}
	r.registerDefaults()
	return r
}

// registerDefaults registers all built-in tools.
func (r *Registry) registerDefaults() {
	r.registerWeb()
	r.registerFiles()
	r.registerShell()
}

// Register adds a tool and appends its schema to the provider payload. It
// mirrors the JSON shape produced by Python's register_tool: an object with
// type "function", and all parameters marked required.
func (r *Registry) Register(t Tool) {
	if t.MaxChars == 0 {
		t.MaxChars = defaultMaxChars
	}
	r.byName[t.Name] = t

	properties := make(map[string]any, len(t.Params))
	required := make([]string, 0, len(t.Params))
	for _, p := range t.Params {
		typ := p.Type
		if typ == "" {
			typ = "string"
		}
		properties[p.Name] = map[string]any{
			"type":        typ,
			"description": p.Description,
		}
		required = append(required, p.Name)
	}

	r.payload = append(r.payload, map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	})
}

// Payload returns the tool schema list sent to the provider in the "tools" key.
func (r *Registry) Payload() []map[string]any {
	return r.payload
}

// Call dispatches a tool by name. It reproduces the Python wrapper: log the
// call, push a status update, run the handler, stringify and truncate.
func (r *Registry) Call(ctx context.Context, name string, args map[string]any) string {
	logging.Log.Info("tool call", "name", name, "args", args)

	tool, ok := r.byName[name]
	if !ok {
		logging.Log.Error("unknown tool requested", "name", name)
		return fmt.Sprintf("Error: unknown tool %q", name)
	}

	if tool.StatusMessage != "" && r.status != nil {
		_ = r.status.Update(ctx, expandArgs(tool.StatusMessage, args))
	}

	result, err := tool.Handler(ctx, args)
	if err != nil {
		logging.Log.Error("tool failed", "name", name, "err", err)
		result = fmt.Sprintf("Error: %v", err)
	}

	if tool.MaxChars > -1 && len(result) > tool.MaxChars {
		result = result[:tool.MaxChars] + "... [truncated]"
	}
	return result
}

// expandArgs substitutes $name placeholders in a status message from args,
// mirroring Python's Template(status_message).safe_substitute(**kwargs).
func expandArgs(tmpl string, args map[string]any) string {
	return os.Expand(tmpl, func(key string) string {
		if v, ok := args[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return "$" + key // leave unknown placeholders intact
	})
}

// argString safely extracts a string argument.
func argString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

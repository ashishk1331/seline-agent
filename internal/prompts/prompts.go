// Package prompts loads the system, compaction and environment-info prompts.
// It is the Go port of prompts/prompt.py. The markdown files are embedded into
// the binary via go:embed instead of being read from disk at runtime.
package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"text/template"

	"github.com/ashishk1331/seline-agent/internal/config"
)

//go:embed system.md soul.md compaction.md
var files embed.FS

// PromptBase builds the prompts from the embedded files and the config.
type PromptBase struct {
	cfg *config.Config
}

// New returns a PromptBase bound to cfg.
func New(cfg *config.Config) *PromptBase {
	return &PromptBase{cfg: cfg}
}

// GetSystemPrompt returns the system prompt followed by the environment info.
func (p *PromptBase) GetSystemPrompt() string {
	return p.read("system.md") + "\n\n" + p.GetEnvironmentInformation()
}

// GetCompactionPrompt returns the conversation-compaction instructions.
func (p *PromptBase) GetCompactionPrompt() string {
	return p.read("compaction.md")
}

// GetEnvironmentInformation renders soul.md as a text/template, substituting the
// runtime environment values. Placeholders use {{.NAME}} syntax, e.g.
// {{.AI_PROVIDER}} or {{.MODEL_NAME}}.
func (p *PromptBase) GetEnvironmentInformation() string {
	data := map[string]string{
		"AI_PROVIDER":          p.cfg.AIProvider,
		"MODEL_NAME":           p.cfg.ModelName,
		"MAX_TOKENS":           strconv.Itoa(p.cfg.MaxTokens),
		"MAX_CONTEXT_TOKENS":   strconv.Itoa(p.cfg.MaxContextTokens),
		"COMPACTION_THRESHOLD": fmt.Sprintf("%d%%", int(p.cfg.CompactionThreshold*100+0.5)),
		"WORKSPACE_DIR":        p.cfg.WorkspaceDir,
		"DATE":                 p.cfg.Date,
		"TIME":                 p.cfg.Time,
		"PLATFORM":             p.cfg.Platform,
		"GO_VERSION":           p.cfg.GoVersion,
		"TIMEZONE":             p.cfg.Timezone,
		"LOCALE":               p.cfg.Locale,
		"MAX_TOOL_CALLS":       strconv.Itoa(p.cfg.MaxToolCalls),
	}
	return p.render("soul.md", data)
}

// read returns the contents of an embedded prompt file.
func (p *PromptBase) read(name string) string {
	b, err := files.ReadFile(name)
	if err != nil {
		// Embedded files are compiled in; this should never happen.
		return ""
	}
	return string(b)
}

// render parses the named embedded file as a text/template and executes it with
// data. Missing keys render as an empty string. On any parse/execute error it
// falls back to the raw file contents so a malformed template never blanks the
// prompt.
func (p *PromptBase) render(name string, data map[string]string) string {
	raw := p.read(name)
	tmpl, err := template.New(name).Option("missingkey=zero").Parse(raw)
	if err != nil {
		return raw
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return raw
	}
	return buf.String()
}

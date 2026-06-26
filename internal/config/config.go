// Package config loads and validates runtime configuration from the
// environment. It is the Go port of config.py. Unlike the Python version it is
// not an import-time singleton: Load() is called explicitly from main and the
// resulting *Config is injected into every component.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ashishk1331/seline-agent/internal/logging"
)

// allowlistPattern validates the comma-separated @username allowlist.
var allowlistPattern = regexp.MustCompile(
	`^(@[a-zA-Z][a-zA-Z0-9_]{4,31})(,@[a-zA-Z][a-zA-Z0-9_]{4,31})*$`,
)

// providerURLs maps a provider name to its default chat-completions endpoint.
var providerURLs = map[string]string{
	"OPENROUTER": "https://openrouter.ai/api/v1/chat/completions",
	"SARVAM":     "https://api.sarvam.ai/v1/chat/completions",
	"NVIDIA":     "https://integrate.api.nvidia.com/v1/chat/completions",
	"CEREBRAS":   "https://api.cerebras.ai/v1/chat/completions",
}

// Config holds all resolved settings.
type Config struct {
	// Required
	AIProvider               string
	TinyfishAPIKey           string
	TelegramBotToken         string
	TelegramAllowlist        string
	TelegramAllowlistCleaned []string
	AIProviderAPIKey         string
	AIProviderLLMURL         string

	// Optional with defaults
	ModelName               string
	MaxTokens               int
	MaxContextTokens        int
	Temperature             float64
	TinyfishSearchURL       string
	TinyfishFetchURL        string
	CompactionThreshold     float64
	CompactionRecentN       int
	MaxToolCalls            int
	MessageDebounceDelay    float64
	MessageDebounceJitter   float64
	MessageDebounceMaxDelay float64

	// Paths
	WorkspaceDir string

	// Environment / runtime details
	Date      string
	Time      string
	Platform  string
	GoVersion string
	Timezone  string
	Locale    string
}

// Load reads, validates and returns the configuration, or an error if any
// required variable is missing or invalid.
func Load() (*Config, error) {
	c := &Config{}

	var err error
	if c.AIProvider, err = required("AI_PROVIDER", ""); err != nil {
		return nil, err
	}
	if c.TinyfishAPIKey, err = required("TINYFISH_API_KEY", ""); err != nil {
		return nil, err
	}
	if c.TelegramBotToken, err = required("TELEGRAM_BOT_TOKEN", ""); err != nil {
		return nil, err
	}
	if c.TelegramAllowlist, err = required(
		"TELEGRAM_ALLOWLIST",
		"Add comma-separated Telegram usernames to allow access to Seline. Example: @john_doe,@jane_doe",
	); err != nil {
		return nil, err
	}
	if c.TelegramAllowlistCleaned, err = sanitizeAllowlist(c.TelegramAllowlist); err != nil {
		return nil, err
	}

	if _, ok := providerURLs[c.AIProvider]; !ok {
		return nil, fmt.Errorf(
			"pick 'OPENROUTER', 'SARVAM', 'NVIDIA' or 'CEREBRAS' for AI_PROVIDER. Found %s instead", c.AIProvider,
		)
	}

	if c.AIProviderAPIKey, err = required(c.AIProvider+"_API_KEY", ""); err != nil {
		return nil, err
	}
	c.AIProviderLLMURL = getenvOr(c.AIProvider+"_URL", providerURLs[c.AIProvider])

	// Optional with defaults
	c.ModelName = getenvOr("MODEL_NAME", "z-ai/glm-4.5-air:free")
	c.MaxTokens = getenvInt("MAX_TOKENS", 1000)
	c.MaxContextTokens = getenvInt("MAX_CONTEXT_TOKENS", 131000)
	c.Temperature = getenvFloat("TEMPERATURE", 0.7)
	c.TinyfishSearchURL = getenvOr("TINYFISH_SEARCH_URL", "https://api.search.tinyfish.ai")
	c.TinyfishFetchURL = getenvOr("TINYFISH_FETCH_URL", "https://api.fetch.tinyfish.ai")
	c.CompactionThreshold = getenvFloat("COMPACTION_THRESHOLD", 0.9)
	c.CompactionRecentN = getenvInt("COMPACTION_RECENT_N", 5)
	c.MaxToolCalls = getenvInt("MAX_TOOL_CALLS", 5)
	c.MessageDebounceDelay = getenvFloat("MESSAGE_DEBOUNCE_DELAY", 1.0)
	c.MessageDebounceJitter = getenvFloat("MESSAGE_DEBOUNCE_JITTER", 0.3)
	c.MessageDebounceMaxDelay = getenvFloat("MESSAGE_DEBOUNCE_MAX_DELAY", 2.0)

	// Paths: note the Python code always appends ".workspace", even when
	// WORKSPACE_DIR is set. Preserve that behavior.
	base := os.Getenv("WORKSPACE_DIR")
	if base == "" {
		if home, herr := os.UserHomeDir(); herr == nil {
			base = home
		}
	}
	c.WorkspaceDir = filepath.Join(base, ".workspace")

	// Environment / runtime details
	now := time.Now()
	c.Date = now.Format("2006-01-02")
	c.Time = now.Format("15:04:05")
	c.Platform = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
	c.GoVersion = runtime.Version()
	zone, _ := now.Zone()
	c.Timezone = zone
	c.Locale = detectLocale()

	return c, nil
}

// Print renders a startup table summarizing the active configuration.
func (c *Config) Print() {
	pct := fmt.Sprintf("%d%%", int(c.CompactionThreshold*100+0.5))
	delay := fmt.Sprintf("[%gs, %gs] (step=%gs)",
		c.MessageDebounceDelay, c.MessageDebounceMaxDelay, c.MessageDebounceJitter)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))).
		Headers("Key", "Value").
		Row("Provider", c.AIProvider).
		Row("Provider LLM Url", c.AIProviderLLMURL).
		Row("Model Name", c.ModelName).
		Row("Max Tokens", strconv.Itoa(c.MaxTokens)).
		Row("Context Window Size", strconv.Itoa(c.MaxContextTokens)).
		Row("Workspace Dir", c.WorkspaceDir).
		Row("Compaction Threshold", pct).
		Row("Message Delay", delay).
		Row("Telegram Allowlist", strings.Join(c.TelegramAllowlistCleaned, ", ")).
		Row("Date", c.Date).
		Row("Time", c.Time).
		Row("Platform", c.Platform).
		Row("Go Version", c.GoVersion).
		Row("Timezone", c.Timezone).
		Row("Locale", c.Locale)

	fmt.Println(t)
}

func required(key, message string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		if message != "" {
			return "", fmt.Errorf("%s", message)
		}
		return "", fmt.Errorf("environment variable %q is required but not set", key)
	}
	return v, nil
}

func sanitizeAllowlist(allowlist string) ([]string, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(allowlist), " ", "")
	if !allowlistPattern.MatchString(cleaned) {
		return nil, fmt.Errorf(
			"invalid TELEGRAM_ALLOWLIST format. Expected: @john_doe,@jane_doe. Got: %s", cleaned,
		)
	}
	return strings.Split(cleaned, ","), nil
}

func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
		logging.Log.Warn("invalid int env var, using default", "key", key, "value", v, "default", def)
	}
	return def
}

func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
		logging.Log.Warn("invalid float env var, using default", "key", key, "value", v, "default", def)
	}
	return def
}

// detectLocale reads the POSIX locale env vars. Go has no stdlib locale.
func detectLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			// strip encoding suffix (e.g. en_US.UTF-8 -> en_US)
			if i := strings.IndexByte(v, '.'); i >= 0 {
				v = v[:i]
			}
			return v
		}
	}
	return "unknown"
}

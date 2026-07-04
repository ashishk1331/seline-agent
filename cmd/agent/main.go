// Command agent is the Seline Telegram agent entrypoint. Go port of main.py +
// __init__.py, with explicit dependency wiring (no import-time singletons) and
// graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/contextmgr"
	"github.com/ashishk1331/seline-agent/internal/gateway"
	"github.com/ashishk1331/seline-agent/internal/llm"
	"github.com/ashishk1331/seline-agent/internal/logging"
	"github.com/ashishk1331/seline-agent/internal/prompts"
	"github.com/ashishk1331/seline-agent/internal/provider"
	"github.com/ashishk1331/seline-agent/internal/tools"
)

func main() {
	// Load .env if present (missing file is not an error — env may be injected).
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logging.Log.Fatal("configuration error", "err", err)
	}
	cfg.Print()

	// Wire dependencies explicitly. Construction order resolves the
	// status -> registry -> resolver -> contextmgr -> agent -> gateway chain.
	prompt := prompts.New(cfg)
	status := gateway.NewStatus()
	registry := tools.NewRegistry(cfg, status)
	resolver := provider.NewResolver(cfg, registry.Payload())

	ctxmgr, err := contextmgr.New(cfg, prompt, resolver)
	if err != nil {
		logging.Log.Fatal("failed to initialize context manager", "err", err)
	}

	agent := llm.NewAgent(ctxmgr, resolver, registry, cfg)
	gw := gateway.New(cfg, agent, ctxmgr, status, resolver)

	// Cancel the root context on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := gw.Run(ctx); err != nil {
		logging.Log.Fatal("gateway error", "err", err)
	}
}

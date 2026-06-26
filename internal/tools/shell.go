package tools

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/ashishk1331/seline-agent/internal/logging"
)

// commandTimeout matches the Python subprocess timeout of 30s.
const commandTimeout = 30 * time.Second

// registerShell registers run_command.
func (r *Registry) registerShell() {
	r.Register(Tool{
		Name:          "run_command",
		Description:   "Run a shell command and return its output.",
		Params:        []Param{{Name: "command", Type: "string", Description: "The shell command to run."}},
		StatusMessage: "Running $command",
		Handler:       runCommand,
	})
}

func runCommand(ctx context.Context, args map[string]any) (string, error) {
	command := argString(args, "command")

	cctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logging.Log.Error("command failed", "command", command, "stderr", stderr.String(), "err", err)
	} else {
		logging.Log.Info("command succeeded", "command", command)
	}

	if out := stdout.String(); out != "" {
		return out, nil
	}
	return stderr.String(), nil
}

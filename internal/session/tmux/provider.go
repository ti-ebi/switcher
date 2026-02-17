package tmux

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type commandRunner func(ctx context.Context, command string, args ...string) ([]byte, error)

// Provider lists tmux session names.
type Provider struct {
	runner commandRunner
}

// NewProvider creates a provider that runs real tmux commands.
func NewProvider() Provider {
	return NewProviderWithRunner(runCommand)
}

// NewProviderWithRunner creates a provider with a custom command runner.
func NewProviderWithRunner(runner commandRunner) Provider {
	if runner == nil {
		runner = runCommand
	}

	return Provider{runner: runner}
}

// List returns available tmux session names.
func (p Provider) List(ctx context.Context) ([]string, error) {
	output, err := p.runner(ctx, "tmux", "list-sessions", "-F", "#{session_name}")
	if err != nil {
		if isNoServerError(output) {
			return []string{}, nil
		}

		outputText := strings.TrimSpace(string(output))
		if outputText != "" {
			return nil, fmt.Errorf("list tmux sessions: %w: %s", err, outputText)
		}

		return nil, fmt.Errorf("list tmux sessions: %w", err)
	}

	return parseSessionNames(output), nil
}

func parseSessionNames(output []byte) []string {
	lines := strings.Split(string(output), "\n")
	sessions := make([]string, 0, len(lines))

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}

		sessions = append(sessions, name)
	}

	return sessions
}

func isNoServerError(output []byte) bool {
	message := strings.ToLower(string(output))
	if strings.Contains(message, "no server running") {
		return true
	}

	return strings.Contains(message, "failed to connect to server")
}

func runCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	return cmd.CombinedOutput()
}

func runInteractiveCommand(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	var stderrBuffer bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuffer)

	err := cmd.Run()
	return stderrBuffer.Bytes(), err
}

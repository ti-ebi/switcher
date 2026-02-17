package tmux

import (
	"context"
	"fmt"
	"strings"
)

var attachCommandRunner commandRunner = runInteractiveCommand

// AttachSession attaches to one tmux session by name.
func AttachSession(ctx context.Context, sessionName string) error {
	return attachSession(ctx, sessionName, attachCommandRunner)
}

func attachSession(ctx context.Context, sessionName string, runner commandRunner) error {
	name := strings.TrimSpace(sessionName)
	if name == "" {
		return fmt.Errorf("attach tmux session: session name is empty")
	}

	output, err := runner(ctx, "tmux", "attach-session", "-t", name)
	if err != nil {
		outputText := strings.TrimSpace(string(output))
		if outputText != "" {
			return fmt.Errorf("attach tmux session: %w: %s", err, outputText)
		}

		return fmt.Errorf("attach tmux session: %w", err)
	}

	return nil
}

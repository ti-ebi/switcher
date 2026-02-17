package tmux

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type commandRunner func(ctx context.Context, command string, args ...string) ([]byte, error)

// Provider lists tmux session names.
type Provider struct {
	runner commandRunner
}

// SessionDetails represents metadata for one tmux session.
type SessionDetails struct {
	Name            string
	WindowCount     int
	AttachedClients int
	CreatedAt       time.Time
	Preview         string
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

// Create creates a new detached tmux session.
func (p Provider) Create(ctx context.Context, sessionName string) error {
	name := strings.TrimSpace(sessionName)
	if name == "" {
		return fmt.Errorf("create tmux session: session name is empty")
	}

	output, err := p.runner(ctx, "tmux", "new-session", "-d", "-s", name)
	if err != nil {
		outputText := strings.TrimSpace(string(output))
		if outputText != "" {
			return fmt.Errorf("create tmux session: %w: %s", err, outputText)
		}

		return fmt.Errorf("create tmux session: %w", err)
	}

	return nil
}

// ListDetails returns metadata for all tmux sessions.
func (p Provider) ListDetails(ctx context.Context) ([]SessionDetails, error) {
	output, err := p.runner(
		ctx,
		"tmux",
		"list-sessions",
		"-F",
		"#{session_name}\t#{session_windows}\t#{session_attached}\t#{session_created}",
	)
	if err != nil {
		if isNoServerError(output) {
			return []SessionDetails{}, nil
		}

		outputText := strings.TrimSpace(string(output))
		if outputText != "" {
			return nil, fmt.Errorf("list tmux session details: %w: %s", err, outputText)
		}

		return nil, fmt.Errorf("list tmux session details: %w", err)
	}

	details, parseErr := parseSessionDetails(output)
	if parseErr != nil {
		return nil, fmt.Errorf("parse tmux session details: %w", parseErr)
	}

	enrichSessionDetailsWithPreviews(ctx, p.runner, details)
	return details, nil
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

func parseSessionDetails(output []byte) ([]SessionDetails, error) {
	lines := strings.Split(string(output), "\n")
	details := make([]SessionDetails, 0, len(lines))

	for lineNumber, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		fields := strings.Split(trimmed, "\t")
		if len(fields) != 4 {
			return nil, fmt.Errorf("line %d: expected 4 fields, got %d", lineNumber+1, len(fields))
		}

		windowCount, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: parse window count %q: %w", lineNumber+1, fields[1], err)
		}

		attachedClients, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: parse attached clients %q: %w", lineNumber+1, fields[2], err)
		}

		createdUnix, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: parse created timestamp %q: %w", lineNumber+1, fields[3], err)
		}

		details = append(details, SessionDetails{
			Name:            fields[0],
			WindowCount:     windowCount,
			AttachedClients: attachedClients,
			CreatedAt:       time.Unix(createdUnix, 0).UTC(),
		})
	}

	return details, nil
}

func enrichSessionDetailsWithPreviews(ctx context.Context, runner commandRunner, details []SessionDetails) {
	for index := range details {
		preview, err := captureSessionPreview(ctx, runner, details[index].Name, 30)
		if err != nil {
			continue
		}

		details[index].Preview = preview
	}
}

func captureSessionPreview(
	ctx context.Context,
	runner commandRunner,
	sessionName string,
	lines int,
) (string, error) {
	if lines <= 0 {
		lines = 30
	}

	output, err := runner(
		ctx,
		"tmux",
		"capture-pane",
		"-p",
		"-J",
		"-t",
		sessionName,
		"-S",
		fmt.Sprintf("-%d", lines),
		"-E",
		"-",
	)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(output), "\n"), nil
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

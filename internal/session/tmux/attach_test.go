package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAttachSessionUsesAttachRunner(t *testing.T) {
	originalRunner := attachCommandRunner
	t.Cleanup(func() {
		attachCommandRunner = originalRunner
	})

	var gotCommand string
	var gotArgs []string

	attachCommandRunner = func(_ context.Context, command string, args ...string) ([]byte, error) {
		gotCommand = command
		gotArgs = append([]string(nil), args...)
		return nil, nil
	}

	err := AttachSession(context.Background(), "dev")
	if err != nil {
		t.Fatalf("AttachSession returned error: %v", err)
	}

	if gotCommand != "tmux" {
		t.Fatalf("command = %q, want %q", gotCommand, "tmux")
	}

	wantArgs := []string{"attach-session", "-t", "dev"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("len(args) = %d, want %d", len(gotArgs), len(wantArgs))
	}

	for index := range wantArgs {
		if gotArgs[index] != wantArgs[index] {
			t.Fatalf("args[%d] = %q, want %q", index, gotArgs[index], wantArgs[index])
		}
	}
}

func TestAttachSessionRunsTmuxAttachCommand(t *testing.T) {
	t.Parallel()

	var gotCommand string
	var gotArgs []string

	err := attachSession(
		context.Background(),
		"dev",
		func(_ context.Context, command string, args ...string) ([]byte, error) {
			gotCommand = command
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("attachSession returned error: %v", err)
	}

	if gotCommand != "tmux" {
		t.Fatalf("command = %q, want %q", gotCommand, "tmux")
	}

	if len(gotArgs) != 3 {
		t.Fatalf("len(args) = %d, want 3", len(gotArgs))
	}

	wantArgs := []string{"attach-session", "-t", "dev"}
	for index := range wantArgs {
		if gotArgs[index] != wantArgs[index] {
			t.Fatalf("args[%d] = %q, want %q", index, gotArgs[index], wantArgs[index])
		}
	}
}

func TestAttachSessionReturnsErrorForEmptySessionName(t *testing.T) {
	t.Parallel()

	err := attachSession(context.Background(), "", func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("attachSession returned nil error, want non-nil")
	}
}

func TestAttachSessionReturnsWrappedError(t *testing.T) {
	t.Parallel()

	err := attachSession(context.Background(), "dev", func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("can't find session: dev\n"), errors.New("exit status 1")
	})
	if err == nil {
		t.Fatal("attachSession returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "attach tmux session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "attach tmux session")
	}

	if !strings.Contains(err.Error(), "can't find session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "can't find session")
	}
}

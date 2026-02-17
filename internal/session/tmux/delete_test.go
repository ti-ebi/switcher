package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProviderDeleteRunsTmuxKillSessionCommand(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "tmux" {
			t.Fatalf("command = %q, want %q", command, "tmux")
		}

		wantArgs := []string{"kill-session", "-t", "dev"}
		if len(args) != len(wantArgs) {
			t.Fatalf("len(args) = %d, want %d", len(args), len(wantArgs))
		}

		for index := range wantArgs {
			if args[index] != wantArgs[index] {
				t.Fatalf("args[%d] = %q, want %q", index, args[index], wantArgs[index])
			}
		}

		return nil, nil
	})

	err := provider.Delete(context.Background(), "dev")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestProviderDeleteReturnsErrorForEmptyName(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, nil
	})

	err := provider.Delete(context.Background(), " ")
	if err == nil {
		t.Fatal("Delete returned nil error, want non-nil")
	}
}

func TestProviderDeleteReturnsWrappedError(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("can't find session\n"), errors.New("exit status 1")
	})

	err := provider.Delete(context.Background(), "dev")
	if err == nil {
		t.Fatal("Delete returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "delete tmux session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "delete tmux session")
	}

	if !strings.Contains(err.Error(), "can't find session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "can't find session")
	}
}

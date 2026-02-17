package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProviderCreateRunsTmuxNewSessionCommand(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "tmux" {
			t.Fatalf("command = %q, want %q", command, "tmux")
		}

		wantArgs := []string{"new-session", "-d", "-s", "dev"}
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

	err := provider.Create(context.Background(), "dev")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
}

func TestProviderCreateReturnsErrorForEmptyName(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, nil
	})

	err := provider.Create(context.Background(), " ")
	if err == nil {
		t.Fatal("Create returned nil error, want non-nil")
	}
}

func TestProviderCreateReturnsWrappedError(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("duplicate session: dev\n"), errors.New("exit status 1")
	})

	err := provider.Create(context.Background(), "dev")
	if err == nil {
		t.Fatal("Create returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "create tmux session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "create tmux session")
	}

	if !strings.Contains(err.Error(), "duplicate session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "duplicate session")
	}
}

package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProviderRenameRunsTmuxRenameSessionCommand(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, command string, args ...string) ([]byte, error) {
		if command != "tmux" {
			t.Fatalf("command = %q, want %q", command, "tmux")
		}

		wantArgs := []string{"rename-session", "-t", "dev", "api"}
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

	err := provider.Rename(context.Background(), "dev", "api")
	if err != nil {
		t.Fatalf("Rename returned error: %v", err)
	}
}

func TestProviderRenameReturnsErrorForEmptyNames(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, nil
	})

	err := provider.Rename(context.Background(), " ", "api")
	if err == nil {
		t.Fatal("Rename returned nil error for empty source name, want non-nil")
	}

	err = provider.Rename(context.Background(), "dev", " ")
	if err == nil {
		t.Fatal("Rename returned nil error for empty target name, want non-nil")
	}
}

func TestProviderRenameReturnsWrappedError(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("duplicate session: api\n"), errors.New("exit status 1")
	})

	err := provider.Rename(context.Background(), "dev", "api")
	if err == nil {
		t.Fatal("Rename returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "rename tmux session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "rename tmux session")
	}

	if !strings.Contains(err.Error(), "duplicate session") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "duplicate session")
	}
}

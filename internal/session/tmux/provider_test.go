package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProviderListParsesSessionNames(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("dev\napi\nlogs\n"), nil
	})

	sessions, err := provider.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("len(sessions) = %d, want 3", len(sessions))
	}

	want := []string{"dev", "api", "logs"}
	for index := range want {
		if sessions[index] != want[index] {
			t.Fatalf("sessions[%d] = %q, want %q", index, sessions[index], want[index])
		}
	}
}

func TestProviderListReturnsEmptyWhenTmuxServerIsNotRunning(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("no server running on /tmp/tmux-501/default\n"), errors.New("exit status 1")
	})

	sessions, err := provider.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestProviderListReturnsWrappedErrorForUnexpectedCommandFailure(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("permission denied\n"), errors.New("exit status 1")
	})

	_, err := provider.List(context.Background())
	if err == nil {
		t.Fatal("List returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "list tmux sessions") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "list tmux sessions")
	}

	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "permission denied")
	}
}

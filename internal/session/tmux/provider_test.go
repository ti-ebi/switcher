package tmux

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestProviderListDetailsParsesFields(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("dev\t3\t1\t1700000000\napi\t2\t0\t1700000600\n"), nil
	})

	details, err := provider.ListDetails(context.Background())
	if err != nil {
		t.Fatalf("ListDetails returned error: %v", err)
	}

	if len(details) != 2 {
		t.Fatalf("len(details) = %d, want 2", len(details))
	}

	if details[0].Name != "dev" {
		t.Fatalf("details[0].Name = %q, want %q", details[0].Name, "dev")
	}

	if details[0].WindowCount != 3 {
		t.Fatalf("details[0].WindowCount = %d, want 3", details[0].WindowCount)
	}

	if details[0].AttachedClients != 1 {
		t.Fatalf("details[0].AttachedClients = %d, want 1", details[0].AttachedClients)
	}

	wantCreatedAt := time.Unix(1700000000, 0).UTC()
	if !details[0].CreatedAt.Equal(wantCreatedAt) {
		t.Fatalf("details[0].CreatedAt = %v, want %v", details[0].CreatedAt, wantCreatedAt)
	}
}

func TestProviderListDetailsReturnsParseError(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("dev\tinvalid\t1\t1700000000\n"), nil
	})

	_, err := provider.ListDetails(context.Background())
	if err == nil {
		t.Fatal("ListDetails returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "parse tmux session details") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "parse tmux session details")
	}
}

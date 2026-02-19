package tmux

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistenceBootstrapInitializesTPMAndConfig(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	calls := make([]commandCall, 0, 2)
	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		command string,
		args ...string,
	) ([]byte, error) {
		calls = append(calls, commandCall{
			command: command,
			args:    append([]string(nil), args...),
		})
		return nil, nil
	})

	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2", len(calls))
	}

	if calls[0].command != "git" {
		t.Fatalf("calls[0].command = %q, want %q", calls[0].command, "git")
	}

	wantCloneArgs := []string{
		"clone",
		"https://github.com/tmux-plugins/tpm",
		filepath.Join(homeDir, ".tmux", "plugins", "tpm"),
	}
	assertArgsEqual(t, calls[0].args, wantCloneArgs)

	if calls[1].command != "bash" {
		t.Fatalf("calls[1].command = %q, want %q", calls[1].command, "bash")
	}

	wantInstallArgs := []string{
		filepath.Join(homeDir, ".tmux", "plugins", "tpm", "bin", "install_plugins"),
	}
	assertArgsEqual(t, calls[1].args, wantInstallArgs)

	configPath := filepath.Join(homeDir, ".tmux.conf")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read tmux config: %v", err)
	}

	config := string(configData)
	for _, snippet := range []string{
		persistenceConfigStartMarker,
		"tmux-plugins/tpm",
		"tmux-plugins/tmux-resurrect",
		"tmux-plugins/tmux-continuum",
		"@continuum-restore 'on'",
		"run '~/.tmux/plugins/tpm/tpm'",
		persistenceConfigEndMarker,
	} {
		if !strings.Contains(config, snippet) {
			t.Fatalf("config must contain %q\nconfig:\n%s", snippet, config)
		}
	}
}

func TestPersistenceBootstrapSkipsTPMCloneWhenDirectoryExists(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	tpmDir := filepath.Join(homeDir, ".tmux", "plugins", "tpm")
	if err := os.MkdirAll(tpmDir, 0o755); err != nil {
		t.Fatalf("create tpm dir: %v", err)
	}

	calls := make([]commandCall, 0, 1)
	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		command string,
		args ...string,
	) ([]byte, error) {
		calls = append(calls, commandCall{
			command: command,
			args:    append([]string(nil), args...),
		})
		return nil, nil
	})

	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}

	if calls[0].command != "bash" {
		t.Fatalf("calls[0].command = %q, want %q", calls[0].command, "bash")
	}
}

func TestPersistenceBootstrapReplacesManagedConfigBlock(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	tpmDir := filepath.Join(homeDir, ".tmux", "plugins", "tpm")
	if err := os.MkdirAll(tpmDir, 0o755); err != nil {
		t.Fatalf("create tpm dir: %v", err)
	}

	configPath := filepath.Join(homeDir, ".tmux.conf")
	initialConfig := strings.Join([]string{
		"set -g mouse on",
		persistenceConfigStartMarker,
		"set -g @plugin 'legacy/plugin'",
		persistenceConfigEndMarker,
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		_ string,
		_ ...string,
	) ([]byte, error) {
		return nil, nil
	})

	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read tmux config: %v", err)
	}

	config := string(configData)
	if strings.Count(config, persistenceConfigStartMarker) != 1 {
		t.Fatalf("managed block count = %d, want 1", strings.Count(config, persistenceConfigStartMarker))
	}

	if strings.Contains(config, "legacy/plugin") {
		t.Fatalf("config still contains legacy block content:\n%s", config)
	}
}

func TestPersistenceBootstrapKeepsExistingEquivalentConfig(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	tpmDir := filepath.Join(homeDir, ".tmux", "plugins", "tpm")
	if err := os.MkdirAll(tpmDir, 0o755); err != nil {
		t.Fatalf("create tpm dir: %v", err)
	}

	configPath := filepath.Join(homeDir, ".tmux.conf")
	initialConfig := strings.Join([]string{
		"set -g @plugin 'tmux-plugins/tpm'",
		"set -g @plugin 'tmux-plugins/tmux-resurrect'",
		"set -g @plugin 'tmux-plugins/tmux-continuum'",
		"set -g @continuum-restore 'on'",
		"run '~/.tmux/plugins/tpm/tpm'",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		_ string,
		_ ...string,
	) ([]byte, error) {
		return nil, nil
	})

	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read tmux config: %v", err)
	}

	config := string(configData)
	if config != initialConfig {
		t.Fatalf("config changed unexpectedly\nwant:\n%s\ngot:\n%s", initialConfig, config)
	}

	if strings.Contains(config, persistenceConfigStartMarker) {
		t.Fatalf("config must not contain managed block marker:\n%s", config)
	}
}

func TestPersistenceBootstrapDoesNotTreatCommentedDirectivesAsConfigured(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	tpmDir := filepath.Join(homeDir, ".tmux", "plugins", "tpm")
	if err := os.MkdirAll(tpmDir, 0o755); err != nil {
		t.Fatalf("create tpm dir: %v", err)
	}

	configPath := filepath.Join(homeDir, ".tmux.conf")
	commentOnlyConfig := strings.Join([]string{
		"# set -g @plugin 'tmux-plugins/tpm'",
		"# set -g @plugin 'tmux-plugins/tmux-resurrect'",
		"# set -g @plugin 'tmux-plugins/tmux-continuum'",
		"# set -g @continuum-restore 'on'",
		"# run '~/.tmux/plugins/tpm/tpm'",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(commentOnlyConfig), 0o644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		_ string,
		_ ...string,
	) ([]byte, error) {
		return nil, nil
	})

	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read tmux config: %v", err)
	}

	config := string(configData)
	if !strings.Contains(config, persistenceConfigStartMarker) {
		t.Fatalf("config must contain managed block start marker:\n%s", config)
	}
}

func TestPersistenceBootstrapNormalizesMultipleManagedBlocks(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	tpmDir := filepath.Join(homeDir, ".tmux", "plugins", "tpm")
	if err := os.MkdirAll(tpmDir, 0o755); err != nil {
		t.Fatalf("create tpm dir: %v", err)
	}

	configPath := filepath.Join(homeDir, ".tmux.conf")
	initialConfig := strings.Join([]string{
		"set -g mouse on",
		"",
		managedPersistenceConfigBlock(),
		"set -g status on",
		"",
		managedPersistenceConfigBlock(),
	}, "\n")
	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		_ string,
		_ ...string,
	) ([]byte, error) {
		return nil, nil
	})

	if err := manager.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read tmux config: %v", err)
	}

	config := string(configData)
	if strings.Count(config, persistenceConfigStartMarker) != 1 {
		t.Fatalf("managed block count = %d, want 1", strings.Count(config, persistenceConfigStartMarker))
	}

	if !strings.Contains(config, "set -g mouse on") {
		t.Fatalf("config must keep pre-existing settings:\n%s", config)
	}

	if !strings.Contains(config, "set -g status on") {
		t.Fatalf("config must keep interleaved settings:\n%s", config)
	}
}

func TestPersistenceSaveRunsResurrectSaveScript(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	calls := make([]commandCall, 0, 1)
	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		command string,
		args ...string,
	) ([]byte, error) {
		calls = append(calls, commandCall{
			command: command,
			args:    append([]string(nil), args...),
		})
		return nil, nil
	})

	if err := manager.Save(context.Background()); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}

	if calls[0].command != "bash" {
		t.Fatalf("calls[0].command = %q, want %q", calls[0].command, "bash")
	}

	wantArgs := []string{
		filepath.Join(homeDir, ".tmux", "plugins", "tmux-resurrect", "scripts", "save.sh"),
	}
	assertArgsEqual(t, calls[0].args, wantArgs)
}

func TestPersistenceRestoreRunsResurrectRestoreScript(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	calls := make([]commandCall, 0, 1)
	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		command string,
		args ...string,
	) ([]byte, error) {
		calls = append(calls, commandCall{
			command: command,
			args:    append([]string(nil), args...),
		})
		return nil, nil
	})

	if err := manager.Restore(context.Background()); err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}

	if calls[0].command != "bash" {
		t.Fatalf("calls[0].command = %q, want %q", calls[0].command, "bash")
	}

	wantArgs := []string{
		filepath.Join(homeDir, ".tmux", "plugins", "tmux-resurrect", "scripts", "restore.sh"),
	}
	assertArgsEqual(t, calls[0].args, wantArgs)
}

func TestPersistenceBootstrapWrapsTPMCloneError(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	manager := newPersistenceManagerForTest(homeDir, func(
		_ context.Context,
		command string,
		_ ...string,
	) ([]byte, error) {
		if command == "git" {
			return []byte("fatal: network timeout\n"), errors.New("exit status 128")
		}

		return nil, nil
	})

	err := manager.Bootstrap(context.Background())
	if err == nil {
		t.Fatal("Bootstrap returned nil error, want non-nil")
	}

	if !strings.Contains(err.Error(), "clone tpm repository") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "clone tpm repository")
	}

	if !strings.Contains(err.Error(), "network timeout") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "network timeout")
	}
}

func TestPersistenceSaveAddsDefaultTimeoutWhenContextHasNoDeadline(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	var hasDeadline bool
	manager := newPersistenceManagerForTest(homeDir, func(
		ctx context.Context,
		_ string,
		_ ...string,
	) ([]byte, error) {
		_, hasDeadline = ctx.Deadline()
		return nil, nil
	})

	if err := manager.Save(context.Background()); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if !hasDeadline {
		t.Fatal("runner context must include deadline")
	}
}

type commandCall struct {
	command string
	args    []string
}

func newPersistenceManagerForTest(homeDir string, runner commandRunner) PersistenceManager {
	return PersistenceManager{
		runner: runner,
		resolveHome: func() (string, error) {
			return homeDir, nil
		},
		readFile: os.ReadFile,
		writeFile: func(path string, data []byte, perm os.FileMode) error {
			return os.WriteFile(path, data, perm)
		},
		mkdirAll: os.MkdirAll,
		stat:     os.Stat,
	}
}

func assertArgsEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(args) = %d, want %d", len(got), len(want))
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("args[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

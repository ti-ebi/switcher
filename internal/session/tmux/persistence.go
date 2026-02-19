package tmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	persistenceConfigStartMarker = "# >>> switcher tmux persistence >>>"
	persistenceConfigEndMarker   = "# <<< switcher tmux persistence <<<"

	defaultTPMCloneTimeout      = 20 * time.Second
	defaultPluginInstallTimeout = 20 * time.Second
	defaultSaveTimeout          = 5 * time.Second
	defaultRestoreTimeout       = 10 * time.Second
)

var persistenceConfigBlockPattern = regexp.MustCompile(
	`(?s)` +
		regexp.QuoteMeta(persistenceConfigStartMarker) +
		`.*?` +
		regexp.QuoteMeta(persistenceConfigEndMarker) +
		`\n?`,
)

type homeResolver func() (string, error)

type readFileFunc func(path string) ([]byte, error)

type writeFileFunc func(path string, data []byte, perm os.FileMode) error

type mkdirAllFunc func(path string, perm os.FileMode) error

type statFunc func(name string) (os.FileInfo, error)

// PersistenceManager bootstraps and controls tmux session persistence plugins.
type PersistenceManager struct {
	runner      commandRunner
	resolveHome homeResolver
	readFile    readFileFunc
	writeFile   writeFileFunc
	mkdirAll    mkdirAllFunc
	stat        statFunc
}

// NewPersistenceManager creates a manager with real system dependencies.
func NewPersistenceManager() PersistenceManager {
	return PersistenceManager{
		runner:      runCommand,
		resolveHome: os.UserHomeDir,
		readFile:    os.ReadFile,
		writeFile: func(path string, data []byte, perm os.FileMode) error {
			return os.WriteFile(path, data, perm)
		},
		mkdirAll: os.MkdirAll,
		stat:     os.Stat,
	}
}

// Bootstrap installs TPM, ensures tmux config directives, and installs plugins.
func (m PersistenceManager) Bootstrap(ctx context.Context) error {
	homeDir, err := m.resolveHome()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	if err := m.ensureTPMInstalled(ctx, homeDir); err != nil {
		return err
	}

	if err := m.ensureTmuxConfig(homeDir); err != nil {
		return err
	}

	if err := m.installPlugins(ctx, homeDir); err != nil {
		return err
	}

	return nil
}

// Save writes the current tmux sessions to resurrect snapshot storage.
func (m PersistenceManager) Save(ctx context.Context) error {
	homeDir, err := m.resolveHome()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	commandCtx, cancel := withDefaultTimeout(ctx, defaultSaveTimeout)
	defer cancel()

	return runExternalCommand(
		commandCtx,
		m.runner,
		"save tmux session snapshot",
		"bash",
		resurrectSaveScriptPath(homeDir),
	)
}

// Restore restores tmux sessions from the latest resurrect snapshot.
func (m PersistenceManager) Restore(ctx context.Context) error {
	homeDir, err := m.resolveHome()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	commandCtx, cancel := withDefaultTimeout(ctx, defaultRestoreTimeout)
	defer cancel()

	return runExternalCommand(
		commandCtx,
		m.runner,
		"restore tmux sessions",
		"bash",
		resurrectRestoreScriptPath(homeDir),
	)
}

func (m PersistenceManager) ensureTPMInstalled(ctx context.Context, homeDir string) error {
	tpmPath := tpmPluginManagerPath(homeDir)
	exists, err := pathExists(m.stat, tpmPath)
	if err != nil {
		return fmt.Errorf("check tpm directory: %w", err)
	}

	if exists {
		return nil
	}

	if err := m.mkdirAll(filepath.Dir(tpmPath), 0o755); err != nil {
		return fmt.Errorf("create tpm parent directory: %w", err)
	}

	commandCtx, cancel := withDefaultTimeout(ctx, defaultTPMCloneTimeout)
	defer cancel()

	return runExternalCommand(
		commandCtx,
		m.runner,
		"clone tpm repository",
		"git",
		"clone",
		"https://github.com/tmux-plugins/tpm",
		tpmPath,
	)
}

func (m PersistenceManager) ensureTmuxConfig(homeDir string) error {
	configPath := tmuxConfigPath(homeDir)
	currentConfig, err := m.readFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read tmux config: %w", err)
	}

	updatedConfig, changed := upsertPersistenceConfig(string(currentConfig))
	if !changed {
		return nil
	}

	if err := m.writeFile(configPath, []byte(updatedConfig), 0o644); err != nil {
		return fmt.Errorf("write tmux config: %w", err)
	}

	return nil
}

func (m PersistenceManager) installPlugins(ctx context.Context, homeDir string) error {
	commandCtx, cancel := withDefaultTimeout(ctx, defaultPluginInstallTimeout)
	defer cancel()

	return runExternalCommand(
		commandCtx,
		m.runner,
		"install tmux plugins",
		"bash",
		tpmInstallScriptPath(homeDir),
	)
}

func runExternalCommand(
	ctx context.Context,
	runner commandRunner,
	action string,
	command string,
	args ...string,
) error {
	output, err := runner(ctx, command, args...)
	if err == nil {
		return nil
	}

	outputText := strings.TrimSpace(string(output))
	if outputText != "" {
		return fmt.Errorf("%s: %w: %s", action, err, outputText)
	}

	return fmt.Errorf("%s: %w", action, err)
}

func upsertPersistenceConfig(current string) (string, bool) {
	if persistenceConfigBlockPattern.MatchString(current) {
		withoutManagedBlocks := persistenceConfigBlockPattern.ReplaceAllString(current, "")
		updated := appendPersistenceConfigBlock(withoutManagedBlocks)
		return updated, updated != current
	}

	if hasRequiredPersistenceDirectives(current) {
		return current, false
	}

	return appendPersistenceConfigBlock(current), true
}

func appendPersistenceConfigBlock(current string) string {
	block := managedPersistenceConfigBlock()
	trimmed := strings.TrimRight(current, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return block
	}

	return trimmed + "\n\n" + block
}

func managedPersistenceConfigBlock() string {
	lines := []string{
		persistenceConfigStartMarker,
		"# Managed by switcher.",
		"set -g @plugin 'tmux-plugins/tpm'",
		"set -g @plugin 'tmux-plugins/tmux-resurrect'",
		"set -g @plugin 'tmux-plugins/tmux-continuum'",
		"set -g @continuum-restore 'on'",
		"run '~/.tmux/plugins/tpm/tpm'",
		persistenceConfigEndMarker,
		"",
	}

	return strings.Join(lines, "\n")
}

func hasRequiredPersistenceDirectives(config string) bool {
	activeConfig := nonCommentConfig(config)
	requiredSnippets := []string{
		"tmux-plugins/tpm",
		"tmux-plugins/tmux-resurrect",
		"tmux-plugins/tmux-continuum",
		"@continuum-restore",
		".tmux/plugins/tpm/tpm",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(activeConfig, snippet) {
			return false
		}
	}

	return true
}

func nonCommentConfig(config string) string {
	lines := strings.Split(config, "\n")
	activeLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		activeLines = append(activeLines, trimmed)
	}

	return strings.Join(activeLines, "\n")
}

func withDefaultTimeout(
	ctx context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, timeout)
}

func pathExists(statPath statFunc, path string) (bool, error) {
	_, err := statPath(path)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func tmuxConfigPath(homeDir string) string {
	return filepath.Join(homeDir, ".tmux.conf")
}

func tpmPluginManagerPath(homeDir string) string {
	return filepath.Join(homeDir, ".tmux", "plugins", "tpm")
}

func tpmInstallScriptPath(homeDir string) string {
	return filepath.Join(tpmPluginManagerPath(homeDir), "bin", "install_plugins")
}

func resurrectSaveScriptPath(homeDir string) string {
	return filepath.Join(homeDir, ".tmux", "plugins", "tmux-resurrect", "scripts", "save.sh")
}

func resurrectRestoreScriptPath(homeDir string) string {
	return filepath.Join(homeDir, ".tmux", "plugins", "tmux-resurrect", "scripts", "restore.sh")
}

package main

import (
	"context"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"switcher/internal/session/tmux"
	"switcher/internal/tui"
)

const detailsRefreshInterval = time.Second

func main() {
	ctx := context.Background()

	if err := tmux.EnsureInstalled(); err != nil {
		log.Fatalf(
			"tmux is required to run switcher.\n"+
				"Install tmux and retry.\n"+
				"- macOS (Homebrew): brew install tmux\n"+
				"- Ubuntu/Debian: sudo apt install tmux\n"+
				"- Fedora: sudo dnf install tmux\n"+
				"- Arch Linux: sudo pacman -S tmux\n"+
				"Details: %v",
			err,
		)
	}

	provider := tmux.NewProvider()

	for {
		finalModel, ok := runSwitcherProgram(ctx, provider)
		if !ok || finalModel.IsQuitting() {
			return
		}

		if processManagementRequests(ctx, provider, finalModel) {
			continue
		}

		if !attachSelectedSession(ctx, finalModel) {
			continue
		}
	}
}

func runSwitcherProgram(ctx context.Context, provider tmux.Provider) (tui.Model, bool) {
	sessionNames, err := provider.List(ctx)
	if err != nil {
		log.Fatalf("failed to load sessions: %v", err)
	}

	model := tui.NewModel(toTUISessions(sessionNames)).EnableColor()
	program := tea.NewProgram(model, tea.WithAltScreen())
	detailsCtx, cancelDetails := context.WithCancel(ctx)
	go streamSessionDetails(detailsCtx, provider, program)

	finalModelAny, err := program.Run()
	cancelDetails()
	if err != nil {
		log.Fatalf("switcher failed: %v", err)
	}

	finalModel, ok := finalModelAny.(tui.Model)
	return finalModel, ok
}

func processManagementRequests(ctx context.Context, provider tmux.Provider, model tui.Model) bool {
	if sessionName, requested := model.CreateRequest(); requested {
		if err := provider.Create(ctx, sessionName); err != nil {
			log.Printf("failed to create session %q: %v", sessionName, err)
		}

		return true
	}

	if fromSessionName, toSessionName, requested := model.RenameRequest(); requested {
		if err := provider.Rename(ctx, fromSessionName, toSessionName); err != nil {
			log.Printf(
				"failed to rename session %q to %q: %v",
				fromSessionName,
				toSessionName,
				err,
			)
		}

		return true
	}

	if sessionName, requested := model.DeleteRequest(); requested {
		if err := provider.Delete(ctx, sessionName); err != nil {
			log.Printf("failed to delete session %q: %v", sessionName, err)
		}

		return true
	}

	return false
}

func attachSelectedSession(ctx context.Context, model tui.Model) bool {
	session, selected := model.SelectedSession()
	if !selected {
		return false
	}

	if err := tmux.AttachSession(ctx, session.Name); err != nil {
		log.Printf("failed to attach session: %v", err)
	}

	return true
}

func streamSessionDetails(ctx context.Context, provider tmux.Provider, program *tea.Program) {
	ticker := time.NewTicker(detailsRefreshInterval)
	defer ticker.Stop()

	sendDetailsUpdate(ctx, provider, program)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendDetailsUpdate(ctx, provider, program)
		}
	}
}

func sendDetailsUpdate(ctx context.Context, provider tmux.Provider, program *tea.Program) {
	details, err := provider.ListDetails(ctx)
	if err != nil {
		program.Send(tui.SessionDetailsUpdatedMsg{
			Details:   map[string]tui.SessionDetails{},
			UpdatedAt: time.Now(),
			Err:       err,
		})
		return
	}

	program.Send(tui.SessionDetailsUpdatedMsg{
		Details:   toTUISessionDetails(details),
		UpdatedAt: time.Now(),
	})
}

func toTUISessions(names []string) []tui.Session {
	sessions := make([]tui.Session, 0, len(names))
	for _, name := range names {
		sessions = append(sessions, tui.Session{Name: name})
	}

	return sessions
}

func toTUISessionDetails(details []tmux.SessionDetails) map[string]tui.SessionDetails {
	result := make(map[string]tui.SessionDetails, len(details))
	for _, detail := range details {
		result[detail.Name] = tui.SessionDetails{
			WindowCount:     detail.WindowCount,
			AttachedClients: detail.AttachedClients,
			CreatedAt:       detail.CreatedAt,
			Preview:         detail.Preview,
		}
	}

	return result
}

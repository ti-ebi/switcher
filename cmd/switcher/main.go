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
	provider := tmux.NewProvider()

	for {
		sessionNames, err := provider.List(ctx)
		if err != nil {
			log.Fatalf("failed to load sessions: %v", err)
		}

		model := tui.NewModel(toTUISessions(sessionNames))
		program := tea.NewProgram(model, tea.WithAltScreen())
		detailsCtx, cancelDetails := context.WithCancel(ctx)
		go streamSessionDetails(detailsCtx, provider, program)

		finalModelAny, err := program.Run()
		cancelDetails()
		if err != nil {
			log.Fatalf("switcher failed: %v", err)
		}

		finalModel, ok := finalModelAny.(tui.Model)
		if !ok {
			return
		}

		if finalModel.IsQuitting() {
			return
		}

		session, selected := finalModel.SelectedSession()
		if !selected {
			continue
		}

		if err := tmux.AttachSession(ctx, session.Name); err != nil {
			log.Printf("failed to attach session: %v", err)
		}
	}
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
		}
	}

	return result
}

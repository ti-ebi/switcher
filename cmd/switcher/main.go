package main

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"switcher/internal/session/tmux"
	"switcher/internal/tui"
)

func main() {
	ctx := context.Background()

	provider := tmux.NewProvider()
	sessionNames, err := provider.List(ctx)
	if err != nil {
		log.Fatalf("failed to load sessions: %v", err)
	}

	model := tui.NewModel(toTUISessions(sessionNames))
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModelAny, err := program.Run()
	if err != nil {
		log.Fatalf("switcher failed: %v", err)
	}

	finalModel, ok := finalModelAny.(tui.Model)
	if !ok {
		return
	}

	session, selected := finalModel.SelectedSession()
	if !selected {
		return
	}

	if err := tmux.AttachSession(ctx, session.Name); err != nil {
		log.Fatalf("failed to attach session: %v", err)
	}

	fmt.Printf("Attached session: %s\n", session.Name)
}

func toTUISessions(names []string) []tui.Session {
	sessions := make([]tui.Session, 0, len(names))
	for _, name := range names {
		sessions = append(sessions, tui.Session{Name: name})
	}

	return sessions
}

package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	if model.Cursor() != 0 {
		t.Fatalf("cursor = %d, want 0", model.Cursor())
	}

	if model.SessionCount() != 2 {
		t.Fatalf("session count = %d, want 2", model.SessionCount())
	}
}

func TestUpdateMovesCursorDownWithJ(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if nextModel.Cursor() != 1 {
		t.Fatalf("cursor = %d, want 1", nextModel.Cursor())
	}
}

func TestUpdateMovesCursorUpWithK(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"}})

	moved, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	nextModel, ok := moved.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", moved)
	}

	updated, _ := nextModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if finalModel.Cursor() != 0 {
		t.Fatalf("cursor = %d, want 0", finalModel.Cursor())
	}
}

func TestUpdateMarksQuittingWithQ(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if !nextModel.IsQuitting() {
		t.Fatal("quitting = false, want true")
	}
}

func TestUpdateSelectsCurrentSessionWithEnter(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	moved, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	nextModel, ok := moved.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", moved)
	}

	updated, _ := nextModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	session, selected := finalModel.SelectedSession()
	if !selected {
		t.Fatal("selected = false, want true")
	}

	if session.Name != "beta" {
		t.Fatalf("selected session = %q, want %q", session.Name, "beta")
	}
}

func TestViewRendersSidebarWithSelection(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	output := model.View()
	want := "Sessions\n\n> alpha\n  beta\n\n[j/k] move  [enter] connect  [q] quit\n"

	if output != want {
		t.Fatalf("view mismatch\nwant:\n%s\n\ngot:\n%s", want, output)
	}
}

func TestUpdateStoresWindowSize(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 64, Height: 10})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if nextModel.Width() != 64 {
		t.Fatalf("width = %d, want 64", nextModel.Width())
	}

	if nextModel.Height() != 10 {
		t.Fatalf("height = %d, want 10", nextModel.Height())
	}
}

func TestViewRendersTwoPaneSidebarLayout(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	resized, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 8})
	nextModel, ok := resized.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", resized)
	}

	output := nextModel.View()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	if len(lines) != 8 {
		t.Fatalf("line count = %d, want 8\noutput:\n%s", len(lines), output)
	}

	if !strings.Contains(output, "Sessions") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Sessions", output)
	}

	if !strings.Contains(output, "Details") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Details", output)
	}

	if !strings.Contains(output, " | ") {
		t.Fatalf("output must contain pane separator %q\noutput:\n%s", " | ", output)
	}

	if !strings.Contains(output, "> alpha") {
		t.Fatalf("output must contain selected marker for alpha\noutput:\n%s", output)
	}

	if !strings.Contains(output, "Session: alpha") {
		t.Fatalf("output must contain details for selected session\noutput:\n%s", output)
	}
}

func TestUpdateStoresSessionDetailsAndViewRendersThem(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(SessionDetailsUpdatedMsg{
		Details: map[string]SessionDetails{
			"alpha": {
				WindowCount:     4,
				AttachedClients: 2,
				CreatedAt:       time.Unix(1700000000, 0).UTC(),
			},
		},
		UpdatedAt: time.Unix(1700001200, 0).UTC(),
	})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	resized, _ := nextModel.Update(tea.WindowSizeMsg{Width: 70, Height: 10})
	sizedModel, ok := resized.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", resized)
	}

	output := sizedModel.View()
	if !strings.Contains(output, "Windows: 4") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Windows: 4", output)
	}

	if !strings.Contains(output, "Attached: 2") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Attached: 2", output)
	}

	wantUpdatedLine := "Updated: " + time.Unix(1700001200, 0).Local().Format("15:04:05")
	if !strings.Contains(output, wantUpdatedLine) {
		t.Fatalf("output must contain %q\noutput:\n%s", wantUpdatedLine, output)
	}
}

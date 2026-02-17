package tui

import (
	"fmt"
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
	want := "Sessions\n\n> alpha\n  beta\n\n[j/k] move  [n] new  [r] rename  [d] delete  [enter] connect  [q] quit\n"

	if output != want {
		t.Fatalf("view mismatch\nwant:\n%s\n\ngot:\n%s", want, output)
	}
}

func TestUpdateStartsCreateModeWithN(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}, {Name: "beta"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if !nextModel.IsCreatingSession() {
		t.Fatal("creating session = false, want true")
	}
}

func TestUpdateBuildsAndSubmitsCreateSessionName(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	createModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	for _, key := range []rune{'d', 'e', 'v'} {
		updated, _ = createModeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		createModeModel, ok = updated.(Model)
		if !ok {
			t.Fatalf("update returned %T, want Model", updated)
		}
	}

	updated, _ = createModeModel.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	createModeModel, ok = updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = createModeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	createModeModel, ok = updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = createModeModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	sessionName, requested := finalModel.CreateRequest()
	if !requested {
		t.Fatal("requested = false, want true")
	}

	if sessionName != "dev" {
		t.Fatalf("sessionName = %q, want %q", sessionName, "dev")
	}
}

func TestUpdateCancelsCreateModeWithEsc(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	createModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = createModeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	createModeModel, ok = updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = createModeModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if finalModel.IsCreatingSession() {
		t.Fatal("creating session = true, want false")
	}

	if finalModel.IsQuitting() {
		t.Fatal("quitting = true, want false")
	}

	_, requested := finalModel.CreateRequest()
	if requested {
		t.Fatal("requested = true, want false")
	}
}

func TestUpdateStartsRenameModeWithR(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if !nextModel.IsRenamingSession() {
		t.Fatal("renaming session = false, want true")
	}
}

func TestUpdateBuildsAndSubmitsRenameSessionName(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	renameModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	for _, key := range []rune{'b', 'e', 't', 'a'} {
		updated, _ = renameModeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		renameModeModel, ok = updated.(Model)
		if !ok {
			t.Fatalf("update returned %T, want Model", updated)
		}
	}

	updated, _ = renameModeModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	from, to, requested := finalModel.RenameRequest()
	if !requested {
		t.Fatal("requested = false, want true")
	}

	if from != "alpha" || to != "beta" {
		t.Fatalf("rename request = (%q -> %q), want (%q -> %q)", from, to, "alpha", "beta")
	}
}

func TestUpdateStartsDeleteConfirmationWithD(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if !nextModel.IsConfirmingDelete() {
		t.Fatal("confirming delete = false, want true")
	}
}

func TestUpdateConfirmsDeleteWithY(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	deleteModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = deleteModeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	sessionName, requested := finalModel.DeleteRequest()
	if !requested {
		t.Fatal("requested = false, want true")
	}

	if sessionName != "alpha" {
		t.Fatalf("sessionName = %q, want %q", sessionName, "alpha")
	}
}

func TestUpdateCancelsDeleteWithEsc(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	deleteModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = deleteModeModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	finalModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	if finalModel.IsConfirmingDelete() {
		t.Fatal("confirming delete = true, want false")
	}

	_, requested := finalModel.DeleteRequest()
	if requested {
		t.Fatal("requested = true, want false")
	}
}

func TestViewShowsCreateSessionPromptInCreateMode(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	createModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = createModeModel.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	sizedModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	output := sizedModel.View()
	if !strings.Contains(output, "Create Session") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Create Session", output)
	}

	if !strings.Contains(output, "Name: _") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Name: _", output)
	}
}

func TestViewShowsRenameSessionPromptInRenameMode(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	renameModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = renameModeModel.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	sizedModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	output := sizedModel.View()
	if !strings.Contains(output, "Rename Session") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Rename Session", output)
	}

	if !strings.Contains(output, "From: alpha") {
		t.Fatalf("output must contain %q\noutput:\n%s", "From: alpha", output)
	}
}

func TestViewShowsDeleteSessionPromptInDeleteMode(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	deleteModeModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	updated, _ = deleteModeModel.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	sizedModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	output := sizedModel.View()
	if !strings.Contains(output, "Delete Session") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Delete Session", output)
	}

	if !strings.Contains(output, "Session: alpha") {
		t.Fatalf("output must contain %q\noutput:\n%s", "Session: alpha", output)
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
				Preview:         "build running...\nall green",
			},
		},
		UpdatedAt: time.Unix(1700001200, 0).UTC(),
	})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	resized, _ := nextModel.Update(tea.WindowSizeMsg{Width: 70, Height: 14})
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

	if !strings.Contains(output, "build running...") {
		t.Fatalf("output must contain %q\noutput:\n%s", "build running...", output)
	}

	wantUpdatedLine := "Updated: " + time.Unix(1700001200, 0).Local().Format("15:04:05")
	if !strings.Contains(output, wantUpdatedLine) {
		t.Fatalf("output must contain %q\noutput:\n%s", wantUpdatedLine, output)
	}
}

func TestViewPrefersLatestPreviewLinesWhenPreviewIsLong(t *testing.T) {
	t.Parallel()

	model := NewModel([]Session{{Name: "alpha"}})

	previewLines := make([]string, 0, 20)
	for index := 1; index <= 20; index++ {
		previewLines = append(previewLines, fmt.Sprintf("preview-%02d", index))
	}

	updated, _ := model.Update(SessionDetailsUpdatedMsg{
		Details: map[string]SessionDetails{
			"alpha": {
				WindowCount:     1,
				AttachedClients: 0,
				CreatedAt:       time.Unix(1700000000, 0).UTC(),
				Preview:         strings.Join(previewLines, "\n"),
			},
		},
		UpdatedAt: time.Unix(1700001200, 0).UTC(),
	})
	nextModel, ok := updated.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", updated)
	}

	resized, _ := nextModel.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	sizedModel, ok := resized.(Model)
	if !ok {
		t.Fatalf("update returned %T, want Model", resized)
	}

	output := sizedModel.View()
	if !strings.Contains(output, "preview-20") {
		t.Fatalf("output must contain %q\noutput:\n%s", "preview-20", output)
	}

	if strings.Contains(output, "preview-01") {
		t.Fatalf("output must not contain %q when preview is truncated\noutput:\n%s", "preview-01", output)
	}
}

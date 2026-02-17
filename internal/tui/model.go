package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Session represents one external terminal session (for example, a tmux session).
type Session struct {
	Name string
}

// SessionDetails represents dynamic metadata shown in the right pane.
type SessionDetails struct {
	WindowCount     int
	AttachedClients int
	CreatedAt       time.Time
	Preview         string
}

// SessionDetailsUpdatedMsg refreshes right-pane details for sessions.
type SessionDetailsUpdatedMsg struct {
	Details   map[string]SessionDetails
	UpdatedAt time.Time
	Err       error
}

// Model is the Bubble Tea state for the switcher sidebar.
type Model struct {
	sessions        []Session
	sessionDetails  map[string]SessionDetails
	detailsUpdated  time.Time
	detailsError    string
	cursor          int
	width           int
	height          int
	selected        bool
	selectedSession Session
	quitting        bool
}

// NewModel builds a model with a stable copy of sessions.
func NewModel(sessions []Session) Model {
	clonedSessions := append([]Session(nil), sessions...)

	return Model{
		sessions:       clonedSessions,
		sessionDetails: map[string]SessionDetails{},
	}
}

// Init starts without side effects.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keyboard interactions for navigation and selection.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if detailsMsg, ok := msg.(SessionDetailsUpdatedMsg); ok {
		m.sessionDetails = cloneSessionDetails(detailsMsg.Details)
		m.detailsUpdated = detailsMsg.UpdatedAt
		m.detailsError = ""
		if detailsMsg.Err != nil {
			m.detailsError = detailsMsg.Err.Error()
		}

		return m, nil
	}

	if windowSizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = windowSizeMsg.Width
		m.height = windowSizeMsg.Height
		return m, nil
	}

	keyMsg, isKeyMsg := msg.(tea.KeyMsg)
	if !isKeyMsg {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.cursor < len(m.sessions)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.sessions) == 0 {
			return m, nil
		}

		m.selected = true
		m.selectedSession = m.sessions[m.cursor]
		return m, tea.Quit
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the list of sessions in a left-sidebar style block.
func (m Model) View() string {
	if m.selected {
		return fmt.Sprintf("Connecting to session: %s\n", m.selectedSession.Name)
	}

	if m.quitting {
		return "Quitting switcher...\n"
	}

	if m.width > 0 && m.height > 0 {
		return m.viewWithSidebar()
	}

	return m.viewWithoutSidebar()
}

func (m Model) viewWithoutSidebar() string {
	if len(m.sessions) == 0 {
		return "Sessions\n\n(no sessions)\n\n[q] quit\n"
	}

	var builder strings.Builder
	builder.WriteString("Sessions\n\n")

	for index, session := range m.sessions {
		prefix := "  "
		if index == m.cursor {
			prefix = "> "
		}

		builder.WriteString(prefix)
		builder.WriteString(session.Name)
		builder.WriteString("\n")
	}

	builder.WriteString("\n[j/k] move  [enter] connect  [q] quit\n")
	return builder.String()
}

// Cursor returns the current highlighted index.
func (m Model) Cursor() int {
	return m.cursor
}

// SessionCount returns the number of available sessions.
func (m Model) SessionCount() int {
	return len(m.sessions)
}

// SelectedSession returns the chosen session and whether one is selected.
func (m Model) SelectedSession() (Session, bool) {
	return m.selectedSession, m.selected
}

// IsQuitting returns whether the user requested quit from the switcher UI.
func (m Model) IsQuitting() bool {
	return m.quitting
}

// Width returns the latest known terminal width.
func (m Model) Width() int {
	return m.width
}

// Height returns the latest known terminal height.
func (m Model) Height() int {
	return m.height
}

func (m Model) viewWithSidebar() string {
	if m.width < 8 || m.height < 4 {
		return m.viewWithoutSidebar()
	}

	sidebarWidth := m.width / 3
	if sidebarWidth < 22 {
		sidebarWidth = 22
	}

	maxSidebarWidth := m.width - 16
	if maxSidebarWidth < 12 {
		maxSidebarWidth = 12
	}

	if sidebarWidth > maxSidebarWidth {
		sidebarWidth = maxSidebarWidth
	}

	rightWidth := m.width - sidebarWidth - 3
	if rightWidth < 1 {
		return m.viewWithoutSidebar()
	}

	leftLines := m.sidebarLines(m.height)
	rightLines := m.detailsLines(m.height)

	var builder strings.Builder
	for index := 0; index < m.height; index++ {
		builder.WriteString(fitLine(leftLines[index], sidebarWidth))
		builder.WriteString(" | ")
		builder.WriteString(fitLine(rightLines[index], rightWidth))
		builder.WriteByte('\n')
	}

	return builder.String()
}

func (m Model) sidebarLines(lineCount int) []string {
	lines := []string{"Sessions", ""}

	for index, session := range m.sessions {
		prefix := "  "
		if index == m.cursor {
			prefix = "> "
		}

		lines = append(lines, prefix+session.Name)
	}

	return withFooter(lines, lineCount, "[j/k] move [enter] connect [q] quit")
}

func (m Model) detailsLines(lineCount int) []string {
	lines := []string{"Details", ""}

	if len(m.sessions) == 0 {
		lines = append(lines, "(no session selected)")
		return withFooter(lines, lineCount, "")
	}

	currentSession := m.sessions[m.cursor]
	lines = append(lines, "Session: "+currentSession.Name, "")

	if detail, ok := m.sessionDetails[currentSession.Name]; ok {
		metadata := []string{
			fmt.Sprintf("Windows: %d", detail.WindowCount),
			fmt.Sprintf("Attached: %d", detail.AttachedClients),
			"Created: " + detail.CreatedAt.Local().Format("2006-01-02 15:04:05"),
		}

		suffix := detailsSuffixLines(m.detailsError, m.detailsUpdated)
		for availablePreviewLineCount(lines, metadata, suffix, lineCount) < 1 && len(metadata) > 0 {
			metadata = metadata[:len(metadata)-1]
		}

		lines = append(lines, metadata...)
		lines = append(lines, "", "Preview:")
		previewCapacity := availablePreviewLineCount(lines[:len(lines)-2], nil, suffix, lineCount)
		lines = append(lines, newestPreviewLines(detail.Preview, previewCapacity)...)
		lines = append(lines, suffix...)
		return withFooter(lines, lineCount, "")
	} else {
		lines = append(lines, "Windows: -", "Attached: -", "Created: -")
	}

	lines = append(lines, detailsSuffixLines(m.detailsError, m.detailsUpdated)...)

	return withFooter(lines, lineCount, "")
}

func withFooter(lines []string, lineCount int, footer string) []string {
	if lineCount <= 0 {
		return []string{}
	}

	result := append([]string(nil), lines...)
	if len(result) > lineCount {
		result = result[:lineCount]
	}

	for len(result) < lineCount {
		result = append(result, "")
	}

	if footer != "" {
		result[lineCount-1] = footer
	}

	return result
}

func fitLine(line string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(line)
	if len(runes) >= width {
		return string(runes[:width])
	}

	return line + strings.Repeat(" ", width-len(runes))
}

func cloneSessionDetails(source map[string]SessionDetails) map[string]SessionDetails {
	if len(source) == 0 {
		return map[string]SessionDetails{}
	}

	cloned := make(map[string]SessionDetails, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func previewLines(preview string) []string {
	trimmed := strings.TrimSpace(preview)
	if trimmed == "" {
		return []string{"(no preview)"}
	}

	return strings.Split(trimmed, "\n")
}

func newestPreviewLines(preview string, maxLines int) []string {
	if maxLines <= 0 {
		return []string{}
	}

	lines := previewLines(preview)
	if len(lines) <= maxLines {
		return lines
	}

	tail := append([]string(nil), lines[len(lines)-maxLines:]...)
	if maxLines > 1 {
		tail[0] = "..."
	}

	return tail
}

func availablePreviewLineCount(prefix, metadata, suffix []string, lineCount int) int {
	return lineCount - (len(prefix) + len(metadata) + 2 + len(suffix))
}

func detailsSuffixLines(detailsError string, detailsUpdated time.Time) []string {
	lines := make([]string, 0, 4)
	if detailsError != "" {
		lines = append(lines, "Refresh error: "+detailsError)
	}

	if !detailsUpdated.IsZero() {
		lines = append(lines, "Updated: "+detailsUpdated.Local().Format("15:04:05"))
	}

	lines = append(lines, "", "Enter to connect")
	return lines
}

# CLAUDE.md

## Project Overview
`switcher` is a terminal product that launches as a TUI and makes switching between multiple terminal sessions fast and simple.

## Chosen Tech Stack
- Language: `Go`
- TUI framework: `github.com/charmbracelet/bubbletea`
- UI components/styling: `github.com/charmbracelet/bubbles` and `github.com/charmbracelet/lipgloss` (as needed)
- Session integration: external terminal session providers (first target is `tmux`) behind an adapter layer
- Testing: Go `testing` package, Bubble Tea model/view tests
- Linting: `golangci-lint` with strict rules
- Formatting: `gofumpt` + `goimports`
- Build: `go build`
- Automation: `Makefile` + CI

## Product Vision (Target UX)
The expected user experience is:

1. The app starts as a TUI in the terminal, with a sidebar on the left.
2. Users move through sessions in the sidebar with `J` and `K` keys.
3. Each listed item maps to an existing terminal session (for example, tmux sessions), so users can switch quickly.
4. Pressing `Enter` jumps into the selected session.

## Development Principles

### 1. Mandatory TDD Workflow
All feature work must follow this order:

1. Write tests first.
2. Implement the feature.
3. Refactor while keeping tests green.

No feature is complete unless it has gone through this full cycle.

### 2. TUI Testing Strategy (No VHS for Now)
Layout and interaction must be tested without relying on VHS in the initial phase.

1. Verify key interactions (`j`/`k`/`enter`) through model update tests.
2. Verify sidebar rendering through `View()` output assertions and golden-style snapshots when useful.
3. Add higher-level visual tooling later only if necessary.

### 3. Quality Gates (Required)
The following checks are mandatory for every change:

1. Tests and lints must run and be enforced.
2. Build must pass.
3. Code formatting must be applied.

## Definition of Done
A task is done only when:

1. Behavior matches the product vision above.
2. Tests were written first and are passing.
3. Lint checks pass.
4. Build succeeds.
5. Formatting is applied.

## Notes for Implementation
- Keep key navigation responsive and consistent (`J`/`K` for movement, `Enter` for selection).
- Design session adapters so external session providers (like tmux) can be integrated cleanly.
- Prefer small, testable units for TUI state transitions and session selection logic.

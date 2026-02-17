# switcher

[![CI](https://github.com/ti-ebi/switcher/actions/workflows/ci.yml/badge.svg)](https://github.com/ti-ebi/switcher/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-555)](#)

A fast terminal session switcher built as a TUI.

`switcher` gives you a sidebar-driven UI to move across tmux sessions, inspect live output, and jump in instantly.

## Why switcher

- Fast keyboard navigation (`j` / `k` + `enter`)
- Live session details in the right pane (windows, attached clients, updated time)
- Live preview of the latest pane output (tail-focused)
- Session lifecycle operations from UI: create, rename, delete
- Returns to switcher after tmux detach (`Ctrl+b`, then `d`)

## Preview

```text
Sessions                         | Details
                                 |
> dev                            | Session: dev
  api                            |
  logs                           | Windows: 3
                                 | Attached: 1
                                 | Created: 2026-02-17 19:10:02
                                 |
[j/k] move [n] new [r] rename    | Preview:
[d] delete [enter] connect [q]   | ...
                                 | latest log line...
```

## Requirements

- Go `1.25+`
- `tmux` installed and available in `PATH`
- A terminal with ANSI color support (recommended)

## Installation

### Install from source

```bash
git clone git@github.com:ti-ebi/switcher.git
cd switcher
make build
```

### Install with `go install`

```bash
go install github.com/ti-ebi/switcher/cmd/switcher@latest
```

## Usage

Run:

```bash
switcher
```

### Keymap

| Key | Action |
|---|---|
| `j` / `k` | Move selection down/up |
| `enter` | Attach selected tmux session |
| `n` | Create a new tmux session |
| `r` | Rename selected tmux session |
| `d` | Delete selected tmux session (with confirmation) |
| `q` / `ctrl+c` | Quit switcher |
| `esc` | Cancel current input/confirmation mode |

### Session operations

- Create: `n` -> type name -> `enter`
- Rename: `r` -> type new name -> `enter`
- Delete: `d` -> `y` to confirm (or `n` / `esc` to cancel)

### Return flow after attach

1. Select a session and press `enter`
2. Work inside tmux
3. Detach with `Ctrl+b`, then `d`
4. You return to switcher automatically

## Development

This project enforces TDD and strict quality gates.

### Workflow

1. Write test first
2. Implement
3. Refactor with tests green

### Commands

```bash
make fmt        # format code (gofumpt + goimports)
make lint       # strict lint checks
make test       # run tests
make build      # compile
make check      # fmt-check + lint + test + build
```

## Architecture (Current)

- `cmd/switcher`: app entrypoint and runtime loop
- `internal/tui`: Bubble Tea model/update/view and interaction modes
- `internal/session/tmux`: tmux integration (list/details/attach/create/rename/delete)

## Roadmap

- Configurable theme profiles
- Fuzzy session search/filter
- Multi-provider session backend (zellij/screen)


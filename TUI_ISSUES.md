# TUI Issues & Context

## Goal

Redesign `tui/tui.go` to look like opencode's UI:
- Near-black background, minimal, lots of breathing room
- No rounded border boxes on the main view
- Input with a left border accent (`│`) instead of a full box
- Small right-aligned hint bar at the bottom: `↑↓ history   tab complete   ctrl+p commands   esc quit`
- **Ctrl+P modal** — centered overlay showing all commands with descriptions, searchable, navigable with ↑↓, Enter pastes the command to input

Reference screenshots are in the repo root (opencode-main.png, opencode-modal.png) or described below.

---

## Current State

The file is at `tui/tui.go`. It compiles and runs via `make cli` (server must be running via `make run` first).

The TUI uses:
- `github.com/charmbracelet/bubbletea` — Bubble Tea TUI framework
- `github.com/charmbracelet/lipgloss` — styling
- `tea.WithAltScreen()` — full-screen alt buffer

---

## Known Issues

### 1. Terminal content bleeding through (main issue)
When the TUI starts, previous terminal content is visible through the TUI — code, text, etc. bleeds through wherever the view doesn't explicitly render content.

**Root cause:** `lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)` is being used but it only fills with whitespace where the content block doesn't reach. The alt screen buffer isn't being cleared properly on startup.

**Attempted fix:** Switched from `lipgloss.JoinVertical` to `lipgloss.Place` — did not fully solve it.

**Possible fixes to try:**
- Return `tea.ClearScreen()` from `Init()` as a startup command
- Use `tea.EnterAltScreen()` manually
- Ensure the content block fills `m.height` lines exactly so lipgloss.Place has nothing to pad
- Check if the height calculation is off — current: `historyHeight := m.height - 5` (header + gap + input + gap + hint = 5 fixed lines)

### 2. Input height accounting
The `inputAccentStyle` uses a left-only border:
```go
inputAccentStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(mauve).
    PaddingLeft(1)
```
This adds 1 char to the left but should not add height. If lipgloss is treating this differently and adding a line, the height math breaks.

### 3. Modal may not look right (untested)
The Ctrl+P modal renders via `viewModal()` using `lipgloss.Place(m.width, m.height, Center, Center, modal)`. This has not been confirmed working visually — the main screen issue needs fixing first.

---

## Architecture Overview

### Model struct
```go
type Model struct {
    client      *client.Client
    input       string
    history     []historyEntry  // list of {command, response, isError}
    historyIdx  int             // -1 = not browsing history
    width       int
    height      int
    showModal   bool
    modalSearch string          // text typed in modal search
    modalIdx    int             // selected item index in modal
}
```

### Key flows
- `Update()` dispatches to `updateModal()` or `updateMain()` based on `showModal`
- `updateMain()` handles all normal typing/commands; CLEAR handled inline
- `updateModal()` handles search typing, ↑↓ nav, Enter (paste to input), ESC (close)
- `handleCommand()` is async — returns a `tea.Cmd` closure that makes network calls
- Responses come back as `responseMsg` and are attached to the last history entry

### Commands handled
SET, GET, DEL, EXISTS, TTL, EXPIRE, KEYS, CLEAR, HELP/?, QUIT/EXIT

### Colors (Catppuccin Mocha)
```go
overlay  = "#6c7086"  // dim/muted
text     = "#cdd6f4"  // main text
mauve    = "#cba6f7"  // accents, input border
pink     = "#f38ba8"  // errors, input prompt
green    = "#a6e3a1"  // success
blue     = "#89b4fa"  // values, modal highlight text
teal     = "#94e2d5"  // key names
yellow   = "#f9e2af"  // warnings
surface0 = "#313244"  // modal selected item background
```

---

## What a Good Fix Looks Like

1. TUI starts clean — no terminal bleed-through
2. Main screen: logo top-left, history fills middle, left-accent input near bottom, minimal hint bar at very bottom right
3. Ctrl+P opens a centered modal with command list, searchable, keyboard navigable
4. ESC closes modal (not quit); ESC when modal closed = quit
5. All existing commands (SET/GET/DEL/EXISTS/TTL/EXPIRE/KEYS) still work
6. Builds clean: `go build ./...`

---

## File Location

```
tui/tui.go   — the only file to change
```

Run with:
```bash
make run     # terminal 1 — starts server on :5001
make cli     # terminal 2 — starts TUI
```

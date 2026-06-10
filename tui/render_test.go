package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func renderDims(t *testing.T, m Model) {
	t.Helper()
	out := m.View()
	rows := strings.Split(out, "\n")
	if len(rows) != m.height {
		t.Errorf("got %d rows, want height %d", len(rows), m.height)
	}
	for i, r := range rows {
		if w := lipgloss.Width(r); w != m.width {
			t.Errorf("row %d width = %d, want %d", i, w, m.width)
		}
	}
}

func TestViewFillsScreen(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	// empty history, two-column (sidebar shown)
	renderDims(t, Model{width: 120, height: 30, historyIdx: -1, addr: ":5001"})
	renderDims(t, Model{width: 80, height: 24, historyIdx: -1, addr: ":5001"})

	// multi-line response (simulating KEYS) that exceeds the history area
	var entries []historyEntry
	for i := 0; i < 50; i++ {
		entries = append(entries, historyEntry{command: "KEYS", response: strings.Repeat("• k\n", 10)})
	}
	renderDims(t, Model{width: 120, height: 30, historyIdx: -1, addr: ":5001", history: entries})

	// narrow terminal — single-column fallback (sidebar dropped)
	renderDims(t, Model{width: 70, height: 24, historyIdx: -1, history: entries})
	renderDims(t, Model{width: 40, height: 6, historyIdx: -1, history: entries})
}

func TestModalFillsScreen(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	renderDims(t, Model{width: 80, height: 24, historyIdx: -1, showModal: true})
	renderDims(t, Model{width: 80, height: 24, historyIdx: -1, showModal: true, modalSearch: "EX", modalIdx: 1})
}

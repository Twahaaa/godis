package tui

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Twahaaa/godis/client"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/superstarryeyes/bit/ansifonts"
)

// ansiRe strips SGR color codes so rendered font art can be recolored to match the theme.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// loadLogo renders "GoDIS" with bit's ansifonts (8bitfortress) at the given scale and
// strips its built-in color so we can paint it in the theme's mauve. Returns nil on error.
func loadLogo(scale float64) []string {
	font, err := ansifonts.LoadFont("8bitfortress")
	if err != nil {
		return nil
	}
	opts := ansifonts.DefaultRenderOptions()
	opts.ScaleFactor = scale
	var out []string
	for _, l := range ansifonts.RenderTextWithOptions("GoDIS", font, opts) {
		out = append(out, strings.TrimRight(ansiRe.ReplaceAllString(l, ""), " "))
	}
	return out
}

// Catppuccin Mocha
var (
	overlay  = lipgloss.Color("#6c7086")
	text     = lipgloss.Color("#cdd6f4")
	mauve    = lipgloss.Color("#cba6f7")
	pink     = lipgloss.Color("#f38ba8")
	green    = lipgloss.Color("#a6e3a1")
	blue     = lipgloss.Color("#89b4fa")
	teal     = lipgloss.Color("#94e2d5")
	yellow   = lipgloss.Color("#f9e2af")
	surface0 = lipgloss.Color("#313244")
	base     = lipgloss.Color("#11111b") // Catppuccin crust — near-black surface
	sidebar  = lipgloss.Color("#1c1c22") // a distinct dark gray panel for the sidebar
)

var screenStyle = lipgloss.NewStyle().Background(base)

var (
	logoStyle = lipgloss.NewStyle().
			Foreground(mauve).
			Background(base).
			Bold(true).
			Italic(true)

	logoBigStyle = lipgloss.NewStyle().
			Foreground(mauve).
			Background(base).
			Bold(true)

	bylineStyle = lipgloss.NewStyle().
			Foreground(overlay).
			Background(base).
			Italic(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(mauve).
			Background(base).
			Bold(true)

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(pink).
				Background(base).
				Bold(true)

	cmdStyle = lipgloss.NewStyle().
			Foreground(text).
			Background(base)

	successStyle = lipgloss.NewStyle().
			Foreground(green).
			Background(base).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(pink).
			Background(base).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(teal).
			Background(base).
			Bold(true)

	valStyle = lipgloss.NewStyle().
			Foreground(blue).
			Background(base)

	dimStyle = lipgloss.NewStyle().
			Foreground(overlay).
			Background(base).
			Italic(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Background(base)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(surface0).
			BorderBackground(base).
			Background(base).
			Padding(0, 1)

	sidebarBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(sidebar).
				BorderBackground(sidebar).
				Background(sidebar)

	sectionStyle = lipgloss.NewStyle().
			Foreground(text).
			Background(base).
			Bold(true)

	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(overlay).
			BorderBackground(base).
			Background(base).
			Padding(1, 2)

	hintStyle    = lipgloss.NewStyle().Foreground(overlay).Background(base)
	hintKeyStyle = lipgloss.NewStyle().Foreground(mauve).Background(base)

	scrollThumbStyle = lipgloss.NewStyle().Foreground(overlay).Background(base)
	scrollTrackStyle = lipgloss.NewStyle().Foreground(surface0).Background(base)
)

// scrollbarCell renders one cell of the history scrollbar for visible row i,
// given a track of trackH rows over total lines with the window starting at start.
func scrollbarCell(i, trackH, total, start int) string {
	thumbLen := trackH * trackH / total
	if thumbLen < 1 {
		thumbLen = 1
	}
	maxStart := total - trackH // > 0 whenever a scrollbar is shown
	thumbTop := 0
	if maxStart > 0 {
		thumbTop = start * (trackH - thumbLen) / maxStart
	}
	if i >= thumbTop && i < thumbTop+thumbLen {
		return scrollThumbStyle.Render("█")
	}
	return scrollTrackStyle.Render("│")
}

// logoBanner is the "GoDIS" wordmark in a clean outline font, shown in the sidebar.
const logoBanner = `  ___     ___ ___ ___
 / __|___|   \_ _/ __|
| (_ / _ \ |) | |\__ \
 \___\___/___/___|___/`

// palette

type paletteEntry struct {
	name  string
	usage string
	desc  string
}

var palette = []paletteEntry{
	{"SET", "SET <key> <value> [ttl]", "store a value, ttl in seconds"},
	{"GET", "GET <key>", "retrieve a value"},
	{"DEL", "DEL <key>", "delete a key"},
	{"EXISTS", "EXISTS <key>", "check if a key exists"},
	{"TTL", "TTL <key>", "time remaining on a key"},
	{"EXPIRE", "EXPIRE <key> <seconds>", "set or update expiry on a key"},
	{"KEYS", "KEYS", "list all keys"},
	{"CLEAR", "CLEAR", "clear the screen"},
}

func filteredPalette(search string) []paletteEntry {
	if search == "" {
		return palette
	}
	upper := strings.ToUpper(search)
	var result []paletteEntry
	for _, e := range palette {
		if strings.Contains(strings.ToUpper(e.name), upper) ||
			strings.Contains(strings.ToUpper(e.usage), upper) {
			result = append(result, e)
		}
	}
	return result
}

// model

type historyEntry struct {
	command  string
	response string
	isError  bool
}

type responseMsg struct {
	response string
	isError  bool
}

type clearMsg struct{}

type Model struct {
	client      *client.Client
	addr        string
	logo        []string
	input       string
	history     []historyEntry
	historyIdx  int
	width       int
	height      int
	scroll      int // history rows scrolled up from the bottom (0 = latest)
	showModal   bool
	modalSearch string
	modalIdx    int
}

func New(addr string) Model {
	return Model{
		client:     client.New(addr),
		addr:       addr,
		logo:       loadLogo(1.0),
		history:    []historyEntry{},
		historyIdx: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.ClearScreen
}

// historyLines flattens the command history into individual visual rows,
// including blank separators between entries. Shared by Update and View so
// scroll bounds and rendering stay in sync.
func (m Model) historyLines() []string {
	var lines []string
	for ei, entry := range m.history {
		if ei > 0 {
			lines = append(lines, "") // blank line between entries
		}
		lines = append(lines, "  "+promptStyle.Render("❯ ")+cmdStyle.Render(entry.command))
		if entry.response != "" {
			mark := successStyle.Render("✓")
			if entry.isError {
				mark = errorStyle.Render("✗")
			}
			for i, sub := range strings.Split(entry.response, "\n") {
				if i == 0 {
					lines = append(lines, "    "+mark+"  "+sub)
				} else {
					lines = append(lines, "       "+sub)
				}
			}
		}
	}
	return lines
}

// historyHeight is the number of visible history rows (the area between the
// top margin and the input box). The input box is always 3 rows tall.
func (m Model) historyHeight() int {
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

// maxScroll is how far up the history can be scrolled.
func (m Model) maxScroll() int {
	over := len(m.historyLines()) - m.historyHeight()
	if over < 0 {
		return 0
	}
	return over
}

func (m Model) clampScroll(s int) int {
	if mx := m.maxScroll(); s > mx {
		s = mx
	}
	if s < 0 {
		s = 0
	}
	return s
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.showModal {
			return m.updateModal(msg)
		}
		return m.updateMain(msg)

	case tea.MouseMsg:
		if !m.showModal {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.scroll = m.clampScroll(m.scroll + 3)
			case tea.MouseButtonWheelDown:
				m.scroll = m.clampScroll(m.scroll - 3)
			}
		}

	case clearMsg:
		m.history = []historyEntry{}

	case responseMsg:
		if len(m.history) > 0 {
			last := &m.history[len(m.history)-1]
			last.response = msg.response
			last.isError = msg.isError
		}
	}

	return m, nil
}

func (m Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := filteredPalette(m.modalSearch)

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlP:
		m.showModal = false
		m.modalSearch = ""
		m.modalIdx = 0

	case tea.KeyEnter:
		if len(filtered) > 0 {
			idx := m.modalIdx
			if idx >= len(filtered) {
				idx = len(filtered) - 1
			}
			m.input = filtered[idx].name + " "
		}
		m.showModal = false
		m.modalSearch = ""
		m.modalIdx = 0

	case tea.KeyUp:
		if m.modalIdx > 0 {
			m.modalIdx--
		}

	case tea.KeyDown:
		if m.modalIdx < len(filtered)-1 {
			m.modalIdx++
		}

	case tea.KeyBackspace:
		if len(m.modalSearch) > 0 {
			m.modalSearch = m.modalSearch[:len(m.modalSearch)-1]
			m.modalIdx = 0
		}

	default:
		m.modalSearch += msg.String()
		m.modalIdx = 0
	}

	return m, nil
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyCtrlP:
		m.showModal = true
		m.modalSearch = ""
		m.modalIdx = 0

	case tea.KeyEnter:
		if m.input == "" {
			return m, nil
		}
		input := m.input
		m.input = ""
		m.historyIdx = -1
		m.scroll = 0 // jump back to the latest output
		// handle CLEAR here so it can modify model state directly
		if strings.ToUpper(strings.TrimSpace(input)) == "CLEAR" {
			m.history = []historyEntry{}
			return m, nil
		}
		return m, m.handleCommand(input)

	case tea.KeyPgUp:
		m.scroll = m.clampScroll(m.scroll + m.historyHeight())

	case tea.KeyPgDown:
		m.scroll = m.clampScroll(m.scroll - m.historyHeight())

	case tea.KeyCtrlU:
		m.scroll = m.clampScroll(m.scroll + m.historyHeight()/2)

	case tea.KeyCtrlD:
		m.scroll = m.clampScroll(m.scroll - m.historyHeight()/2)

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	case tea.KeyCtrlH:
		if len(m.input) > 0 {
			trimmed := strings.TrimRight(m.input, " ")
			idx := strings.LastIndex(trimmed, " ")
			if idx == -1 {
				m.input = ""
			} else {
				m.input = m.input[:idx+1]
			}
		}

	case tea.KeyCtrlL:
		m.history = []historyEntry{}
		m.historyIdx = -1
		m.scroll = 0

	case tea.KeyUp:
		if len(m.history) > 0 && m.historyIdx < len(m.history)-1 {
			m.historyIdx++
			m.input = m.history[len(m.history)-1-m.historyIdx].command
		}

	case tea.KeyDown:
		if m.historyIdx > 0 {
			m.historyIdx--
			m.input = m.history[len(m.history)-1-m.historyIdx].command
		} else if m.historyIdx == 0 {
			m.historyIdx = -1
			m.input = ""
		}

	case tea.KeyTab:
		m.input = tabComplete(m.input)

	default:
		m.input += msg.String()
	}

	return m, nil
}

func (m *Model) handleCommand(input string) tea.Cmd {
	m.history = append(m.history, historyEntry{command: input})

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	return func() tea.Msg {
		switch strings.ToUpper(parts[0]) {
		case "SET":
			if len(parts) != 3 && len(parts) != 4 {
				return responseMsg{response: "usage: SET <key> <value> [ttl]", isError: true}
			}
			var err error
			if len(parts) == 4 {
				ttl, convErr := strconv.Atoi(parts[3])
				if convErr != nil {
					return responseMsg{response: "TTL must be a number of seconds", isError: true}
				}
				err = m.client.Set(context.Background(), parts[1], parts[2], ttl)
			} else {
				err = m.client.Set(context.Background(), parts[1], parts[2])
			}
			if err != nil {
				return responseMsg{response: err.Error(), isError: true}
			}
			response := fmt.Sprintf("%s  %s  %s",
				successStyle.Render("OK"),
				keyStyle.Render(parts[1]),
				valStyle.Render("← "+parts[2]),
			)
			if len(parts) == 4 {
				response += dimStyle.Render(fmt.Sprintf("  (expires in %ss)", parts[3]))
			}
			return responseMsg{response: response}

		case "GET":
			if len(parts) != 2 {
				return responseMsg{response: "usage: GET <key>", isError: true}
			}
			val, err := m.client.Get(context.Background(), parts[1])
			if err != nil {
				return responseMsg{response: err.Error(), isError: true}
			}
			return responseMsg{
				response: fmt.Sprintf("%s  →  %s",
					keyStyle.Render(parts[1]),
					valStyle.Render(val),
				),
			}

		case "TTL":
			if len(parts) != 2 {
				return responseMsg{response: "usage: TTL <key>", isError: true}
			}
			ttl, err := m.client.TTL(context.Background(), parts[1])
			if err != nil {
				return responseMsg{
					response: fmt.Sprintf("%s  %s", keyStyle.Render(parts[1]), dimStyle.Render(err.Error())),
					isError:  true,
				}
			}
			return responseMsg{
				response: fmt.Sprintf("%s  %s",
					keyStyle.Render(parts[1]),
					valStyle.Render("expires in "+ttl),
				),
			}

		case "EXPIRE":
			if len(parts) != 3 {
				return responseMsg{response: "usage: EXPIRE <key> <seconds>", isError: true}
			}
			ttl, convErr := strconv.Atoi(parts[2])
			if convErr != nil {
				return responseMsg{response: "seconds must be a number", isError: true}
			}
			ok, err := m.client.Expire(context.Background(), parts[1], ttl)
			if err != nil {
				return responseMsg{response: err.Error(), isError: true}
			}
			if !ok {
				return responseMsg{response: fmt.Sprintf("key %s not found", keyStyle.Render(parts[1])), isError: true}
			}
			return responseMsg{
				response: fmt.Sprintf("%s  %s",
					keyStyle.Render(parts[1]),
					dimStyle.Render(fmt.Sprintf("expiry set to %ss", parts[2])),
				),
			}

		case "DEL":
			if len(parts) != 2 {
				return responseMsg{response: "usage: DEL <key>", isError: true}
			}
			deleted, err := m.client.Del(context.Background(), parts[1])
			if err != nil {
				return responseMsg{response: err.Error(), isError: true}
			}
			if !deleted {
				return responseMsg{response: fmt.Sprintf("key %s not found", keyStyle.Render(parts[1])), isError: true}
			}
			return responseMsg{
				response: fmt.Sprintf("%s  %s",
					successStyle.Render("DEL"),
					keyStyle.Render(parts[1]),
				),
			}

		case "EXISTS":
			if len(parts) != 2 {
				return responseMsg{response: "usage: EXISTS <key>", isError: true}
			}
			exists, err := m.client.Exists(context.Background(), parts[1])
			if err != nil {
				return responseMsg{response: err.Error(), isError: true}
			}
			if exists {
				return responseMsg{response: fmt.Sprintf("%s  %s", keyStyle.Render(parts[1]), successStyle.Render("exists"))}
			}
			return responseMsg{response: fmt.Sprintf("%s  %s", keyStyle.Render(parts[1]), dimStyle.Render("does not exist")), isError: true}

		case "KEYS":
			keys, err := m.client.Keys(context.Background())
			if err != nil {
				return responseMsg{response: err.Error(), isError: true}
			}
			var lines []string
			for _, k := range keys {
				lines = append(lines, keyStyle.Render("• ")+cmdStyle.Render(k))
			}
			return responseMsg{response: strings.Join(lines, "\n")}

		case "?", "HELP", "/HELP":
			return responseMsg{response: dimStyle.Render("press ctrl+p to see all commands")}

		case "QUIT", "EXIT":
			return tea.Quit()

		default:
			return responseMsg{
				response: fmt.Sprintf("unknown command %s — press ctrl+p for help",
					warningStyle.Render("'"+parts[0]+"'")),
				isError: true,
			}
		}
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.showModal {
		return viewModal(m)
	}

	// Two columns: a main pane and a right sidebar. Below 80 cols the sidebar
	// is dropped so narrow terminals keep a usable single-column layout.
	sideW := 30
	if m.width < 80 {
		sideW = 0
	}
	mainW := m.width - sideW

	content := m.viewMain(mainW)
	if sideW > 0 {
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, m.viewSidebar(sideW))
	}

	return screenStyle.Width(m.width).Height(m.height).
		MaxWidth(m.width).MaxHeight(m.height).Render(content)
}

// viewMain renders the conversation pane as exactly m.height rows, each colW wide.
func (m Model) viewMain(colW int) string {
	rowStyle := lipgloss.NewStyle().Width(colW).MaxWidth(colW).MaxHeight(1).Background(base)
	blank := rowStyle.Render("")

	// history — flattened to visual rows so multi-line / wrapping responses
	// are counted (and clamped) correctly and never overflow the screen.
	lines := m.historyLines()

	// input box — rounded border, pinned to the bottom of the pane.
	// total box width = boxW + 2 (padding) + 2 (border); +2 left margin +2 right = colW.
	boxW := colW - 8
	if boxW < 8 {
		boxW = 8
	}
	cursor := lipgloss.NewStyle().Foreground(pink).Background(base).Render("█")
	box := boxStyle.Width(boxW).Render(inputPromptStyle.Render("❯ ") + cmdStyle.Render(m.input) + cursor)
	var boxRows []string
	for _, l := range strings.Split(box, "\n") {
		boxRows = append(boxRows, rowStyle.Render("  "+l))
	}

	// status line under the box
	status := lipgloss.NewStyle().Width(colW).MaxWidth(colW).MaxHeight(1).Background(base).
		Align(lipgloss.Right).PaddingRight(3).
		Render(hintKeyStyle.Render("ctrl+p") + hintStyle.Render(" commands"))

	// The middle area lives between the top margin and the input box:
	// topMargin(1) + area + box + status(1) + bottomMargin(1) = m.height
	areaH := m.height - len(boxRows) - 3
	if areaH < 1 {
		areaH = 1
	}

	var area []string
	if len(lines) == 0 {
		area = m.emptyState(colW, areaH, rowStyle, blank)
	} else {
		total := len(lines)
		showBar := total > areaH

		// pick the visible window based on the scroll offset (0 = bottom)
		scroll := m.clampScroll(m.scroll)
		start := total - areaH - scroll
		if start < 0 {
			start = 0
		}
		end := start + areaH
		if end > total {
			end = total
		}
		window := lines[start:end]

		histW := colW
		if showBar {
			histW = colW - 1
		}
		histRow := lipgloss.NewStyle().Width(histW).MaxWidth(histW).MaxHeight(1).Background(base)

		for i := 0; i < areaH; i++ {
			row := blank
			if showBar {
				row = histRow.Render("")
			}
			if i < len(window) {
				if showBar {
					row = histRow.Render(window[i])
				} else {
					row = rowStyle.Render(window[i])
				}
			}
			if showBar {
				row += scrollbarCell(i, areaH, total, start)
			}
			area = append(area, row)
		}
	}

	rows := make([]string, 0, m.height)
	rows = append(rows, blank)
	rows = append(rows, area...)
	rows = append(rows, boxRows...)
	rows = append(rows, status, blank)

	return strings.Join(rows, "\n")
}

// emptyState renders the centered "GoDIS" logo (bit ansifonts) into an areaH-tall
// block, falling back to a one-line hint when the logo can't fit.
func (m Model) emptyState(colW, areaH int, rowStyle lipgloss.Style, blank string) []string {
	center := func(s string) string {
		pad := (colW - lipgloss.Width(s)) / 2
		if pad < 0 {
			pad = 0
		}
		return rowStyle.Render(strings.Repeat(" ", pad) + s)
	}

	logoW := 0
	for _, l := range m.logo {
		if w := lipgloss.Width(l); w > logoW {
			logoW = w
		}
	}

	var block []string
	if len(m.logo) > 0 && colW >= logoW+2 && areaH >= len(m.logo)+2 {
		for _, l := range m.logo {
			block = append(block, center(logoBigStyle.Render(l)))
		}
		block = append(block, blank)
		block = append(block, center(dimStyle.Render("type a command  ·  ctrl+p for all")))
	} else {
		block = append(block, center(dimStyle.Render("no commands yet — ctrl+p for all")))
	}

	top := (areaH - len(block)) / 2
	if top < 0 {
		top = 0
	}
	rows := make([]string, 0, areaH)
	for i := 0; i < top; i++ {
		rows = append(rows, blank)
	}
	rows = append(rows, block...)
	for len(rows) < areaH {
		rows = append(rows, blank)
	}
	if len(rows) > areaH {
		rows = rows[:areaH]
	}
	return rows
}

// viewSidebar renders the context panel as exactly m.height rows, each colW wide
// (including its 1-col left border that doubles as the separator).
func (m Model) viewSidebar(colW int) string {
	innerW := colW - 1
	line := func(s string) string {
		return lipgloss.NewStyle().Width(innerW).MaxWidth(innerW).MaxHeight(1).Background(sidebar).Render(s)
	}
	blank := line("")

	// rebuild the text styles on the darker sidebar surface so the text cells
	// match the sidebar background instead of the lighter main base.
	sectionSt := sectionStyle.Background(sidebar)
	keySt := keyStyle.Background(sidebar)
	cmdSt := cmdStyle.Background(sidebar)
	dimSt := dimStyle.Background(sidebar)
	hkSt := hintKeyStyle.Background(sidebar)
	hsSt := hintStyle.Background(sidebar)

	commands := len(m.history)
	errors := 0
	for _, e := range m.history {
		if e.isError {
			errors++
		}
	}
	errVal := cmdSt.Render(strconv.Itoa(errors))
	if errors > 0 {
		errVal = errorStyle.Background(sidebar).Render(strconv.Itoa(errors))
	}

	top := []string{blank}
	logoSt := logoBigStyle.Background(sidebar)
	for _, l := range strings.Split(logoBanner, "\n") {
		top = append(top, line(" "+logoSt.Render(l)))
	}
	top = append(top,
		blank,
		line(" "+sectionSt.Render("Server")),
		line(" "+keySt.Render(m.addr)),
		blank,
		line(" "+sectionSt.Render("Session")),
		line(" "+dimSt.Render("commands  ")+cmdSt.Render(strconv.Itoa(commands))),
		line(" "+dimSt.Render("errors    ")+errVal),
		blank,
		line(" "+sectionSt.Render("Shortcuts")),
		line(" "+hkSt.Render("↑↓")+hsSt.Render(" history")),
		line(" "+hkSt.Render("tab")+hsSt.Render(" complete")),
		line(" "+hkSt.Render("ctrl+p")+hsSt.Render(" commands")),
		line(" "+hkSt.Render("esc")+hsSt.Render(" quit")),
	)
	bottom := []string{line(" " + dimSt.Render(shortDir(innerW-2))), blank}

	rows := make([]string, 0, m.height)
	rows = append(rows, top...)
	for len(rows) < m.height-len(bottom) {
		rows = append(rows, blank)
	}
	rows = append(rows, bottom...)
	if len(rows) > m.height {
		rows = rows[:m.height]
	}

	return sidebarBorderStyle.Render(strings.Join(rows, "\n"))
}

// shortDir returns the working directory with $HOME collapsed to ~, truncated to max cells.
func shortDir(max int) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(dir, home) {
		dir = "~" + dir[len(home):]
	}
	if max > 1 && lipgloss.Width(dir) > max {
		dir = "…" + dir[lipgloss.Width(dir)-max+1:]
	}
	return dir
}

func viewModal(m Model) string {
	modalWidth := 60
	if modalWidth > m.width-4 {
		modalWidth = m.width - 4
	}

	filtered := filteredPalette(m.modalSearch)

	idx := m.modalIdx
	if len(filtered) > 0 && idx >= len(filtered) {
		idx = len(filtered) - 1
	}

	// header line
	commandsLabel := lipgloss.NewStyle().Foreground(text).Background(base).Bold(true).Render("Commands")
	escLabel := dimStyle.Render("esc")
	gap := lipgloss.NewStyle().Background(base).Render(strings.Repeat(" ", max(0, modalWidth-lipgloss.Width(commandsLabel)-lipgloss.Width(escLabel))))
	headerLine := commandsLabel + gap + escLabel

	// search line
	cursor := lipgloss.NewStyle().Foreground(mauve).Background(base).Render("█")
	searchLine := dimStyle.Render("> ") + cmdStyle.Render(m.modalSearch) + cursor

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, "")
	lines = append(lines, searchLine)
	lines = append(lines, "")

	if len(filtered) == 0 {
		lines = append(lines, "  "+dimStyle.Render("no commands match"))
	}

	bar := lipgloss.NewStyle().Foreground(mauve).Background(base).Render("▎ ")
	gapPrefix := lipgloss.NewStyle().Background(base).Render("  ")
	usageSel := lipgloss.NewStyle().Foreground(mauve).Background(base).Bold(true)
	usageOff := lipgloss.NewStyle().Foreground(overlay).Background(base)
	for i, entry := range filtered {
		if i == idx {
			lines = append(lines, bar+usageSel.Render(entry.usage))
			lines = append(lines, bar+dimStyle.Render(entry.desc))
		} else {
			lines = append(lines, gapPrefix+usageOff.Render(entry.usage))
			lines = append(lines, gapPrefix+dimStyle.Render(entry.desc))
		}
		if i < len(filtered)-1 {
			lines = append(lines, "")
		}
	}

	content := strings.Join(lines, "\n")
	modal := modalBoxStyle.Width(modalWidth).Render(content)

	placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceBackground(base))
	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(placed)
}

func tabComplete(input string) string {
	if input == "" {
		return input
	}
	upper := strings.ToUpper(input)
	for _, cmd := range []string{"SET ", "GET ", "DEL ", "EXISTS ", "TTL ", "EXPIRE ", "KEYS", "CLEAR "} {
		if strings.HasPrefix(cmd, upper) {
			return cmd
		}
	}
	return input
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Start(addr string) error {
	m := New(addr)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

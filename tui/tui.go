package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Twahaaa/godis/client"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha palette
var (
	overlay = lipgloss.Color("#6c7086")
	text    = lipgloss.Color("#cdd6f4")
	mauve   = lipgloss.Color("#cba6f7")
	pink    = lipgloss.Color("#f38ba8")
	green   = lipgloss.Color("#a6e3a1")
	blue    = lipgloss.Color("#89b4fa")
	teal    = lipgloss.Color("#94e2d5")
	yellow  = lipgloss.Color("#f9e2af")
)

var (
	headerBarStyle = lipgloss.NewStyle().
			Padding(0, 2)

	logoTextStyle = lipgloss.NewStyle().
			Foreground(mauve).
			Bold(true).
			Italic(true)

	bylineTextStyle = lipgloss.NewStyle().
			Foreground(overlay).
			Italic(true)

	subtitleTextStyle = lipgloss.NewStyle().
				Foreground(overlay)

	historyBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mauve).
			Padding(0, 1)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(pink).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Foreground(mauve).
			Bold(true)

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(pink).
				Bold(true)

	cmdStyle = lipgloss.NewStyle().
			Foreground(text)

	successStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(pink).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(teal).
			Bold(true)

	valStyle = lipgloss.NewStyle().
			Foreground(blue)

	dimStyle = lipgloss.NewStyle().
			Foreground(overlay).
			Italic(true)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(mauve).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(overlay)

	helpBarStyle = lipgloss.NewStyle().
			Padding(0, 2)

	warningStyle = lipgloss.NewStyle().
			Foreground(yellow)
)

type historyEntry struct {
	command  string
	response string
	isError  bool
}

type responseMsg struct {
	response string
	isError  bool
}

type Model struct {
	client     *client.Client
	input      string
	history    []historyEntry
	historyIdx int // -1 = not browsing history
	width      int
	height     int
}

func New(addr string) Model {
	return Model{
		client:     client.New(addr),
		history:    []historyEntry{},
		historyIdx: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.input == "" {
				return m, nil
			}
			cmd := m.input
			m.input = ""
			m.historyIdx = -1
			return m, m.handleCommand(cmd)

		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		// Ctrl+Backspace — delete last word
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

		// Ctrl+L — clear screen
		case tea.KeyCtrlL:
			m.history = []historyEntry{}
			m.historyIdx = -1

		// ↑ — previous command in history
		case tea.KeyUp:
			if len(m.history) > 0 && m.historyIdx < len(m.history)-1 {
				m.historyIdx++
				m.input = m.history[len(m.history)-1-m.historyIdx].command
			}

		// ↓ — next command in history
		case tea.KeyDown:
			if m.historyIdx > 0 {
				m.historyIdx--
				m.input = m.history[len(m.history)-1-m.historyIdx].command
			} else if m.historyIdx == 0 {
				m.historyIdx = -1
				m.input = ""
			}

		// Tab — complete command name
		case tea.KeyTab:
			m.input = tabComplete(m.input)

		default:
			m.input += msg.String()
		}

	case responseMsg:
		if len(m.history) > 0 {
			last := &m.history[len(m.history)-1]
			last.response = msg.response
			last.isError = msg.isError
		}
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

		case "CLEAR":
			m.history = []historyEntry{}
			return responseMsg{response: ""}

		case "?", "HELP", "/HELP":
			return responseMsg{
				response: keyStyle.Render("SET") + valStyle.Render(" <key> <value> [ttl]") + dimStyle.Render("  store a value, ttl in seconds") + "\n" +
					"  " + keyStyle.Render("GET") + valStyle.Render(" <key>") + dimStyle.Render("           retrieve a value") + "\n" +
					"  " + keyStyle.Render("DEL") + valStyle.Render(" <key>") + dimStyle.Render("           delete a key") + "\n" +
					"  " + keyStyle.Render("EXISTS") + valStyle.Render(" <key>") + dimStyle.Render("         check if a key exists") + "\n" +
					"  " + keyStyle.Render("KEYS") + dimStyle.Render("                   list all keys") + "\n" +
					"  " + keyStyle.Render("CLEAR") + dimStyle.Render("                  clear the screen") + "\n" +
					"  " + keyStyle.Render("↑ ↓") + dimStyle.Render("                  navigate command history") + "\n" +
					"  " + keyStyle.Render("Tab") + dimStyle.Render("                   autocomplete command") + "\n" +
					"  " + keyStyle.Render("Ctrl+L") + dimStyle.Render("                clear screen") + "\n" +
					"  " + keyStyle.Render("Ctrl+Backspace") + dimStyle.Render("        delete last word") + "\n" +
					"  " + keyStyle.Render("ESC") + dimStyle.Render("                   quit"),
			}

		case "QUIT", "EXIT":
			return tea.Quit()

		default:
			return responseMsg{
				response: fmt.Sprintf("unknown command %s — try SET or GET",
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

	innerWidth := m.width - 6

	// header bar — full width surface background
	logoText := logoTextStyle.Render("GoDIS")
	byText := bylineTextStyle.Render("  by TWAHaaa  ")
	subtitleText := subtitleTextStyle.Render("")
	rightText := subtitleTextStyle.Render("")

	leftSide := lipgloss.JoinHorizontal(lipgloss.Center, logoText, byText, subtitleText)
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(leftSide)-lipgloss.Width(rightText)-4))
	headerContent := leftSide + gap + rightText
	header := headerBarStyle.Width(m.width).Render(headerContent)

	// history box
	var lines []string
	for _, entry := range m.history {
		lines = append(lines,
			promptStyle.Render("❯ ")+cmdStyle.Render(entry.command),
		)
		if entry.response != "" {
			if entry.isError {
				lines = append(lines, cmdStyle.Render("  ")+errorStyle.Render("✗")+" "+entry.response)
			} else {
				lines = append(lines, cmdStyle.Render("  ")+successStyle.Render("✓")+" "+entry.response)
			}
		}
	}
	if len(lines) == 0 {
		lines = append(lines,
			dimStyle.Render("no commands yet — try ")+
				keyStyle.Render("SET foo bar")+
				dimStyle.Render(" or ")+
				keyStyle.Render("GET foo"),
		)
	}

	innerHeight := m.height - 7
	if innerHeight < 1 {
		innerHeight = 1
	}
	if len(lines) > innerHeight {
		lines = lines[len(lines)-innerHeight:]
	}

	historyBox := historyBoxStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(strings.Join(lines, "\n"))

	// input box
	cursor := lipgloss.NewStyle().Foreground(pink).Render("█")
	inputBox := inputBoxStyle.Width(innerWidth).
		Render(inputPromptStyle.Render("❯ ") + cmdStyle.Render(m.input) + cursor)

	// help bar
	help := helpBarStyle.Width(m.width).Render(
		helpKeyStyle.Render("SET")+" "+helpDescStyle.Render("key val")+"   "+
			helpKeyStyle.Render("GET")+" "+helpDescStyle.Render("key")+"   "+
			helpKeyStyle.Render("DEL")+" "+helpDescStyle.Render("key")+"   "+
			helpKeyStyle.Render("EXISTS")+" "+helpDescStyle.Render("key")+"   "+
			helpKeyStyle.Render("KEYS")+"   "+
			helpKeyStyle.Render("CLEAR")+"   "+
			helpKeyStyle.Render("↑↓")+" "+helpDescStyle.Render("history")+"   "+
			helpKeyStyle.Render("Tab")+" "+helpDescStyle.Render("complete")+"   "+
			helpKeyStyle.Render("?")+" "+helpDescStyle.Render("help")+"   "+
			helpKeyStyle.Render("ESC")+" "+helpDescStyle.Render("quit"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		historyBox,
		inputBox,
		help,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Left, lipgloss.Top,
		content,
	)
}

func tabComplete(input string) string {
	if input == "" {
		return input
	}
	upper := strings.ToUpper(input)
	for _, cmd := range []string{"SET ", "GET ", "DEL ", "EXISTS ", "KEYS", "CLEAR ", "HELP "} {
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
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

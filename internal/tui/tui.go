// Package tui implements the interactive kp terminal UI.
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/azcpcf/kp-go/internal/ui"
)

// refreshInterval is how often the UI rescans listening sockets.
const refreshInterval = 2 * time.Second

// Column widths of the listener table.
const (
	widthPort    = 6
	widthProto   = 6
	widthAddr    = 24
	widthProcess = 18
	widthPID     = 8
)

// Scanner provides the listeners shown by the UI.
type Scanner interface {
	List() ([]scan.Listener, error)
}

// Model is the Bubble Tea model for the interactive listener table.
type Model struct {
	scanner   Scanner
	theme     ui.Theme
	table     table.Model
	listeners []scan.Listener
	err       error
	width     int
	height    int
}

// tickMsg triggers a periodic refresh.
type tickMsg struct{}

// refreshMsg carries the result of a scan.
type refreshMsg struct {
	listeners []scan.Listener
	err       error
}

// rowKey identifies a selected listener across refreshes.
type rowKey struct {
	proto string
	port  int
	pid   string
}

// New returns a Model that reads listeners from s.
func New(s Scanner, theme ui.Theme) Model {
	styles := table.DefaultStyles()
	styles.Header = theme.Header.Padding(0, 1)
	styles.Cell = lipgloss.NewStyle().Padding(0, 1)
	styles.Selected = theme.Port.Bold(true).Reverse(true)

	m := Model{
		scanner: s,
		theme:   theme,
		table: table.New(
			table.WithColumns([]table.Column{
				{Title: "PORT", Width: widthPort},
				{Title: "PROTO", Width: widthProto},
				{Title: "ADDR", Width: widthAddr},
				{Title: "PROCESS", Width: widthProcess},
				{Title: "PID", Width: widthPID},
			}),
			table.WithFocused(true),
			table.WithStyles(styles),
		),
		width:  80,
		height: 24,
	}
	m.resize()
	return m
}

// Run starts the interactive UI and blocks until the user quits.
func Run(s Scanner, theme ui.Theme) error {
	_, err := tea.NewProgram(New(s, theme)).Run()
	return err
}

// Init starts the initial scan and the refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), tick())
}

// Update handles key presses, refreshes and terminal resizes.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.table.MoveUp(1)
		case "down", "j":
			m.table.MoveDown(1)
		case "r":
			return m, m.refresh()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tickMsg:
		return m, tea.Batch(m.refresh(), tick())
	case refreshMsg:
		m.apply(msg)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
	}
	return m, nil
}

// View renders the header, listener table and key hints.
func (m Model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", m.theme.Header.Render(fmt.Sprintf("kp ui - listening ports (%d)", len(m.listeners))))
	if m.err != nil {
		fmt.Fprintf(&b, "%s\n", m.theme.Error.Render("scan error: "+m.err.Error()))
	}
	b.WriteString(m.table.View())
	b.WriteString("\n")
	b.WriteString(m.theme.Muted.Render("j/k up/down  r refresh  q quit"))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// refresh scans for listeners in a command so Update and View never block.
func (m Model) refresh() tea.Cmd {
	s := m.scanner
	return func() tea.Msg {
		ls, err := s.List()
		return refreshMsg{listeners: ls, err: err}
	}
}

// tick schedules the next periodic refresh.
func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// apply replaces the rows, keeping the selected listener when it still exists.
func (m *Model) apply(msg refreshMsg) {
	if msg.err != nil {
		m.err = msg.err
		m.resize()
		return
	}

	m.err = nil
	prev, hadSelection := m.selectedKey()

	listeners := make([]scan.Listener, len(msg.listeners))
	copy(listeners, msg.listeners)
	group.SortByPort(listeners)
	m.listeners = listeners

	rows := make([]table.Row, len(m.listeners))
	for i, l := range m.listeners {
		rows[i] = table.Row{
			strconv.Itoa(l.Port),
			l.Proto,
			l.Addr,
			l.Process,
			l.PID,
		}
	}
	m.table.SetRows(rows)

	if hadSelection {
		for i, l := range m.listeners {
			if keyOf(l) == prev {
				m.table.SetCursor(i)
				break
			}
		}
	}
	m.resize()
}

// selectedKey returns the identity of the selected listener.
func (m Model) selectedKey() (rowKey, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.listeners) {
		return rowKey{}, false
	}
	return keyOf(m.listeners[i]), true
}

func keyOf(l scan.Listener) rowKey {
	return rowKey{proto: l.Proto, port: l.Port, pid: l.PID}
}

// resize fits the table between the header and footer lines.
func (m *Model) resize() {
	h := m.height - 2
	if m.err != nil {
		h--
	}
	if h < 1 {
		h = 1
	}
	m.table.SetWidth(m.width)
	m.table.SetHeight(h)
}

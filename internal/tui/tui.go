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

// statusTTL is how long a status line stays visible.
const statusTTL = 3 * time.Second

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

// Killer terminates the process behind a listener.
type Killer interface {
	Kill(l scan.Listener, force bool) error
}

// Model is the Bubble Tea model for the interactive listener table.
type Model struct {
	scanner   Scanner
	killer    Killer
	theme     ui.Theme
	table     table.Model
	listeners []scan.Listener
	pending   *scan.Listener
	status    string
	statusErr bool
	statusSeq int
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

// killMsg carries the result of a kill command.
type killMsg struct {
	listener scan.Listener
	force    bool
	err      error
}

// clearStatusMsg clears the status line if it is still the current one.
type clearStatusMsg struct {
	seq int
}

// rowKey identifies a selected listener across refreshes.
type rowKey struct {
	proto string
	port  int
	pid   string
}

// New returns a Model that reads listeners from s and kills through k.
func New(s Scanner, k Killer, theme ui.Theme) Model {
	styles := table.DefaultStyles()
	styles.Header = theme.Header.Padding(0, 1)
	styles.Cell = lipgloss.NewStyle().Padding(0, 1)
	styles.Selected = theme.Port.Bold(true).Reverse(true)

	m := Model{
		scanner: s,
		killer:  k,
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
func Run(s Scanner, k Killer, theme ui.Theme) error {
	_, err := tea.NewProgram(New(s, k, theme)).Run()
	return err
}

// Init starts the initial scan and the refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), tick())
}

// Update handles key presses, refreshes, kills and terminal resizes.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.pending != nil {
			switch msg.String() {
			case "t":
				return m.startKill(false)
			case "f":
				return m.startKill(true)
			case "esc":
				m.pending = nil
			case "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			m.table.MoveUp(1)
		case "down", "j":
			m.table.MoveDown(1)
		case "x", "d":
			return m, m.openConfirm()
		case "r":
			return m, m.refresh()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tickMsg:
		return m, tea.Batch(m.refresh(), tick())
	case refreshMsg:
		m.apply(msg)
	case killMsg:
		return m, m.killDone(msg)
	case clearStatusMsg:
		if msg.seq == m.statusSeq {
			m.status = ""
			m.statusErr = false
			m.resize()
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
	}
	return m, nil
}

// View renders the header, listener table and key hints, with a centered
// confirmation box over the table while a kill is pending.
func (m Model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", m.theme.Header.Render(fmt.Sprintf("kp ui - listening ports (%d)", len(m.listeners))))
	if m.err != nil {
		fmt.Fprintf(&b, "%s\n", m.theme.Error.Render("scan error: "+m.err.Error()))
	}
	if m.pending != nil {
		b.WriteString(m.overlay(m.modalView()))
	} else {
		b.WriteString(m.table.View())
	}
	b.WriteString("\n")
	if m.status != "" {
		style := m.theme.Success
		if m.statusErr {
			style = m.theme.Error
		}
		b.WriteString(style.Render(m.status))
		b.WriteString("\n")
	}
	b.WriteString(m.theme.Muted.Render("j/k up/down  x kill  r refresh  q quit"))

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

// openConfirm opens the kill confirmation for the selected listener, or shows
// an error when its owner is unknown.
func (m *Model) openConfirm() tea.Cmd {
	l, ok := m.selectedListener()
	if !ok {
		return nil
	}
	if l.PID == "-" {
		return m.setStatus(fmt.Sprintf("pid unknown on port %d: run with sudo to see the owner", l.Port), true)
	}
	m.pending = &l
	return nil
}

// startKill closes the confirmation and runs the kill as a command.
func (m Model) startKill(force bool) (tea.Model, tea.Cmd) {
	if m.pending == nil {
		return m, nil
	}
	l := *m.pending
	m.pending = nil
	return m, m.kill(l, force)
}

// kill terminates l in a command and reports the result as a killMsg.
func (m Model) kill(l scan.Listener, force bool) tea.Cmd {
	k := m.killer
	return func() tea.Msg {
		return killMsg{listener: l, force: force, err: k.Kill(l, force)}
	}
}

// killDone reports the outcome and refreshes immediately after a success.
func (m *Model) killDone(msg killMsg) tea.Cmd {
	if msg.err != nil {
		return m.setStatus("kill failed: "+msg.err.Error(), true)
	}
	text := fmt.Sprintf("killed %s (pid %s) on port %d", msg.listener.Process, msg.listener.PID, msg.listener.Port)
	return tea.Batch(m.refresh(), m.setStatus(text, false))
}

// setStatus shows text and schedules its removal. The sequence number keeps an
// older timer from clearing a newer status.
func (m *Model) setStatus(text string, isErr bool) tea.Cmd {
	m.status = text
	m.statusErr = isErr
	m.statusSeq++
	m.resize()
	seq := m.statusSeq
	return tea.Tick(statusTTL, func(time.Time) tea.Msg {
		return clearStatusMsg{seq: seq}
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

// selectedListener returns the listener under the cursor.
func (m Model) selectedListener() (scan.Listener, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.listeners) {
		return scan.Listener{}, false
	}
	return m.listeners[i], true
}

// selectedKey returns the identity of the selected listener.
func (m Model) selectedKey() (rowKey, bool) {
	l, ok := m.selectedListener()
	if !ok {
		return rowKey{}, false
	}
	return keyOf(l), true
}

func keyOf(l scan.Listener) rowKey {
	return rowKey{proto: l.Proto, port: l.Port, pid: l.PID}
}

// modalView renders the confirmation box.
func (m Model) modalView() string {
	l := *m.pending
	title := fmt.Sprintf("Kill %s (pid %s) on port %d?", l.Process, l.PID, l.Port)
	content := fmt.Sprintf("%s\n\n[t] SIGTERM   [f] SIGKILL   %s", title, m.theme.Muted.Render("[esc] cancel"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(content)
}

// overlay draws content centered over the table.
func (m Model) overlay(content string) string {
	height := m.bodyHeight()
	base := lipgloss.Place(m.width, height, lipgloss.Left, lipgloss.Top, m.table.View())
	w, h := lipgloss.Width(content), lipgloss.Height(content)
	modal := lipgloss.NewLayer(content).
		X(max(0, (m.width-w)/2)).
		Y(max(0, (height-h)/2)).
		Z(1)
	return lipgloss.NewCompositor(lipgloss.NewLayer(base), modal).Render()
}

// bodyHeight is the number of lines available to the table or modal.
func (m Model) bodyHeight() int {
	h := m.height - 2
	if m.err != nil {
		h--
	}
	if m.status != "" {
		h--
	}
	return max(1, h)
}

// resize fits the table and modal between the header and footer lines.
func (m *Model) resize() {
	m.table.SetWidth(m.width)
	m.table.SetHeight(m.bodyHeight())
}

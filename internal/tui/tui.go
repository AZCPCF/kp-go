// Package tui implements the interactive kp terminal UI.
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
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

// sortMode selects the row order.
type sortMode int

const (
	sortPort sortMode = iota
	sortProcess
	sortPID
)

// String returns the name shown in the header.
func (s sortMode) String() string {
	switch s {
	case sortProcess:
		return "process"
	case sortPID:
		return "pid"
	default:
		return "port"
	}
}

// protoCycle is the order the p key cycles through.
var protoCycle = []string{scan.ProtoAll, scan.ProtoTCP, scan.ProtoUDP}

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
	scanner      Scanner
	killer       Killer
	theme        ui.Theme
	keys         keyMap
	table        table.Model
	filter       textinput.Model
	help         help.Model
	filtering    bool
	helpOpen     bool
	all          []scan.Listener
	listeners    []scan.Listener
	groups       []group.Group
	sort         sortMode
	proto        string
	grouped      bool
	pending      *scan.Listener
	pendingPorts int
	status       string
	statusErr    bool
	statusSeq    int
	err          error
	width        int
	height       int
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
	ports    int
	err      error
}

// clearStatusMsg clears the status line if it is still the current one.
type clearStatusMsg struct {
	seq int
}

// rowKey identifies a selected listener across refreshes and row rebuilds.
type rowKey struct {
	process string
	proto   string
	port    int
	pid     string
}

// New returns a Model that reads listeners from s and kills through k.
func New(s Scanner, k Killer, theme ui.Theme) Model {
	styles := table.DefaultStyles()
	styles.Header = theme.Header.Padding(0, 1)
	styles.Cell = lipgloss.NewStyle().Padding(0, 1)
	styles.Selected = theme.Port.Bold(true).Reverse(true)

	f := textinput.New()
	f.Prompt = "/ "
	f.Placeholder = "port, process or address"
	fs := textinput.DefaultStyles(true)
	fs.Focused.Text = theme.Header
	fs.Focused.Prompt = theme.Header
	fs.Focused.Placeholder = theme.Muted
	fs.Blurred = fs.Focused
	f.SetStyles(fs)

	h := help.New()
	h.Styles = help.Styles{
		Ellipsis:       theme.Muted,
		ShortKey:       theme.Header,
		ShortDesc:      theme.Muted,
		ShortSeparator: theme.Muted,
		FullKey:        theme.Header,
		FullDesc:       theme.Muted,
		FullSeparator:  theme.Muted,
	}

	m := Model{
		scanner: s,
		killer:  k,
		theme:   theme,
		keys:    defaultKeyMap(),
		table: table.New(
			table.WithColumns(flatColumns()),
			table.WithFocused(true),
			table.WithStyles(styles),
		),
		filter: f,
		help:   h,
		proto:  scan.ProtoAll,
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
		return m.handleKey(msg)
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

	// Keep forwarding the blink/paste messages the filter input relies on.
	if m.filtering {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleKey routes a key press to the overlay, filter, confirmation or table.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.helpOpen {
		switch {
		case key.Matches(msg, m.keys.Help, m.keys.Cancel):
			m.helpOpen = false
		case key.Matches(msg, m.keys.Quit, m.keys.Interrupt):
			return m, tea.Quit
		}
		return m, nil
	}

	if m.filtering {
		switch {
		case key.Matches(msg, m.keys.Interrupt):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Cancel):
			m.filter.Reset()
			m.filtering = false
			m.filter.Blur()
			m.rebuild()
			return m, nil
		case key.Matches(msg, m.keys.Accept):
			m.filtering = false
			m.filter.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.rebuild()
		return m, cmd
	}

	if m.pending != nil {
		switch {
		case key.Matches(msg, m.keys.Term):
			return m.startKill(false)
		case key.Matches(msg, m.keys.Force):
			return m.startKill(true)
		case key.Matches(msg, m.keys.Cancel):
			m.pending = nil
			m.pendingPorts = 0
		case key.Matches(msg, m.keys.Quit, m.keys.Interrupt):
			return m, tea.Quit
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Up):
		m.table.MoveUp(1)
	case key.Matches(msg, m.keys.Down):
		m.table.MoveDown(1)
	case key.Matches(msg, m.keys.Kill):
		return m, m.openConfirm()
	case key.Matches(msg, m.keys.Refresh):
		return m, m.refresh()
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		return m, m.filter.Focus()
	case key.Matches(msg, m.keys.Sort):
		m.sort = (m.sort + 1) % 3
		m.rebuild()
	case key.Matches(msg, m.keys.Group):
		prev, had := m.selectedKey()
		cursor := m.table.Cursor()
		m.grouped = !m.grouped
		m.rebuildRows(prev, had, cursor)
	case key.Matches(msg, m.keys.Proto):
		m.proto = nextProto(m.proto)
		m.rebuild()
	case key.Matches(msg, m.keys.Help):
		m.helpOpen = true
	case key.Matches(msg, m.keys.Quit, m.keys.Interrupt):
		return m, tea.Quit
	}
	return m, nil
}

// View renders the header, listener table and key hints, with the help or
// confirmation box centered over the table when open.
func (m Model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", m.theme.Header.Render(m.header()))
	if m.err != nil {
		fmt.Fprintf(&b, "%s\n", m.theme.Error.Render("scan error: "+m.err.Error()))
	}
	if m.filtering {
		b.WriteString(m.filter.View())
		b.WriteString("\n")
	}
	switch {
	case m.helpOpen:
		b.WriteString(m.overlay(m.helpView()))
	case m.pending != nil:
		b.WriteString(m.overlay(m.modalView()))
	default:
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
	b.WriteString(m.help.ShortHelpView(m.keys.ShortHelp()))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// header describes the visible rows and the active view options.
func (m Model) header() string {
	h := fmt.Sprintf("kp ui - listening ports (%d)", len(m.listeners))
	if m.grouped {
		h += "  grouped"
	}
	h += "  sort: " + m.sort.String()
	h += "  proto: " + m.proto
	if q := m.filter.Value(); q != "" && !m.filtering {
		h += fmt.Sprintf("  filter: %q", q)
	}
	return h
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
	l, ports, ok := m.selectedTarget()
	if !ok {
		return nil
	}
	if l.PID == "-" {
		return m.setStatus(fmt.Sprintf("pid unknown on port %d: run with sudo to see the owner", l.Port), true)
	}
	m.pending = &l
	m.pendingPorts = ports
	return nil
}

// startKill closes the confirmation and runs the kill as a command.
func (m Model) startKill(force bool) (tea.Model, tea.Cmd) {
	if m.pending == nil {
		return m, nil
	}
	l := *m.pending
	ports := m.pendingPorts
	m.pending = nil
	m.pendingPorts = 0
	return m, m.kill(l, force, ports)
}

// kill terminates l in a command and reports the result as a killMsg.
func (m Model) kill(l scan.Listener, force bool, ports int) tea.Cmd {
	k := m.killer
	return func() tea.Msg {
		return killMsg{listener: l, force: force, ports: ports, err: k.Kill(l, force)}
	}
}

// killDone reports the outcome and refreshes immediately after a success.
func (m *Model) killDone(msg killMsg) tea.Cmd {
	if msg.err != nil {
		return m.setStatus("kill failed: "+msg.err.Error(), true)
	}
	text := fmt.Sprintf("killed %s (pid %s) on %s", msg.listener.Process, msg.listener.PID, portLabel(msg.ports, msg.listener.Port))
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

// apply replaces the scanned listeners and rebuilds the visible rows.
func (m *Model) apply(msg refreshMsg) {
	if msg.err != nil {
		m.err = msg.err
		m.resize()
		return
	}

	m.err = nil
	m.all = append([]scan.Listener(nil), msg.listeners...)
	m.rebuild()
}

// rebuild reapplies the filters, sort and view mode, keeping the selected row
// when it still exists.
func (m *Model) rebuild() {
	prev, had := m.selectedKey()
	m.rebuildRows(prev, had, m.table.Cursor())
}

// rebuildRows derives the visible listeners from the raw scan and fills the
// table with either flat listener rows or one row per process group.
func (m *Model) rebuildRows(prev rowKey, had bool, cursor int) {
	query := m.filter.Value()
	ls := make([]scan.Listener, 0, len(m.all))
	for _, l := range m.all {
		if m.proto != scan.ProtoAll && l.Proto != m.proto {
			continue
		}
		if !matchesFilter(l, query) {
			continue
		}
		ls = append(ls, l)
	}
	sortListeners(ls, m.sort)
	m.listeners = ls

	var rows []table.Row
	if m.grouped {
		m.groups = group.ByProcess(ls)
		rows = make([]table.Row, len(m.groups))
		for i, g := range m.groups {
			rows[i] = table.Row{g.Process, g.PID, joinedPorts(g.Listeners)}
		}
	} else {
		m.groups = nil
		rows = make([]table.Row, len(ls))
		for i, l := range ls {
			rows[i] = table.Row{
				strconv.Itoa(l.Port),
				l.Proto,
				l.Addr,
				l.Process,
				l.PID,
			}
		}
	}
	m.setColumns()
	m.table.SetRows(rows)
	m.restoreSelection(prev, had, cursor)
	m.resize()
}

// setColumns switches the table columns when the view mode changes, clearing
// the old rows first so they are never rendered against the new columns.
func (m *Model) setColumns() {
	want, cols := 5, flatColumns()
	if m.grouped {
		want, cols = 3, groupColumns()
	}
	if len(m.table.Columns()) == want {
		return
	}
	m.table.SetRows(nil)
	m.table.SetColumns(cols)
}

// restoreSelection moves the cursor back to prev when it is still visible and
// otherwise puts it back at cursor, clamped to the new list.
func (m *Model) restoreSelection(prev rowKey, had bool, cursor int) {
	n := m.rowCount()
	if had {
		for i := 0; i < n; i++ {
			if m.rowKey(i) == prev {
				m.table.SetCursor(i)
				return
			}
		}
	}
	if n == 0 {
		return
	}
	m.table.SetCursor(min(max(cursor, 0), n-1))
}

// rowCount is the number of visible table rows.
func (m Model) rowCount() int {
	if m.grouped {
		return len(m.groups)
	}
	return len(m.listeners)
}

// rowKey returns the identity of a visible row.
func (m Model) rowKey(i int) rowKey {
	if m.grouped {
		g := m.groups[i]
		return rowKey{process: g.Process, pid: g.PID}
	}
	return keyOf(m.listeners[i])
}

// selectedTarget returns the listener behind the cursor, plus how many ports
// it covers (more than one for a process group).
func (m Model) selectedTarget() (scan.Listener, int, bool) {
	i := m.table.Cursor()
	if m.grouped {
		if i < 0 || i >= len(m.groups) {
			return scan.Listener{}, 0, false
		}
		g := m.groups[i]
		return g.Listeners[0], len(g.Listeners), true
	}
	if i < 0 || i >= len(m.listeners) {
		return scan.Listener{}, 0, false
	}
	return m.listeners[i], 1, true
}

// selectedKey returns the identity of the selected row.
func (m Model) selectedKey() (rowKey, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= m.rowCount() {
		return rowKey{}, false
	}
	return m.rowKey(i), true
}

func keyOf(l scan.Listener) rowKey {
	return rowKey{process: l.Process, proto: l.Proto, port: l.Port, pid: l.PID}
}

// matchesFilter reports whether l matches query on port, process or address.
func matchesFilter(l scan.Listener, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	return strings.Contains(strconv.Itoa(l.Port), q) ||
		strings.Contains(strings.ToLower(l.Process), q) ||
		strings.Contains(strings.ToLower(l.Addr), q)
}

// sortListeners orders ls in place using mode.
func sortListeners(ls []scan.Listener, mode sortMode) {
	switch mode {
	case sortProcess:
		group.SortByProcess(ls)
	case sortPID:
		group.SortByPID(ls)
	default:
		group.SortByPort(ls)
	}
}

// nextProto returns the protocol after p in the all/tcp/udp cycle.
func nextProto(p string) string {
	for i, v := range protoCycle {
		if v == p {
			return protoCycle[(i+1)%len(protoCycle)]
		}
	}
	return protoCycle[0]
}

// portLabel describes the kill target for the modal and status line.
func portLabel(ports, port int) string {
	if ports > 1 {
		return fmt.Sprintf("%d ports", ports)
	}
	return fmt.Sprintf("port %d", port)
}

func flatColumns() []table.Column {
	return []table.Column{
		{Title: "PORT", Width: widthPort},
		{Title: "PROTO", Width: widthProto},
		{Title: "ADDR", Width: widthAddr},
		{Title: "PROCESS", Width: widthProcess},
		{Title: "PID", Width: widthPID},
	}
}

func groupColumns() []table.Column {
	return []table.Column{
		{Title: "PROCESS", Width: widthProcess},
		{Title: "PID", Width: widthPID},
		{Title: "PORTS", Width: widthAddr},
	}
}

func joinedPorts(ls []scan.Listener) string {
	ports := make([]string, len(ls))
	for i, l := range ls {
		ports[i] = strconv.Itoa(l.Port)
	}
	return strings.Join(ports, ", ")
}

// helpView renders the full key reference in a bordered box.
func (m Model) helpView() string {
	content := m.theme.Header.Render("Keys") + "\n\n" + m.help.FullHelpView(m.keys.FullHelp())
	content += "\n\n" + m.theme.Muted.Render("? or esc to close")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(content)
}

// modalView renders the confirmation box.
func (m Model) modalView() string {
	l := *m.pending
	title := fmt.Sprintf("Kill %s (pid %s) on %s?", l.Process, l.PID, portLabel(m.pendingPorts, l.Port))
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
	if m.filtering {
		h--
	}
	if m.err != nil {
		h--
	}
	if m.status != "" {
		h--
	}
	return max(1, h)
}

// resize fits the table and overlays between the header and footer lines.
func (m *Model) resize() {
	m.table.SetWidth(m.width)
	m.table.SetHeight(m.bodyHeight())
	m.help.SetWidth(m.width)
	m.filter.SetWidth(max(10, m.width-4))
}

package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/azcpcf/kp-go/internal/ui"
)

var errBoom = errors.New("boom")

var testListeners = []scan.Listener{
	{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
	{Addr: "127.0.0.1", Port: 5432, Proto: "tcp", Process: "postgres", PID: "20"},
	{Addr: "0.0.0.0", Port: 5353, Proto: "udp", Process: "avahi", PID: "30"},
}

type fakeScanner struct {
	listeners []scan.Listener
	err       error
	calls     int
}

func (f *fakeScanner) List() ([]scan.Listener, error) {
	f.calls++
	return f.listeners, f.err
}

type killCall struct {
	listener scan.Listener
	force    bool
}

type fakeKiller struct {
	calls []killCall
	err   error
}

func (f *fakeKiller) Kill(l scan.Listener, force bool) error {
	f.calls = append(f.calls, killCall{listener: l, force: force})
	return f.err
}

func newTestModel() (Model, *fakeScanner, *fakeKiller) {
	fs := &fakeScanner{listeners: testListeners}
	fk := &fakeKiller{}
	return New(fs, fk, ui.New(false)), fs, fk
}

// update feeds msg to the model and returns the concrete Model type.
func update(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	tm, cmd := m.Update(msg)
	got, ok := tm.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", tm)
	}
	return got, cmd
}

func applyRefresh(t *testing.T, m Model, ls []scan.Listener) Model {
	t.Helper()
	m, _ = update(t, m, refreshMsg{listeners: ls})
	return m
}

func press(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text}
}

func TestNavigation(t *testing.T) {
	m, _, _ := newTestModel()
	m = applyRefresh(t, m, testListeners)

	if got := m.table.Cursor(); got != 0 {
		t.Fatalf("initial cursor = %d, want 0", got)
	}

	steps := []struct {
		name string
		msg  tea.KeyPressMsg
		want int
	}{
		{"j", press('j', "j"), 1},
		{"down", press(tea.KeyDown, ""), 2},
		{"down clamps", press(tea.KeyDown, ""), 2},
		{"k", press('k', "k"), 1},
		{"up", press(tea.KeyUp, ""), 0},
		{"up clamps", press(tea.KeyUp, ""), 0},
	}
	for _, step := range steps {
		m, _ = update(t, m, step.msg)
		if got := m.table.Cursor(); got != step.want {
			t.Errorf("%s: cursor = %d, want %d", step.name, got, step.want)
		}
	}
}

func TestRefreshPreservesSelection(t *testing.T) {
	m, _, _ := newTestModel()
	m = applyRefresh(t, m, testListeners)

	m, _ = update(t, m, press(tea.KeyDown, ""))
	m, _ = update(t, m, press(tea.KeyDown, ""))
	if got := m.table.Cursor(); got != 2 {
		t.Fatalf("cursor after j j = %d, want 2", got)
	}

	refreshed := []scan.Listener{
		{Addr: "0.0.0.0", Port: 9000, Proto: "tcp", Process: "new", PID: "99"},
		testListeners[1],
		testListeners[2],
		testListeners[0],
	}
	m = applyRefresh(t, m, refreshed)

	want := keyOf(testListeners[1])
	if got := keyOf(m.listeners[m.table.Cursor()]); got != want {
		t.Errorf("selected key = %+v, want %+v", got, want)
	}
	if row := m.table.SelectedRow(); row[3] != "postgres" {
		t.Errorf("selected row = %v, want postgres", row)
	}
	if len(m.table.Rows()) != len(refreshed) {
		t.Errorf("rows = %d, want %d", len(m.table.Rows()), len(refreshed))
	}
}

func TestRefreshSelectedRowGone(t *testing.T) {
	m, _, _ := newTestModel()
	m = applyRefresh(t, m, testListeners)

	m, _ = update(t, m, press(tea.KeyDown, ""))
	if got := m.table.Cursor(); got != 1 {
		t.Fatalf("cursor = %d, want 1", got)
	}

	m = applyRefresh(t, m, testListeners[:2])

	if got := m.table.Cursor(); got != 1 {
		t.Errorf("cursor after removal = %d, want 1", got)
	}
	if row := m.table.SelectedRow(); row[3] != "postgres" {
		t.Errorf("selected row = %v, want postgres", row)
	}
}

func TestRefreshEmpty(t *testing.T) {
	m, _, _ := newTestModel()
	m = applyRefresh(t, m, nil)

	if m.table.SelectedRow() != nil {
		t.Errorf("selected row = %v, want nil", m.table.SelectedRow())
	}
	m, _ = update(t, m, press('j', "j"))
	if got := m.table.Cursor(); got != -1 {
		t.Errorf("cursor = %d, want -1 for no rows", got)
	}
}

func TestWindowSize(t *testing.T) {
	m, _, _ := newTestModel()
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if m.width != 100 || m.height != 30 {
		t.Errorf("size = %dx%d, want 100x30", m.width, m.height)
	}
	if got := m.table.Width(); got != 100 {
		t.Errorf("table width = %d, want 100", got)
	}
	// The table reserves one line for its header inside the 28-line area.
	if got := m.table.Height(); got != 27 {
		t.Errorf("table height = %d, want 27", got)
	}
}

func TestQuitKeys(t *testing.T) {
	keys := []tea.KeyPressMsg{
		press('q', "q"),
		{Code: 'c', Mod: tea.ModCtrl},
	}
	for _, msg := range keys {
		m, _, _ := newTestModel()
		_, cmd := update(t, m, msg)
		if cmd == nil {
			t.Fatalf("%q: quit command is nil", msg.String())
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%q: command did not return tea.QuitMsg", msg.String())
		}
	}
}

func TestRefreshKeyRunsScanner(t *testing.T) {
	m, fs, _ := newTestModel()
	_, cmd := update(t, m, press('r', "r"))
	if cmd == nil {
		t.Fatal("refresh command is nil")
	}

	msg := cmd()
	rm, ok := msg.(refreshMsg)
	if !ok {
		t.Fatalf("command returned %T, want refreshMsg", msg)
	}
	if fs.calls != 1 {
		t.Errorf("scanner calls = %d, want 1", fs.calls)
	}
	if rm.err != nil || len(rm.listeners) != len(testListeners) {
		t.Errorf("refresh = %+v, want %d listeners", rm, len(testListeners))
	}
}

func TestScanErrorShown(t *testing.T) {
	m, fs, _ := newTestModel()
	m = applyRefresh(t, m, testListeners)

	fs.err = errBoom
	_, cmd := update(t, m, press('r', "r"))
	msg, ok := cmd().(refreshMsg)
	if !ok {
		t.Fatalf("command returned %T, want refreshMsg", msg)
	}
	m, _ = update(t, m, msg)

	if m.err == nil || !strings.Contains(m.err.Error(), "boom") {
		t.Errorf("model error = %v, want boom", m.err)
	}
	if len(m.listeners) != len(testListeners) {
		t.Errorf("listeners = %d, want the previous %d", len(m.listeners), len(testListeners))
	}
	view := m.View().Content
	if !strings.Contains(view, "scan error: boom") {
		t.Errorf("view missing error line:\n%s", view)
	}
	if !strings.Contains(view, "node") {
		t.Errorf("view dropped previous rows:\n%s", view)
	}
}

func TestView(t *testing.T) {
	m, _, _ := newTestModel()
	m = applyRefresh(t, m, testListeners)

	view := m.View()
	for _, want := range []string{
		"kp ui",
		"listening ports (3)",
		"PORT", "PROTO", "ADDR", "PROCESS", "PID",
		"node", "postgres", "avahi",
		"x kill", "r refresh", "q quit",
	} {
		if !strings.Contains(view.Content, want) {
			t.Errorf("view missing %q:\n%s", want, view.Content)
		}
	}
	if !view.AltScreen {
		t.Error("view should use the alternate screen")
	}
}

func TestInit(t *testing.T) {
	m, fs, _ := newTestModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil, want the scan and tick commands")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init command returned %T, want tea.BatchMsg", cmd())
	}
	if len(batch) != 2 {
		t.Fatalf("Init batch has %d commands, want 2", len(batch))
	}
	if _, ok := batch[0]().(refreshMsg); !ok {
		t.Errorf("first Init command did not return a refreshMsg")
	}
	if fs.calls != 1 {
		t.Errorf("scanner calls = %d, want 1", fs.calls)
	}
}

func openModal(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = update(t, m, press('x', "x"))
	if m.pending == nil {
		t.Fatal("modal did not open")
	}
	return m
}

func TestKillModalOpens(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{press('x', "x"), press('d', "d")} {
		m, _, fk := newTestModel()
		m = applyRefresh(t, m, testListeners)

		m, cmd := update(t, m, key)
		if cmd != nil {
			t.Errorf("%q: modal opened with a command", key.String())
		}
		if m.pending == nil {
			t.Fatalf("%q: modal did not open", key.String())
		}
		if len(fk.calls) != 0 {
			t.Errorf("%q: killer called before confirmation", key.String())
		}
		view := m.View().Content
		for _, want := range []string{
			"Kill node (pid 10) on port 3000?",
			"[t] SIGTERM", "[f] SIGKILL", "[esc] cancel",
			"PORT", "node",
		} {
			if !strings.Contains(view, want) {
				t.Errorf("%q: view missing %q:\n%s", key.String(), want, view)
			}
		}
	}
}

func TestKillModalCancel(t *testing.T) {
	m, _, fk := newTestModel()
	m = applyRefresh(t, m, testListeners)
	m = openModal(t, m)

	m, cmd := update(t, m, press(tea.KeyEsc, ""))

	if cmd != nil {
		t.Errorf("cancel returned a command")
	}
	if m.pending != nil {
		t.Error("cancel left the modal open")
	}
	if len(fk.calls) != 0 {
		t.Errorf("cancel called the killer %d times", len(fk.calls))
	}
	if m.status != "" {
		t.Errorf("cancel set status %q", m.status)
	}
}

func TestKillConfirm(t *testing.T) {
	tests := []struct {
		name  string
		key   tea.KeyPressMsg
		force bool
	}{
		{"sigterm", press('t', "t"), false},
		{"sigkill", press('f', "f"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, fs, fk := newTestModel()
			m = applyRefresh(t, m, testListeners)
			m = openModal(t, m)

			m, cmd := update(t, m, tt.key)
			if m.pending != nil {
				t.Fatal("confirmation did not close the modal")
			}
			if cmd == nil {
				t.Fatal("confirmation returned no kill command")
			}
			msg, ok := cmd().(killMsg)
			if !ok {
				t.Fatalf("kill command returned %T, want killMsg", msg)
			}
			if msg.force != tt.force {
				t.Errorf("killMsg force = %v, want %v", msg.force, tt.force)
			}
			if len(fk.calls) != 1 {
				t.Fatalf("killer calls = %d, want 1", len(fk.calls))
			}
			if got := fk.calls[0]; got.force != tt.force || got.listener.Port != 3000 {
				t.Errorf("kill call = %+v, want port 3000 force %v", got, tt.force)
			}

			m, cmd = update(t, m, msg)
			if m.statusErr {
				t.Errorf("status is an error: %q", m.status)
			}
			if want := "killed node (pid 10) on port 3000"; m.status != want {
				t.Errorf("status = %q, want %q", m.status, want)
			}

			batch, ok := cmd().(tea.BatchMsg)
			if !ok {
				t.Fatalf("result command returned %T, want tea.BatchMsg", cmd())
			}
			if len(batch) != 2 {
				t.Fatalf("result batch has %d commands, want refresh and clear timer", len(batch))
			}
			if _, ok := batch[0]().(refreshMsg); !ok {
				t.Error("success did not schedule an immediate refresh")
			}
			if fs.calls != 1 {
				t.Errorf("scanner calls = %d, want 1", fs.calls)
			}
		})
	}
}

func TestKillError(t *testing.T) {
	m, fs, fk := newTestModel()
	m = applyRefresh(t, m, testListeners)
	fk.err = errBoom
	m = openModal(t, m)

	m, cmd := update(t, m, press('t', "t"))
	msg, ok := cmd().(killMsg)
	if !ok {
		t.Fatalf("kill command returned %T, want killMsg", msg)
	}
	if msg.err == nil {
		t.Fatal("killMsg error = nil, want boom")
	}

	m, cmd = update(t, m, msg)

	if !m.statusErr {
		t.Errorf("status %q is not marked as an error", m.status)
	}
	if want := "kill failed: boom"; m.status != want {
		t.Errorf("status = %q, want %q", m.status, want)
	}
	if fs.calls != 0 {
		t.Errorf("failed kill triggered a refresh: scanner calls = %d", fs.calls)
	}
	if cmd == nil {
		t.Error("failed kill did not schedule the status clear")
	}
	if view := m.View().Content; !strings.Contains(view, "kill failed: boom") {
		t.Errorf("view missing error status:\n%s", view)
	}
}

func TestKillUnknownPID(t *testing.T) {
	m, _, fk := newTestModel()
	m = applyRefresh(t, m, []scan.Listener{
		{Addr: "0.0.0.0", Port: 22, Proto: "tcp", Process: "-", PID: "-"},
	})

	m, cmd := update(t, m, press('x', "x"))

	if m.pending != nil {
		t.Error("unknown PID opened the modal")
	}
	if len(fk.calls) != 0 {
		t.Errorf("unknown PID called the killer %d times", len(fk.calls))
	}
	if cmd == nil {
		t.Error("unknown PID did not schedule the status clear")
	}
	if !m.statusErr {
		t.Errorf("status %q is not marked as an error", m.status)
	}
	if want := "run with sudo to see the owner"; !strings.Contains(m.status, want) {
		t.Errorf("status = %q, want it to contain %q", m.status, want)
	}
	view := m.View().Content
	if strings.Contains(view, "SIGTERM") {
		t.Errorf("unknown PID rendered the modal:\n%s", view)
	}
	if !strings.Contains(view, "run with sudo to see the owner") {
		t.Errorf("view missing unknown PID status:\n%s", view)
	}
}

func TestKillStatusAutoClears(t *testing.T) {
	m, _, _ := newTestModel()
	m = applyRefresh(t, m, testListeners)
	m = openModal(t, m)

	m, cmd := update(t, m, press('t', "t"))
	m, _ = update(t, m, cmd().(killMsg))
	if m.status == "" {
		t.Fatal("success did not set a status")
	}

	seq := m.statusSeq
	m, _ = update(t, m, clearStatusMsg{seq: seq - 1})
	if m.status == "" {
		t.Error("a stale clear timer removed the status")
	}

	m, _ = update(t, m, clearStatusMsg{seq: seq})
	if m.status != "" {
		t.Errorf("status = %q, want it cleared", m.status)
	}
}

func TestTickSchedulesRefresh(t *testing.T) {
	m, fs, _ := newTestModel()
	_, cmd := update(t, m, tickMsg{})
	if cmd == nil {
		t.Fatal("tick command is nil")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("tick command returned %T, want tea.BatchMsg", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("tick batch has %d commands, want refresh and tick", len(batch))
	}
	if _, ok := batch[0]().(refreshMsg); !ok {
		t.Errorf("tick did not schedule a refresh")
	}
	if fs.calls != 1 {
		t.Errorf("scanner calls = %d, want 1", fs.calls)
	}
}

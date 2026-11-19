package tui

import "charm.land/bubbles/v2/key"

// keyMap holds every key binding used by the TUI. Keeping them in one place
// keeps the help output in sync with the actual controls.
type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Kill      key.Binding
	Refresh   key.Binding
	Filter    key.Binding
	Accept    key.Binding
	Sort      key.Binding
	Group     key.Binding
	Proto     key.Binding
	Help      key.Binding
	Quit      key.Binding
	Interrupt key.Binding
	Term      key.Binding
	Force     key.Binding
	Cancel    key.Binding
}

// defaultKeyMap returns the bindings shown by the short and full help.
func defaultKeyMap() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Kill:      key.NewBinding(key.WithKeys("x", "d"), key.WithHelp("x", "kill")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Accept:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply filter")),
		Sort:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Group:     key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "group")),
		Proto:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "proto")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Interrupt: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Term:      key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "SIGTERM")),
		Force:     key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "SIGKILL")),
		Cancel:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear/cancel")),
	}
}

// ShortHelp is the one-line key hint shown in the footer.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Filter, k.Sort, k.Group, k.Proto, k.Help, k.Kill, k.Refresh, k.Quit,
	}
}

// FullHelp is the grouped key reference shown by the help overlay.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Filter, k.Accept, k.Cancel},
		{k.Sort, k.Group, k.Proto, k.Refresh, k.Help},
		{k.Kill, k.Term, k.Force, k.Quit, k.Interrupt},
	}
}

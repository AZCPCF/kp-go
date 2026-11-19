Task: make the TUI pleasant to navigate.

- `/` opens a filter input (bubbles/textinput) that filters rows live by port, process name, or address. esc clears it.
- `s` cycles sort: port, process, pid. Show the active sort in the header.
- `g` toggles grouped view (reuse internal/group) and back.
- `p` cycles protocol filter: all, tcp, udp.
- `?` toggles a full help overlay (bubbles/help + key.Binding for every key). Footer shows a short help.
- Define all keys in one `keys.go` using key.Binding so help stays in sync.
- Tests: filtering, sort cycling, group toggle, protocol cycling, and that selection stays valid when the filter shrinks the list.

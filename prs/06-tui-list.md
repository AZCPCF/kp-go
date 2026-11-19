Task: add `kp ui` (alias: `kp tui`), an interactive terminal UI. Read-only in this PR.

- Add github.com/charmbracelet/bubbletea and github.com/charmbracelet/bubbles (verify imports/APIs with `go doc`; use matching major versions that work together).
- Package internal/tui. A `Model` struct with Init/Update/View.
- Show a table (bubbles/table) with columns PORT, PROTO, ADDR, PROCESS, PID, using the theme from internal/ui.
- Refresh every 2 seconds via a tick command. Preserve the selected row across refreshes (match by proto+port+pid).
- Keys: up/down and j/k to move, q or ctrl+c to quit, r to refresh now.
- Header line with title and total count. Footer with key hints.
- Scanning happens through an interface (`type Scanner interface { List() ([]scan.Listener, error) }`) so tests can inject fakes.
- Tests: feed messages to Update (key presses, a refresh result message, a window size message) and assert model state: cursor position, rows, selection preserved after refresh, quit command.
- Handle scan errors by showing an error line, not crashing.

Task: add killing from the TUI.

- Key `x` (or `d`) on the selected row opens a confirmation modal: "Kill <process> (pid N) on port P?" with choices
  [t] SIGTERM  [f] SIGKILL  [esc] cancel. Style it as a centered bordered box (lipgloss).
- Killing runs as a tea.Cmd using the existing internal/kill Killer (injectable). Result arrives as a message.
- Show a status line after the action (green success or red error, auto-clears after 3 seconds).
- Rows with unknown PID ("-") show an explanatory error instead of the modal ("run with sudo to see the owner").
- After a successful kill, trigger an immediate refresh.
- Tests: Update-level tests with a fake killer: open modal, cancel, confirm term, confirm force, error path, unknown PID path.

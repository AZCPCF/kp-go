Task: add colorful static output.

- Add github.com/charmbracelet/lipgloss (use the latest stable major; verify import path and API with `go doc`).
- Create internal/ui/theme.go with a small palette and named styles (header, port, process, muted, error, success).
- Style the table output: bold header, colored ports, muted addresses, distinct color per protocol.
- Style kill output (green success line, red error line).
- Color must be disabled when NO_COLOR is set, when stdout is not a TTY, or when `--color=never`.
  Add a global `--color` flag with auto|always|never.
- Keep --json output completely unstyled.
- Tests: with color disabled, output must equal the old plain format (assert exact strings).
  Test the color-mode decision function separately (env, tty, flag combinations).

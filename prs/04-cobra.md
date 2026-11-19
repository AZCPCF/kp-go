Task: replace the `flag` package with Cobra, keeping all current behavior.

Commands:
  kp                      list (default), flags: -g/--group, -p/--port, --proto, --json
  kp kill <port...>       flags: -f/--force
  kp <port...>            shorthand for `kp kill <port...>` (must keep working)
  kp version              prints "kp <version>"
  kp --version            same

Requirements:
- Add github.com/spf13/cobra. Verify the API with `go doc` before using it.
- Commands go in internal/cli (one file per command). cmd/kp/main.go only calls cli.Execute().
- Cobra's built-in `completion` command stays enabled.
- Handle the `kp 3000` shorthand explicitly (numeric first arg routes to kill).
- Exit codes: 0 success, 1 runtime error, 2 usage error.
- Tests: execute commands through cobra with buffers for out/err and a fake Killer.

Task: show details for the selected process.

- In internal/scan (or internal/procinfo), add `Info(pid int) (ProcInfo, error)` reading /proc/<pid>/:
  cmdline (NUL-separated, join with spaces), user (uid from /proc/<pid>/status mapped via os/user, fallback to the numeric uid),
  start time / uptime (from /proc/<pid>/stat with btime from /proc/stat), RSS memory (VmRSS in /proc/<pid>/status), thread count.
- Put the /proc root path behind a variable or parameter so tests can read fixture files from testdata/.
- TUI: key `enter` or `i` toggles a details pane (right side or below, depending on width) for the selected row.
- CLI: `kp info <port>` prints the same details, with `--json` support.
- Missing permission or vanished process must produce a clear partial result, not a crash.
- Tests: parser tests against fixture files in testdata/, plus pane toggle tests.

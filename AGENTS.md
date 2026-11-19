# kp: project rules

## What this is
`kp` is a Linux CLI that lists listening TCP/UDP ports and kills the processes on them.
Final goal: a complete CLI plus a friendly, colorful terminal UI (Bubble Tea).

## Current state
- Go, single `package main` in the repo root (will be restructured in PR 1)
- `proc.go`: reads /proc/net/tcp{,6} and maps socket inodes to PIDs
- `scan.go`: `listListeners()` tries /proc first, falls back to `ss -tlnpH`
- `group.go`: sortByPort, filterByPort, groupByProcess
- `render.go`: plain table output (tabwriter)
- `kill.go`: `Killer` struct with an injectable `Signal` func (SIGTERM, optional SIGKILL, waits for exit)
- Tests: table-driven, use fakes, never touch real processes
- CI: GitHub Actions (vet, gofmt, test -race). Releases: GoReleaser on `v*` tags, linux amd64/arm64

## Hard rules
1. One task = one PR-sized change. Do ONLY what the prompt asks. Do not refactor unrelated code.
2. Before finishing, run `make check`. It must pass. Never say "done" if it fails.
3. Never guess a library API. After `go get`, verify signatures with `go doc <pkg> <Symbol>`
   or by reading the module source in the module cache. Use the latest stable major version
   and use the exact import path that version documents.
4. Keep business logic separate from I/O and UI. Functions take `io.Writer` / interfaces, not `os.Stdout`.
5. Never kill real processes in tests. Use the fakes pattern from kill_test.go.
6. Every new behavior needs tests. For the TUI, test the model's `Update` with messages and
   assert on the resulting model state. Do not test by rendering to a real terminal.
7. Errors: wrap with `%w`, lowercase messages, no panics in normal paths.
8. Respect `NO_COLOR` and disable colors when stdout is not a TTY.
9. Linux-only code goes in files ending `_linux.go`. Do not break `go vet` on other OSes once PR 10 lands.
10. Commit messages: Conventional Commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `ci:`).
11. No new dependency unless the task allows it. List any dependency you add in your final message.

## Style
- Standard library first. gofmt. Small functions. Comments only where intent is not obvious.
- Exported identifiers get a one-line doc comment once code moves into `internal/`.

## Final message format
End every task with: files changed, dependencies added, `make check` result, and anything uncertain.

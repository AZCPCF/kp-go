Task: restructure the repo without changing behavior.

Target layout:
  cmd/kp/main.go          flag parsing and wiring only
  internal/scan/          Listener type, listListeners, proc.go, ss fallback (+ tests)
  internal/group/         sortByPort, filterByPort, groupByProcess, Group type (+ tests)
  internal/render/        printTable, printGrouped (+ tests if any)
  internal/kill/          Killer, runKill (+ tests)

Requirements:
- Export what other packages need, with one-line doc comments.
- Update .goreleaser.yaml: builds[0].main must be ./cmd/kp, ldflags still set main.version.
- Update the Makefile build target to ./cmd/kp.
- All existing tests must pass, moved with their code. No behavior change.
- `go run ./cmd/kp`, `go run ./cmd/kp -g`, `go run ./cmd/kp -p 3000` behave as before.
- Do NOT add dependencies.

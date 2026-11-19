Task: add `--json` output.

- Add a `--json` flag. With it, the list and grouped views print JSON to stdout instead of a table.
- Listener JSON fields: addr, port, process, pid (pid as a number, or null when unknown).
- Grouped JSON: array of {process, pid, ports:[...]}.
- Implement in internal/render with io.Writer. Use encoding/json with struct tags and indentation.
- Empty result prints `[]`, never `null`.
- Tests: golden-style tests comparing exact output for list, grouped, and empty cases.
- No new dependencies.

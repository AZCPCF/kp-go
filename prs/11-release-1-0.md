Task: release polish.

- README.md: install instructions, usage with examples for every command, TUI key table, platform support, FAQ about sudo and unknown PIDs.
- Add a `vhs` tape file (tapes/demo.tape) to record a demo GIF. Do not generate the GIF in this task.
- GoReleaser: add shell completions to the archives (generate with `kp completion`), add a Homebrew tap section commented out with instructions.
- `kp` with no arguments in a TTY keeps listing; add `--no-header` for scripting.
- Make sure `kp --help` text is consistent and has examples in each subcommand's `Example` field.
- Audit: run `go vet ./...`, `go test -race ./...`, and check there is no leftover TODO or dead code.

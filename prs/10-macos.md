Task: make kp work on macOS.

- Split OS-specific code behind build tags: scan_linux.go / scan_darwin.go (and stubs elsewhere).
- Darwin implementation: run `lsof -nP -iTCP -sTCP:LISTEN -FpcnP` (and -iUDP for UDP) and parse the field-format output (-F).
- Process info on darwin: use `ps -o ...` or leave fields empty gracefully.
- Signals use syscall.Kill on both Linux and Darwin. Windows is explicitly unsupported (build constraint with a clear error message, or just do not build it).
- Add darwin amd64/arm64 to .goreleaser.yaml.
- CI: add a macOS job (`macos-latest`) that runs `go vet` and `go test`.
- Tests: parser tests on lsof -F fixture strings. No test may call real lsof.

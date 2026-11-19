Task: support UDP listeners.

- Extend Listener with a Proto field ("tcp" or "udp").
- Parse /proc/net/udp and /proc/net/udp6. UDP "listening" sockets are state 07 (TCP LISTEN is 0A).
  Reuse the existing hex address parsing.
- Add `--proto` flag: tcp (default), udp, or all. Show a PROTO column in the table and a proto field in JSON.
- Update the ss fallback to use `ss -tulnpH` when proto includes udp (use -t or -u as appropriate).
- Tests: add a sample /proc/net/udp fixture string and table-driven tests for state filtering.
- kill must work for UDP ports too: `kp 5353` kills whichever process owns that port in the selected proto.
- No new dependencies.

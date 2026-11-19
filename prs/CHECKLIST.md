# Review checklist (for you, do NOT paste to the agent)

Loop per PR: new branch -> opencode Plan mode (Tab) -> read plan -> Build mode -> `make check` -> read `git diff main` -> manual test -> push -> green CI -> squash merge.

01 package-layout  : diff is mostly renames, no logic changes, tests moved not deleted.
                     Test: kp, kp -g, kp -p 3000 behave as before.
02 json-output     : empty list prints [], unknown pid is null.
                     Test: kp --json | jq . ; kp --json -g
03 udp             : UDP uses state 07 (not 0A). Compare counts with ss -ulnpH.
                     Test: kp --proto udp ; kp --proto all
04 cobra           : kp 3000 -f and kp -f 3000 both work, no leftover flag package.
                     Test: kp --help ; kp kill 99999 ; kp completion bash | head
05 styled-output   : no escape codes when piped, JSON never styled.
                     Test: kp | cat -v ; NO_COLOR=1 kp
06 tui-list        : scanning runs inside a tea.Cmd, never inside Update/View.
                     Test: kp ui, start python3 -m http.server 8123 elsewhere, it appears in ~2s, resize window.
07 tui-kill        : modal cannot be confirmed by a stray key, kill never runs directly in Update.
                     Test: kill a throwaway server with t; force path with a SIGTERM-ignoring process.
08 tui-navigation  : all keys only in keys.go, cursor never past the end after filtering.
                     Test: filter "node", toggle group, open ?
09 process-details : /proc/<pid>/stat parsed from the LAST ")" (names can contain spaces/parens).
                     Test: kp info 3000 ; open pane on a root-owned process without sudo.
10 macos           : GOOS=darwin go vet ./... passes on Linux, Linux path untouched. Rely on macOS CI.
11 release-1-0     : record the GIF yourself (vhs tapes/demo.tape), then tag v1.0.0.

Always watch: kill logic, /proc parsing, anything that blocks inside Update.
When the agent repeats a mistake, add one rule line to AGENTS.md.

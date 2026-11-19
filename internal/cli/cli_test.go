package cli

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/azcpcf/kp-go/internal/kill"
	"github.com/azcpcf/kp-go/internal/scan"
)

var errBoom = errors.New("boom")

type fakeScan struct {
	ls     []scan.Listener
	err    error
	protos []string
}

func (f *fakeScan) scan(proto string) ([]scan.Listener, error) {
	f.protos = append(f.protos, proto)
	return f.ls, f.err
}

type signalCall struct {
	pid int
	sig syscall.Signal
}

type fakeProc struct {
	alive map[int]bool
	calls []signalCall
}

func (f *fakeProc) signal(pid int, sig syscall.Signal) error {
	f.calls = append(f.calls, signalCall{pid, sig})
	if !f.alive[pid] {
		return syscall.ESRCH
	}
	if sig != 0 {
		f.alive[pid] = false
	}
	return nil
}

func (f *fakeProc) kills() []signalCall {
	var res []signalCall
	for _, c := range f.calls {
		if c.sig != 0 {
			res = append(res, c)
		}
	}
	return res
}

var testListeners = []scan.Listener{
	{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
	{Addr: "127.0.0.1", Port: 5432, Proto: "tcp", Process: "postgres", PID: "20"},
}

func testOptions(ls []scan.Listener) (options, *fakeScan, *fakeProc) {
	fs := &fakeScan{ls: ls}
	proc := &fakeProc{alive: map[int]bool{}}
	for _, l := range ls {
		if pid, err := strconv.Atoi(l.PID); err == nil {
			proc.alive[pid] = true
		}
	}
	return options{
		scan:    fs.scan,
		killer:  &kill.Killer{Signal: proc.signal, Wait: 10 * time.Millisecond, Poll: time.Millisecond},
		version: "test",
	}, fs, proc
}

func execCLI(args []string, opts options) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb, opts)
	return code, out.String(), errb.String()
}

func TestListOutput(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   []string
		absent []string
	}{
		{
			name: "table",
			args: nil,
			want: []string{"PORT", "PROTO", "3000", "node"},
		},
		{
			name:   "grouped",
			args:   []string{"-g"},
			want:   []string{"PROCESS", "PORTS", "node", "postgres"},
			absent: []string{"PROTO"},
		},
		{
			name:   "port short",
			args:   []string{"-p", "5432"},
			want:   []string{"5432", "postgres"},
			absent: []string{"3000"},
		},
		{
			name:   "port long",
			args:   []string{"--port", "5432"},
			want:   []string{"5432", "postgres"},
			absent: []string{"3000"},
		},
		{
			name: "json",
			args: []string{"--json"},
			want: []string{`"port": 3000`, `"process": "node"`, `"pid": 10`},
		},
		{
			name: "grouped json",
			args: []string{"--json", "-g"},
			want: []string{`"process": "node"`, `"ports": [`, `"pid": 20`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _, _ := testOptions(testListeners)
			code, out, errb := execCLI(tt.args, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("stdout = %q, want it to contain %q", out, want)
				}
			}
			for _, absent := range tt.absent {
				if strings.Contains(out, absent) {
					t.Errorf("stdout = %q, want it not to contain %q", out, absent)
				}
			}
		})
	}
}

func TestEmptyList(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   string
		absent string
	}{
		{"plain", nil, "no listeners found", ""},
		{"grouped", []string{"-g"}, "no listeners found", ""},
		{"json", []string{"--json"}, "[]", "no listeners found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _, _ := testOptions(nil)
			code, out, errb := execCLI(tt.args, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("stdout = %q, want it to contain %q", out, tt.want)
			}
			if tt.absent != "" && strings.Contains(out, tt.absent) {
				t.Errorf("stdout = %q, want it not to contain %q", out, tt.absent)
			}
		})
	}
}

func TestProtoFlag(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "tcp"},
		{[]string{"--proto", "udp"}, "udp"},
		{[]string{"--proto", "all"}, "all"},
	}
	for _, tt := range tests {
		opts, fs, _ := testOptions(testListeners)
		execCLI(tt.args, opts)
		if len(fs.protos) != 1 || fs.protos[0] != tt.want {
			t.Errorf("args %v: scanned protos = %v, want [%s]", tt.args, fs.protos, tt.want)
		}
	}
}

func TestScanError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"list", nil},
		{"kill command", []string{"kill", "3000"}},
		{"shorthand", []string{"3000"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _, _ := testOptions(nil)
			opts.scan = func(string) ([]scan.Listener, error) {
				return nil, errBoom
			}
			code, out, errb := execCLI(tt.args, opts)
			if code != 1 {
				t.Errorf("code = %d, want 1", code)
			}
			if !strings.Contains(errb, "error: boom") {
				t.Errorf("stderr = %q, want the scan error", errb)
			}
			if strings.Contains(out, "listeners found") {
				t.Errorf("unexpected stdout: %q", out)
			}
		})
	}
}

func TestShorthandKill(t *testing.T) {
	tests := []struct {
		name string
		args []string
		sig  syscall.Signal
	}{
		{"plain", []string{"3000"}, syscall.SIGTERM},
		{"force before port", []string{"-f", "3000"}, syscall.SIGKILL},
		{"force after port", []string{"3000", "-f"}, syscall.SIGKILL},
		{"force long", []string{"--force", "3000"}, syscall.SIGKILL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _, proc := testOptions(testListeners)
			code, out, errb := execCLI(tt.args, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			if !strings.Contains(out, "killed node (pid 10) on port 3000") {
				t.Errorf("unexpected stdout: %q", out)
			}
			want := []signalCall{{10, tt.sig}}
			if got := proc.kills(); len(got) != 1 || got[0] != want[0] {
				t.Errorf("signals = %v, want %v", got, want)
			}
		})
	}
}

func TestKillCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		sig  syscall.Signal
	}{
		{"plain", []string{"kill", "3000"}, syscall.SIGTERM},
		{"force", []string{"kill", "3000", "-f"}, syscall.SIGKILL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, fs, proc := testOptions(testListeners)
			code, out, errb := execCLI(tt.args, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			if !strings.Contains(out, "killed node (pid 10) on port 3000") {
				t.Errorf("unexpected stdout: %q", out)
			}
			if len(fs.protos) != 1 || fs.protos[0] != "tcp" {
				t.Errorf("scanned protos = %v, want [tcp]", fs.protos)
			}
			if got := proc.kills(); len(got) != 1 || got[0].sig != tt.sig {
				t.Errorf("signals = %v, want one %v", got, tt.sig)
			}
		})
	}
}

func TestKillErrors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  string
	}{
		{"nothing listening", []string{"9999"}, 1, "port 9999: nothing is listening"},
		{"invalid port in command", []string{"kill", "abc"}, 1, "invalid port: abc"},
		{"invalid port shorthand", []string{"70000"}, 1, "invalid port: 70000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _, proc := testOptions(testListeners)
			code, out, errb := execCLI(tt.args, opts)
			if code != tt.wantCode {
				t.Errorf("code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(errb, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", errb, tt.wantErr)
			}
			if len(proc.kills()) != 0 {
				t.Errorf("unexpected signals: %v", proc.kills())
			}
			if out != "" {
				t.Errorf("unexpected stdout: %q", out)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"command", []string{"version"}},
		{"flag", []string{"--version"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _, _ := testOptions(nil)
			code, out, errb := execCLI(tt.args, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			if out != "kp test\n" {
				t.Errorf("stdout = %q, want %q", out, "kp test\n")
			}
		})
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown command", []string{"foo"}, `unknown command "foo"`},
		{"unknown long flag", []string{"--nope"}, "unknown flag"},
		{"unknown short flag", []string{"-z"}, "unknown shorthand flag"},
		{"kill without ports", []string{"kill"}, "requires at least 1 arg(s)"},
		{"version with args", []string{"version", "extra"}, "unknown command"},
		{"version shorthand disabled", []string{"-v"}, "unknown shorthand flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, fs, _ := testOptions(testListeners)
			code, _, errb := execCLI(tt.args, opts)
			if code != 2 {
				t.Errorf("code = %d, want 2", code)
			}
			if !strings.Contains(errb, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", errb, tt.wantErr)
			}
			if !strings.Contains(errb, "run 'kp --help' for usage") {
				t.Errorf("stderr = %q, want the usage hint", errb)
			}
			if len(fs.protos) != 0 {
				t.Errorf("scan should not run, got protos %v", fs.protos)
			}
		})
	}
}

func TestHelpAndCompletion(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		opts, _, _ := testOptions(nil)
		code, out, errb := execCLI([]string{"--help"}, opts)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, errb)
		}
		for _, want := range []string{"Usage:", "kill", "version", "completion"} {
			if !strings.Contains(out, want) {
				t.Errorf("help output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("completion bash", func(t *testing.T) {
		opts, _, _ := testOptions(nil)
		code, out, errb := execCLI([]string{"completion", "bash"}, opts)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, errb)
		}
		if len(out) == 0 || !strings.Contains(out, "bash") {
			t.Errorf("unexpected completion output: %q", out)
		}
	})
}

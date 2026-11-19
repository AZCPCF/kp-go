package kill

import (
	"bytes"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/azcpcf/kp-go/internal/scan"
)

type call struct {
	pid int
	sig syscall.Signal
}

type fakeProc struct {
	alive    map[int]bool
	stubborn map[int]bool
	errs     map[int]error
	calls    []call
}

func (f *fakeProc) signal(pid int, sig syscall.Signal) error {
	f.calls = append(f.calls, call{pid, sig})
	if err, ok := f.errs[pid]; ok {
		return err
	}
	if !f.alive[pid] {
		return syscall.ESRCH
	}
	if sig != 0 && !f.stubborn[pid] {
		f.alive[pid] = false
	}
	return nil
}

func (f *fakeProc) killCalls() []call {
	var res []call
	for _, c := range f.calls {
		if c.sig != 0 {
			res = append(res, c)
		}
	}
	return res
}

func newTestKiller(f *fakeProc) *Killer {
	return &Killer{
		Signal: f.signal,
		Wait:   50 * time.Millisecond,
		Poll:   time.Millisecond,
	}
}

func TestKillSuccess(t *testing.T) {
	f := &fakeProc{alive: map[int]bool{10: true}}
	ls := []scan.Listener{{Port: 3000, Process: "node", PID: "10"}}
	var out, errb bytes.Buffer

	code := Run(&out, &errb, newTestKiller(f), ls, []string{"3000"}, false)

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "killed node (pid 10) on port 3000") {
		t.Errorf("unexpected stdout: %q", out.String())
	}
	want := []call{{10, syscall.SIGTERM}}
	if !reflect.DeepEqual(f.killCalls(), want) {
		t.Errorf("got %v, want %v", f.killCalls(), want)
	}
}

func TestKillForceUsesSIGKILL(t *testing.T) {
	f := &fakeProc{alive: map[int]bool{10: true}}
	ls := []scan.Listener{{Port: 3000, Process: "node", PID: "10"}}
	var out, errb bytes.Buffer

	Run(&out, &errb, newTestKiller(f), ls, []string{"3000"}, true)

	want := []call{{10, syscall.SIGKILL}}
	if !reflect.DeepEqual(f.killCalls(), want) {
		t.Errorf("got %v, want %v", f.killCalls(), want)
	}
}

func TestKillDedupPID(t *testing.T) {
	f := &fakeProc{alive: map[int]bool{10: true}}
	ls := []scan.Listener{
		{Addr: "0.0.0.0", Port: 3000, Process: "node", PID: "10"},
		{Addr: "::", Port: 3000, Process: "node", PID: "10"},
	}
	var out, errb bytes.Buffer

	Run(&out, &errb, newTestKiller(f), ls, []string{"3000"}, false)

	if n := len(f.killCalls()); n != 1 {
		t.Errorf("expected 1 kill call, got %d", n)
	}
}

func TestKillErrors(t *testing.T) {
	node := []scan.Listener{{Port: 3000, Process: "node", PID: "10"}}
	tests := []struct {
		name     string
		ls       []scan.Listener
		args     []string
		proc     *fakeProc
		wantText string
	}{
		{"not a number", node, []string{"abc"}, &fakeProc{}, "invalid port: abc"},
		{"out of range", node, []string{"70000"}, &fakeProc{}, "invalid port: 70000"},
		{"zero", node, []string{"0"}, &fakeProc{}, "invalid port: 0"},
		{"nothing listening", node, []string{"9999"}, &fakeProc{}, "nothing is listening"},
		{
			"unknown pid",
			[]scan.Listener{{Port: 22, Process: "-", PID: "-"}},
			[]string{"22"}, &fakeProc{}, "process unknown",
		},
		{
			"permission denied", node, []string{"3000"},
			&fakeProc{errs: map[int]error{10: syscall.EPERM}}, "permission denied",
		},
		{"already gone", node, []string{"3000"}, &fakeProc{}, "already gone"},
		{
			"stubborn process", node, []string{"3000"},
			&fakeProc{alive: map[int]bool{10: true}, stubborn: map[int]bool{10: true}},
			"still running",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := Run(&out, &errb, newTestKiller(tt.proc), tt.ls, tt.args, false)
			if code != 1 {
				t.Errorf("code = %d, want 1", code)
			}
			if !strings.Contains(errb.String(), tt.wantText) {
				t.Errorf("stderr = %q, want it to contain %q", errb.String(), tt.wantText)
			}
			if strings.Contains(out.String(), "killed") {
				t.Errorf("unexpected stdout: %q", out.String())
			}
		})
	}
}

func TestKillContinuesAfterFailure(t *testing.T) {
	f := &fakeProc{alive: map[int]bool{20: true}}
	ls := []scan.Listener{{Port: 4000, Process: "app", PID: "20"}}
	var out, errb bytes.Buffer

	code := Run(&out, &errb, newTestKiller(f), ls, []string{"3000", "4000"}, false)

	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "killed app (pid 20) on port 4000") {
		t.Errorf("second port was not killed: %q", out.String())
	}
	if !strings.Contains(errb.String(), "port 3000: nothing is listening") {
		t.Errorf("missing error for first port: %q", errb.String())
	}
}

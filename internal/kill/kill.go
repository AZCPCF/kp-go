// Package kill terminates processes listening on a port.
package kill

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"syscall"
	"time"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/scan"
)

// Killer sends signals and waits for the process to exit.
type Killer struct {
	Signal func(pid int, sig syscall.Signal) error
	Wait   time.Duration
	Poll   time.Duration
}

// New returns a Killer that sends SIGTERM (or SIGKILL when forced), waits up
// to 2s for the process to exit and polls every 50ms.
func New() *Killer {
	return &Killer{
		Signal: syscall.Kill,
		Wait:   2 * time.Second,
		Poll:   50 * time.Millisecond,
	}
}

// Run kills whatever listens on each port in args and returns a process exit
// code: 0 on success, 1 if any port failed.
func Run(out, errw io.Writer, k *Killer, ls []scan.Listener, args []string, force bool) int {
	code := 0
	for _, a := range args {
		port, err := strconv.Atoi(a)
		if err != nil || port < 1 || port > 65535 {
			fmt.Fprintf(errw, "invalid port: %s\n", a)
			code = 1
			continue
		}
		if err := k.killPort(out, ls, port, force); err != nil {
			fmt.Fprintf(errw, "port %d: %v\n", port, err)
			code = 1
		}
	}
	return code
}

// Kill terminates the process owning l, using SIGKILL when force is set.
func (k *Killer) Kill(l scan.Listener, force bool) error {
	if l.PID == "-" {
		return errors.New("process unknown (try sudo)")
	}
	pid, err := strconv.Atoi(l.PID)
	if err != nil {
		return fmt.Errorf("invalid pid %q", l.PID)
	}
	if err := k.terminate(pid, force); err != nil {
		return fmt.Errorf("%s (pid %d): %w", l.Process, pid, err)
	}
	return nil
}

func (k *Killer) killPort(out io.Writer, ls []scan.Listener, port int, force bool) error {
	matches := group.FilterByPort(ls, port)
	if len(matches) == 0 {
		return errors.New("nothing is listening")
	}
	seen := map[string]bool{}
	for _, l := range matches {
		if seen[l.PID] {
			continue
		}
		seen[l.PID] = true
		if err := k.Kill(l, force); err != nil {
			return err
		}
		fmt.Fprintf(out, "killed %s (pid %s) on port %d\n", l.Process, l.PID, port)
	}
	return nil
}

func (k *Killer) terminate(pid int, force bool) error {
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	if err := k.Signal(pid, sig); err != nil {
		switch {
		case errors.Is(err, syscall.ESRCH):
			return errors.New("process already gone")
		case errors.Is(err, syscall.EPERM):
			return errors.New("permission denied (try sudo)")
		}
		return err
	}
	deadline := time.Now().Add(k.Wait)
	for time.Now().Before(deadline) {
		if k.Signal(pid, 0) != nil {
			return nil
		}
		time.Sleep(k.Poll)
	}
	return fmt.Errorf("still running after %v (use -f)", k.Wait)
}

package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"syscall"
	"time"
)

type Killer struct {
	Signal func(pid int, sig syscall.Signal) error
	Wait   time.Duration
	Poll   time.Duration
}

func newKiller() *Killer {
	return &Killer{
		Signal: syscall.Kill,
		Wait:   2 * time.Second,
		Poll:   50 * time.Millisecond,
	}
}

func runKill(out, errw io.Writer, k *Killer, ls []Listener, args []string, force bool) int {
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

func (k *Killer) killPort(out io.Writer, ls []Listener, port int, force bool) error {
	matches := filterByPort(ls, port)
	if len(matches) == 0 {
		return errors.New("nothing is listening")
	}
	seen := map[string]bool{}
	for _, l := range matches {
		if seen[l.PID] {
			continue
		}
		seen[l.PID] = true
		if l.PID == "-" {
			return errors.New("process unknown (try sudo)")
		}
		pid, _ := strconv.Atoi(l.PID)
		if err := k.terminate(pid, force); err != nil {
			return fmt.Errorf("%s (pid %d): %w", l.Process, pid, err)
		}
		fmt.Fprintf(out, "killed %s (pid %d) on port %d\n", l.Process, pid, port)
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

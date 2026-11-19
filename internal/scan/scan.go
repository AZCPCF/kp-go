// Package scan discovers listening TCP sockets.
package scan

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var usersRe = regexp.MustCompile(`\("([^"]+)",pid=(\d+)`)

// Listener is a single listening socket and the process that owns it.
type Listener struct {
	Addr    string
	Port    int
	Process string
	PID     string
}

// List returns all listening TCP sockets, preferring /proc and falling back to
// "ss -tlnpH".
func List() ([]Listener, error) {
	ls, err := listListenersProc()
	if err != nil {
		return listListenersSS()
	}
	return ls, nil
}

func listListenersSS() ([]Listener, error) {
	out, err := exec.Command("ss", "-tlnpH").Output()
	if err != nil {
		return nil, err
	}
	var res []Listener
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		local := f[3]
		i := strings.LastIndex(local, ":")
		if i < 0 {
			continue
		}
		port, err := strconv.Atoi(local[i+1:])
		if err != nil {
			continue
		}
		l := Listener{Addr: local[:i], Port: port, Process: "-", PID: "-"}
		if m := usersRe.FindStringSubmatch(line); m != nil {
			l.Process, l.PID = m[1], m[2]
		}
		res = append(res, l)
	}
	return res, nil
}

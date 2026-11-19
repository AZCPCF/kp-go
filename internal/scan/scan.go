// Package scan discovers listening TCP and UDP sockets.
package scan

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Protocol values accepted by List.
const (
	ProtoTCP = "tcp"
	ProtoUDP = "udp"
	ProtoAll = "all"
)

var usersRe = regexp.MustCompile(`\("([^"]+)",pid=(\d+)`)

// Listener is a single listening socket and the process that owns it.
type Listener struct {
	Addr    string
	Port    int
	Proto   string
	Process string
	PID     string
}

// ParseProto validates a protocol selector, returning ProtoTCP for an empty
// string.
func ParseProto(s string) (string, error) {
	switch s {
	case "":
		return ProtoTCP, nil
	case ProtoTCP, ProtoUDP, ProtoAll:
		return s, nil
	}
	return "", fmt.Errorf("invalid proto %q (want tcp, udp or all)", s)
}

// List returns all listening sockets for proto ("tcp", "udp" or "all"),
// preferring /proc and falling back to ss.
func List(proto string) ([]Listener, error) {
	p, err := ParseProto(proto)
	if err != nil {
		return nil, err
	}
	ls, err := listListenersProc(p)
	if err != nil {
		return listListenersSS(p)
	}
	return ls, nil
}

func ssArgs(proto string) []string {
	switch proto {
	case ProtoUDP:
		return []string{"-ulnpH"}
	case ProtoAll:
		return []string{"-tulnpH"}
	default:
		return []string{"-tlnpH"}
	}
}

func listListenersSS(proto string) ([]Listener, error) {
	out, err := exec.Command("ss", ssArgs(proto)...).Output()
	if err != nil {
		return nil, err
	}
	return parseSS(string(out), proto), nil
}

func parseSS(data string, proto string) []Listener {
	var res []Listener
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		// ss prints a Netid column when more than one socket type is selected.
		p, i := proto, 3
		if f[0] == ProtoTCP || f[0] == ProtoUDP {
			p, i = f[0], 4
		}
		if len(f) <= i {
			continue
		}
		local := f[i]
		colon := strings.LastIndex(local, ":")
		if colon < 0 {
			continue
		}
		port, err := strconv.Atoi(local[colon+1:])
		if err != nil {
			continue
		}
		l := Listener{Addr: local[:colon], Port: port, Proto: p, Process: "-", PID: "-"}
		if m := usersRe.FindStringSubmatch(line); m != nil {
			l.Process, l.PID = m[1], m[2]
		}
		res = append(res, l)
	}
	return res
}

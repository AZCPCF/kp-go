// Package group sorts, filters and groups listening sockets.
package group

import (
	"sort"

	"github.com/azcpcf/kp-go/internal/scan"
)

// Group is a process and the sockets it listens on.
type Group struct {
	Process   string
	PID       string
	Listeners []scan.Listener
}

// SortByPort sorts ls in place by ascending port, then protocol.
func SortByPort(ls []scan.Listener) {
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].Port != ls[j].Port {
			return ls[i].Port < ls[j].Port
		}
		return ls[i].Proto < ls[j].Proto
	})
}

// FilterByPort returns the listeners on port.
func FilterByPort(ls []scan.Listener, port int) []scan.Listener {
	var res []scan.Listener
	for _, l := range ls {
		if l.Port == port {
			res = append(res, l)
		}
	}
	return res
}

// ByProcess groups ls by process name and PID, preserving input order.
func ByProcess(ls []scan.Listener) []Group {
	idx := map[string]int{}
	var gs []Group
	for _, l := range ls {
		key := l.Process + ":" + l.PID
		i, ok := idx[key]
		if !ok {
			i = len(gs)
			idx[key] = i
			gs = append(gs, Group{Process: l.Process, PID: l.PID})
		}
		gs[i].Listeners = append(gs[i].Listeners, l)
	}
	return gs
}

package main

import "sort"

func sortByPort(ls []Listener) {
	sort.Slice(ls, func(i, j int) bool {
		return ls[i].Port < ls[j].Port
	})
}

func filterByPort(ls []Listener, port int) []Listener {
	var res []Listener
	for _, l := range ls {
		if l.Port == port {
			res = append(res, l)
		}
	}
	return res
}

func groupByProcess(ls []Listener) []Group {
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

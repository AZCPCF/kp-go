package main

import (
	"reflect"
	"testing"
)

var sampleListeners = []Listener{
	{Addr: "0.0.0.0", Port: 3000, Process: "node", PID: "10"},
	{Addr: "::", Port: 3000, Process: "node", PID: "10"},
	{Addr: "127.0.0.1", Port: 5432, Process: "postgres", PID: "20"},
	{Addr: "0.0.0.0", Port: 80, Process: "nginx", PID: "30"},
}

func TestSortByPort(t *testing.T) {
	ls := append([]Listener(nil), sampleListeners...)
	sortByPort(ls)
	var ports []int
	for _, l := range ls {
		ports = append(ports, l.Port)
	}
	want := []int{80, 3000, 3000, 5432}
	if !reflect.DeepEqual(ports, want) {
		t.Errorf("got %v, want %v", ports, want)
	}
}

func TestFilterByPort(t *testing.T) {
	if got := filterByPort(sampleListeners, 3000); len(got) != 2 {
		t.Errorf("port 3000: got %d, want 2", len(got))
	}
	if got := filterByPort(sampleListeners, 9999); len(got) != 0 {
		t.Errorf("port 9999: got %d, want 0", len(got))
	}
}

func TestGroupByProcess(t *testing.T) {
	gs := groupByProcess(sampleListeners)
	if len(gs) != 3 {
		t.Fatalf("got %d groups, want 3", len(gs))
	}
	if gs[0].Process != "node" || len(gs[0].Listeners) != 2 {
		t.Errorf("first group wrong: %+v", gs[0])
	}
	if gs[1].Process != "postgres" || gs[2].Process != "nginx" {
		t.Errorf("order not preserved: %+v", gs)
	}
}

func TestGroupSamePIDDifferentName(t *testing.T) {
	ls := []Listener{
		{Port: 1, Process: "a", PID: "1"},
		{Port: 2, Process: "a", PID: "2"},
	}
	if gs := groupByProcess(ls); len(gs) != 2 {
		t.Errorf("same name with different PIDs must be separate groups, got %d", len(gs))
	}
}

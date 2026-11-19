package group

import (
	"reflect"
	"testing"

	"github.com/azcpcf/kp-go/internal/scan"
)

var sampleListeners = []scan.Listener{
	{Addr: "0.0.0.0", Port: 3000, Process: "node", PID: "10"},
	{Addr: "::", Port: 3000, Process: "node", PID: "10"},
	{Addr: "127.0.0.1", Port: 5432, Process: "postgres", PID: "20"},
	{Addr: "0.0.0.0", Port: 80, Process: "nginx", PID: "30"},
}

func TestSortByPort(t *testing.T) {
	ls := append([]scan.Listener(nil), sampleListeners...)
	SortByPort(ls)
	var ports []int
	for _, l := range ls {
		ports = append(ports, l.Port)
	}
	want := []int{80, 3000, 3000, 5432}
	if !reflect.DeepEqual(ports, want) {
		t.Errorf("got %v, want %v", ports, want)
	}
}

func TestSortByPortProtoTiebreak(t *testing.T) {
	ls := []scan.Listener{
		{Port: 53, Proto: "udp"},
		{Port: 80, Proto: "tcp"},
		{Port: 53, Proto: "tcp"},
	}
	SortByPort(ls)
	want := []scan.Listener{
		{Port: 53, Proto: "tcp"},
		{Port: 53, Proto: "udp"},
		{Port: 80, Proto: "tcp"},
	}
	if !reflect.DeepEqual(ls, want) {
		t.Errorf("got %+v, want %+v", ls, want)
	}
}

func TestFilterByPort(t *testing.T) {
	if got := FilterByPort(sampleListeners, 3000); len(got) != 2 {
		t.Errorf("port 3000: got %d, want 2", len(got))
	}
	if got := FilterByPort(sampleListeners, 9999); len(got) != 0 {
		t.Errorf("port 9999: got %d, want 0", len(got))
	}
}

func TestGroupByProcess(t *testing.T) {
	gs := ByProcess(sampleListeners)
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
	ls := []scan.Listener{
		{Port: 1, Process: "a", PID: "1"},
		{Port: 2, Process: "a", PID: "2"},
	}
	if gs := ByProcess(ls); len(gs) != 2 {
		t.Errorf("same name with different PIDs must be separate groups, got %d", len(gs))
	}
}

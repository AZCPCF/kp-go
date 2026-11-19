package render

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/azcpcf/kp-go/internal/ui"
)

// lines splits rendered output into its trailing-newline-terminated lines.
func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, []scan.Listener{
		{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
	}, ui.New(false))

	got := lines(buf.String())
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(got), buf.String())
	}
	if fields := strings.Fields(got[0]); len(fields) != 5 || fields[0] != "PORT" || fields[1] != "PROTO" || fields[2] != "ADDR" || fields[3] != "PROCESS" || fields[4] != "PID" {
		t.Errorf("header = %q", got[0])
	}
	want := []string{"3000", "tcp", "0.0.0.0", "node", "10"}
	fields := strings.Fields(got[1])
	if len(fields) != len(want) {
		t.Fatalf("row = %q, want %d columns", got[1], len(want))
	}
	for i := range want {
		if fields[i] != want[i] {
			t.Errorf("column %d = %q, want %q", i, fields[i], want[i])
		}
	}
}

func TestPrintTableMixedProtos(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, []scan.Listener{
		{Addr: "0.0.0.0", Port: 53, Proto: "tcp", Process: "dns", PID: "10"},
		{Addr: "0.0.0.0", Port: 53, Proto: "udp", Process: "dns", PID: "10"},
	}, ui.New(false))

	got := lines(buf.String())
	if len(got) != 3 {
		t.Fatalf("got %d lines, want 3: %q", len(got), buf.String())
	}
	if fields := strings.Fields(got[1]); fields[1] != "tcp" {
		t.Errorf("first row proto = %q, want tcp", fields[1])
	}
	if fields := strings.Fields(got[2]); fields[1] != "udp" {
		t.Errorf("second row proto = %q, want udp", fields[1])
	}
}

func TestPrintTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, nil, ui.New(false))

	if got := lines(buf.String()); len(got) != 1 {
		t.Fatalf("got %d lines, want header only: %q", len(got), buf.String())
	}
	if !strings.HasPrefix(buf.String(), "PORT") {
		t.Errorf("missing header: %q", buf.String())
	}
}

func TestPrintGrouped(t *testing.T) {
	var buf bytes.Buffer
	PrintGrouped(&buf, []group.Group{
		{
			Process: "node",
			PID:     "10",
			Listeners: []scan.Listener{
				{Addr: "0.0.0.0", Port: 3000, Process: "node", PID: "10"},
				{Addr: "::", Port: 3001, Process: "node", PID: "10"},
			},
		},
	}, ui.New(false))

	got := lines(buf.String())
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(got), buf.String())
	}
	if fields := strings.Fields(got[0]); len(fields) != 3 || fields[0] != "PROCESS" || fields[1] != "PID" || fields[2] != "PORTS" {
		t.Errorf("header = %q", got[0])
	}
	if !strings.HasPrefix(got[1], "node") || !strings.Contains(got[1], "10") {
		t.Errorf("row = %q, want process node and pid 10", got[1])
	}
	if !strings.Contains(got[1], "3000, 3001") {
		t.Errorf("row = %q, want ports joined as \"3000, 3001\"", got[1])
	}
}

func TestPrintGroupedEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintGrouped(&buf, nil, ui.New(false))

	if got := lines(buf.String()); len(got) != 1 {
		t.Fatalf("got %d lines, want header only: %q", len(got), buf.String())
	}
	if !strings.HasPrefix(buf.String(), "PROCESS") {
		t.Errorf("missing header: %q", buf.String())
	}
}

func TestPrintTablePlainExact(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, []scan.Listener{
		{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
		{Addr: "127.0.0.1", Port: 5432, Proto: "udp", Process: "postgres", PID: "-"},
	}, ui.New(false))

	want := "PORT  PROTO  ADDR       PROCESS   PID\n" +
		"3000  tcp    0.0.0.0    node      10\n" +
		"5432  udp    127.0.0.1  postgres  -\n"
	if got := buf.String(); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPrintGroupedPlainExact(t *testing.T) {
	var buf bytes.Buffer
	PrintGrouped(&buf, []group.Group{
		{
			Process: "node",
			PID:     "10",
			Listeners: []scan.Listener{
				{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
				{Addr: "::", Port: 3001, Proto: "tcp", Process: "node", PID: "10"},
			},
		},
	}, ui.New(false))

	want := "PROCESS  PID  PORTS\nnode     10   3000, 3001\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrintTableColorKeepsLayout(t *testing.T) {
	ls := []scan.Listener{
		{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
		{Addr: "127.0.0.1", Port: 5432, Proto: "udp", Process: "postgres", PID: "-"},
	}

	var plain, colored bytes.Buffer
	PrintTable(&plain, ls, ui.New(false))
	PrintTable(&colored, ls, ui.New(true))

	if !strings.Contains(colored.String(), "\x1b[") {
		t.Fatalf("colored output has no ANSI escapes: %q", colored.String())
	}
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	if got, want := ansi.ReplaceAllString(colored.String(), ""), plain.String(); got != want {
		t.Errorf("stripped colored output:\n%q\nwant plain:\n%q", got, want)
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	PrintJSON(&buf, []scan.Listener{
		{Addr: "0.0.0.0", Port: 3000, Proto: "tcp", Process: "node", PID: "10"},
		{Addr: "0.0.0.0", Port: 5353, Proto: "udp", Process: "avahi", PID: "-"},
	})

	want := `[
  {
    "addr": "0.0.0.0",
    "port": 3000,
    "proto": "tcp",
    "process": "node",
    "pid": 10
  },
  {
    "addr": "0.0.0.0",
    "port": 5353,
    "proto": "udp",
    "process": "avahi",
    "pid": null
  }
]
`
	if got := buf.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintGroupedJSON(t *testing.T) {
	var buf bytes.Buffer
	PrintGroupedJSON(&buf, []group.Group{
		{
			Process: "node",
			PID:     "10",
			Listeners: []scan.Listener{
				{Addr: "0.0.0.0", Port: 3000, Process: "node", PID: "10"},
				{Addr: "::", Port: 3001, Process: "node", PID: "10"},
			},
		},
		{
			Process:   "postgres",
			PID:       "-",
			Listeners: []scan.Listener{{Addr: "127.0.0.1", Port: 5432, Process: "postgres", PID: "-"}},
		},
	})

	want := `[
  {
    "process": "node",
    "pid": 10,
    "ports": [
      3000,
      3001
    ]
  },
  {
    "process": "postgres",
    "pid": null,
    "ports": [
      5432
    ]
  }
]
`
	if got := buf.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintJSONEmpty(t *testing.T) {
	tests := []struct {
		name string
		ls   []scan.Listener
	}{
		{"nil", nil},
		{"empty", []scan.Listener{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintJSON(&buf, tt.ls)
			if got, want := buf.String(), "[]\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestPrintGroupedJSONEmpty(t *testing.T) {
	tests := []struct {
		name string
		gs   []group.Group
	}{
		{"nil", nil},
		{"empty", []group.Group{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PrintGroupedJSON(&buf, tt.gs)
			if got, want := buf.String(), "[]\n"; got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

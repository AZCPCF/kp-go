package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/scan"
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
		{Addr: "0.0.0.0", Port: 3000, Process: "node", PID: "10"},
	})

	got := lines(buf.String())
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(got), buf.String())
	}
	if fields := strings.Fields(got[0]); len(fields) != 4 || fields[0] != "PORT" || fields[1] != "ADDR" || fields[2] != "PROCESS" || fields[3] != "PID" {
		t.Errorf("header = %q", got[0])
	}
	want := []string{"3000", "0.0.0.0", "node", "10"}
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

func TestPrintTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintTable(&buf, nil)

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
	})

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
	PrintGrouped(&buf, nil)

	if got := lines(buf.String()); len(got) != 1 {
		t.Fatalf("got %d lines, want header only: %q", len(got), buf.String())
	}
	if !strings.HasPrefix(buf.String(), "PROCESS") {
		t.Errorf("missing header: %q", buf.String())
	}
}

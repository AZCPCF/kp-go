package ui

import (
	"bytes"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func namedStyles(theme Theme) []struct {
	name  string
	style lipgloss.Style
} {
	return []struct {
		name  string
		style lipgloss.Style
	}{
		{"header", theme.Header},
		{"port", theme.Port},
		{"process", theme.Process},
		{"muted", theme.Muted},
		{"error", theme.Error},
		{"success", theme.Success},
		{"tcp", theme.TCP},
		{"udp", theme.UDP},
	}
}

func TestThemeDisabledIsPlain(t *testing.T) {
	for _, tt := range namedStyles(New(false)) {
		if got, want := tt.style.Render("x"), "x"; got != want {
			t.Errorf("%s style = %q, want %q", tt.name, got, want)
		}
	}
}

func TestThemeEnabledUsesANSI(t *testing.T) {
	for _, tt := range namedStyles(New(true)) {
		got := tt.style.Render("x")
		if !strings.Contains(got, "\x1b[") {
			t.Errorf("%s style = %q, want ANSI escape", tt.name, got)
		}
		if !strings.Contains(got, "x") {
			t.Errorf("%s style = %q, want it to contain the text", tt.name, got)
		}
	}
}

func TestProtoStyles(t *testing.T) {
	theme := New(true)
	if theme.Proto("tcp").Render("tcp") == theme.Proto("udp").Render("udp") {
		t.Error("tcp and udp should render differently")
	}
	if got, want := theme.Proto("sctp").Render("sctp"), theme.Muted.Render("sctp"); got != want {
		t.Errorf("unknown proto = %q, want muted %q", got, want)
	}
}

func TestStyleWriterDisabled(t *testing.T) {
	var buf bytes.Buffer
	w := StyleWriter(&buf, New(false).Success)
	in := "killed node (pid 10) on port 3000\nkilled node (pid 11) on port 3001\n"
	if _, err := w.Write([]byte(in)); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != in {
		t.Errorf("got %q, want %q", got, in)
	}
}

func TestStyleWriterEnabled(t *testing.T) {
	var buf bytes.Buffer
	w := StyleWriter(&buf, New(true).Success)
	in := "killed node (pid 10) on port 3000\n"
	if _, err := w.Write([]byte(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\x1b[") {
		t.Errorf("output = %q, want a leading ANSI escape", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output = %q, want a trailing newline", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("output = %q, want exactly one newline", out)
	}
	if !strings.Contains(out, "killed node (pid 10) on port 3000") {
		t.Errorf("output = %q, want the message", out)
	}
}

func TestIsTTYBuffer(t *testing.T) {
	var buf bytes.Buffer
	if IsTTY(&buf) {
		t.Error("a bytes.Buffer should not be a TTY")
	}
}

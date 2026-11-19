package ui

import "testing"

func TestParseColorMode(t *testing.T) {
	tests := []struct {
		in      string
		want    ColorMode
		wantErr bool
	}{
		{"auto", ColorAuto, false},
		{"always", ColorAlways, false},
		{"never", ColorNever, false},
		{"", "", true},
		{"AUTO", "", true},
		{"sometimes", "", true},
	}
	for _, tt := range tests {
		got, err := ParseColorMode(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseColorMode(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("ParseColorMode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUseColor(t *testing.T) {
	tests := []struct {
		name    string
		mode    ColorMode
		noColor bool
		tty     bool
		want    bool
	}{
		{"auto tty", ColorAuto, false, true, true},
		{"auto no color", ColorAuto, true, true, false},
		{"auto not a tty", ColorAuto, false, false, false},
		{"auto no color and not a tty", ColorAuto, true, false, false},
		{"always", ColorAlways, false, false, true},
		{"always overrides no color", ColorAlways, true, false, true},
		{"never", ColorNever, false, true, false},
		{"never with no color", ColorNever, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UseColor(tt.mode, tt.noColor, tt.tty); got != tt.want {
				t.Errorf("UseColor(%q, %v, %v) = %v, want %v", tt.mode, tt.noColor, tt.tty, got, tt.want)
			}
		})
	}
}

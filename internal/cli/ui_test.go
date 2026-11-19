package cli

import (
	"strings"
	"testing"

	"github.com/azcpcf/kp-go/internal/tui"
	"github.com/azcpcf/kp-go/internal/ui"
)

func TestUICommandAliases(t *testing.T) {
	tests := [][]string{
		{"ui", "--help"},
		{"tui", "--help"},
	}
	for _, args := range tests {
		t.Run(args[0], func(t *testing.T) {
			opts, _, _ := testOptions(testListeners)
			code, out, errb := execCLI(args, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			if !strings.Contains(out, "ui") {
				t.Errorf("help output missing ui command:\n%s", out)
			}
		})
	}
}

func TestUICommandRuns(t *testing.T) {
	for _, name := range []string{"ui", "tui"} {
		t.Run(name, func(t *testing.T) {
			opts, fs, _ := testOptions(testListeners)
			var scanner tui.Scanner
			opts.ui = func(s tui.Scanner, _ ui.Theme) error {
				scanner = s
				return nil
			}

			code, _, errb := execCLI([]string{name}, opts)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, errb)
			}
			if scanner == nil {
				t.Fatal("ui runner was not called")
			}
			ls, err := scanner.List()
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(ls) != len(testListeners) {
				t.Errorf("listeners = %d, want %d", len(ls), len(testListeners))
			}
			if len(fs.protos) != 1 || fs.protos[0] != "all" {
				t.Errorf("scanned protos = %v, want [all]", fs.protos)
			}
		})
	}
}

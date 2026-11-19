package cli

import (
	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/azcpcf/kp-go/internal/tui"
	"github.com/azcpcf/kp-go/internal/ui"
	"github.com/spf13/cobra"
)

func newUICmd(opts options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ui",
		Aliases: []string{"tui"},
		Short:   "open the interactive terminal UI",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			theme, err := themeFor(cmd)
			if err != nil {
				return err
			}
			run := opts.ui
			if run == nil {
				k := opts.killer
				run = func(s tui.Scanner, theme ui.Theme) error {
					return tui.Run(s, k, theme)
				}
			}
			if err := run(scanAll{list: opts.scan}, theme); err != nil {
				return &exitError{code: 1, err: err}
			}
			return nil
		},
	}
	return cmd
}

// scanAll adapts the proto-aware scan function to tui.Scanner.
type scanAll struct {
	list func(proto string) ([]scan.Listener, error)
}

func (s scanAll) List() ([]scan.Listener, error) {
	return s.list(scan.ProtoAll)
}

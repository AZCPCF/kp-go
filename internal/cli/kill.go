package cli

import (
	"github.com/azcpcf/kp-go/internal/kill"
	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/azcpcf/kp-go/internal/ui"
	"github.com/spf13/cobra"
)

func newKillCmd(opts options) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "kill <port...>",
		Short: "kill processes listening on ports",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			theme, err := themeFor(cmd)
			if err != nil {
				return err
			}
			return runKill(cmd, opts, args, scan.ProtoTCP, force, theme)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "kill with SIGKILL")
	return cmd
}

// runKill scans for listeners and kills whatever listens on ports. The exit
// code from kill.Run is propagated as an error that carries no message.
func runKill(cmd *cobra.Command, opts options, ports []string, proto string, force bool, theme ui.Theme) error {
	ls, err := opts.scan(proto)
	if err != nil {
		return &exitError{code: 1, err: err}
	}
	out := ui.StyleWriter(cmd.OutOrStdout(), theme.Success)
	errw := ui.StyleWriter(cmd.ErrOrStderr(), theme.Error)
	if code := kill.Run(out, errw, opts.killer, ls, ports, force); code != 0 {
		return &exitError{code: code}
	}
	return nil
}

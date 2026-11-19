package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "kp", opts.version)
		},
	}
}

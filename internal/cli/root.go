package cli

import (
	"fmt"
	"strconv"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/render"
	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/spf13/cobra"
)

// listFlags holds the flags of the default list command.
type listFlags struct {
	group bool
	port  int
	proto string
	json  bool
}

func newRoot(opts options) *cobra.Command {
	var (
		flags listFlags
		force bool
	)

	cmd := &cobra.Command{
		Use:           "kp [port...]",
		Short:         "list listening ports, or kill what is on them",
		Long:          "List listening TCP and UDP ports.\n\nWith ports, kp kills the processes listening on them; this is shorthand for kp kill <port...>.",
		Args:          cobra.ArbitraryArgs,
		Version:       opts.version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if _, err := strconv.Atoi(args[0]); err != nil {
					return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
				}
				return runKill(cmd, opts, args, flags.proto, force)
			}
			return runList(cmd, opts, flags)
		},
	}

	cmd.SetVersionTemplate("kp {{.Version}}\n")
	// Define the flag ourselves so cobra does not add a -v shorthand.
	cmd.Flags().Bool("version", false, "print version")
	cmd.Flags().BoolVarP(&flags.group, "group", "g", false, "group by process")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 0, "filter by port")
	cmd.Flags().StringVar(&flags.proto, "proto", scan.ProtoTCP, "protocol to list: tcp, udp or all")
	cmd.Flags().BoolVar(&flags.json, "json", false, "output JSON")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "kill with SIGKILL (shorthand kill mode)")

	cmd.AddCommand(newKillCmd(opts), newVersionCmd(opts))
	return cmd
}

func runList(cmd *cobra.Command, opts options, flags listFlags) error {
	ls, err := opts.scan(flags.proto)
	if err != nil {
		return &exitError{code: 1, err: err}
	}

	if flags.port != 0 {
		ls = group.FilterByPort(ls, flags.port)
	}
	group.SortByPort(ls)

	out := cmd.OutOrStdout()
	if len(ls) == 0 && !flags.json {
		fmt.Fprintln(out, "no listeners found")
		return nil
	}

	switch {
	case flags.json && flags.group:
		render.PrintGroupedJSON(out, group.ByProcess(ls))
	case flags.json:
		render.PrintJSON(out, ls)
	case flags.group:
		render.PrintGrouped(out, group.ByProcess(ls))
	default:
		render.PrintTable(out, ls)
	}
	return nil
}

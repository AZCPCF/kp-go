// Package render prints listener tables to an io.Writer.
package render

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/scan"
)

// PrintTable writes ls as a plain port/address/process/pid table.
func PrintTable(w io.Writer, ls []scan.Listener) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "PORT\tADDR\tPROCESS\tPID")
	for _, l := range ls {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", l.Port, l.Addr, l.Process, l.PID)
	}
	tw.Flush()
}

// PrintGrouped writes gs as a process/pid/ports table.
func PrintGrouped(w io.Writer, gs []group.Group) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "PROCESS\tPID\tPORTS")
	for _, g := range gs {
		ports := make([]string, len(g.Listeners))
		for i, l := range g.Listeners {
			ports[i] = strconv.Itoa(l.Port)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", g.Process, g.PID, strings.Join(ports, ", "))
	}
	tw.Flush()
}

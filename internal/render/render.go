// Package render prints listener tables and JSON to an io.Writer.
package render

import (
	"encoding/json"
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

type jsonListener struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	Process string `json:"process"`
	PID     *int   `json:"pid"`
}

type jsonGroup struct {
	Process string `json:"process"`
	PID     *int   `json:"pid"`
	Ports   []int  `json:"ports"`
}

// PrintJSON writes ls as an indented JSON array.
func PrintJSON(w io.Writer, ls []scan.Listener) {
	out := make([]jsonListener, 0, len(ls))
	for _, l := range ls {
		out = append(out, jsonListener{
			Addr:    l.Addr,
			Port:    l.Port,
			Process: l.Process,
			PID:     pidInt(l.PID),
		})
	}
	writeJSON(w, out)
}

// PrintGroupedJSON writes gs as an indented JSON array.
func PrintGroupedJSON(w io.Writer, gs []group.Group) {
	out := make([]jsonGroup, 0, len(gs))
	for _, g := range gs {
		ports := make([]int, 0, len(g.Listeners))
		for _, l := range g.Listeners {
			ports = append(ports, l.Port)
		}
		out = append(out, jsonGroup{
			Process: g.Process,
			PID:     pidInt(g.PID),
			Ports:   ports,
		})
	}
	writeJSON(w, out)
}

// pidInt converts a PID string to a number, returning nil when it is unknown.
func pidInt(s string) *int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

func writeJSON(w io.Writer, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	fmt.Fprintln(w, string(b))
}

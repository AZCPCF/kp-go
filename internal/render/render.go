// Package render prints listener tables and JSON to an io.Writer.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/azcpcf/kp-go/internal/group"
	"github.com/azcpcf/kp-go/internal/scan"
	"github.com/azcpcf/kp-go/internal/ui"
)

// cell is one table cell. The style is applied after the text is padded so
// ANSI sequences never affect column alignment.
type cell struct {
	text  string
	style lipgloss.Style
}

// writeTable writes header and rows as left-aligned columns with two spaces
// of padding, matching text/tabwriter's output for plain text.
func writeTable(w io.Writer, header []cell, rows [][]cell) {
	widths := make([]int, len(header))
	for i, c := range header {
		widths[i] = utf8.RuneCountInString(c.text)
	}
	for _, row := range rows {
		for i, c := range row {
			if n := utf8.RuneCountInString(c.text); n > widths[i] {
				widths[i] = n
			}
		}
	}
	writeRow := func(row []cell) {
		for i, c := range row {
			text := c.text
			if i < len(row)-1 {
				text += strings.Repeat(" ", widths[i]-utf8.RuneCountInString(text)+2)
			}
			fmt.Fprint(w, c.style.Render(text))
		}
		fmt.Fprintln(w)
	}
	writeRow(header)
	for _, row := range rows {
		writeRow(row)
	}
}

// PrintTable writes ls as a styled port/proto/address/process/pid table.
func PrintTable(w io.Writer, ls []scan.Listener, t ui.Theme) {
	header := []cell{
		{"PORT", t.Header},
		{"PROTO", t.Header},
		{"ADDR", t.Header},
		{"PROCESS", t.Header},
		{"PID", t.Header},
	}
	rows := make([][]cell, 0, len(ls))
	for _, l := range ls {
		rows = append(rows, []cell{
			{strconv.Itoa(l.Port), t.Port},
			{l.Proto, t.Proto(l.Proto)},
			{l.Addr, t.Muted},
			{l.Process, t.Process},
			{l.PID, lipgloss.Style{}},
		})
	}
	writeTable(w, header, rows)
}

// PrintGrouped writes gs as a styled process/pid/ports table.
func PrintGrouped(w io.Writer, gs []group.Group, t ui.Theme) {
	header := []cell{
		{"PROCESS", t.Header},
		{"PID", t.Header},
		{"PORTS", t.Header},
	}
	rows := make([][]cell, 0, len(gs))
	for _, g := range gs {
		ports := make([]string, len(g.Listeners))
		for i, l := range g.Listeners {
			ports[i] = strconv.Itoa(l.Port)
		}
		rows = append(rows, []cell{
			{g.Process, t.Process},
			{g.PID, lipgloss.Style{}},
			{strings.Join(ports, ", "), t.Port},
		})
	}
	writeTable(w, header, rows)
}

type jsonListener struct {
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
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
			Proto:   l.Proto,
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

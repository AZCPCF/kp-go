// Package ui holds the terminal palette, styles and color rules for kp.
package ui

import "charm.land/lipgloss/v2"

// Palette used by the kp theme.
const (
	colorHeader  = "212"
	colorPort    = "39"
	colorProcess = "214"
	colorMuted   = "244"
	colorError   = "196"
	colorSuccess = "40"
	colorTCP     = "42"
	colorUDP     = "213"
)

// Theme is the set of named styles used by kp output. The zero value renders
// plain text, so a disabled theme needs no special casing.
type Theme struct {
	Header  lipgloss.Style // table headers
	Port    lipgloss.Style // port numbers
	Process lipgloss.Style // process names
	Muted   lipgloss.Style // addresses and other secondary text
	Error   lipgloss.Style // failures
	Success lipgloss.Style // successful kills
	TCP     lipgloss.Style // TCP protocol labels
	UDP     lipgloss.Style // UDP protocol labels
}

// New returns the kp theme. When enabled is false every style is a no-op.
func New(enabled bool) Theme {
	if !enabled {
		return Theme{}
	}
	return Theme{
		Header:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorHeader)),
		Port:    lipgloss.NewStyle().Foreground(lipgloss.Color(colorPort)),
		Process: lipgloss.NewStyle().Foreground(lipgloss.Color(colorProcess)),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)),
		TCP:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorTCP)),
		UDP:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorUDP)),
	}
}

// Proto returns the style for a protocol name, falling back to muted.
func (t Theme) Proto(proto string) lipgloss.Style {
	switch proto {
	case "tcp":
		return t.TCP
	case "udp":
		return t.UDP
	}
	return t.Muted
}

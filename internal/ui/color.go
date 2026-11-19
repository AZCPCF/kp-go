package ui

import (
	"fmt"
	"io"
	"os"
)

// ColorMode controls when kp emits ANSI color.
type ColorMode string

// Supported color modes.
const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// ParseColorMode validates a --color value.
func ParseColorMode(s string) (ColorMode, error) {
	switch ColorMode(s) {
	case ColorAuto, ColorAlways, ColorNever:
		return ColorMode(s), nil
	}
	return "", fmt.Errorf("invalid color mode %q (want auto, always or never)", s)
}

// UseColor reports whether output should be colored. An explicit always wins
// over NO_COLOR and a non-terminal; auto needs both an unset NO_COLOR and a
// terminal; never is always off.
func UseColor(mode ColorMode, noColor, isTTY bool) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	return !noColor && isTTY
}

// IsTTY reports whether w is a terminal. Writers that are not *os.File, such
// as pipes and buffers, are not terminals.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

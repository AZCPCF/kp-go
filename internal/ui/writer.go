package ui

import (
	"io"
	"strings"

	"charm.land/lipgloss/v2"
)

// StyleWriter returns a writer that renders every line with s. It is meant
// for line-oriented output such as kill messages; a zero style is a no-op.
func StyleWriter(w io.Writer, s lipgloss.Style) io.Writer {
	return &styleWriter{w: w, style: s}
}

type styleWriter struct {
	w     io.Writer
	style lipgloss.Style
}

func (s *styleWriter) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = s.style.Render(line)
		}
	}
	if _, err := io.WriteString(s.w, strings.Join(lines, "\n")); err != nil {
		return 0, err
	}
	return len(p), nil
}

package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// boxed returns the part of p inside a one-cell border: where the session's
// program draws once the pane has a box around it.
func boxed(p sessionPane) sessionPane {
	return sessionPane{s: p.s, x: p.x + 1, y: p.y + 1, w: max(1, p.w-2), h: max(1, p.h-2)}
}

// renderPaneBox draws a rounded border in the border style around a session's
// screen, which is w×h cells inside the border.
func renderPaneBox(screen string, w, h int, border lipgloss.Style) string {
	edge := lipgloss.RoundedBorder()
	lines := splitLines(screen)
	left, right := border.Render(edge.Left), border.Render(edge.Right)

	rows := make([]string, 0, h+2)
	rows = append(rows, border.Render(edge.TopLeft+strings.Repeat(edge.Top, w)+edge.TopRight))
	for i := range h {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		// The reset keeps a colour the program left on (a background, say)
		// from running into the right border.
		rows = append(rows, left+padLine(l, w)+"\x1b[m"+right)
	}
	rows = append(rows, border.Render(edge.BottomLeft+strings.Repeat(edge.Bottom, w)+edge.BottomRight))

	return strings.Join(rows, "\n")
}

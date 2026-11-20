package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// boxed returns the part of p inside a one-cell border: where the session's
// program draws once the pane has a box around it.
func boxed(p sessionPane) sessionPane {
	return sessionPane{s: p.s, x: p.x + 1, y: p.y + 1, w: max(1, p.w-2), h: max(1, p.h-2)}
}

// paneTitle is the name in a pane's top border: "nvim", or the agent's name.
func (m Model) paneTitle(s *session.Session) string {
	if m.current != nil && m.current.ws != nil && s == m.current.ws.Editor {
		return "nvim"
	}
	return agentTitle(normalizedProvider(s.Provider), s.Title)
}

// renderPaneBox draws a rounded border in the border style around a session's
// screen, which is w×h cells inside the border, with the title in the top
// border after the corner and one line.
func renderPaneBox(screen, title string, w, h int, border lipgloss.Style) string {
	edge := lipgloss.RoundedBorder()
	lines := splitLines(screen)
	left, right := border.Render(edge.Left), border.Render(edge.Right)

	// One line before the title and at least one after it.
	title = truncateTo(title, w-2)
	fill := strings.Repeat(edge.Top, w-1-lipgloss.Width(title))

	rows := make([]string, 0, h+2)
	rows = append(rows, border.Render(edge.TopLeft+edge.Top+title+fill+edge.TopRight))
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

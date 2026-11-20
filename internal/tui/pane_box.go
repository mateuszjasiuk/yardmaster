package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// framed is the part of p a session's program draws in: inside its box's
// one-cell border when panes have boxes, else all of p.
func (m Model) framed(p sessionPane) sessionPane {
	if !m.paneBoxes {
		return p
	}
	return sessionPane{s: p.s, x: p.x + 1, y: p.y + 1, w: max(1, p.w-2), h: max(1, p.h-2)}
}

// paneFrame is a pane's screen as drawn: in its box with the border in the
// border style when panes have boxes, else as it is.
func (m Model) paneFrame(screen string, p sessionPane, border lipgloss.Style) string {
	if !m.paneBoxes {
		return screen
	}
	return renderPaneBox(screen, paneTitle(p.s), p.w, p.h, border)
}

// paneTitle is the name in a pane's top border: "nvim", or the agent's name.
func paneTitle(s *session.Session) string {
	if s.Kind == session.Editor {
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
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		// The reset keeps a colour the program left on (a background, say)
		// from running into the right border.
		rows = append(rows, left+padLine(line, w)+"\x1b[m"+right)
	}
	rows = append(rows, bottomBorder(edge, border, "", w+2))

	return strings.Join(rows, "\n")
}

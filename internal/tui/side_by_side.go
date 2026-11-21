package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/mateuszjasiuk/yardmaster/internal/config"
	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// sideBySideMinWidth is the narrowest room for sessions that fits nvim and the
// agent next to each other, with the divider between them or in two boxes.
const sideBySideMinWidth = 2*minSessionWidth + 1

// toggleSideBySide shows nvim and the agent next to each other, or one at a
// time again, resizes them, and remembers the choice.
func (m Model) toggleSideBySide() (tea.Model, tea.Cmd) {
	m.sideBySide = !m.sideBySide
	if m.state != nil {
		m.state.SideBySide = m.sideBySide
	}

	m.resizeSessions()
	return m, m.writeStateCmd()
}

// sideBySideIn reports whether ws lays out its nvim and agent side by side:
// turned on, room for both, and an agent to show. It doesn't depend on the
// screen, so switching views never resizes a session.
func (m Model) sideBySideIn(ws *session.Workspace) bool {
	return m.sideBySideFits() && ws != nil && ws.Editor != nil && ws.ActiveAgentSession() != nil
}

// sideBySideFits reports whether side by side is on and the terminal has room
// for it. The side panel hides first when both don't fit.
func (m Model) sideBySideFits() bool {
	return m.sideBySide && m.width >= sideBySideMinWidth
}

// sideBySideShown reports whether nvim and the agent are on screen together:
// the current worktree is side by side and one of the two has focus.
func (m Model) sideBySideShown() bool {
	return (m.screen == screenEditor || m.screen == screenAgent) &&
		m.current != nil && m.sideBySideIn(m.current.ws)
}

// cycleScreen moves to the next session view, or the previous one. While nvim
// and the agent are side by side they are one stop, and coming back from the
// terminal focuses the one used last.
func (m Model) cycleScreen(forward bool) Model {
	switch {
	case m.sideBySideShown():
		m = m.rememberSideBySideFocus()
		m.screen = screenTerminal
	case m.screen == screenTerminal && m.sideBySideIn(m.current.ws):
		m.screen = screenEditor
		if s, ok := m.sideBySideFocus[m.current.key]; ok {
			m.screen = s
		}
	case forward:
		m.screen = m.screen.next()
	default:
		m.screen = m.screen.prev()
	}

	if m.screen == screenAgent {
		m.clearWorkspaceAttention()
	}
	return m
}

// rememberSideBySideFocus records which of nvim and the agent has focus, when
// they are side by side, before the screen changes.
func (m Model) rememberSideBySideFocus() Model {
	if !m.sideBySideShown() {
		return m
	}

	if m.sideBySideFocus == nil {
		m.sideBySideFocus = map[string]screen{}
	}
	m.sideBySideFocus[m.current.key] = m.screen
	return m
}

// sideBySideRects splits the room for sessions into where nvim and the agent
// draw: inside two boxes next to each other, or in two panes with a divider
// column between them.
func (m Model) sideBySideRects(room sessionPane) (editor, agent sessionPane) {
	divider := 1
	if m.paneBoxes {
		divider = 0
	}

	left := (room.w - divider) / 2
	editor = m.framed(sessionPane{x: room.x, y: room.y, w: left, h: room.h})
	agent = m.framed(sessionPane{x: room.x + left + divider, y: room.y, w: room.w - divider - left, h: room.h})

	return editor, agent
}

// renderSideBySide draws nvim and the agent next to each other, the one
// without focus in gray. In boxes, the focused one's border is lit; else the
// divider always borders the focused pane, so it is lit, as between two
// terminal panes.
func (m Model) renderSideBySide() string {
	panes := m.sessionPanes()
	if !m.paneBoxes {
		left, right := panes[0], panes[1]
		return joinVerticalSplit(m.sideBySidePane(left), m.sideBySidePane(right), cAccent, left.h, left.w)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, m.sideBySideBox(panes[0]), m.sideBySideBox(panes[1]))
}

// sideBySideBox is a pane's screen in its box, the border lit when the pane
// has focus.
func (m Model) sideBySideBox(p sessionPane) string {
	border := barSep
	if p.s == m.activeSession() {
		border = accentStyle
	}
	return m.paneFrame(m.sideBySidePane(p), p, border)
}

// sideBySidePane is a pane's screen, dimmed when it doesn't have focus: faint
// over its own colours, or plain gray text (display.side_by_side_dim).
func (m Model) sideBySidePane(p sessionPane) string {
	if p.s == m.activeSession() {
		return m.paneScreen(p.s)
	}

	if m.sideBySideDim == config.DimGray {
		lines := splitLines(ansi.Strip(m.paneScreen(p.s)))
		for i, l := range lines {
			lines[i] = dimPaneStyle.Render(l)
		}
		return strings.Join(lines, "\n")
	}
	return faintLines(m.paneScreen(p.s))
}

// faintLines turns on the terminal's faint style at the start of each line and
// again after each SGR sequence, since the program's own resets turn it off.
// Each line ends with a reset, so faint doesn't spill into the border after it.
func faintLines(s string) string {
	const faint = "\x1b[2m"
	lines := splitLines(s)
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(faint)
		for {
			j := strings.Index(l, "\x1b[")
			if j < 0 {
				break
			}
			end := j + 2
			for end < len(l) && (l[end] < 0x40 || l[end] > 0x7e) {
				end++
			}
			if end == len(l) {
				break
			}
			b.WriteString(l[:end+1])
			if l[end] == 'm' {
				b.WriteString(faint)
			}
			l = l[end+1:]
		}
		b.WriteString(l)
		b.WriteString("\x1b[m")
	}
	return b.String()
}

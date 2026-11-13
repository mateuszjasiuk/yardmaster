package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/KCaverly/caretaker/internal/session"
)

// sideBySideMinWidth is the narrowest room for sessions that fits nvim and the
// agent next to each other, with the divider between them.
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

// sideBySideWidths splits w columns into nvim's and the agent's, with one
// column for the divider between them.
func sideBySideWidths(w int) (left, right int) {
	left = (w - 1) / 2
	return left, w - 1 - left
}

// renderSideBySide draws nvim and the agent next to each other, the one
// without focus in gray. The divider always borders the focused pane, so it is
// lit, as between two terminal panes.
func (m Model) renderSideBySide() string {
	panes := m.sessionPanes()
	left, right := panes[0], panes[1]
	return joinVerticalSplit(m.sideBySidePane(left), m.sideBySidePane(right), cAccent, left.h, left.w)
}

// sideBySidePane is a pane's screen, dropped to plain gray text when it
// doesn't have focus.
func (m Model) sideBySidePane(p sessionPane) string {
	if p.s == m.activeSession() {
		return p.s.Render()
	}

	lines := splitLines(ansi.Strip(p.s.Render()))
	for i, l := range lines {
		lines[i] = dimPaneStyle.Render(l)
	}
	return strings.Join(lines, "\n")
}

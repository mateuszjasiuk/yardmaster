package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// sessionPane is one session on screen and the cells it fills, in screen
// coordinates.
type sessionPane struct {
	s          *session.Session
	x, y, w, h int
}

// contains reports whether screen cell (x, y) is inside the pane.
func (p sessionPane) contains(x, y int) bool {
	return x >= p.x && x < p.x+p.w && y >= p.y && y < p.y+p.h
}

// sessionPanes lists the sessions drawn on a session screen with their
// rectangles: nvim and the agent side by side, every pane of an unzoomed
// terminal split, else the active session alone. When nvim and the agent sit
// in a box, their rectangle is the part inside its border. The cursor and the
// mouse read it, so they agree on where each session is. Empty when no session
// is on screen.
func (m Model) sessionPanes() []sessionPane {
	s := m.activeSession()
	if s == nil {
		return nil
	}
	x, y := m.sidePanelW(), m.barHeight()
	w, h := m.sessionSize()
	room := sessionPane{s: s, x: x, y: y, w: w, h: h}

	if m.sideBySideShown() {
		ws := m.current.ws
		editor, agent := m.sideBySideRects(room)
		editor.s, agent.s = ws.Editor, ws.ActiveAgentSession()
		return []sessionPane{editor, agent}
	}
	if m.screen == screenTerminal {
		ws := m.current.ws
		if !ws.TermZoomed && len(ws.Terms) > 1 && ws.TermLayout != nil {
			var panes []sessionPane
			for _, b := range session.ComputePaneBounds(ws.TermLayout, x, y, w, h) {
				if b.Idx < len(ws.Terms) && ws.Terms[b.Idx] != nil {
					panes = append(panes, sessionPane{s: ws.Terms[b.Idx], x: b.X, y: b.Y, w: b.W, h: b.H})
				}
			}
			return panes
		}
		return []sessionPane{room}
	}
	return []sessionPane{m.framed(room)}
}

// paneAt returns the pane under screen cell (x, y), if any.
func (m Model) paneAt(x, y int) (sessionPane, bool) {
	for _, p := range m.sessionPanes() {
		if p.contains(x, y) {
			return p, true
		}
	}
	return sessionPane{}, false
}

// paneOf returns the pane session s is drawn in, if it is on screen.
func (m Model) paneOf(s *session.Session) (sessionPane, bool) {
	for _, p := range m.sessionPanes() {
		if p.s == s {
			return p, true
		}
	}
	return sessionPane{}, false
}

// focusPane gives the keyboard to s, one of the current worktree's sessions on
// screen. Focusing the agent marks its news as seen, like opening its view.
func (m Model) focusPane(s *session.Session) Model {
	if m.current == nil || m.current.ws == nil {
		return m
	}

	ws := m.current.ws
	switch s {
	case ws.Editor:
		m.screen = screenEditor
	case ws.ActiveAgentSession():
		m.screen = screenAgent
		m.clearWorkspaceAttention()
	default:
		for i, t := range ws.Terms {
			if t == s {
				ws.ActiveTerm = i
			}
		}
	}
	return m
}

// sessionCursor places the focused session's cursor on screen, or returns nil
// when it hides its cursor.
func (m Model) sessionCursor() *tea.Cursor {
	p, ok := m.paneOf(m.activeSession())
	if !ok {
		return nil
	}
	cx, cy, visible := p.s.Cursor()
	if !visible || showsSign(p.s) {
		return nil
	}
	return tea.NewCursor(p.x+cx, p.y+cy)
}

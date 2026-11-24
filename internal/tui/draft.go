package tui

import (
	"fmt"
	"time"

	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// draftEscWindow is how soon a second Esc must follow the first to count as the
// double Esc that empties Claude's input box. A guess: Claude does not document
// its own window.
const draftEscWindow = time.Second

// draft marks an agent session whose input box may hold text the user typed.
// ym cannot read the box; it only sees the keys it forwards, so the mark is
// kept from those keys. While it is set, queued messages wait.
type draft struct {
	// entered is set by an Enter while the agent was not busy. The text counts
	// as sent once the agent turns busy: an Enter that picks an @file or
	// /command item leaves the agent idle and the text in the box.
	entered bool
	// escAt is when the last Esc came, to spot a double Esc.
	escAt time.Time
}

// noteDraftKey updates s's draft mark for a key the user sends it. ctrl+c and a
// double Esc empty the box; a single Esc or an Enter only changes a draft that
// is already marked; every other key marks one.
func (m Model) noteDraftKey(s *session.Session, key string) {
	if s.Kind != session.Agent {
		return
	}
	// On a permission prompt or a question, keys answer it and leave the box
	// alone.
	if m.agentStatus[s.Pid()].Status == "waiting" {
		return
	}

	d, has := m.drafts[s]
	switch key {
	// Two Esc that arrive in one read come in as a single alt+esc.
	case "ctrl+c", "alt+esc":
		delete(m.drafts, s)
	case "esc":
		if !has {
			return
		}
		doubleEsc := !d.escAt.IsZero() && time.Since(d.escAt) <= draftEscWindow
		if doubleEsc {
			delete(m.drafts, s)
			return
		}
		d.escAt = time.Now()
		m.drafts[s] = d
	case "enter":
		if !has {
			return
		}
		// A busy Claude queues the typed message itself and empties the box.
		if m.agentBusy(s) {
			delete(m.drafts, s)
			return
		}
		m.drafts[s] = draft{entered: true}
	default:
		m.drafts[s] = draft{}
	}
}

// noteDraftPaste marks a draft in s when the user pastes into an agent.
func (m Model) noteDraftPaste(s *session.Session) {
	if s.Kind == session.Agent {
		m.drafts[s] = draft{}
	}
}

// settleDrafts drops the marks of drafts the agent has taken (it turned busy
// after the Enter) and of sessions that have ended.
func (m Model) settleDrafts() {
	for s, d := range m.drafts {
		taken := d.entered && m.agentBusy(s)
		if !s.Alive() || taken {
			delete(m.drafts, s)
		}
	}
}

// queueHeld reports whether key's queue waits because its active agent has a
// draft.
func (m Model) queueHeld(key string) bool {
	if m.state == nil || len(m.state.Queue(key)) == 0 {
		return false
	}
	ws, ok := m.mgr.Workspace(key)
	if !ok {
		return false
	}

	return m.hasDraft(ws.ActiveAgentSession())
}

// heldNote is the sign in an agent pane's top border while its queue waits
// for the user's draft, or "".
func (m Model) heldNote(s *session.Session) string {
	if m.current == nil || m.current.ws == nil {
		return ""
	}
	onActiveAgent := s == m.current.ws.ActiveAgentSession()
	if !onActiveAgent || !m.queueHeld(m.current.key) {
		return ""
	}

	return fmt.Sprintf("%d queued · waiting for your draft", len(m.state.Queue(m.current.key)))
}

// hasDraft reports whether agent s may hold text the user typed.
func (m Model) hasDraft(s *session.Session) bool {
	_, has := m.drafts[s]
	return has
}

// agentBusy reports whether ym last saw agent s in a turn.
func (m Model) agentBusy(s *session.Session) bool {
	return m.agentStatus[s.Pid()].Status == "busy"
}

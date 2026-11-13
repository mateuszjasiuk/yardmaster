package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/KCaverly/caretaker/internal/session"
)

// sideBySideModel is sidePanelModel on the editor screen, with an agent.
func sideBySideModel(t *testing.T, width int) Model {
	t.Helper()
	m, _, mgr := sidePanelModel(t, width)
	m.screen = screenEditor
	if len(m.current.ws.Agents) == 0 {
		if _, err := mgr.SpawnAgent("repo/wt", t.TempDir(), session.Spec{Kind: session.Agent, Argv: []string{"cat"}}, 10, 10); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func pressSideBySide(t *testing.T, m Model) Model {
	t.Helper()
	mm, _ := m.handleKey(tea.KeyPressMsg{Code: '\\', Mod: tea.ModCtrl | tea.ModShift})
	return mm.(Model)
}

// sizeOf is a session's width and height.
func sizeOf(s *session.Session) session.Size {
	w, h := s.Size()
	return session.Size{W: w, H: h}
}

func TestSideBySideSplitsTheWidthAndIsSaved(t *testing.T) {
	m := sideBySideModel(t, 150)
	ws := m.current.ws

	m = pressSideBySide(t, m)
	if !m.state.SideBySide {
		t.Fatal("the choice should be kept in the state")
	}
	w, h := m.sessionSize()
	left, right := sideBySideWidths(w)
	if left+1+right != 150 || left < minSessionWidth || right < minSessionWidth {
		t.Fatalf("widths %d+1+%d should fill 150 columns, each at least %d", left, right, minSessionWidth)
	}
	if got := sizeOf(ws.Editor); got != (session.Size{W: left, H: h}) {
		t.Fatalf("nvim is %+v, want %dx%d", got, left, h)
	}
	if got := sizeOf(ws.ActiveAgentSession()); got != (session.Size{W: right, H: h}) {
		t.Fatalf("the agent is %+v, want %dx%d", got, right, h)
	}
	if got := sizeOf(ws.Terms[0]); got.W != 150 {
		t.Fatalf("the terminal should keep the full width, got %d", got.W)
	}

	if vis := m.visibleSessions(); len(vis) != 2 || vis[0] != ws.Editor || vis[1] != ws.ActiveAgentSession() {
		t.Fatalf("nvim and the agent should both be visible, got %v", vis)
	}
	lines := strings.Split(m.View().Content, "\n")
	if row := []rune(ansi.Strip(lines[m.barHeight()])); len(row) <= left || row[left] != '│' {
		t.Fatalf("the divider should be at column %d: %q", left, string(row))
	}
	if c := m.View().Cursor; c != nil && c.X >= left {
		t.Fatalf("nvim has focus, so its cursor should be on the left, got x=%d", c.X)
	}

	m = pressSideBySide(t, m)
	if got := sizeOf(ws.Editor); got.W != 150 {
		t.Fatalf("turning it off should give nvim the full width, got %d", got.W)
	}
	if vis := m.visibleSessions(); len(vis) != 1 {
		t.Fatalf("only nvim should be visible again, got %d sessions", len(vis))
	}
}

func TestSideBySideKeepsSizesAcrossViews(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	ws := m.current.ws
	before := sizeOf(ws.Editor)

	mm, _ := m.gotoScreen(screenAgent)
	m = mm.(Model)
	if !m.sideBySideShown() || m.activeSession() != ws.ActiveAgentSession() {
		t.Fatal("the agent key should move focus to the agent, both still on screen")
	}
	mm, _ = m.gotoScreen(screenTerminal)
	m = mm.(Model)
	if m.sideBySideShown() || len(m.visibleSessions()) != 1 {
		t.Fatal("the terminal screen shows the terminal alone")
	}
	if got := sizeOf(ws.Editor); got != before {
		t.Fatalf("switching views should not resize nvim: %+v, was %+v", got, before)
	}
}

func TestSideBySideInANarrowTerminal(t *testing.T) {
	for _, c := range []struct {
		width         int
		panel, joined bool
	}{
		{width: 160, panel: true, joined: true},
		{width: 140, panel: false, joined: true},
		{width: 110, panel: false, joined: false},
	} {
		m := sideBySideModel(t, c.width)
		m = pressSidePanel(t, m)
		m = pressSideBySide(t, m)
		if got := m.sidePanelW() > 0; got != c.panel {
			t.Errorf("%d columns: session list shown = %v, want %v", c.width, got, c.panel)
		}
		if got := m.sideBySideShown(); got != c.joined {
			t.Errorf("%d columns: side by side shown = %v, want %v", c.width, got, c.joined)
		}
		w, _ := m.sessionSize()
		if got := sizeOf(m.current.ws.Editor).W; c.joined && got >= w || !c.joined && got != w {
			t.Errorf("%d columns: nvim is %d wide in %d columns of room", c.width, got, w)
		}
		if !m.sideBySide || !m.sidePanel {
			t.Errorf("%d columns: both settings should stay on", c.width)
		}
	}
}

func TestSideBySideWithoutAnAgent(t *testing.T) {
	m, _, mgr := sidePanelModel(t, 150)
	m.screen = screenEditor
	for len(m.current.ws.Agents) > 0 {
		mgr.CloseAgent("repo/wt", 0)
	}

	m = pressSideBySide(t, m)
	if m.sideBySideShown() {
		t.Fatal("with no agent nvim should be alone")
	}
	if got := sizeOf(m.current.ws.Editor).W; got != 150 {
		t.Fatalf("nvim alone should get the full width, got %d", got)
	}
}

package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/KCaverly/caretaker/internal/config"
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

func pressCycle(t *testing.T, m Model, forward bool) Model {
	t.Helper()
	code := ']'
	if !forward {
		code = '['
	}
	mm, _ := m.handleKey(tea.KeyPressMsg{Code: code, Mod: tea.ModAlt})
	return mm.(Model)
}

func TestSideBySideIsOneStopWhenCycling(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))

	m = pressCycle(t, m, true)
	if m.screen != screenTerminal {
		t.Fatalf("from nvim side by side, cycle should go to the terminal, got %v", m.screen)
	}
	m = pressCycle(t, m, true)
	if m.screen != screenEditor {
		t.Fatalf("back from the terminal, nvim should have focus, got %v", m.screen)
	}

	mm, _ := m.gotoScreen(screenAgent)
	m = pressCycle(t, mm.(Model), false)
	if m.screen != screenTerminal {
		t.Fatalf("from the agent side by side, cycle back should go to the terminal, got %v", m.screen)
	}
	m = pressCycle(t, m, false)
	if m.screen != screenAgent {
		t.Fatalf("back from the terminal, the agent used last should have focus, got %v", m.screen)
	}

	m = pressSideBySide(t, m)
	if m = pressCycle(t, m, true); m.screen != screenTerminal {
		t.Fatalf("one at a time, cycle goes agent → terminal as before, got %v", m.screen)
	}
	if m = pressCycle(t, m, true); m.screen != screenEditor {
		t.Fatalf("one at a time, cycle goes terminal → nvim as before, got %v", m.screen)
	}
	if m = pressCycle(t, m, true); m.screen != screenAgent {
		t.Fatalf("one at a time, cycle goes nvim → agent as before, got %v", m.screen)
	}
}

func TestSideBySideClickMovesFocus(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	agent := m.current.ws.ActiveAgentSession()
	m.attention[agent.Pid()] = attnEntry{level: attnDone, key: m.current.key}
	panes := m.sessionPanes()
	left, right := panes[0], panes[1]

	click := func(x int) {
		t.Helper()
		mm, _ := m.handleMouseClick(tea.MouseClickMsg{X: x, Y: left.y + 1, Button: tea.MouseLeft})
		m = mm.(Model)
	}
	click(left.x + left.w) // the divider
	if m.screen != screenEditor {
		t.Fatalf("a click on the divider should not move focus, got %v", m.screen)
	}
	if m.attention[agent.Pid()].level != attnDone {
		t.Fatal("the agent's news is unread while only nvim has had focus")
	}

	click(right.x + 2)
	if m.screen != screenAgent || !m.sideBySideShown() {
		t.Fatalf("a click in the agent should focus it, both still shown; screen %v", m.screen)
	}
	if m.attention[agent.Pid()].level != attnSeen {
		t.Fatal("focusing the agent should mark its news as seen")
	}

	click(left.x + 2)
	if m.screen != screenEditor {
		t.Fatalf("a click in nvim should focus it, got %v", m.screen)
	}
}

func TestClickFocusesTerminalPanes(t *testing.T) {
	m, ctrl, mgr := sidePanelModel(t, 120)
	w, h := m.sessionSize()
	if _, err := mgr.SplitTermPane("repo/wt", t.TempDir(), ctrl.TermSpec(), session.SplitV, w, h); err != nil {
		t.Fatal(err)
	}
	mgr.ResizeTermPanes("repo/wt", w, h)
	ws := m.current.ws
	left := m.sessionPanes()[0]
	click := func(x int) {
		t.Helper()
		mm, _ := m.handleMouseClick(tea.MouseClickMsg{X: x, Y: left.y + 1, Button: tea.MouseLeft})
		m = mm.(Model)
	}

	click(left.x + left.w) // the divider
	if ws.ActiveTerm != 1 {
		t.Fatalf("a click on the divider should not move focus, active=%d", ws.ActiveTerm)
	}
	click(left.x + 2)
	if ws.ActiveTerm != 0 {
		t.Fatalf("a click in the left pane should focus it, active=%d", ws.ActiveTerm)
	}

	mgr.ZoomTermPane("repo/wt")
	click(w - 2)
	if ws.ActiveTerm != 0 {
		t.Fatalf("a zoomed pane fills the screen, so a click keeps its focus, active=%d", ws.ActiveTerm)
	}
}

func TestSideBySideFollowsTheFirstAndLastAgent(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	ws := m.current.ws
	w, _ := m.sessionSize()
	left, right := sideBySideWidths(w)

	for len(ws.Agents) > 0 {
		agent := ws.Agents[0]
		mm, _ := m.closeBoardAgent(boardRow{kind: rowAgent, key: m.current.key, agentIdx: 0, pid: agent.Pid()})
		m = mm.(Model)
	}
	if got := sizeOf(ws.Editor).W; got != w {
		t.Fatalf("with its last agent closed, nvim should get the full width, got %d", got)
	}

	sz := m.sizesFor(m.current.key).Agent
	if sz.W != right {
		t.Fatalf("the first agent should start at its half, %d, got %d", right, sz.W)
	}
	if _, err := m.mgr.SpawnAgent(m.current.key, t.TempDir(), session.Spec{Kind: session.Agent, Argv: []string{"cat"}}, sz.W, sz.H); err != nil {
		t.Fatal(err)
	}
	m.resizeSessions()
	if got := sizeOf(ws.Editor).W; got != left || !m.sideBySideShown() {
		t.Fatalf("once an agent runs, nvim should be back on the left at %d, got %d", left, got)
	}
}

func TestSideBySideShowsTheAgentPickedWithF4(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	sz := m.sizesFor(m.current.key).Agent
	second, err := m.mgr.SpawnAgent(m.current.key, t.TempDir(), session.Spec{Kind: session.Agent, Argv: []string{"cat"}}, sz.W, sz.H)
	if err != nil {
		t.Fatal(err)
	}
	m.current.ws.ActiveAgent = 0

	mm, _ := m.handleKey(tea.KeyPressMsg{Code: tea.KeyF4})
	m = mm.(Model)
	panes := m.sessionPanes()
	if len(panes) != 2 || panes[1].s != second {
		t.Fatal("F4 should put the next agent on the right")
	}
	if got := sizeOf(second).W; got != panes[1].w {
		t.Fatalf("the next agent should already have the right pane's width %d, got %d", panes[1].w, got)
	}
}

func TestSideBySideStaysOutOfOverlays(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))

	m.helpOpen = true
	if len(m.visibleSessions()) != 0 {
		t.Fatal("help covers the sessions, so none is visible")
	}
	if got, want := m.View().Content, m.renderBar()+"\n"+m.renderHelp(m.height-m.barHeight()); got != want {
		t.Fatal("help should be drawn instead of the two panes")
	}
}

func TestSideBySideLightsBothBarIcons(t *testing.T) {
	m := sideBySideModel(t, 150)
	icons := func() (editor, agent string) {
		t.Helper()
		for _, z := range m.barZones() {
			switch z.s {
			case screenEditor:
				editor = z.glyph
			case screenAgent:
				agent = z.glyph
			}
		}
		return editor, agent
	}
	eg, ag := m.destinationGlyph(screenEditor), m.destinationGlyph(screenAgent)

	if _, agent := icons(); agent != boldDim.Render(ag) {
		t.Fatal("one at a time, the agent icon is dim while nvim is on screen")
	}

	m = pressSideBySide(t, m)
	for _, s := range []screen{screenEditor, screenAgent} {
		mm, _ := m.gotoScreen(s)
		m = mm.(Model)
		if editor, agent := icons(); editor != boldGreen.Render(eg) || agent != boldPurple.Render(ag) {
			t.Fatalf("side by side with %v focused, both icons should be lit", s)
		}
	}
}

func TestSideBySideDimsThePaneWithoutFocus(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	left, right := m.sessionPanes()[0], m.sessionPanes()[1]
	if m.sideBySideDim != config.DimFaint {
		t.Fatalf("the default should be faint, got %q", m.sideBySideDim)
	}

	if got := m.sideBySidePane(left); got != left.s.Render() {
		t.Fatal("the focused pane is drawn as it is")
	}
	if got, want := m.sideBySidePane(right), faintLines(right.s.Render()); got != want {
		t.Fatal("the pane without focus should be faint over its own colours")
	}
}

func TestFaintLinesKeepsColoursAndFaintThroughResets(t *testing.T) {
	const f, reset = "\x1b[2m", "\x1b[m"
	for in, want := range map[string]string{
		"plain":                f + "plain" + reset,
		"a\nb":                 f + "a" + reset + "\n" + f + "b" + reset,
		"\x1b[32mgo\x1b[0m on": f + "\x1b[32m" + f + "go\x1b[0m" + f + " on" + reset,
		"\x1b[38;2;1;2;3mx":    f + "\x1b[38;2;1;2;3m" + f + "x" + reset,
		"\x1b[2Kcleared":       f + "\x1b[2Kcleared" + reset,
		"cut \x1b[3":           f + "cut \x1b[3" + reset,
	} {
		if got := faintLines(in); got != want {
			t.Errorf("faintLines(%q) =\n%q\nwant\n%q", in, got, want)
		}
	}
}

func TestSideBySideGraysThePaneWithoutFocus(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	m.sideBySideDim = config.DimGray
	left, right := m.sessionPanes()[0], m.sessionPanes()[1]

	if got := m.sideBySidePane(left); got != left.s.Render() {
		t.Fatal("the focused pane keeps its own colours")
	}
	got := m.sideBySidePane(right)
	if ansi.Strip(got) != ansi.Strip(right.s.Render()) {
		t.Fatal("the pane without focus keeps its text")
	}
	for _, l := range strings.Split(got, "\n") {
		if l != dimPaneStyle.Render(ansi.Strip(l)) {
			t.Fatalf("every line of the pane without focus should be plain gray: %q", l)
		}
	}
}

func TestKeyLabelShowsTheShiftedSymbol(t *testing.T) {
	for key, want := range map[string]string{
		"ctrl+shift+\\": "ctrl+|",
		"ctrl+\\":       "ctrl+\\",
		"alt+shift+/":   "alt+?",
		"f3 / f4":       "f3 / f4",
		"shift+pgup":    "shift+pgup",
	} {
		if got := keyLabel(key); got != want {
			t.Errorf("keyLabel(%q) = %q, want %q", key, got, want)
		}
	}

	m := sideBySideModel(t, 150)
	if help := ansi.Strip(m.renderHelp(200)); !strings.Contains(help, "ctrl+|") { // tall enough for every row
		t.Fatal("help should show the side by side key as ctrl+|")
	}
}

func TestSideBySidePaneFocusKeys(t *testing.T) {
	m := sideBySideModel(t, 150)
	press := func(code rune) {
		t.Helper()
		mm, _ := m.handleKey(tea.KeyPressMsg{Code: code, Mod: tea.ModAlt})
		m = mm.(Model)
	}

	press('l')
	if m.screen != screenEditor {
		t.Fatalf("one at a time, the focus key goes to nvim and the view stays, got %v", m.screen)
	}

	m = pressSideBySide(t, m)
	press('l')
	if m.screen != screenAgent {
		t.Fatalf("side by side, focus right from nvim should focus the agent, got %v", m.screen)
	}
	press('l')
	if m.screen != screenAgent {
		t.Fatalf("there is nothing right of the agent, so focus stays, got %v", m.screen)
	}
	press('h')
	if m.screen != screenEditor {
		t.Fatalf("focus left from the agent should focus nvim, got %v", m.screen)
	}
}

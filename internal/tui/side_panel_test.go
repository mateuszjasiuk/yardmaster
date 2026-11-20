package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/mateuszjasiuk/yardmaster/internal/config"
	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// sidePanelModel is a model with one open workspace on the terminal screen.
func sidePanelModel(t *testing.T, width int) (Model, *Controller, *session.Manager) {
	t.Helper()
	ctrl := &Controller{cfg: config.Config{
		Editor: "cat", Agent: "cat", Shell: "sh", Keys: config.Default().Keys,
	}}
	mgr := session.NewManager()
	t.Cleanup(mgr.CloseAll)

	m := New(ctrl, mgr)
	m.sidePanel, m.sideBySide = false, false
	mm, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
	m = mm.(Model)
	mm, _ = m.activate("repo", "wt", t.TempDir())
	m = mm.(Model)
	m.screen = screenTerminal
	return m, ctrl, mgr
}

func pressSidePanel(t *testing.T, m Model) Model {
	t.Helper()
	mm, _ := m.handleKey(tea.KeyPressMsg{Code: '\\', Mod: tea.ModCtrl})
	return mm.(Model)
}

func TestSidePanelNarrowsSessionsAndIsSaved(t *testing.T) {
	m, _, _ := sidePanelModel(t, 120)

	m = pressSidePanel(t, m)
	if w, _ := m.sessionSize(); w != 120-sidePanelWidth-1 {
		t.Fatalf("session width with the panel = %d, want %d", w, 120-sidePanelWidth-1)
	}
	if !m.state.SidePanel {
		t.Fatal("the choice should be kept in the state")
	}
	if cols, _ := m.current.ws.Terms[0].Size(); cols != 120-sidePanelWidth-1 {
		t.Fatalf("terminal width = %d, want it resized to %d", cols, 120-sidePanelWidth-1)
	}
	lines := strings.Split(m.View().Content, "\n")
	if row := ansi.Strip(lines[m.barHeight()+1]); !strings.HasPrefix(row, "  repo ") || ansi.StringWidth(row) > 120 {
		t.Fatalf("the second body row should start with the panel and fit the screen: %q", row)
	}

	m = pressSidePanel(t, m)
	if w, _ := m.sessionSize(); w != 120 {
		t.Fatalf("closing the panel should give the full width back, got %d", w)
	}
}

func TestSidePanelHidesInANarrowTerminal(t *testing.T) {
	m, _, _ := sidePanelModel(t, 80)

	m = pressSidePanel(t, m)
	if w, _ := m.sessionSize(); w != 80 {
		t.Fatalf("an 80-column terminal has no room for the panel, got session width %d", w)
	}
	if !m.sidePanel {
		t.Fatal("the panel should stay on, to show again when the terminal is wider")
	}
}

func TestSidePanelShiftsClicksIntoPanes(t *testing.T) {
	m, ctrl, mgr := sidePanelModel(t, 120)
	m = pressSidePanel(t, m)
	w, h := m.sessionSize()
	if _, err := mgr.SplitTermPane("repo/wt", t.TempDir(), ctrl.TermSpec(), session.SplitV, w, h); err != nil {
		t.Fatal(err)
	}
	ws := m.current.ws

	// A click on the panel stays out of the panes.
	m.handleMouseClick(tea.MouseClickMsg{X: 5, Y: m.barHeight() + 3, Button: tea.MouseLeft})
	if ws.ActiveTerm != 1 {
		t.Fatalf("a click on the panel should not move pane focus, active=%d", ws.ActiveTerm)
	}

	m.handleMouseClick(tea.MouseClickMsg{X: m.sidePanelW() + 5, Y: m.barHeight() + 3, Button: tea.MouseLeft})
	if ws.ActiveTerm != 0 {
		t.Fatalf("a click just right of the panel should focus the left pane, active=%d", ws.ActiveTerm)
	}
}

// panelRows is the panel's rows as plain text, without the padding.
func panelRows(m Model) []string {
	var rows []string
	for _, r := range m.sidePanelRows() {
		rows = append(rows, strings.TrimRight(ansi.Strip(r.text), " "))
	}
	return rows
}

// boxRows is a box as plain text: the title in the top border, then each row.
func boxRows(title string, rows ...string) string {
	w := sidePanelWidth - 1
	out := []string{" ╭─" + title + strings.Repeat("─", w-3-ansi.StringWidth(title)) + "╮"}
	for _, r := range rows {
		out = append(out, " │"+r+strings.Repeat(" ", w-2-ansi.StringWidth(r))+"│")
	}
	out = append(out, " ╰"+strings.Repeat("─", w-2)+"╯")
	return strings.Join(out, "\n")
}

// activeBoxRows is boxRows with the heavy border and the accent bar of the
// active session.
func activeBoxRows(title string, rows ...string) string {
	heavy := strings.NewReplacer("╭", "┏", "╮", "┓", "╰", "┗", "╯", "┛", "─", "━", "│", "┃").Replace(boxRows(title, rows...))
	lines := strings.Split(heavy, "\n")
	for i, l := range lines {
		lines[i] = "▌" + strings.TrimPrefix(l, " ")
	}

	return strings.Join(lines, "\n")
}

// panelLines joins rows of the panel, as plain text, into one string.
func panelLines(parts ...string) string { return strings.Join(parts, "\n") }

// firstPanelRows is the panel's first rows as plain text, as many as want has.
func firstPanelRows(m Model, want string) string {
	return panelLines(panelRows(m)[:strings.Count(want, "\n")+1]...)
}

func TestSidePanelListsSessionsUnderTheirRepo(t *testing.T) {
	m := stripModelKeys(t, 140, "api/zeta", "api/alpha", "web/main")
	m.sidePanel = true
	ws, _ := m.mgr.Workspace("api/zeta")
	if len(ws.Agents) == 0 {
		t.Skip("workspace started without an agent")
	}
	m.agentStatus = map[int]AgentStatus{ws.Agents[0].Pid(): {Status: "waiting"}}

	want := panelLines("", "  api", boxRows("claude", "alpha"), boxRows("! claude", "zeta"), "  web", activeBoxRows("claude", "main"))
	got := firstPanelRows(m, want)
	if got != want {
		t.Fatalf("panel rows:\n%s\nwant:\n%s", got, want)
	}
	rows := m.sidePanelRows()
	if red := " " + lipgloss.NewStyle().Foreground(cRed).Render("╭─"); !strings.HasPrefix(rows[5].text, red) {
		t.Fatalf("a waiting session's border should be red: %q", rows[5].text)
	}
	for _, r := range m.sidePanelRows() {
		if w := ansi.StringWidth(r.text); w != sidePanelWidth {
			t.Fatalf("every row should be %d wide, got %d: %q", sidePanelWidth, w, r.text)
		}
	}
}

func TestSidePanelFollowsTheStackToggle(t *testing.T) {
	m := stackStripModel(t)
	m.sidePanel = true

	want := panelLines("", "  repo", boxRows("⛁ #4", "4 worktrees"), activeBoxRows("claude", "beta"), boxRows("claude", "zed"))
	got := firstPanelRows(m, want)
	if got != want {
		t.Fatalf("a stack you're not in should be one box:\n%s\nwant:\n%s", got, want)
	}

	m = pressCtrl(t, m, 'y')
	want = panelLines("", "  repo", " ⛁ #4", boxRows("claude", "a"), boxRows("claude", "b"), boxRows("claude", "c"), boxRows("claude", "d"),
		"", activeBoxRows("claude", "beta"))
	got = firstPanelRows(m, want)
	if got != want {
		t.Fatalf("ctrl+y should show the stack in full:\n%s\nwant:\n%s", got, want)
	}
}

// longPanelModel has 30 worktrees with one session each; wt29, the last, is
// current. 22 panel rows fit the top row and 7 boxes.
func longPanelModel(t *testing.T) Model {
	t.Helper()
	var names []string
	for i := range 30 {
		names = append(names, fmt.Sprintf("wt%02d", i))
	}
	m := stripModel(t, 140, names...)
	// Open the panel with its key, through Update, as the app does.
	mm, _ := m.Update(tea.KeyPressMsg{Code: '\\', Mod: tea.ModCtrl})

	return mm.(Model)
}

// wheel turns the mouse wheel once over the side panel.
func wheel(t *testing.T, m Model, button tea.MouseButton) Model {
	t.Helper()
	mm, _ := m.Update(tea.MouseWheelMsg{X: 4, Y: m.barHeight() + 5, Button: button})

	return mm.(Model)
}

func TestSidePanelScrollsToTheActiveSession(t *testing.T) {
	m := longPanelModel(t)

	rows := panelRows(m)
	if len(rows) != m.height-m.barHeight() {
		t.Fatalf("the panel should fill the height: %d rows, want %d", len(rows), m.height-m.barHeight())
	}
	// The repo's header is above the window, so its name stays on the top row.
	if want := "  repo             ↑ 23 more"; rows[0] != want {
		t.Fatalf("top row = %q, want %q", rows[0], want)
	}
	if want := activeBoxRows("claude", "wt29"); panelLines(rows[19:22]...) != want {
		t.Fatalf("the active session's box should be the last one shown:\n%s", panelLines(rows...))
	}
}

func TestSidePanelWheelScrollsUntilTheActiveSessionChanges(t *testing.T) {
	m := longPanelModel(t)

	m = wheel(t, m, tea.MouseWheelUp)
	rows := panelRows(m)
	if rows[1] != " ╭─claude──────────────────╮" || rows[2] != " │wt22                     │" {
		t.Fatalf("one wheel turn up should show one box more above:\n%s", panelLines(rows...))
	}
	// The "↓" row takes a row, so wt28 and wt29 are below now.
	if rows[19] != "  ↓ 2 more" {
		t.Fatalf("the boxes below the window should be counted:\n%s", panelLines(rows...))
	}

	// A click on "↓ 2 more" scrolls a page down, here back to the end.
	mm, _ := m.handleMouseClick(tea.MouseClickMsg{X: 4, Y: m.barHeight() + 19, Button: tea.MouseLeft})
	if got := panelRows(mm.(Model))[0]; !strings.HasSuffix(got, "↑ 23 more") {
		t.Fatalf("a click on the ↓ row should scroll down, top row = %q", got)
	}

	// A status change redraws the panel; it stays where you scrolled.
	ws, _ := m.mgr.Workspace("repo/wt05")
	m.agentStatus = map[int]AgentStatus{ws.Agents[0].Pid(): {Status: "waiting"}}
	if got := panelRows(m); got[2] != rows[2] {
		t.Fatalf("a redraw should keep the scroll, got:\n%s", panelLines(got...))
	}
	if got := panelRows(m)[0]; !strings.HasSuffix(got, "↑ 22 more !1") {
		t.Fatalf("a hidden waiting session should be counted on the top row: %q", got)
	}

	// Switching sessions brings the new active one into view.
	mm, _ = m.Update(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl}) // previous worktree
	m = mm.(Model)
	if !strings.Contains(panelLines(panelRows(m)...), "▌┃wt28") {
		t.Fatalf("the panel should follow the active session:\n%s", panelLines(panelRows(m)...))
	}
}

func TestSidePanelClickOpensTheSession(t *testing.T) {
	m := stripModel(t, 140, "alpha", "beta", "gamma") // gamma is current
	m.sidePanel = true
	click := func(m Model, row int) Model {
		mm, _ := m.handleMouseClick(tea.MouseClickMsg{X: 4, Y: m.barHeight() + row, Button: tea.MouseLeft})
		return mm.(Model)
	}

	// Rows: blank, "repo", alpha, beta, gamma.
	if m = click(m, 1); m.current.worktree != "gamma" {
		t.Fatalf("a click on the repo's row should do nothing, current = %q", m.current.worktree)
	}
	if m = click(m, 2); m.current.worktree != "alpha" {
		t.Fatalf("a click on alpha's row should open it, current = %q", m.current.worktree)
	}

	m.helpOpen = true
	if m = click(m, 3); m.current.worktree != "alpha" {
		t.Fatalf("the panel isn't drawn under help, so a click should do nothing, current = %q", m.current.worktree)
	}
}

func TestSidePanelBoxOpensItsSession(t *testing.T) {
	m := stripModel(t, 140, "alpha", "beta") // beta is current, on the editor
	m.sidePanel = true

	// Rows: blank, "repo", then the top border and the body of alpha's box.
	mm, _ := m.handleMouseClick(tea.MouseClickMsg{X: 4, Y: m.barHeight() + 3, Button: tea.MouseLeft})
	m = mm.(Model)
	if m.current.worktree != "alpha" || m.screen != screenAgent {
		t.Fatalf("a click inside a session's box should open that session, got %q on screen %v", m.current.worktree, m.screen)
	}
}

func TestSidePanelGivesEachSessionABoxAndMarksTheActiveOne(t *testing.T) {
	m := stripModel(t, 140, "alpha") // alpha is current
	m.sidePanel = true
	if _, err := m.mgr.SpawnAgent("repo/alpha", t.TempDir(), session.Spec{Kind: session.Agent, Title: "second", Argv: []string{"cat"}}, 10, 10); err != nil {
		t.Fatal(err)
	}
	m.current.ws.ActiveAgent = 0

	want := panelLines("", "  repo", activeBoxRows("claude", "alpha"), boxRows("second", "alpha"))
	if got := firstPanelRows(m, want); got != want {
		t.Fatalf("panel rows:\n%s\nwant:\n%s", got, want)
	}
	rows := m.sidePanelRows()
	accent := accentStyle.Render("▌") + accentStyle.Render("┏━")
	if !strings.HasPrefix(rows[2].text, accent) || strings.HasPrefix(rows[5].text, accent) {
		t.Fatalf("only the active session's box should have the accent bar and heavy border:\n%q\n%q", rows[2].text, rows[5].text)
	}
}

func TestSidePanelShowsTheStageInTheBottomBorder(t *testing.T) {
	m := stripModel(t, 140, "alpha", "beta") // beta is current
	m.sidePanel = true
	ws, _ := m.mgr.Workspace("repo/alpha")
	ws.Agents[0].Stage = "step 2"

	// Rows: the top row, "repo", then alpha's box.
	got := panelRows(m)[4]
	want := " ╰" + strings.Repeat("─", sidePanelWidth-1-5-len("step 2")) + " step 2 ─╯"
	if got != want {
		t.Fatalf("bottom border = %q, want %q", got, want)
	}
	if w := ansi.StringWidth(m.sidePanelRows()[4].text); w != sidePanelWidth {
		t.Fatalf("the row should stay %d wide, got %d", sidePanelWidth, w)
	}
}

func TestSidePanelHidesTheAgentStrip(t *testing.T) {
	m := stripModel(t, 140, "alpha", "beta")
	if m.barHeight() != 4 || !strings.Contains(barStripRow(m), "alpha · beta") {
		t.Fatalf("with the panel closed the strip has its own row: %q", m.renderBar())
	}

	m.sidePanel = true
	if m.barHeight() != 2 || strings.Contains(m.renderBar(), "alpha") {
		t.Fatalf("the strip should be gone while the panel lists the sessions: %q", m.renderBar())
	}
	if row := barFirstRow(m); !strings.Contains(row, "repo / beta") {
		t.Fatalf("the bar should keep its other parts: %q", row)
	}
	if _, _, ok := m.stripClick(10, stripRow); ok {
		t.Fatal("a hidden strip should take no clicks")
	}

	// The bar keeps its height under an overlay and on the deck.
	m.helpOpen = true
	if m.barHeight() != 2 || strings.Contains(m.renderBar(), "alpha") {
		t.Fatalf("under an overlay the bar should not change: %q", m.renderBar())
	}
}

func TestStripMoreOpensTheSidePanel(t *testing.T) {
	var names []string
	for i := range 12 {
		names = append(names, fmt.Sprintf("worktree-%02d", i))
	}
	m := stripModel(t, 120, names...)

	_, items := m.agentStripLayout()
	more := items[len(items)-1]
	if !more.more {
		t.Fatalf("the strip's last item should be \"+N\": %+v", more)
	}

	mm, _, ok := m.stripClick(more.start, stripRow)
	m = mm.(Model)
	if !ok || !m.sidePanel {
		t.Fatal(`a click on "+N" should open the side panel`)
	}

	// "+N" can't show the panel on the deck, so it takes no click there.
	m.sidePanel, m.sideBySide = false, false
	m.screen = screenPicker
	_, items = m.agentStripLayout()
	mm, _, ok = m.stripClick(items[len(items)-1].start, 1)
	if m = mm.(Model); ok || m.sidePanel {
		t.Fatal(`on the deck "+N" should do nothing`)
	}
}

func TestSidePanelInATinyTerminalDoesNotCrash(t *testing.T) {
	m := stripModel(t, 140, "alpha", "beta", "gamma")
	m.sidePanel = true
	for _, h := range []int{3, 2, 1} {
		m.height = h
		m.handleMouseClick(tea.MouseClickMsg{X: 4, Y: h - 1, Button: tea.MouseLeft})
	}
}

func TestSidePanelNamesTheAccountInsteadOfABackground(t *testing.T) {
	m := stackStripModel(t)
	m.sidePanel = true
	m.height = 40 // room for every box

	mm, _ := m.activate("repo", "c", m.active[2].view.WT.Path) // inside the stack
	m = mm.(Model)
	var zedPath string
	for _, it := range m.active {
		if it.view.WT.Name == "zed" {
			zedPath = it.view.WT.Path
		}
	}
	m.ctrl.cfg.ClaudeAccounts = map[string]config.ClaudeAccount{
		"private": {ConfigDir: t.TempDir(), Folders: []string{zedPath}},
	}

	want := panelLines("", "  repo", " ⛁ #4", boxRows("claude", "a"), boxRows("claude", "b"), activeBoxRows("claude", "c"), boxRows("claude", "d"),
		"", boxRows("claude", "beta"), "  repo · private", boxRows("claude", "zed"))
	got := firstPanelRows(m, want)
	if got != want {
		t.Fatalf("panel rows:\n%s\nwant:\n%s", got, want)
	}
	for _, r := range m.sidePanelRows() {
		if strings.Contains(r.text, "48;2;") {
			t.Fatalf("panel rows should have no background: %q", r.text)
		}
	}
}

package tui

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/mateuszjasiuk/yardmaster/internal/config"
	"github.com/mateuszjasiuk/yardmaster/internal/repo"
	"github.com/mateuszjasiuk/yardmaster/internal/session"
	"github.com/mateuszjasiuk/yardmaster/internal/state"
)

// restoreModel is a model whose last run left worktrees r/w1…r/wN open, w1
// opened most recently, each with one saved agent named after it. agent is the
// command every restored agent and editor runs.
func restoreModel(t *testing.T, agent string, n int) Model {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	ctrl := &Controller{cfg: config.Config{Editor: agent, Agent: agent, Shell: "sh", Keys: config.Default().Keys}}
	mgr := session.NewManager()
	t.Cleanup(mgr.CloseAll)

	m := New(ctrl, mgr)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = mm.(Model)
	g := Group{Repo: repo.Repo{Name: "r", Path: t.TempDir()}}
	m.state = &state.State{LastOpened: map[string]int64{}, Workspaces: map[string]*state.WorkspaceState{}}
	var open []string
	for i := 1; i <= n; i++ {
		name := "w" + strconv.Itoa(i)
		g.Worktrees = append(g.Worktrees, WorktreeView{WT: repo.Worktree{Repo: "r", Name: name, Path: t.TempDir(), Branch: name}})
		key := "r/" + name
		m.state.LastOpened[key] = int64(1000 - i)
		m.state.SetAgents(key, []state.AgentState{{SessionID: "s-" + name, Label: "agent-" + name}}, 0)
		open = append([]string{key}, open...) // oldest first, so the order must come from the times
	}
	m.state.SetOpen(open)
	m.groups = []Group{g}
	m.groupsLoaded = true
	m.recomputeActive()

	return m
}

// hangingAgent never draws, so its workspace stays starting.
func hangingAgent(t *testing.T) string {
	t.Helper()
	return scriptAgent(t, "exec sleep 30")
}

func TestRestoreStartsTheLastOpenedFirstAFewAtATime(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 5)
	before := m.state.Opened("r/w5")

	m, _ = m.restoreSessions()

	if got := m.mgr.Keys(); len(got) != restoreAtOnce || !slices.Contains(got, "r/w1") || !slices.Contains(got, "r/w3") {
		t.Errorf("open = %v, want the %d most recently opened", got, restoreAtOnce)
	}
	if !slices.Equal(m.restoreWaiting(), []string{"r/w4", "r/w5"}) {
		t.Errorf("waiting = %v, want the older ones, newest first", m.restoreWaiting())
	}
	if m.state.Opened("r/w1") != 999 || m.state.Opened("r/w5") != before {
		t.Error("the restore changed the last-opened times; the next start would lose the order")
	}
	if !m.boardOpen {
		t.Error("the restore should show the board")
	}
}

func TestRestoreStartsTheNextOnceOneIsUp(t *testing.T) {
	m := restoreModel(t, drawingAgent(t), 4)
	m, _ = m.restoreSessions()
	waitRestoredUp(t, m)

	m, _ = m.startQueuedRestores()

	if waiting := m.restoreWaiting(); len(waiting) != 0 || !m.mgr.Has("r/w4") {
		t.Errorf("waiting = %v; the last worktree should start once the others are up", waiting)
	}
}

func TestRestoreGivesAHangingWorktreesPlaceUp(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 4)
	m, _ = m.restoreSessions()
	for key := range m.restoreStartedAt {
		m.restoreStartedAt[key] = time.Now().Add(-startingTooLong)
	}

	m, _ = m.startQueuedRestores()

	if !m.mgr.Has("r/w4") {
		t.Error("worktrees that hang past startingTooLong should not hold the queue up")
	}
}

func TestBoardListsTheQueuedWorktreesAsStarting(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 4)
	m, _ = m.restoreSessions()

	rows, _ := m.buildBoard()
	i := slices.IndexFunc(rows, func(r boardRow) bool { return r.key == "r/w4" && r.kind == rowQueued })
	if i < 0 {
		t.Fatalf("the queued worktree has no row: %+v", rows)
	}
	if rows[i].label != "agent-w4" || !rows[i].spinning() {
		t.Errorf("row = %+v, want its saved name and the spinner", rows[i])
	}
	if got := ansi.Strip(m.renderBoard(36)); !strings.Contains(got, "restoring 0 of 4…") {
		t.Errorf("the board lacks the restore line:\n%s", got)
	}
}

func TestEnterOnAQueuedRowOpensItAtOnce(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 4)
	m, _ = m.restoreSessions()
	m.boardInput.SetValue("w4")

	mm, _ := m.handleBoard(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mm.(Model)

	if m.current == nil || m.current.key != "r/w4" || m.screen != screenAgent {
		t.Fatalf("enter should open the queued worktree's agent, current = %+v", m.current)
	}
	m, _ = m.startQueuedRestores()
	if m.restoreWaitsFor("r/w4") {
		t.Error("an opened worktree should leave the queue")
	}
}

func TestRestoreClosesTheBoardNobodyTouched(t *testing.T) {
	m := restoreModel(t, drawingAgent(t), 2)
	m, _ = m.restoreSessions()
	waitRestoredUp(t, m)

	m, _ = m.finishRestore()

	if m.boardOpen || m.restoring != nil {
		t.Error("the board should close once the restore is done")
	}
}

func TestRestoreKeepsTheBoardTheUserUsed(t *testing.T) {
	m := restoreModel(t, drawingAgent(t), 2)
	m, _ = m.restoreSessions()
	mm, _ := m.handleBoard(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(Model)
	waitRestoredUp(t, m)

	m, _ = m.finishRestore()

	if !m.boardOpen {
		t.Fatal("the restore closed a board the user was using")
	}
	if got := ansi.Strip(m.renderBoard(36)); !strings.Contains(got, "restored 2 workspaces") {
		t.Errorf("the board should say how the restore ended:\n%s", got)
	}
}

func TestQuittingMidRestoreKeepsTheQueuedWorktreesOpen(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 5)
	m, _ = m.restoreSessions()

	m.recordOpen()

	if len(m.state.Open) != 5 {
		t.Errorf("open = %v; the queued worktrees would not come back next time", m.state.Open)
	}
}

// waitRestoredUp waits until every workspace the restore has opened is up.
func waitRestoredUp(t *testing.T, m Model) {
	t.Helper()
	for _, key := range m.mgr.Keys() {
		ws, _ := m.mgr.Workspace(key)
		if ws.Editor != nil {
			waitStarted(t, ws.Editor)
		}
		for _, a := range ws.Agents {
			waitStarted(t, a)
		}
	}
}

func TestStoppingAWaitingWorktreeKeepsTheRestoreFromStartingIt(t *testing.T) {
	m := restoreModel(t, drawingAgent(t), 4)
	m, _ = m.restoreSessions()

	m.stopWorkspace("r/w4")
	waitRestoredUp(t, m)
	m, _ = m.startQueuedRestores()

	if m.mgr.Has("r/w4") {
		t.Error("the restore started a worktree the user stopped while it waited")
	}
}

func TestRestoreWhereEveryStartFailedClosesItsBoard(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 1)
	m, _ = m.showRestoreBoard()
	m.restoring, m.restoreStartedAt = nil, map[string]time.Time{"r/w1": time.Now()}

	m, _ = m.finishRestore()

	if m.boardOpen || m.restoreStartedAt != nil {
		t.Error("a restore with nothing left should end and close its board")
	}
}

func TestPastingIntoTheRestoreBoardKeepsItOpen(t *testing.T) {
	m := restoreModel(t, drawingAgent(t), 2)
	m, _ = m.restoreSessions()

	mm, _ := m.typeBoardQuery(tea.PasteMsg{Content: "w2"})
	m = mm.(Model)

	if m.restoreBoardUntouched {
		t.Error("a paste should count as using the board, or the restore closes it under the user")
	}
}

func TestAWaitingWorktreeWithNoSavedAgentStillHasARow(t *testing.T) {
	m := restoreModel(t, hangingAgent(t), 4)
	m.state.SetAgents("r/w4", nil, 0)
	m, _ = m.restoreSessions()

	rows, _ := m.buildBoard()

	if !slices.ContainsFunc(rows, func(r boardRow) bool { return r.key == "r/w4" && r.kind == rowQueued }) {
		t.Error("a worktree that will start one default agent should have its row meanwhile")
	}
}

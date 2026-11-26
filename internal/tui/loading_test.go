package tui

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/mateuszjasiuk/yardmaster/internal/agent"
	"github.com/mateuszjasiuk/yardmaster/internal/config"
	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// silentSession starts a program of kind that never draws, so it stays
// starting.
func silentSession(t *testing.T, kind session.Kind) *session.Session {
	t.Helper()
	s, err := session.StartSpec(session.Spec{Kind: kind, Argv: []string{"sleep", "30"}, Provider: agent.Claude}, t.TempDir(), 60, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	return s
}

// eventually polls cond until it holds, or fails with what.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// loadingModel is the sample deck with the default keys, for the restart hint.
func loadingModel() Model {
	m := sampleModel()
	m.keys = config.Default().Keys

	return m
}

func TestStartingSignWaitsBeforeShowing(t *testing.T) {
	m := loadingModel()
	s := silentSession(t, session.Agent)

	if showsSign(s) {
		t.Error("a pane that just started shows its sign; want nothing for 150 ms")
	}
	eventually(t, "the sign shows after the delay", func() bool { return showsSign(s) })
	if got := ansi.Strip(m.startingSignAfter(s, time.Second)); !strings.HasSuffix(got, " starting claude…") {
		t.Errorf("after a second the sign is %q", got)
	}
}

func TestStartingSignAddsSecondsThenSaysStillStarting(t *testing.T) {
	m := loadingModel()
	s := silentSession(t, session.Agent)

	if got := ansi.Strip(m.startingSignAfter(s, 7*time.Second)); !strings.HasSuffix(got, "starting claude… 7s") {
		t.Errorf("after 7 s the sign is %q", got)
	}
	got := ansi.Strip(m.startingSignAfter(s, 34*time.Second))
	if !strings.Contains(got, "still starting claude… 34s") || !strings.Contains(got, `then "restart agent"`) {
		t.Errorf("after 34 s the sign is %q, want still starting and the restart hint", got)
	}
}

func TestStartingSignHasNoRestartHintForNvim(t *testing.T) {
	m := loadingModel()
	s := silentSession(t, session.Editor)

	got := ansi.Strip(m.startingSignAfter(s, 34*time.Second))
	if !strings.Contains(got, "still starting nvim…") || strings.Contains(got, "restart") {
		t.Errorf("a slow nvim's sign is %q", got)
	}
}

func TestSilentTerminalShowsItsOwnScreen(t *testing.T) {
	s := silentSession(t, session.Terminal)
	time.Sleep(2 * startingSignDelay)

	if showsSign(s) {
		t.Error("a terminal that has not drawn shows a sign; a shell may never draw")
	}
}

func TestStartingSignFitsANarrowPane(t *testing.T) {
	m := loadingModel()
	s, err := session.StartSpec(session.Spec{Kind: session.Agent, Argv: []string{"sleep", "30"}, Provider: agent.Claude}, t.TempDir(), 12, 4, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	eventually(t, "the sign shows", func() bool { return showsSign(s) })

	for _, line := range strings.Split(m.paneScreen(s), "\n") {
		if w := ansi.StringWidth(line); w > 12 {
			t.Errorf("a sign line is %d cells wide in a 12-cell pane: %q", w, ansi.Strip(line))
		}
	}
}

func TestStartingSignCountsHeldKeys(t *testing.T) {
	m := loadingModel()
	s := silentSession(t, session.Agent)
	s.SendKey(uv.KeyPressEvent{Code: 'h', Text: "h"})
	s.SendKey(uv.KeyPressEvent{Code: 'i', Text: "i"})

	if got := ansi.Strip(m.startingSignAfter(s, time.Second)); !strings.HasSuffix(got, "(2 keys waiting)") {
		t.Errorf("the sign is %q, want the held keys counted", got)
	}
}

func TestPaneOfAnAgentThatExitedBeforeStartingSaysSo(t *testing.T) {
	m := loadingModel()
	s, err := session.StartSpec(session.Spec{Kind: session.Agent, Argv: []string{"/usr/bin/true"}, Provider: agent.Claude}, t.TempDir(), 60, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	waitStarted(t, s)

	got := ansi.Strip(m.paneScreen(s))
	if !strings.Contains(got, "claude exited before it started") || !strings.Contains(got, `then "restart agent"`) {
		t.Errorf("the pane shows:\n%s", got)
	}
}

func TestStartingAgentShowsPendingOnTheStrip(t *testing.T) {
	sign, _ := attnSign(attnNone, false, true, dimStyle)
	if sign != "…" {
		t.Errorf("a starting agent's sign is %q, want …", sign)
	}
	sign, _ = attnSign(attnWaiting, false, true, dimStyle)
	if sign != "!" {
		t.Errorf("waiting outranks starting, got %q", sign)
	}
}

func TestAgentThatExitedBeforeStartingNeedsTheUser(t *testing.T) {
	m := loadingModel()
	specs := []session.Spec{{Kind: session.Agent, Argv: []string{"/usr/bin/true"}, Provider: agent.Claude}}
	ws, err := m.mgr.Activate("r/a", t.TempDir(), specs, session.SameSize(60, 10))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.mgr.CloseAll)
	waitStarted(t, ws.Agents[0])

	g, _ := m.agentGroup("r/a", "r", "a", "")
	if g.attn != attnWaiting || g.agents[0].status != "exited before it started" {
		t.Errorf("attn = %v, status = %q; want waiting and the exit named", g.attn, g.agents[0].status)
	}
}

// openAgent opens workspace key with one agent running argv.
func openAgent(t *testing.T, m Model, key string, argv ...string) *session.Workspace {
	t.Helper()
	specs := []session.Spec{{Kind: session.Agent, Argv: argv, Provider: agent.Claude}}
	ws, err := m.mgr.Activate(key, t.TempDir(), specs, session.SameSize(60, 10))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.mgr.CloseAll)

	return ws
}

func TestRestoreSignCountsTheWorkspacesThatCameUp(t *testing.T) {
	m := loadingModel()
	openAgent(t, m, "r/slow", "sleep", "30")
	ready := openAgent(t, m, "r/ready", "sh", "-c", "echo ready; exec cat")
	waitStarted(t, ready.Agents[0])
	m = restoringSince(m, time.Now(), "r/slow", "r/ready", "r/closed")

	if got := ansi.Strip(barFirstRow(m)); !strings.Contains(got, "restoring 2 of 3…") {
		t.Errorf("bar = %q, want the closed and the ready workspace counted", got)
	}
	m, cmd := m.finishRestore()
	if cmd != nil || len(m.restoring) != 3 {
		t.Error("the restore finished while a workspace was still starting")
	}
}

func TestRestoreFlashesOnceEveryWorkspaceCameUp(t *testing.T) {
	m := loadingModel()
	ws := openAgent(t, m, "r/ready", "sh", "-c", "echo ready; exec cat")
	waitStarted(t, ws.Agents[0])
	m = restoringSince(m, time.Now(), "r/ready")

	m, cmd := m.finishRestore()
	if cmd == nil || m.restoring != nil {
		t.Fatal("the restore should end with its flash once every workspace is up")
	}
	if got := barFirstRow(m); strings.Contains(got, "restoring") {
		t.Errorf("the bar still shows the restore: %q", got)
	}
}

func TestNotificationZoneStaysClickableBesideTheRestoreSign(t *testing.T) {
	m := loadingModel()
	openAgent(t, m, "r/slow", "sleep", "30")
	m.restoring = []string{"r/slow"}
	m.attention = map[int]attnEntry{1: {level: attnDone, key: "r/slow"}}
	m.width = 160

	bar := ansi.Strip(strings.SplitN(m.renderBar(), "\n", 2)[0])
	notifAt, restoreAt := strings.Index(bar, "✓"), strings.Index(bar, "restoring")
	if notifAt < 0 || restoreAt < 0 {
		t.Fatalf("the bar lacks the restore sign or the notifications: %q", bar)
	}
	notifCol, restoreCol := len([]rune(bar[:notifAt])), len([]rune(bar[:restoreAt]))

	if !m.notifZoneAt(notifCol, 0) {
		t.Errorf("a click on the ✓ in %q misses the notification zone", bar)
	}
	if m.notifZoneAt(restoreCol, 0) {
		t.Error("a click on the restore sign opens the notifications")
	}
}

func TestRestoreStopsWaitingForAPaneThatHangs(t *testing.T) {
	m := loadingModel()
	openAgent(t, m, "r/slow", "sleep", "30")
	m = restoringSince(m, time.Now().Add(-startingTooLong), "r/slow")

	m, cmd := m.finishRestore()
	if cmd == nil || m.restoring != nil {
		t.Error("the restore should end after startingTooLong, even with a pane still starting")
	}
}

func TestRestoreNamesTheWorkspacesThatFailed(t *testing.T) {
	m := loadingModel()
	ready := openAgent(t, m, "r/ready", "sh", "-c", "echo ready; exec cat")
	failed := openAgent(t, m, "r/failed", "/usr/bin/true")
	waitStarted(t, ready.Agents[0])
	waitStarted(t, failed.Agents[0])
	m = restoringSince(m, time.Now(), "r/ready", "r/failed")

	m, _ = m.finishRestore()

	if m.status != "restored 1 of 2 workspaces" {
		t.Errorf("flash = %q, want the failed workspace left out of the count", m.status)
	}
}

func TestRestoreKeepsAnErrorOnTheBar(t *testing.T) {
	m := loadingModel()
	m = restoringSince(m, time.Now(), "r/closed")
	m.setError("restore error (r/x): gone")

	m, cmd := m.finishRestore()

	if cmd != nil || m.status != "restore error (r/x): gone" || m.restoring != nil {
		t.Errorf("status = %q; the restore should end without covering the error", m.status)
	}
}

// restoringSince marks keys as restored workspaces that the restore started at.
func restoringSince(m Model, at time.Time, keys ...string) Model {
	m.restoring = keys
	m.restoreStartedAt = map[string]time.Time{}
	for _, key := range keys {
		m.restoreStartedAt[key] = at
	}

	return m
}

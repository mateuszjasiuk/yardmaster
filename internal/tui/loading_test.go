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

func loadingModel() Model {
	m := sampleModel()
	m.keys = config.Default().Keys

	return m
}

func TestStartingSignWaitsBeforeShowing(t *testing.T) {
	m := loadingModel()
	s := silentSession(t, session.Agent)

	if got := m.startingSignAfter(s, 100*time.Millisecond); got != "" {
		t.Errorf("a pane that just started shows %q, want nothing yet", got)
	}
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

func TestStartingSignHasNoRestartHintOutsideAgents(t *testing.T) {
	m := loadingModel()
	s := silentSession(t, session.Terminal)

	got := ansi.Strip(m.startingSignAfter(s, 34*time.Second))
	if !strings.Contains(got, "still starting terminal…") || strings.Contains(got, "restart") {
		t.Errorf("a slow terminal's sign is %q", got)
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
	m.restoring = []string{"r/slow", "r/ready", "r/closed"}

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
	m.restoring = []string{"r/ready"}

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

	z := m.barRightZones()
	bar := ansi.Strip(strings.SplitN(m.renderBar(), "\n", 2)[0])
	notifAt := strings.Index(bar, "✓")
	if z.notif == "" || notifAt < 0 {
		t.Fatalf("no notification zone in %q", bar)
	}
	if !m.notifZoneAt(len([]rune(bar[:notifAt])), 0) {
		t.Errorf("a click on the ✓ in %q misses the notification zone", bar)
	}
	if m.notifZoneAt(len([]rune(bar[:strings.Index(bar, "restoring")])), 0) {
		t.Error("a click on the restore sign opens the notifications")
	}
}

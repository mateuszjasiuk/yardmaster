package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

func TestSessionPanesBesideTheSidePanel(t *testing.T) {
	m, ctrl, mgr := sidePanelModel(t, 120)
	m = pressSidePanel(t, m)
	w, h := m.sessionSize()
	if _, err := mgr.SplitTermPane("repo/wt", t.TempDir(), ctrl.TermSpec(), session.SplitV, w, h); err != nil {
		t.Fatal(err)
	}
	mgr.ResizeTermPanes("repo/wt", w, h)

	panes := m.sessionPanes()
	if len(panes) != 2 {
		t.Fatalf("a split should give two panes, got %d", len(panes))
	}
	left, right := panes[0], panes[1]
	if left.x != m.sidePanelW() || left.y != m.barHeight() {
		t.Fatalf("left pane at %d,%d, want it right of the panel and under the bar at %d,%d",
			left.x, left.y, m.sidePanelW(), m.barHeight())
	}
	if right.x != left.x+left.w+1 || left.w+1+right.w != w {
		t.Fatalf("panes %d+1+%d at x=%d,%d should fill the %d columns with one divider",
			left.w, right.w, left.x, right.x, w)
	}
	if p, ok := m.paneAt(right.x, right.y); !ok || p.s != m.current.ws.Terms[1] {
		t.Fatal("the right pane's first cell should be in the right pane")
	}
	if _, ok := m.paneAt(right.x-1, right.y); ok {
		t.Fatal("the divider should be in no pane")
	}

	// The new pane has focus, so the cursor is in it.
	if c := m.View().Cursor; c == nil || !right.contains(c.X, c.Y) {
		t.Fatalf("cursor %+v should be inside the focused right pane %+v", c, right)
	}
}

// mouseLogShell writes a shell script that turns on mouse reporting, prints
// "ready", and writes every byte it reads to the returned log file.
func mouseLogShell(t *testing.T) (shell, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "mouse.log")
	shell = filepath.Join(dir, "mouse.sh")
	script := "#!/bin/sh\nstty raw -echo\nprintf '\\033[?1002h\\033[?1006hready'\nexec dd bs=1 of=" + log + " 2>/dev/null\n"
	if err := os.WriteFile(shell, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return shell, log
}

// waitForLog waits until the log file holds want, and fails with what it holds.
func waitForLog(t *testing.T, log, want string) {
	t.Helper()
	var got []byte
	for range 100 {
		got, _ = os.ReadFile(log)
		if strings.Contains(string(got), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the session got %q, want it to contain %q", got, want)
}

func TestForwardMouseKeepsADragThatLeavesThePane(t *testing.T) {
	m, ctrl, _ := sidePanelModel(t, 120)
	shell, log := mouseLogShell(t)
	ctrl.cfg.Shell = shell
	mm, _ := m.activate("repo", "mouse", t.TempDir())
	m = pressSidePanel(t, mm.(Model))
	m.screen = screenTerminal
	waitForSession(t, m.current.ws.Terms[0], "ready")

	pw, by := m.sidePanelW(), m.barHeight()
	// A click on the panel and a release with no press in the session stay out
	// of it; a press in the session arrives at its own cell (SGR is 1-based).
	m.forwardMouse(tea.MouseClickMsg{X: 3, Y: by + 2, Button: tea.MouseLeft})
	m.forwardMouse(tea.MouseReleaseMsg{X: pw + 5, Y: by + 2, Button: tea.MouseLeft})
	mm, _ = m.handleMouseClick(tea.MouseClickMsg{X: pw + 5, Y: by + 2, Button: tea.MouseLeft})
	m = mm.(Model)
	waitForLog(t, log, "\x1b[<0;6;3M")

	// The release over the panel lands at the session's left edge.
	m.forwardMouse(tea.MouseReleaseMsg{X: 3, Y: by + 2, Button: tea.MouseLeft})
	waitForLog(t, log, "\x1b[<0;1;3m")

	got, _ := os.ReadFile(log)
	if strings.Count(string(got), "M") != 1 || strings.Count(string(got), "m") != 1 {
		t.Fatalf("only the press in the session and its release should reach it, got %q", got)
	}
}

func TestForwardMouseKeepsADragInThePaneItStartedIn(t *testing.T) {
	m, ctrl, mgr := sidePanelModel(t, 150)
	shell, log := mouseLogShell(t)
	ctrl.cfg.Editor = shell
	mm, _ := m.activate("repo", "drag", t.TempDir())
	m = mm.(Model)
	m.screen = screenEditor
	if len(m.current.ws.Agents) == 0 {
		if _, err := mgr.SpawnAgent("repo/drag", t.TempDir(), session.Spec{Kind: session.Agent, Argv: []string{"cat"}}, 10, 10); err != nil {
			t.Fatal(err)
		}
	}
	m = pressSideBySide(t, m)
	waitForSession(t, m.current.ws.Editor, "ready")

	left, right := m.sessionPanes()[0], m.sessionPanes()[1]
	y := left.y + 1
	mm, _ = m.handleMouseClick(tea.MouseClickMsg{X: left.x + 4, Y: y, Button: tea.MouseLeft})
	m = mm.(Model)
	waitForLog(t, log, "\x1b[<0;5;2M")

	// Dragging into the agent's pane still moves and releases in nvim, at its
	// last column.
	edge := strconv.Itoa(left.w)
	m.forwardMouse(tea.MouseMotionMsg{X: right.x + 10, Y: y, Button: tea.MouseLeft})
	waitForLog(t, log, "\x1b[<32;"+edge+";2M")
	mm, _ = m.Update(tea.MouseReleaseMsg{X: right.x + 10, Y: y, Button: tea.MouseLeft})
	m = mm.(Model)
	waitForLog(t, log, "\x1b[<0;"+edge+";2m")
	if m.mouseDown != nil {
		t.Fatal("the release should end the drag")
	}
}

func TestForwardMouseKeepsADragThatEndsOnADivider(t *testing.T) {
	m, ctrl, mgr := sidePanelModel(t, 120)
	shell, log := mouseLogShell(t)
	ctrl.cfg.Shell = shell
	mm, _ := m.activate("repo", "mouse", t.TempDir())
	m = mm.(Model)
	m.screen = screenTerminal
	waitForSession(t, m.current.ws.Terms[0], "ready")
	ctrl.cfg.Shell = "sh"
	w, h := m.sessionSize()
	if _, err := mgr.SplitTermPane("repo/mouse", t.TempDir(), ctrl.TermSpec(), session.SplitV, w, h); err != nil {
		t.Fatal(err)
	}
	mgr.ResizeTermPanes("repo/mouse", w, h)

	left := m.sessionPanes()[0]
	mm, _ = m.handleMouseClick(tea.MouseClickMsg{X: 4, Y: left.y + 1, Button: tea.MouseLeft})
	m = mm.(Model)
	waitForLog(t, log, "\x1b[<0;5;2M")

	// The divider is the column after the left pane; the release lands on the
	// pane's last column.
	m.forwardMouse(tea.MouseReleaseMsg{X: left.x + left.w, Y: left.y + 1, Button: tea.MouseLeft})
	waitForLog(t, log, "\x1b[<0;"+strconv.Itoa(left.w)+";2m")
}

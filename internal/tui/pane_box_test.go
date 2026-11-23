package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPaneBoxFramesNvimAndKeepsTheCursorInside(t *testing.T) {
	m, _, _ := sidePanelModel(t, 120)
	m.screen = screenEditor
	x, y := m.sidePanelW(), m.barHeight()
	w, h := m.sessionSize()

	panes := m.sessionPanes()
	if len(panes) != 1 {
		t.Fatalf("nvim alone should be one pane, got %d", len(panes))
	}
	p := panes[0]
	if p.x != x+1 || p.y != y+1 || p.w != w-2 || p.h != h-2 {
		t.Fatalf("nvim's pane is %+v, want the inside of its box at %d,%d %dx%d", p, x+1, y+1, w-2, h-2)
	}
	if got := sizeOf(p.s); got.W != p.w || got.H != p.h {
		t.Fatalf("nvim is %+v, want its pane's size %dx%d", got, p.w, p.h)
	}

	v := m.View()
	lines := strings.Split(v.Content, "\n")
	top := []rune(ansi.Strip(lines[y]))
	bottom := []rune(ansi.Strip(lines[y+h-1]))
	if len(top) < x+w || top[x] != '╭' || top[x+w-1] != '╮' || bottom[x] != '╰' || bottom[x+w-1] != '╯' {
		t.Fatalf("nvim should sit in a rounded box:\n%q\n%q", string(top), string(bottom))
	}
	if got := string(top[x : x+6]); got != "╭─nvim" {
		t.Fatalf("nvim's top border should name it, got %q", got)
	}
	cx, cy, visible := p.s.Cursor()
	if visible && (v.Cursor == nil || v.Cursor.X != p.x+cx || v.Cursor.Y != p.y+cy) {
		t.Fatalf("the cursor should be inside the box at %d,%d, got %+v", p.x+cx, p.y+cy, v.Cursor)
	}
}

func TestPaneBoxLeavesTheTerminalUnboxed(t *testing.T) {
	m, _, _ := sidePanelModel(t, 120)
	m.screen = screenTerminal
	w, h := m.sessionSize()

	p := m.sessionPanes()[0]
	if p.x != m.sidePanelW() || p.y != m.barHeight() || p.w != w || p.h != h {
		t.Fatalf("the terminal should fill the room, got %+v", p)
	}
}

func TestPaneBoxNamesTheAgentSideBySide(t *testing.T) {
	m := pressSideBySide(t, sideBySideModel(t, 150))
	agent := m.current.ws.ActiveAgentSession()
	right := m.sessionPanes()[1]

	top := []rune(ansi.Strip(strings.Split(m.View().Content, "\n")[m.barHeight()]))
	want := "╭─" + paneTitle(agent) + "─"
	if got := string(top[right.x-1 : right.x-1+len([]rune(want))]); got != want {
		t.Fatalf("the agent's top border should name it, got %q, want %q", got, want)
	}
}

func TestRenderPaneBoxCutsALongTitle(t *testing.T) {
	top := ansi.Strip(strings.Split(renderPaneBox("", "a-very-long-name", "", 8, 1, barSep), "\n")[0])
	if top != "╭─a-ver…─╮" {
		t.Fatalf("a long title should be cut to fit, got %q", top)
	}
}

func TestPaneBoxTakesThePlaceOfTheLineUnderTheBar(t *testing.T) {
	m, _, _ := sidePanelModel(t, 120)
	m.screen = screenEditor

	lines := strings.Split(m.View().Content, "\n")
	if m.barHeight() != 1 {
		t.Fatalf("without the line the bar should be one row, got %d", m.barHeight())
	}
	if !strings.HasPrefix(ansi.Strip(lines[1]), "╭─nvim") {
		t.Fatalf("nvim's box should start right under the bar, got %q", ansi.Strip(lines[1]))
	}
}

func TestPaneBoxesOffKeepsTheLineAndTheDivider(t *testing.T) {
	m, _, _ := sidePanelModel(t, 150)
	m.paneBoxes = false
	m.screen = screenEditor
	m.resizeSessions()
	w, h := m.sessionSize()

	if m.barHeight() != 2 {
		t.Fatalf("the bar should keep its line, got %d rows", m.barHeight())
	}
	if p := m.sessionPanes()[0]; p.x != 0 || p.y != 2 || p.w != w || p.h != h {
		t.Fatalf("nvim should fill the room, got %+v", p)
	}
	lines := strings.Split(m.View().Content, "\n")
	if !strings.HasPrefix(ansi.Strip(lines[1]), "───") || strings.Contains(ansi.Strip(lines[2]), "╭") {
		t.Fatalf("the line under the bar should be back and no box drawn:\n%q\n%q", ansi.Strip(lines[1]), ansi.Strip(lines[2]))
	}

	m = pressSideBySide(t, m)
	panes := m.sessionPanes()
	left, right := panes[0], panes[1]
	if left.w+1+right.w != w || right.x != left.x+left.w+1 {
		t.Fatalf("nvim and the agent should share the room around a divider, got %+v %+v", left, right)
	}
	if got := sizeOf(left.s); got.W != left.w || got.H != h {
		t.Fatalf("nvim is %+v, want %dx%d", got, left.w, h)
	}
	row := []rune(ansi.Strip(strings.Split(m.View().Content, "\n")[m.barHeight()]))
	if row[left.w] != '│' {
		t.Fatalf("the divider should be at column %d: %q", left.w, string(row))
	}
}

func TestRenderPaneBoxKeepsTheBorderInPlace(t *testing.T) {
	box := renderPaneBox("\x1b[41mabcdefghij", "", "", 4, 1, barSep)
	row := strings.Split(box, "\n")[1]

	if got := ansi.Strip(row); got != "│abcd│" {
		t.Fatalf("a line wider than the box should be cut, got %q", got)
	}
	short := strings.Split(renderPaneBox("\x1b[41mab", "", "", 4, 1, barSep), "\n")[1]
	if !strings.Contains(short, "ab\x1b[m  ") {
		t.Fatalf("the colour should be reset before the padding, got %q", short)
	}
}

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

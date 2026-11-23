package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/mateuszjasiuk/yardmaster/internal/ctl"
	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// draftModel is sidePanelModel on the agent screen of repo/wt, with a live cat
// agent that ym sees as idle.
func draftModel(t *testing.T) (Model, *session.Session) {
	t.Helper()
	m, _, mgr := sidePanelModel(t, 100)
	m.screen = screenAgent
	s, err := mgr.SpawnAgent("repo/wt", t.TempDir(), session.Spec{Kind: session.Agent, Argv: []string{"cat"}}, 80, 20)
	if err != nil {
		t.Fatal(err)
	}
	ws := m.current.ws
	ws.ActiveAgent = len(ws.Agents) - 1
	m.agentStatus = map[int]AgentStatus{s.Pid(): {Status: "idle"}}

	return m, s
}

func typeKeys(t *testing.T, m Model, keys ...tea.KeyPressMsg) Model {
	t.Helper()
	for _, k := range keys {
		m, _ = press(t, m, k)
	}

	return m
}

var (
	keyH     = tea.KeyPressMsg{Code: 'h', Text: "h"}
	keyEnter = tea.KeyPressMsg{Code: tea.KeyEnter}
	keyEsc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	keyCtrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

func TestDeliverQueuedHoldsForTheUsersDraftUntilItIsSent(t *testing.T) {
	m, s := draftModel(t)
	const key = "repo/wt"

	m = typeKeys(t, m, keyH)
	m.state.Enqueue(key, "a note")
	m.deliverQueued()
	if got := len(m.state.Queue(key)); got != 1 {
		t.Fatalf("a queued message must not be pasted into the user's draft; queue has %d", got)
	}
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "1 queued · waiting for your draft") {
		t.Fatalf("the agent pane should show the held queue:\n%s", view)
	}

	// Enter on an idle agent may only pick an @file item: still held.
	m = typeKeys(t, m, keyEnter)
	m.deliverQueued()
	if got := len(m.state.Queue(key)); got != 1 {
		t.Fatalf("an Enter the agent did not act on should keep the hold; queue has %d", got)
	}

	// The agent took the message: the next idle gets the queue.
	m.agentStatus[s.Pid()] = AgentStatus{Status: "busy"}
	m.deliverQueued()
	m.agentStatus[s.Pid()] = AgentStatus{Status: "idle"}
	m.deliverQueued()
	if got := len(m.state.Queue(key)); got != 0 {
		t.Fatalf("after the draft was sent the queue should be delivered; queue has %d", got)
	}
	if view := ansi.Strip(m.View().Content); strings.Contains(view, "waiting for your draft") {
		t.Fatal("the sign should go once the queue is delivered")
	}
}

func TestDraftMarkKeys(t *testing.T) {
	tests := []struct {
		name      string
		keys      []tea.KeyPressMsg
		wantDraft bool
	}{
		{"typing marks a draft", []tea.KeyPressMsg{keyH}, true},
		{"Up can recall a prompt", []tea.KeyPressMsg{{Code: tea.KeyUp}}, true},
		{"shift+enter adds a line", []tea.KeyPressMsg{keyH, {Code: tea.KeyEnter, Mod: tea.ModShift}}, true},
		{"ctrl+c empties the box", []tea.KeyPressMsg{keyH, keyCtrlC}, false},
		{"one Esc keeps the text", []tea.KeyPressMsg{keyH, keyEsc}, true},
		{"a double Esc empties the box", []tea.KeyPressMsg{keyH, keyEsc, keyEsc}, false},
		{"a key between two Esc", []tea.KeyPressMsg{keyH, keyEsc, keyH, keyEsc}, true},
		{"Esc alone marks nothing", []tea.KeyPressMsg{keyEsc}, false},
		{"Enter alone marks nothing", []tea.KeyPressMsg{keyEnter}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, s := draftModel(t)
			m = typeKeys(t, m, tt.keys...)
			if _, got := m.drafts[s]; got != tt.wantDraft {
				t.Fatalf("draft = %t, want %t", got, tt.wantDraft)
			}
		})
	}
}

func TestDraftMarkEdges(t *testing.T) {
	t.Run("Enter on a busy agent clears at once", func(t *testing.T) {
		m, s := draftModel(t)
		m.agentStatus[s.Pid()] = AgentStatus{Status: "busy"}
		m = typeKeys(t, m, keyH, keyEnter)
		if _, got := m.drafts[s]; got {
			t.Fatal("a busy Claude queues the typed message itself; the mark should go")
		}
	})
	t.Run("a slow second Esc keeps the text", func(t *testing.T) {
		m, s := draftModel(t)
		m = typeKeys(t, m, keyH, keyEsc)
		m.drafts[s] = draft{escAt: time.Now().Add(-2 * draftEscWindow)}
		m = typeKeys(t, m, keyEsc)
		if _, got := m.drafts[s]; !got {
			t.Fatal("two Esc far apart are not a double Esc")
		}
	})
	t.Run("a paste marks a draft", func(t *testing.T) {
		m, s := draftModel(t)
		mm, _ := m.Update(tea.PasteMsg{Content: "pasted"})
		if _, got := mm.(Model).drafts[s]; !got {
			t.Fatal("pasted text sits in the box like typed text")
		}
	})
	t.Run("nvim keys mark nothing", func(t *testing.T) {
		m, _ := draftModel(t)
		m.screen = screenEditor
		m = typeKeys(t, m, keyH)
		if len(m.drafts) != 0 {
			t.Fatalf("only agent sessions have drafts: %v", m.drafts)
		}
	})
}

func TestCtlSendSaysTheQueueIsHeldForTheDraft(t *testing.T) {
	ws, _, m := stageModel(t)
	s := ws.ActiveAgentSession()
	m.agentStatus = map[int]AgentStatus{s.Pid(): {Status: "idle"}}
	m.drafts[s] = draft{}

	p, ch := pending(ctl.Request{Op: ctl.OpSend, Worktree: "r/a", Text: "a note"})
	mm, _ := m.handleCtl(p)
	m = mm.(Model)

	if resp := <-ch; !resp.OK || !strings.Contains(resp.Message, "held: the user is typing") {
		t.Fatalf("the reply should say the message waits for the user's draft: %+v", resp)
	}
	if got := len(m.state.Queue("r/a")); got != 1 {
		t.Fatalf("the message should stay queued; queue has %d", got)
	}
	it, _ := m.activeByKey("r/a")
	if info := m.ctlInfo(it); !info.Held || info.Queued != 1 {
		t.Fatalf("ym ctl list should show the held queue: %+v", info)
	}
}

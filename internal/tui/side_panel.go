package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// panelIdleStyle is a session with nothing to show. The panel is a list to
// read, so it is lighter than the bar's dim gray, but still darker than a
// seen or highlighted session.
var panelIdleStyle = lipgloss.NewStyle().Foreground(cFg3)

// sidePanelBlank is an empty row of the side panel.
var sidePanelBlank = sidePanelRow{text: strings.Repeat(" ", sidePanelWidth)}

const (
	// sidePanelWidth is the side panel's width, without the blank column after it.
	sidePanelWidth = 28
	// minSessionWidth is the narrowest a session gets next to the side panel.
	// In a narrower terminal the panel doesn't show.
	minSessionWidth = 60
)

// sidePanelW is how many columns the side panel takes from the sessions, the
// blank column after it included: 0 when it is closed or the terminal is too
// narrow. While side by side is on, the panel needs room for two sessions, so
// it hides before they do.
func (m Model) sidePanelW() int {
	need := minSessionWidth
	if m.sideBySide {
		need = sideBySideMinWidth
	}
	if !m.sidePanel || m.width-sidePanelWidth-1 < need {
		return 0
	}
	return sidePanelWidth + 1
}

// toggleSidePanel opens or closes the side panel, gives the sessions the width
// that is left, and remembers the choice.
func (m Model) toggleSidePanel() (tea.Model, tea.Cmd) {
	m.sidePanel = !m.sidePanel
	if m.state != nil {
		m.state.SidePanel = m.sidePanel
	}
	m.resizeSessions()
	return m, m.writeStateCmd()
}

// withSidePanel draws the side panel to the left of a session body.
func (m Model) withSidePanel(body string) string {
	h := m.height - m.barHeight()
	rows := m.sidePanelRows(h)
	lines := splitLines(body)
	out := make([]string, len(rows))
	for i, r := range rows {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out[i] = r.text + " " + line
	}

	return strings.Join(out, "\n")
}

// sidePanelClick handles a left click on the side panel: a click on a box opens
// its session, and on a collapsed gh stack's box opens the stack like a click
// on the agent strip. Other rows do nothing.
func (m Model) sidePanelClick(x, y int) (tea.Model, tea.Cmd, bool) {
	if x >= m.sidePanelW() || y < m.barHeight() || !m.sidePanelShown() {
		return m, nil, false
	}

	rows := m.sidePanelRows(m.height - m.barHeight())
	i := y - m.barHeight()
	if i >= len(rows) {
		return m, nil, true
	}

	switch r := rows[i]; {
	case r.agent != nil:
		mm, cmd := m.focusBoardAgent(*r.agent)
		return mm, cmd, true
	case r.group != nil:
		mm, cmd := m.openStripGroup(*r.group)
		return mm, cmd, true
	}

	return m, nil, true
}

// sidePanelShown reports whether the side panel is on screen: open, with room,
// and next to a session rather than under the deck or an overlay.
func (m Model) sidePanelShown() bool {
	return m.sidePanelW() > 0 && m.screen != screenPicker && m.screen != screenSetup &&
		!m.overlayOpen() && !m.confirmActive() && m.activeSession() != nil
}

// sidePanelRows lays out the panel's h rows, each sidePanelWidth wide: a blank
// row under the bar, then the boxes, with "+N more" when they don't fit. A box
// is never cut: one that doesn't fit counts in "+N more" whole.
func (m Model) sidePanelRows(h int) []sidePanelRow {
	if h < 2 {
		return nil
	}

	layout := m.sidePanelLayout()

	if room := h - 1; len(layout) > room {
		cut := room - 1
		for cut > 0 && layout[cut].inBox {
			cut--
		}
		hidden := 0
		for _, r := range layout[cut:] {
			if !r.inBox && (r.agent != nil || r.group != nil) {
				hidden++
			}
		}
		more := sidePanelRow{text: padLine(dimStyle.Render(fmt.Sprintf(" +%d more", hidden)), sidePanelWidth)}
		layout = append(layout[:cut], more)
	}

	rows := append([]sidePanelRow{sidePanelBlank}, layout...)
	for len(rows) < h {
		rows = append(rows, sidePanelBlank)
	}
	return rows[:h]
}

// sidePanelRow is one row of the side panel. On a box's rows, agent is the
// session the box opens, or group the collapsed gh stack; both are nil on other
// rows. inBox marks a box's rows under its top border.
type sidePanelRow struct {
	text  string
	group *agentGroup
	agent *boardRow
	inBox bool
}

// sidePanelLayout lists the panel's rows: the agent strip's entries in the
// same order, a box per session, each repo under its name, and the worktrees
// of the gh stack you're in under the stack's name. A blank row ends that
// stack when a box of the same repo follows, as nothing else shows where it
// ends.
func (m Model) sidePanelLayout() []sidePanelRow {
	groups := m.stripEntries()
	var rows []sidePanelRow
	for i, g := range groups {
		inStack := inGhStack(g)
		continuesStack := i > 0 && inStack && sameGhStack(groups[i-1], g)
		startsStack := inStack && !continuesStack
		stackEnded := i > 0 && inGhStack(groups[i-1]) && !continuesStack
		newRepo := i == 0 || groups[i-1].repo != g.repo || groups[i-1].account != g.account

		if stackEnded && !newRepo {
			rows = append(rows, sidePanelBlank)
		}
		if newRepo && g.key != m.homeWSKey {
			rows = append(rows, sidePanelRow{text: sidePanelRepo(g)})
		}
		if startsStack {
			name := " " + dimStyle.Render(truncateTo(ghStackIcon+" "+g.stack.Name(), sidePanelWidth-1))
			rows = append(rows, sidePanelRow{text: padLine(name, sidePanelWidth)})
		}

		if g.members != nil {
			rows = append(rows, m.stackBox(&groups[i])...)
		} else {
			rows = append(rows, m.sessionBoxes(&groups[i])...)
		}
	}

	return rows
}

// stackBox is a collapsed gh stack's box: the stack's sign and name in the top
// border, and how many worktrees it holds inside.
func (m Model) stackBox(g *agentGroup) []sidePanelRow {
	sign, signSt, _ := m.stripLook(*g, panelIdleStyle)
	box := sidePanelBoxLook{sign: sign, signSt: signSt, bodySt: dimStyle}

	return box.render(ghStackIcon+" "+g.worktree, fmt.Sprintf("%d worktrees", len(g.members)), sidePanelRow{group: g})
}

// sessionBoxes are a worktree's boxes, one per session: the session's sign and
// name in the top border and the worktree inside. The active session has a
// heavy border.
func (m Model) sessionBoxes(g *agentGroup) []sidePanelRow {
	worktree := g.worktree
	if g.key == m.homeWSKey {
		worktree = "~"
	}
	worktreeSt := dimStyle
	if g.flag != nil {
		worktreeSt = worktreeSt.Foreground(g.flag)
	}

	var rows []sidePanelRow
	for i := range g.agents {
		a := &g.agents[i]
		sign, signSt := attnSign(a.attn, a.busy, panelIdleStyle)
		box := sidePanelBoxLook{heavy: m.isActiveSession(*a), sign: sign, signSt: signSt, bodySt: worktreeSt}
		rows = append(rows, box.render(a.label, worktree, sidePanelRow{agent: a})...)
	}

	return rows
}

// isActiveSession reports whether a is the session you'd come back to: the
// current worktree's active agent.
func (m Model) isActiveSession(a boardRow) bool {
	return m.current != nil && m.current.ws != nil && a.key == m.current.key && a.agentIdx == m.current.ws.ActiveAgent
}

// sidePanelBoxLook is how one box in the side panel looks: its weight, its
// sign, and the colours of the sign and the body. A box with a sign has its
// border and title in the sign's colour; a heavy box without one is in the
// accent colour; any other box has a faint border.
type sidePanelBoxLook struct {
	heavy          bool
	sign           string
	signSt, bodySt lipgloss.Style
}

// render draws the box one column in from the panel's edge: the corner, the
// sign (if any) and the title, the border to the right edge, then one row of
// body and the bottom border. A heavy box has an accent bar in that first
// column, which a status colour on its border can't hide. Every row opens
// what target opens.
func (b sidePanelBoxLook) render(title, body string, target sidePanelRow) []sidePanelRow {
	const boxW = sidePanelWidth - 1
	edge := lipgloss.RoundedBorder()
	if b.heavy {
		edge = lipgloss.ThickBorder()
	}
	border, titleSt := barSep, b.signSt
	switch {
	case b.sign != "":
		border = lipgloss.NewStyle().Foreground(b.signSt.GetForeground())
	case b.heavy:
		border, titleSt = accentStyle, b.signSt.Foreground(cAccent)
	}

	head := ""
	if b.sign != "" {
		head = b.signSt.Render(b.sign) + " "
	}
	// The corner and one line before the title, at least one line and the
	// corner after it.
	head += titleSt.Render(truncateTo(title, boxW-4-lipgloss.Width(head)))
	fill := strings.Repeat(edge.Top, boxW-3-lipgloss.Width(head))
	body = padLine(b.bodySt.Render(truncateTo(body, boxW-2)), boxW-2)
	margin := " "
	if b.heavy {
		margin = accentStyle.Render("▌")
	}

	rows := []sidePanelRow{target, target, target}
	rows[0].text = margin + border.Render(edge.TopLeft+edge.Top) + head + border.Render(fill+edge.TopRight)
	rows[1].text = margin + border.Render(edge.Left) + body + border.Render(edge.Right)
	rows[2].text = margin + border.Render(edge.BottomLeft+strings.Repeat(edge.Bottom, boxW-2)+edge.BottomRight)
	rows[1].inBox, rows[2].inBox = true, true

	return rows
}

// sidePanelRepo renders a repo's header row. An extra Claude account's block
// names the account after the repo, as the panel has no background to mark it.
func sidePanelRepo(g agentGroup) string {
	tag := accountTag(g)
	repo := truncateTo(g.repo, sidePanelWidth-2-lipgloss.Width(tag))
	return padLine("  "+repoHdrStyle.Render(repo)+dimStyle.Render(tag), sidePanelWidth)
}

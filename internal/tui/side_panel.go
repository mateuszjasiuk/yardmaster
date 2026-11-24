package tui

import (
	"fmt"
	"slices"
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
	rows := m.sidePanelRows()
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
// on the agent strip. A click on "↑ N more" or "↓ N more" scrolls a page.
// Other rows do nothing.
func (m Model) sidePanelClick(x, y int) (tea.Model, tea.Cmd, bool) {
	if !m.overSidePanel(x, y) {
		return m, nil, false
	}

	rows := m.sidePanelRows()
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
	case r.scroll != 0:
		return m.scrollSidePanel(r.scroll), nil, true
	}

	return m, nil, true
}

// sidePanelWheel scrolls the side panel by one box when the wheel turns over
// it. It reports false when the wheel is elsewhere.
func (m Model) sidePanelWheel(msg tea.MouseWheelMsg) (Model, bool) {
	mo := msg.Mouse()
	if !m.overSidePanel(mo.X, mo.Y) {
		return m, false
	}

	switch mo.Button {
	case tea.MouseWheelUp:
		return m.scrollSidePanel(-1), true
	case tea.MouseWheelDown:
		return m.scrollSidePanel(1), true
	}

	return m, true
}

// overSidePanel reports whether the point (x, y) is on the side panel while it
// is drawn.
func (m Model) overSidePanel(x, y int) bool {
	return x < m.sidePanelW() && y >= m.barHeight() && m.sidePanelShown()
}

// sidePanelH is how many rows the side panel has: all of them under the bar.
func (m Model) sidePanelH() int {
	return m.height - m.barHeight()
}

// scrollSidePanel moves the panel's window by delta boxes. It stays there until
// the active session changes.
func (m Model) scrollSidePanel(delta int) Model {
	blocks := m.sidePanelBlocks()
	m.panelTop = clampPanelTop(blocks, m.panelTop+delta, m.sidePanelH())

	return m
}

// syncSidePanel moves the panel's window just enough to show the active
// session's box when the active session has changed since the window last
// moved. Update runs it after every message, so every way of switching
// sessions is covered.
func (m *Model) syncSidePanel() {
	session := m.activeSessionID()
	if session == m.panelTopSession || !m.sidePanelShown() {
		return
	}

	blocks := m.sidePanelBlocks()
	m.panelTop = followActiveBlock(blocks, m.panelTop, m.sidePanelH())
	m.panelTopSession = session
}

// sidePanelShown reports whether the side panel is on screen: open, with room,
// and next to a session rather than under the deck or an overlay.
func (m Model) sidePanelShown() bool {
	return m.sidePanelW() > 0 && m.screen != screenPicker && m.screen != screenSetup &&
		!m.overlayOpen() && !m.confirmActive() && m.activeSession() != nil
}

// sidePanelRows lays out the panel's rows, each sidePanelWidth wide: the top
// row, then the boxes that fit in the window, then "↓ N more" when some are
// below it. The top row holds "↑ N more" when boxes are above the window, and
// the repo's name when its header is above it. A box is never cut, and a
// header always shows with its first box. Only the boxes in the window are
// drawn.
func (m Model) sidePanelRows() []sidePanelRow {
	h := m.sidePanelH()
	if h < 2 {
		return nil
	}

	blocks := m.sidePanelBlocks()
	top := clampPanelTop(blocks, m.panelTop, h)
	end := panelWindowEnd(blocks, top, h)
	page := max(1, end-top)

	rows := []sidePanelRow{sidePanelTopRow(blocks, top, page)}
	for _, b := range blocks[top:end] {
		rows = append(rows, b.rows()...)
	}
	if end < len(blocks) {
		rows = append(rows, sidePanelMoreRow("↓", blocks[end:], page))
	}
	for len(rows) < h {
		rows = append(rows, sidePanelBlank)
	}

	return rows
}

// sidePanelRow is one row of the side panel. On a box's rows, agent is the
// session the box opens, or group the collapsed gh stack; both are nil on other
// rows. scroll is how many boxes a click on the row scrolls: a page back on
// "↑ N more", a page on "↓ N more", 0 elsewhere.
type sidePanelRow struct {
	text   string
	group  *agentGroup
	agent  *boardRow
	scroll int
}

// sidePanelBlock is one box of the side panel with the rows above it that
// belong to it (a blank row that ends a stack, the repo's or the stack's
// header), so they scroll and fit together. The box is drawn only when it is
// in the window.
type sidePanelBlock struct {
	lead        []sidePanelRow
	look        sidePanelBoxLook
	title, body string
	stage       string
	target      sidePanelRow
	// repoAbove is the worktree whose repo header is above this box, out of
	// its own rows; nil when the box holds the header or has none (home).
	repoAbove *agentGroup
	attn      attnLevel
	active    bool
}

// height is how many rows the block takes: its lead rows and the box's three.
func (b sidePanelBlock) height() int {
	return len(b.lead) + 3
}

// rows draws the block: its lead rows, then the box.
func (b sidePanelBlock) rows() []sidePanelRow {
	return append(slices.Clone(b.lead), b.look.render(b.title, b.body, b.stage, b.target)...)
}

// sidePanelBlocks lists the panel's boxes as blocks: the agent strip's entries
// in the same order, a box per session, each repo under its name, and the
// worktrees of the gh stack you're in under the stack's name.
func (m Model) sidePanelBlocks() []sidePanelBlock {
	groups := m.stripEntries()
	var blocks []sidePanelBlock
	for i := range groups {
		g := &groups[i]
		lead, showsRepo := m.sidePanelLeadRows(groups, i)

		var boxes []sidePanelBlock
		if g.members != nil {
			boxes = []sidePanelBlock{m.stackBox(g)}
		} else {
			boxes = m.sessionBoxes(g)
		}
		for j := range boxes {
			if g.key != m.homeWSKey {
				boxes[j].repoAbove = g
			}
		}
		if len(boxes) > 0 {
			boxes[0].lead = lead
			if showsRepo {
				boxes[0].repoAbove = nil
			}
		}
		blocks = append(blocks, boxes...)
	}

	return blocks
}

// sidePanelLeadRows are the rows above the first box of groups[i]: a blank row
// that ends a feature or the gh stack you're in when a box of the same repo
// follows, as nothing else shows where it ends; the repo's header for a new
// repo; the feature's or the stack's header where it starts. showsRepo reports
// whether they hold the repo's header.
func (m Model) sidePanelLeadRows(groups []agentGroup, i int) (lead []sidePanelRow, showsRepo bool) {
	g := groups[i]
	var prev agentGroup
	if i > 0 {
		prev = groups[i-1]
	}
	inStack := inGhStack(g)
	continuesStack := i > 0 && inStack && sameGhStack(prev, g)
	startsFeature := g.feature != "" && prev.feature != g.feature
	stackEnded := i > 0 && inGhStack(prev) && !continuesStack
	featureEnded := prev.feature != "" && prev.feature != g.feature
	newRepo := i == 0 || prev.repo != g.repo || prev.account != g.account
	showsRepo = newRepo && g.key != m.homeWSKey

	if (stackEnded || featureEnded || startsFeature) && !newRepo {
		lead = append(lead, sidePanelBlank)
	}
	if showsRepo {
		lead = append(lead, sidePanelRow{text: sidePanelRepo(g, sidePanelWidth)})
	}
	if startsFeature {
		_, name, _ := strings.Cut(g.feature, "/")
		lead = append(lead, sidePanelGroupHeader(featureIcon+" "+name))
	}
	if inStack && !continuesStack {
		lead = append(lead, sidePanelGroupHeader(ghStackIcon+" "+g.stack.Name()))
	}

	return lead, showsRepo
}

// sidePanelGroupHeader is the header row above a feature's or a gh stack's
// boxes, in the grouping colour.
func sidePanelGroupHeader(name string) sidePanelRow {
	text := " " + groupStyle.Render(truncateTo(name, sidePanelWidth-1))

	return sidePanelRow{text: padLine(text, sidePanelWidth)}
}

// featureIcon marks a feature's header in the session list.
const featureIcon = "⚑"

// managerIcon marks the sessions of a feature's manager (its coordinator).
const managerIcon = "★"

// stackBox is a collapsed gh stack's box: the stack's sign and name in the top
// border, and how many worktrees it holds inside.
func (m Model) stackBox(g *agentGroup) sidePanelBlock {
	sign, signSt, _ := m.stripLook(*g, panelIdleStyle)
	bodySt := dimStyle
	if g.flag != nil {
		bodySt = bodySt.Foreground(g.flag)
	}

	return sidePanelBlock{
		look: sidePanelBoxLook{
			sign: sign, seen: g.attn == attnSeen,
			signSt: signSt, titleSt: groupStyle, bodySt: bodySt,
		},
		title: ghStackIcon + " " + g.worktree, body: fmt.Sprintf("%d worktrees", len(g.members)),
		target: sidePanelRow{group: g}, attn: g.attn,
	}
}

// sessionBoxes are a worktree's boxes, one per session: the session's sign and
// name in the top border and the worktree inside. The active session has a
// heavy border; a feature's manager has its mark before the name.
func (m Model) sessionBoxes(g *agentGroup) []sidePanelBlock {
	worktree := g.worktree
	if g.key == m.homeWSKey {
		worktree = "~"
	}
	worktreeSt := dimStyle
	if g.flag != nil {
		worktreeSt = worktreeSt.Foreground(g.flag)
	}

	var blocks []sidePanelBlock
	for i := range g.agents {
		a := &g.agents[i]
		mark := ""
		if g.manager {
			mark = managerIcon
		}
		active := m.isActiveSession(*a)
		sign, signSt := attnSign(a.attn, a.busy, a.starting, panelIdleStyle)
		blocks = append(blocks, sidePanelBlock{
			look: sidePanelBoxLook{
				heavy: active, sign: sign, mark: mark, seen: a.attn == attnSeen,
				signSt: signSt, titleSt: identityStyle, bodySt: worktreeSt,
			},
			title: a.label, body: worktree, stage: a.stage, target: sidePanelRow{agent: a},
			attn: a.attn, active: active,
		})
	}

	return blocks
}

// activeSessionID names the session you'd come back to, so the panel can tell
// when it changes: the current worktree and its active agent.
func (m Model) activeSessionID() string {
	if m.current == nil || m.current.ws == nil {
		return ""
	}

	return fmt.Sprintf("%s#%d", m.current.key, m.current.ws.ActiveAgent)
}

// followActiveBlock moves top just enough that the active block fits in the
// window. Without an active block it leaves top as it is.
func followActiveBlock(blocks []sidePanelBlock, top, h int) int {
	active := slices.IndexFunc(blocks, func(b sidePanelBlock) bool { return b.active })
	if active < 0 {
		return clampPanelTop(blocks, top, h)
	}

	top = clampPanelTop(blocks, top, h)
	if active < top {
		return active
	}
	for top < active && panelWindowEnd(blocks, top, h) <= active {
		top++
	}

	return top
}

// clampPanelTop keeps top inside the list, and no lower than needed to show
// the last block, so the window never ends in empty rows while boxes are
// above it.
func clampPanelTop(blocks []sidePanelBlock, top, h int) int {
	lowestTop := len(blocks)
	used := 0
	for lowestTop > 0 && used+blocks[lowestTop-1].height() <= h-1 {
		used += blocks[lowestTop-1].height()
		lowestTop--
	}

	return clamp(top, 0, lowestTop)
}

// panelWindowEnd is the end of the window that starts at top in h rows: one
// row goes to the top row, and one to "↓ N more" when blocks are left below.
func panelWindowEnd(blocks []sidePanelBlock, top, h int) int {
	end := fitBlocks(blocks, top, h-1)
	if end == len(blocks) {
		return end
	}

	return fitBlocks(blocks, top, h-2)
}

// fitBlocks is the end of the run of whole blocks from top that fits in room
// rows.
func fitBlocks(blocks []sidePanelBlock, top, room int) int {
	end := top
	for end < len(blocks) && blocks[end].height() <= room {
		room -= blocks[end].height()
		end++
	}

	return end
}

// sidePanelTopRow is the panel's first row: the repo's name when its header is
// above the window, and "↑ N more" when boxes are.
func sidePanelTopRow(blocks []sidePanelBlock, top, page int) sidePanelRow {
	if top == 0 {
		return sidePanelBlank
	}

	more := sidePanelMoreRow("↑", blocks[:top], -page)
	firstShown := blocks[min(top, len(blocks)-1)]
	if firstShown.repoAbove == nil {
		return more
	}

	count := moreCount("↑", blocks[:top])
	more.text = sidePanelRepo(*firstShown.repoAbove, sidePanelWidth-lipgloss.Width(count)-1) + " " + count

	return more
}

// sidePanelMoreRow is the "↑ N more" or "↓ N more" row for the hidden boxes. A
// click on it scrolls by scroll boxes.
func sidePanelMoreRow(arrow string, hidden []sidePanelBlock, scroll int) sidePanelRow {
	return sidePanelRow{text: padLine("  "+moreCount(arrow, hidden), sidePanelWidth), scroll: scroll}
}

// moreCount says how many boxes are hidden on one side of the window, with how
// many of them wait for you ("!") or are done ("✓"), so a session that needs
// you never hides.
func moreCount(arrow string, hidden []sidePanelBlock) string {
	waiting, done := 0, 0
	for _, b := range hidden {
		switch b.attn {
		case attnWaiting:
			waiting++
		case attnDone:
			done++
		}
	}

	text := dimStyle.Render(fmt.Sprintf("%s %d more", arrow, len(hidden)))
	if waiting > 0 {
		text += " " + boldRed.Render(fmt.Sprintf("!%d", waiting))
	}
	if done > 0 {
		text += " " + boldGreen.Render(fmt.Sprintf("✓%d", done))
	}

	return text
}

// isActiveSession reports whether a is the session you'd come back to: the
// current worktree's active agent.
func (m Model) isActiveSession(a boardRow) bool {
	return m.current != nil && m.current.ws != nil && a.key == m.current.key && a.agentIdx == m.current.ws.ActiveAgent
}

// sidePanelBoxLook is how one box in the side panel looks: its weight, its
// sign, a mark before the title (if any, always in the grouping colour), and
// the colours of the sign, the title and the body. A box with a sign has its
// border in the sign's colour; a heavy box without one is in the accent
// colour; a seen box (finished, viewed, still your turn) has its border in the
// sign style's colour too, as it has no sign; any other box has a faint
// border. The title keeps its own colour, so status never hides who it is.
type sidePanelBoxLook struct {
	heavy, seen             bool
	sign, mark              string
	signSt, titleSt, bodySt lipgloss.Style
}

// render draws the box one column in from the panel's edge: the corner, the
// sign (if any) and the title, the border to the right edge, then one row of
// body and the bottom border, with the stage (if any) at its right end. A
// heavy box has an accent bar in that first column, which a status colour on
// its border can't hide. Every row opens what target opens.
func (b sidePanelBoxLook) render(title, body, stage string, target sidePanelRow) []sidePanelRow {
	const boxW = sidePanelWidth - 1
	edge := lipgloss.RoundedBorder()
	if b.heavy {
		edge = lipgloss.ThickBorder()
	}
	border := barSep
	switch {
	case b.sign != "":
		border = lipgloss.NewStyle().Foreground(b.signSt.GetForeground())
	case b.heavy:
		border = accentStyle
	case b.seen:
		border = lipgloss.NewStyle().Foreground(b.signSt.GetForeground())
	}

	head := ""
	if b.sign != "" {
		head = b.signSt.Render(b.sign) + " "
	}
	if b.mark != "" {
		head += groupStyle.Render(b.mark) + " "
	}
	// The corner and one line before the title, at least one line and the
	// corner after it.
	head += b.titleSt.Render(truncateTo(title, boxW-4-lipgloss.Width(head)))
	fill := strings.Repeat(edge.Top, boxW-3-lipgloss.Width(head))
	body = padLine(b.bodySt.Render(truncateTo(body, boxW-2)), boxW-2)
	margin := " "
	if b.heavy {
		margin = accentStyle.Render("▌")
	}

	rows := []sidePanelRow{target, target, target}
	rows[0].text = margin + border.Render(edge.TopLeft+edge.Top) + head + border.Render(fill+edge.TopRight)
	rows[1].text = margin + border.Render(edge.Left) + body + border.Render(edge.Right)
	rows[2].text = margin + bottomBorder(edge, border, stage, boxW)

	return rows
}

// bottomBorder is a box's bottom border, boxW wide, with the stage (if any) at
// its right end in a dim colour.
func bottomBorder(edge lipgloss.Border, border lipgloss.Style, stage string, boxW int) string {
	if stage == "" {
		return border.Render(edge.BottomLeft + strings.Repeat(edge.Bottom, boxW-2) + edge.BottomRight)
	}

	// Around the stage: the corner and at least one line before it, a space on
	// each side, one line and the corner after it.
	const around = 6
	stage = truncateTo(stage, boxW-around)
	line := strings.Repeat(edge.Bottom, boxW-around+1-lipgloss.Width(stage))

	return border.Render(edge.BottomLeft+line+" ") + dimStyle.Render(stage) + border.Render(" "+edge.Bottom+edge.BottomRight)
}

// sidePanelRepo renders a repo's header, w columns wide. An extra Claude
// account's block names the account after the repo, as the panel has no
// background to mark it.
func sidePanelRepo(g agentGroup, w int) string {
	tag := accountTag(g)
	repo := truncateTo(g.repo, w-2-lipgloss.Width(tag))

	return padLine("  "+placeHdrStyle.Render(repo)+dimStyle.Render(tag), w)
}

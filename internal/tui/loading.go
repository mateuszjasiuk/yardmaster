package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mateuszjasiuk/yardmaster/internal/config"
	"github.com/mateuszjasiuk/yardmaster/internal/session"
)

// A starting pane stays blank this long before its sign shows, so a program
// that draws at once never flickers.
const startingSignDelay = 150 * time.Millisecond

// After these waits the sign adds the seconds, then says "still starting" and
// how to restart.
const (
	startingShowSeconds = 5 * time.Second
	startingTooLong     = 30 * time.Second
)

// loadingTickMsg advances every loading spinner on screen. Like the plasma tick
// it is armed only while something is loading, so no timer runs at rest.
type loadingTickMsg struct{}

// loadingTickInterval paces the spinner. Fast enough to read as motion — the
// whole point is to tell "working" from "hung" — and slow enough that a long
// wait costs a handful of redraws per second.
const loadingTickInterval = 120 * time.Millisecond

func scheduleLoadingTick() tea.Cmd {
	return tea.Tick(loadingTickInterval, func(time.Time) tea.Msg { return loadingTickMsg{} })
}

// spinnerFrames are the braille frames a loading sign animates through.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// loading reports whether anything on screen shows a spinner, so the tick
// keeps running.
func (m Model) loading() bool {
	if m.scanning() || len(m.creating) > 0 {
		return true
	}
	for _, s := range m.visibleSessions() {
		if s.Starting() {
			return true
		}
	}

	return false
}

// scanning reports whether the deck is still waiting for its first scan.
func (m Model) scanning() bool {
	return m.screen != screenSetup && !m.groupsLoaded && !m.scanFailed
}

// spinner is the current spinner frame. Every spinner shows the same frame,
// taken from the clock, so several on screen move together.
func (m Model) spinner() string {
	if m.iconMode == config.IconsASCII || m.iconMode == config.IconsText {
		return "*"
	}
	frame := time.Now().UnixMilli() / loadingTickInterval.Milliseconds()

	return spinnerFrames[int(frame%int64(len(spinnerFrames)))]
}

// loadingLine is a spinner in the working colour, then dim text.
func (m Model) loadingLine(text string) string {
	return stackWaitStyle.Render(m.spinner()) + " " + dimStyle.Render(text)
}

// paneScreen is what a pane shows: the program's screen, or a sign in the
// middle of the pane while the program starts or after it failed to.
func (m Model) paneScreen(s *session.Session) string {
	sign := m.startingSign(s)
	if sign == "" {
		return s.Render()
	}
	w, h := s.Size()

	return centerBlock(sign, w, h)
}

// startingSign is the sign for a pane whose program has not drawn yet, or ""
// when the program's own screen should show.
func (m Model) startingSign(s *session.Session) string {
	if s.ExitedBeforeStart() {
		sign := errStyle.Render(paneProgram(s) + " exited before it started")
		if s.Kind == session.Agent {
			sign += "\n" + m.restartHint()
		}
		return sign
	}
	if !s.Starting() {
		return ""
	}

	return m.startingSignAfter(s, time.Since(s.StartedAt()))
}

// startingSignAfter is the sign of a starting pane that has waited this long.
func (m Model) startingSignAfter(s *session.Session, waited time.Duration) string {
	if waited < startingSignDelay {
		return ""
	}
	text := "starting " + paneProgram(s) + "…"
	if waited >= startingTooLong {
		text = "still " + text
	}
	if waited >= startingShowSeconds {
		text += fmt.Sprintf(" %ds", int(waited.Seconds()))
	}
	if held := s.HeldKeys(); held > 0 {
		text += fmt.Sprintf(" (%s waiting)", pluralize(held, "key"))
	}
	sign := m.loadingLine(text)
	if waited >= startingTooLong && s.Kind == session.Agent {
		sign += "\n" + m.restartHint()
	}

	return sign
}

// restartHint says how to restart the agent: through the command palette,
// since the board's restart key works only while the board is open.
func (m Model) restartHint() string {
	return dimStyle.Render(m.keys.CommandPalette + " then \"restart agent\"")
}

// paneProgram names what runs in a pane, in the words the user knows it by.
func paneProgram(s *session.Session) string {
	switch s.Kind {
	case session.Editor:
		return "nvim"
	case session.Terminal:
		return "terminal"
	default:
		return normalizedProvider(s.Provider).String()
	}
}

package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mateuszjasiuk/yardmaster/internal/config"
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
	if m.scanning() {
		return true
	}

	return len(m.creating) > 0
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

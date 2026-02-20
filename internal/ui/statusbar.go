package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

const statusIdleText = "Press Ctrl+/ for commands"

// revertDelay is how long (ms) a status message stays before reverting to the
// idle hint.
const revertDelay = 5000

// StatusBar displays transformation results and the idle usage hint at the
// bottom of the application window.
type StatusBar struct {
	Box      *gtk.Box
	label    *gtk.Label
	timerTag glib.SourceHandle // 0 when no timer is pending
}

// NewStatusBar creates a status bar widget that is always visible and shows the
// usage hint by default.
func NewStatusBar() *StatusBar {
	label := gtk.NewLabel(statusIdleText)
	label.SetXAlign(0)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetHExpand(true)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("statusbar")
	box.AddCSSClass("statusbar-idle")
	box.SetMarginTop(4)
	box.SetMarginBottom(4)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	box.Append(label)

	return &StatusBar{Box: box, label: label}
}

// ShowError displays an error message and schedules a revert to the idle hint.
func (s *StatusBar) ShowError(message, logPath string) {
	s.cancelTimer()
	text := message
	if logPath != "" {
		text += "  (log: " + logPath + ")"
	}
	s.label.SetText(text)
	s.Box.RemoveCSSClass("statusbar-success")
	s.Box.RemoveCSSClass("statusbar-idle")
	s.Box.AddCSSClass("statusbar-error")
	s.scheduleRevert()
}

// ShowSuccess displays a success message and schedules a revert to the idle
// hint. The caller provides the full display text.
func (s *StatusBar) ShowSuccess(message string) {
	s.cancelTimer()
	s.label.SetText(message)
	s.Box.RemoveCSSClass("statusbar-error")
	s.Box.RemoveCSSClass("statusbar-idle")
	s.Box.AddCSSClass("statusbar-success")
	s.scheduleRevert()
}

// Clear immediately reverts the status bar to the idle hint.
func (s *StatusBar) Clear() {
	s.cancelTimer()
	s.revertToIdle()
}

func (s *StatusBar) scheduleRevert() {
	s.timerTag = glib.TimeoutAdd(revertDelay, func() bool {
		s.revertToIdle()
		return false // SOURCE_REMOVE — do not repeat
	})
}

func (s *StatusBar) cancelTimer() {
	if s.timerTag != 0 {
		glib.SourceRemove(s.timerTag)
		s.timerTag = 0
	}
}

func (s *StatusBar) revertToIdle() {
	s.timerTag = 0
	s.label.SetText(statusIdleText)
	s.Box.RemoveCSSClass("statusbar-error")
	s.Box.RemoveCSSClass("statusbar-success")
	s.Box.AddCSSClass("statusbar-idle")
}

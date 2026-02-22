package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

const defaultIdleText = "Press Ctrl+/ for commands"

// successRevertDelay is how long (ms) a success message stays before reverting
// to the idle hint. Error messages do NOT auto-revert — they stay until the
// next user action so there is enough time to read and act on them.
const successRevertDelay = 5000

// StatusBar displays transformation results and the idle usage hint at the
// bottom of the application window. It contains three independent zones:
//   - notification zone (left): transient event messages that auto-revert
//   - spinner (centre-right): animates while a script is executing
//   - syntax zone (right): persistent detected-language indicator
type StatusBar struct {
	Box         *gtk.Box
	label       *gtk.Label        // notification zone
	spinner     *gtk.Spinner      // busy indicator
	syntaxLabel *gtk.Label        // syntax zone — right-aligned, empty when inactive
	timerTag    glib.SourceHandle // 0 when no timer is pending
	idleText    string
	isIdle      bool
}

// NewStatusBar creates a status bar widget that is always visible and shows the
// usage hint by default. The bar has three independent zones: a left
// notification zone, a centre-right busy spinner, and a right syntax-language
// indicator zone.
func NewStatusBar() *StatusBar {
	// Notification zone — left-aligned, expands to fill available space.
	label := gtk.NewLabel(defaultIdleText)
	label.SetXAlign(0)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetHExpand(true)

	// Busy spinner — hidden until SetBusy(true) is called.
	spinner := gtk.NewSpinner()
	spinner.SetMarginStart(6)
	spinner.SetMarginEnd(4)
	spinner.SetVisible(false)

	// Syntax zone — right-aligned, shows the detected language name when active.
	syntaxLabel := gtk.NewLabel("")
	syntaxLabel.SetXAlign(1)
	syntaxLabel.SetMarginStart(8)
	syntaxLabel.AddCSSClass("statusbar-syntax")

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("statusbar")
	box.AddCSSClass("statusbar-idle")
	box.SetMarginTop(4)
	box.SetMarginBottom(4)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	box.Append(label)
	box.Append(spinner)
	box.Append(syntaxLabel)

	return &StatusBar{
		Box:         box,
		label:       label,
		spinner:     spinner,
		syntaxLabel: syntaxLabel,
		idleText:    defaultIdleText,
		isIdle:      true,
	}
}

// SetBusy shows or hides the busy spinner. Call with true before launching a
// script goroutine and with false inside the glib.IdleAdd callback that
// delivers the result. Must be called on the GTK main thread.
func (s *StatusBar) SetBusy(busy bool) {
	if busy {
		s.spinner.SetVisible(true)
		s.spinner.Start()
	} else {
		s.spinner.Stop()
		s.spinner.SetVisible(false)
	}
}

// SetSyntaxLanguage shows the detected language name in the right-aligned
// syntax zone. Calling this never affects the notification zone.
func (s *StatusBar) SetSyntaxLanguage(name string) {
	s.syntaxLabel.SetText(name)
}

// ClearSyntaxLanguage removes any language indicator from the syntax zone.
// Safe to call when no language is active.
func (s *StatusBar) ClearSyntaxLanguage() {
	s.syntaxLabel.SetText("")
}

// SetIdleHint updates the idle-state hint text. If the bar is currently
// showing the idle hint it refreshes immediately.
func (s *StatusBar) SetIdleHint(text string) {
	s.idleText = text
	if s.isIdle {
		s.label.SetText(text)
	}
}

// ShowError displays an error message. Unlike ShowSuccess, errors do NOT
// auto-dismiss — they persist until the next call to ShowSuccess, Clear, or
// SetBusy(false), giving the user enough time to read and act on them.
func (s *StatusBar) ShowError(message, logPath string) {
	s.cancelTimer()
	s.isIdle = false
	text := message
	if logPath != "" {
		text += "  (log: " + logPath + ")"
	}
	s.label.SetText(text)
	s.Box.RemoveCSSClass("statusbar-success")
	s.Box.RemoveCSSClass("statusbar-idle")
	s.Box.AddCSSClass("statusbar-error")
	// No scheduleRevert — errors are sticky.
}

// ShowSuccess displays a success message and schedules a revert to the idle
// hint after successRevertDelay ms. The caller provides the full display text.
func (s *StatusBar) ShowSuccess(message string) {
	s.cancelTimer()
	s.isIdle = false
	s.label.SetText(message)
	s.Box.RemoveCSSClass("statusbar-error")
	s.Box.RemoveCSSClass("statusbar-idle")
	s.Box.AddCSSClass("statusbar-success")
	s.scheduleRevert()
}

func (s *StatusBar) scheduleRevert() {
	s.timerTag = glib.TimeoutAdd(successRevertDelay, func() bool {
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
	s.isIdle = true
	s.label.SetText(s.idleText)
	s.Box.RemoveCSSClass("statusbar-error")
	s.Box.RemoveCSSClass("statusbar-success")
	s.Box.AddCSSClass("statusbar-idle")
}

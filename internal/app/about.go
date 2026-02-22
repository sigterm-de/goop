package app

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	gtksource "libdb.so/gotk4-sourceview/pkg/gtksource/v5"
)

// ShowAboutDialog opens the About window, transient to parent.
func ShowAboutDialog(parent *gtk.ApplicationWindow, version string) {
	gtkMaj := gtk.GetMajorVersion()
	gtkMin := gtk.GetMinorVersion()
	gtkMic := gtk.GetMicroVersion()
	svMaj := gtksource.GetMajorVersion()
	svMin := gtksource.GetMinorVersion()
	svMic := gtksource.GetMicroVersion()

	dialog := gtk.NewAboutDialog()
	dialog.SetTransientFor(&parent.Window)
	dialog.SetModal(true)
	dialog.SetProgramName("goop")
	dialog.SetVersion(version)
	dialog.SetComments(fmt.Sprintf(
		"A Boop-compatible text transformation tool.\n\nGTK %d.%d.%d · GtkSourceView %d.%d.%d",
		gtkMaj, gtkMin, gtkMic, svMaj, svMin, svMic,
	))
	dialog.SetWebsite("https://codeberg.org/sigterm-de/goop")
	dialog.SetWebsiteLabel("codeberg.org/sigterm-de/goop")
	dialog.SetLogoIconName("goop")
	dialog.Present()
}

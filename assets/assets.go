// Package assets exposes the embedded Boop scripts, CSS stylesheet, and application icon.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed scripts style.css
var embedded embed.FS

//go:embed icons/goop.png
var iconPNG []byte

// Scripts returns a sub-filesystem rooted at the scripts/ directory.
// The returned fs.FS contains all bundled .js script files.
func Scripts() fs.FS {
	sub, err := fs.Sub(embedded, "scripts")
	if err != nil {
		panic("assets: sub scripts: " + err.Error())
	}
	return sub
}

// StyleCSS returns the contents of the application CSS stylesheet.
func StyleCSS() []byte {
	data, err := embedded.ReadFile("style.css")
	if err != nil {
		panic("assets: read style.css: " + err.Error())
	}
	return data
}

// Icon returns the application icon as PNG bytes.
func Icon() []byte { return iconPNG }

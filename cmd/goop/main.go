package main

import (
	"flag"
	"fmt"
	"os"

	"codeberg.org/sigterm-de/goop/internal/app"
)

// Injected at build time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("goop %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}

	os.Exit(app.Run(version))
}

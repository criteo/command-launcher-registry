package main

import (
	"os"

	"github.com/criteo/command-launcher-registry/internal/branding"
	"github.com/criteo/command-launcher-registry/internal/client/commands"
)

// Version information - set via ldflags at build time
var (
	version  = "dev"
	buildNum = "local"
	appName  = "cola-regctl"
)

func main() {
	// Initialize branding from build-time appName
	branding.Init(appName)

	commands.SetVersionInfo(version, buildNum, appName)
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}

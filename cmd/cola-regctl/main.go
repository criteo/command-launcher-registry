package main

import (
	"os"

	"github.com/criteo/command-launcher-registry/internal/client/commands"
)

// Version information - set via ldflags at build time
var (
	version     = "dev"
	buildNum    = "local"
	appName     = "cola-regctl"
	appLongName = "Command Launcher Registry CLI"
)

func main() {
	commands.SetVersionInfo(version, buildNum, appName, appLongName)
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}

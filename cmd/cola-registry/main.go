package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/criteo/command-launcher-registry/internal/cli"
)

// Version information - set via ldflags at build time
var (
	version     = "dev"
	buildNum    = "local"
	appName     = "cola-registry"
	appLongName = "Command Launcher Registry Server"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "cola-registry",
	Short: "Command Launcher Registry Server",
	Long: `COLA Registry Server provides a REST API for managing Command Launcher
remote registries. It serves registry indexes and provides full CRUD operations
for registries, packages, and versions.`,
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(cli.ServerCmd)
	rootCmd.AddCommand(cli.AuthCmd)
}

// versionTemplate returns the version output template
func versionTemplate() string {
	if buildNum != "" && buildNum != "local" {
		return fmt.Sprintf("%s version %s (build %s)\n", appName, version, buildNum)
	}
	return fmt.Sprintf("%s version %s\n", appName, version)
}

func main() {
	// Set version info
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(versionTemplate())

	// Pass version info to cli package for server startup logging
	cli.SetVersionInfo(version, buildNum, appName, appLongName)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

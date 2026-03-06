package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/criteo/command-launcher-registry/internal/branding"
)

// Version information - set from main.go via SetVersionInfo
var (
	version  = "dev"
	buildNum = "local"
	appName  = "cola-regctl"
)

// SetVersionInfo sets version information from the main package.
func SetVersionInfo(ver, build, name string) {
	version = ver
	buildNum = build
	appName = name
}

var (
	// Global flags
	flagURL     string
	flagToken   string
	flagJSON    bool
	flagVerbose bool
	flagTimeout time.Duration
	flagYes     bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "cola-regctl",
	Short: "Command Launcher Registry CLI Client",
}

// Execute configures branding-dependent metadata and flags, then runs the
// root command. Must be called after branding.Init().
func Execute() error {
	rootCmd.Use = appName
	rootCmd.Long = fmt.Sprintf(`%s is a command-line client for managing Command Launcher remote registries.

It provides full CRUD operations for registries, packages, and versions via the REST API.`, appName)

	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", fmt.Sprintf("Server URL (or use %s env var)", branding.URLEnvVar()))
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", fmt.Sprintf("Authentication token: 'user:password' for basic auth or JWT token (or use %s env var)", branding.TokenEnvVar()))
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second, "HTTP request timeout")
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation prompts")

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(versionTemplate())
	return rootCmd.Execute()
}

// versionTemplate returns the version output template
func versionTemplate() string {
	if buildNum != "" && buildNum != "local" {
		return fmt.Sprintf("%s version %s (build %s)\n", appName, version, buildNum)
	}
	return fmt.Sprintf("%s version %s\n", appName, version)
}

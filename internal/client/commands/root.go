package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// Version information - set from main.go via SetVersionInfo
var (
	version     = "dev"
	buildNum    = "local"
	appName     = "cola-regctl"
	appLongName = "Command Launcher Registry CLI"
)

// SetVersionInfo sets version information from main package
func SetVersionInfo(v, b, n, l string) {
	version = v
	buildNum = b
	appName = n
	appLongName = l
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
	Long: `cola-regctl is a command-line client for managing Command Launcher remote registries.

It provides full CRUD operations for registries, packages, and versions via the REST API.`,
}

// Execute executes the root command
func Execute() error {
	// Set version info before executing
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

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "Server URL (or use COLA_REGISTRY_URL env var)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Authentication token in 'user:password' format (or use COLA_REGISTRY_SESSION_TOKEN env var)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second, "HTTP request timeout")
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation prompts")

	// Add subcommands
	// These will be implemented in subsequent tasks
	// rootCmd.AddCommand(loginCmd)
	// rootCmd.AddCommand(logoutCmd)
	// rootCmd.AddCommand(whoamiCmd)
	// rootCmd.AddCommand(registryCmd)
	// rootCmd.AddCommand(packageCmd)
	// rootCmd.AddCommand(versionCmd)
	// rootCmd.AddCommand(completionCmd)
}

// getGlobalFlags returns the global flag values
func getGlobalFlags() (url, token string, jsonOutput, verbose bool, timeout time.Duration, yes bool) {
	return flagURL, flagToken, flagJSON, flagVerbose, flagTimeout, flagYes
}

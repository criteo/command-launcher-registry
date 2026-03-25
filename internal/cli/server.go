package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/branding"
	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/server"
	"github.com/criteo/command-launcher-registry/internal/server/handlers"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

// Exit codes
const (
	ExitCodeOK                  = 0
	ExitCodeInvalidConfig       = 1
	ExitCodeStorageInitFailed   = 2
	ExitCodeServerStartupFailed = 3
)

// Version information - set from main.go via SetVersionInfo
var (
	version  = "dev"
	buildNum = "local"
)

// SetVersionInfo sets version information from the main package.
func SetVersionInfo(ver, build string) {
	version = ver
	buildNum = build
}

var v *viper.Viper

// ServerCmd represents the server command
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the registry HTTP server",
	Long:  `Start the HTTP server that serves Command Launcher registry index and provides REST API for registry management.`,
	RunE:  runServer,
}

// SetupFlags registers all branding-dependent flags on the server command
// and binds them to viper. Must be called after branding.Init() and before
// rootCmd.Execute().
func SetupFlags() {
	v = config.NewViper()

	ev := branding.EnvVar // shorthand

	// CLI flags - these take precedence over environment variables
	// Defaults here match viper defaults so --help shows them; viper uses flag.Changed for precedence.
	ServerCmd.Flags().String("storage-uri", "file://./data/registry.json", fmt.Sprintf("Storage URI (or %s)", ev("STORAGE_URI")))
	ServerCmd.Flags().String("storage-token", "", fmt.Sprintf("Storage authentication token (or %s)", ev("STORAGE_TOKEN")))
	ServerCmd.Flags().Int("port", 8080, fmt.Sprintf("Server port (or %s)", ev("SERVER_PORT")))
	ServerCmd.Flags().String("host", "0.0.0.0", fmt.Sprintf("Bind address (or %s)", ev("SERVER_HOST")))
	ServerCmd.Flags().String("log-level", "info", fmt.Sprintf("Log level: debug|info|warn|error (or %s)", ev("LOGGING_LEVEL")))
	ServerCmd.Flags().String("log-format", "json", fmt.Sprintf("Log format: json|text (or %s)", ev("LOGGING_FORMAT")))
	ServerCmd.Flags().String("auth-type", "none", fmt.Sprintf("Authentication type: none|basic|ldap|custom_jwt (or %s)", ev("AUTH_TYPE")))
	ServerCmd.Flags().String("auth-ldap-server", "", fmt.Sprintf("LDAP server URL, e.g., ldap://ldap.example.com (or %s)", ev("AUTH_LDAP_SERVER")))
	ServerCmd.Flags().Int("auth-ldap-timeout", 30, fmt.Sprintf("LDAP connection timeout in seconds (or %s)", ev("AUTH_LDAP_TIMEOUT")))
	ServerCmd.Flags().String("auth-ldap-bind-dn", "", fmt.Sprintf("LDAP bind DN for service account (or %s)", ev("AUTH_LDAP_BIND_DN")))
	ServerCmd.Flags().String("auth-ldap-user-base-dn", "", fmt.Sprintf("LDAP base DN for user searches (or %s)", ev("AUTH_LDAP_USER_BASE_DN")))
	ServerCmd.Flags().String("auth-custom-jwt-script", "", fmt.Sprintf("Path to JWT validator script (or %s)", ev("AUTH_CUSTOM_JWT_SCRIPT")))
	ServerCmd.Flags().String("auth-custom-jwt-required-group", "", fmt.Sprintf("Required group for authorization (or %s)", ev("AUTH_CUSTOM_JWT_REQUIRED_GROUP")))

	// Bind CLI flags to viper
	v.BindPFlag("storage.uri", ServerCmd.Flags().Lookup("storage-uri"))
	v.BindPFlag("storage.token", ServerCmd.Flags().Lookup("storage-token"))
	v.BindPFlag("server.port", ServerCmd.Flags().Lookup("port"))
	v.BindPFlag("server.host", ServerCmd.Flags().Lookup("host"))
	v.BindPFlag("logging.level", ServerCmd.Flags().Lookup("log-level"))
	v.BindPFlag("logging.format", ServerCmd.Flags().Lookup("log-format"))
	v.BindPFlag("auth.type", ServerCmd.Flags().Lookup("auth-type"))
	v.BindPFlag("auth.ldap.server", ServerCmd.Flags().Lookup("auth-ldap-server"))
	v.BindPFlag("auth.ldap.timeout", ServerCmd.Flags().Lookup("auth-ldap-timeout"))
	v.BindPFlag("auth.ldap.bind_dn", ServerCmd.Flags().Lookup("auth-ldap-bind-dn"))
	v.BindPFlag("auth.ldap.user_base_dn", ServerCmd.Flags().Lookup("auth-ldap-user-base-dn"))
	v.BindPFlag("auth.custom_jwt.script", ServerCmd.Flags().Lookup("auth-custom-jwt-script"))
	v.BindPFlag("auth.custom_jwt.required_group", ServerCmd.Flags().Lookup("auth-custom-jwt-required-group"))
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration (CLI flags > env vars > defaults)
	cfg, err := config.LoadWithViper(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load configuration: %v\n", err)
		os.Exit(ExitCodeInvalidConfig)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid configuration: %v\n", err)
		os.Exit(ExitCodeInvalidConfig)
	}

	// Create logger
	logger := server.NewLogger(cfg.Logging.Level, cfg.Logging.Format)

	// Log effective configuration at startup (with masked token)
	logEffectiveConfig(cfg, logger)

	// Parse storage URI
	storageURI, err := cfg.GetParsedStorageURI()
	if err != nil {
		logger.Error("Failed to parse storage URI",
			"error", err,
			"storage_uri", cfg.Storage.URI)
		os.Exit(ExitCodeInvalidConfig)
	}

	// Initialize storage using factory
	store, err := storage.NewStorage(storageURI, cfg.Storage.Token, logger)
	if err != nil {
		logger.Error("Failed to initialize storage",
			"error", err,
			"storage_uri", cfg.Storage.URI,
			"scheme", storageURI.Scheme)
		os.Exit(ExitCodeStorageInitFailed)
	}

	// Initialize authenticator
	var authenticator auth.Authenticator
	switch cfg.Auth.Type {
	case "none":
		authenticator = auth.NewNoAuth()
		logger.Info("Authentication disabled (auth.type=none)")
	case "basic":
		authenticator, err = auth.NewBasicAuth(cfg.Auth.UsersFile, logger)
		if err != nil {
			logger.Error("Failed to initialize basic auth",
				"error", err,
				"users_file", cfg.Auth.UsersFile)
			os.Exit(ExitCodeStorageInitFailed)
		}
	case "ldap":
		authenticator, err = auth.NewLDAPAuth(cfg.Auth.LDAP, logger)
		if err != nil {
			logger.Error("Failed to initialize LDAP auth",
				"error", err,
				"server", cfg.Auth.LDAP.Server)
			os.Exit(ExitCodeStorageInitFailed)
		}
		logger.Info("LDAP authentication enabled",
			"server", cfg.Auth.LDAP.Server,
			"user_base_dn", cfg.Auth.LDAP.UserBaseDN)
	case "custom_jwt":
		authenticator, err = auth.NewCustomJWTAuth(cfg.Auth.CustomJWT, logger)
		if err != nil {
			logger.Error("Failed to initialize custom JWT auth",
				"error", err)
			os.Exit(ExitCodeStorageInitFailed)
		}
		logger.Info("Custom JWT authentication enabled")
	default:
		logger.Error("Unsupported auth type", "auth_type", cfg.Auth.Type)
		os.Exit(ExitCodeInvalidConfig)
	}

	// Create server
	srv := server.NewServer(cfg, logger, store, authenticator)

	// Create all handlers
	indexHandler := handlers.NewIndexHandler(store, logger)
	registryHandler := handlers.NewRegistryHandler(store, logger)
	packageHandler := handlers.NewPackageHandler(store, logger)
	versionHandler := handlers.NewVersionHandler(store, logger)
	healthHandler := handlers.NewHealthHandler(store, logger)
	metricsHandler := handlers.NewMetricsHandler(logger)
	whoamiHandler := handlers.NewWhoamiHandler(authenticator, logger)

	// Set all handlers
	srv.SetHandlers(server.HandlerSet{
		IndexGet:       indexHandler.GetIndex,
		IndexOptions:   indexHandler.HandleOptions,
		Health:         healthHandler.GetHealth,
		Metrics:        metricsHandler.GetMetrics,
		Whoami:         whoamiHandler.GetWhoami,
		ListRegistries: registryHandler.ListRegistries,
		CreateRegistry: registryHandler.CreateRegistry,
		GetRegistry:    registryHandler.GetRegistry,
		UpdateRegistry: registryHandler.UpdateRegistry,
		DeleteRegistry: registryHandler.DeleteRegistry,
		ListPackages:   packageHandler.ListPackages,
		CreatePackage:  packageHandler.CreatePackage,
		GetPackage:     packageHandler.GetPackage,
		UpdatePackage:  packageHandler.UpdatePackage,
		DeletePackage:  packageHandler.DeletePackage,
		ListVersions:   versionHandler.ListVersions,
		CreateVersion:  versionHandler.CreateVersion,
		GetVersion:     versionHandler.GetVersion,
		DeleteVersion:  versionHandler.DeleteVersion,
	})

	// Start server
	logger.Info("Server ready to accept connections",
		"address", fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port))

	if err := srv.Start(); err != nil {
		logger.Error("Server stopped with error", "error", err)
		os.Exit(ExitCodeServerStartupFailed)
	}

	return nil
}

// logEffectiveConfig logs the effective configuration at startup
func logEffectiveConfig(cfg *config.Config, logger *slog.Logger) {
	tokenDisplay := cfg.MaskToken()
	if tokenDisplay == "" {
		tokenDisplay = "(not set)"
	}

	// Format version string
	versionStr := version
	if buildNum != "" && buildNum != "local" {
		versionStr = fmt.Sprintf("%s (build %s)", version, buildNum)
	}

	logger.Info("Server starting with configuration",
		"version", versionStr,
		"storage_uri", cfg.Storage.URI,
		"storage_token", tokenDisplay,
		"port", cfg.Server.Port,
		"host", cfg.Server.Host,
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format,
		"auth_type", cfg.Auth.Type,
		"auth_users_file", cfg.Auth.UsersFile,
	)
}

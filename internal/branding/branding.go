package branding

import (
	"strings"
	"sync"
)

var (
	mu      sync.RWMutex
	prefix  = "cola"
	appName = "cola-registry"
)

// Init sets the branding from the appName. The prefix is derived by taking
// the part before the first "-" (e.g., "cola-registry" → "cola").
// Must be called before any other branding function.
func Init(name string) {
	mu.Lock()
	defer mu.Unlock()
	if name == "" {
		return
	}
	appName = name
	if idx := strings.Index(name, "-"); idx > 0 {
		prefix = name[:idx]
	} else {
		prefix = name
	}
}

// Prefix returns the raw prefix (e.g., "cola" or "cdt").
func Prefix() string {
	mu.RLock()
	defer mu.RUnlock()
	return prefix
}

// AppName returns the full app name (e.g., "cola-registry").
func AppName() string {
	mu.RLock()
	defer mu.RUnlock()
	return appName
}

// EnvPrefix returns the environment variable prefix for viper (e.g., "COLA_REGISTRY").
func EnvPrefix() string {
	return strings.ToUpper(Prefix()) + "_REGISTRY"
}

// EnvVar returns the full environment variable name for a given config suffix
// (e.g., EnvVar("URL") returns "COLA_REGISTRY_URL").
func EnvVar(suffix string) string {
	return EnvPrefix() + "_" + suffix
}

// URLEnvVar returns the URL environment variable name (e.g., "COLA_REGISTRY_URL").
// Convenience alias for EnvVar("URL").
func URLEnvVar() string {
	return EnvVar("URL")
}

// TokenEnvVar returns the token environment variable name (e.g., "COLA_REGISTRY_TOKEN").
// Convenience alias for EnvVar("TOKEN").
func TokenEnvVar() string {
	return EnvVar("TOKEN")
}

// ConfigDir returns the config directory relative path (e.g., ".config/cola-registry").
func ConfigDir() string {
	return ".config/" + Prefix() + "-registry"
}

// KeychainService returns the keychain/credential manager service name (e.g., "cola-registry").
func KeychainService() string {
	return Prefix() + "-registry"
}

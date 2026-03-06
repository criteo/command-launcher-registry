package auth

import (
	"fmt"
	"os"

	"github.com/criteo/command-launcher-registry/internal/branding"
)

// ResolveToken resolves the authentication token using precedence:
// 1. flagToken (--token flag, if provided)
// 2. Token environment variable (e.g., COLA_REGISTRY_TOKEN)
// 3. Stored credentials
// Returns empty string if no token found
func ResolveToken(flagToken string) (string, error) {
	// Priority 1: CLI flag (highest priority for explicit override)
	if flagToken != "" {
		return flagToken, nil
	}

	// Priority 2: Environment variable
	if envToken := os.Getenv(branding.TokenEnvVar()); envToken != "" {
		return envToken, nil
	}

	// Priority 3: Stored credentials
	storedToken, err := LoadStoredToken()
	if err != nil {
		// If error is "not found", return empty string
		if err == ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to load stored token: %w", err)
	}

	return storedToken, nil
}

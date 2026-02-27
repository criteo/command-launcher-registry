package auth

import (
	"fmt"
	"os"
)

const (
	// TokenEnvVar is the environment variable for authentication token
	TokenEnvVar = "COLA_REGISTRY_TOKEN"
)

// ResolveToken resolves the authentication token using precedence:
// 1. flagToken (--token flag, if provided)
// 2. COLA_REGISTRY_TOKEN environment variable
// 3. Stored credentials
// Returns empty string if no token found
func ResolveToken(flagToken string) (string, error) {
	// Priority 1: CLI flag (highest priority for explicit override)
	if flagToken != "" {
		return flagToken, nil
	}

	// Priority 2: Environment variable
	if envToken := os.Getenv(TokenEnvVar); envToken != "" {
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

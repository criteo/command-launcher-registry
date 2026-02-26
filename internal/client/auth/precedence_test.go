package auth

import (
	"os"
	"testing"
)

func TestResolveToken_FlagPriority(t *testing.T) {
	// Flag token should take highest priority
	flagToken := "flag-token"

	// Set env vars that should be ignored when flag is set
	os.Setenv(TokenEnvVar, "env-token")
	defer os.Unsetenv(TokenEnvVar)

	token, err := ResolveToken(flagToken)
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != flagToken {
		t.Errorf("ResolveToken with flag = %q, expected %q", token, flagToken)
	}
}

func TestResolveToken_NewEnvVar(t *testing.T) {
	// COLA_REGISTRY_TOKEN should be used when flag is empty
	expectedToken := "jwt-token-from-env"
	os.Setenv(TokenEnvVar, expectedToken)
	defer os.Unsetenv(TokenEnvVar)

	token, err := ResolveToken("")
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != expectedToken {
		t.Errorf("ResolveToken with COLA_REGISTRY_TOKEN = %q, expected %q", token, expectedToken)
	}
}

func TestResolveToken_EmptyFlag(t *testing.T) {
	// Empty flag should not take precedence, should fall through to env var
	expectedToken := "env-token"
	os.Setenv(TokenEnvVar, expectedToken)
	defer os.Unsetenv(TokenEnvVar)

	token, err := ResolveToken("")
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != expectedToken {
		t.Errorf("ResolveToken with empty flag = %q, expected %q", token, expectedToken)
	}
}

func TestResolveToken_NoFlagOrEnv(t *testing.T) {
	// When flag and env var are not set, should fall through to stored credentials (or empty if none)
	os.Unsetenv(TokenEnvVar)

	token, err := ResolveToken("")
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	// Token should be either from stored credentials or empty
	// We can't assume no stored credentials exist on the test machine
	// This test verifies that the function completes successfully
	// when flag and env var are not set
	t.Logf("ResolveToken with no flag/env returned: %q", token)
}

func TestEnvironmentVariableNames(t *testing.T) {
	// Verify the environment variable constant is correct
	if TokenEnvVar != "COLA_REGISTRY_TOKEN" {
		t.Errorf("TokenEnvVar = %q, expected COLA_REGISTRY_TOKEN", TokenEnvVar)
	}
}

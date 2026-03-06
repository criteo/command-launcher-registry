package auth

import (
	"os"
	"testing"

	"github.com/criteo/command-launcher-registry/internal/branding"
)

func TestResolveToken_FlagPriority(t *testing.T) {
	// Flag token should take highest priority
	flagToken := "flag-token"

	// Set env vars that should be ignored when flag is set
	os.Setenv(branding.TokenEnvVar(), "env-token")
	defer os.Unsetenv(branding.TokenEnvVar())

	token, err := ResolveToken(flagToken)
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != flagToken {
		t.Errorf("ResolveToken with flag = %q, expected %q", token, flagToken)
	}
}

func TestResolveToken_NewEnvVar(t *testing.T) {
	// Token env var should be used when flag is empty
	expectedToken := "jwt-token-from-env"
	os.Setenv(branding.TokenEnvVar(), expectedToken)
	defer os.Unsetenv(branding.TokenEnvVar())

	token, err := ResolveToken("")
	if err != nil {
		t.Fatalf("ResolveToken failed: %v", err)
	}

	if token != expectedToken {
		t.Errorf("ResolveToken with %s = %q, expected %q", branding.TokenEnvVar(), token, expectedToken)
	}
}

func TestResolveToken_EmptyFlag(t *testing.T) {
	// Empty flag should not take precedence, should fall through to env var
	expectedToken := "env-token"
	os.Setenv(branding.TokenEnvVar(), expectedToken)
	defer os.Unsetenv(branding.TokenEnvVar())

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
	os.Unsetenv(branding.TokenEnvVar())

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
	// Verify the default environment variable name is correct
	if branding.TokenEnvVar() != "COLA_REGISTRY_TOKEN" {
		t.Errorf("branding.TokenEnvVar() = %q, expected COLA_REGISTRY_TOKEN", branding.TokenEnvVar())
	}
}

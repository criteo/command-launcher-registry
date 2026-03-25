package branding

import "testing"

// resetBranding restores default state before each test to avoid order dependency.
func resetBranding() {
	mu.Lock()
	defer mu.Unlock()
	prefix = "cola"
	appName = "cola-registry"
}

func TestInit_ColaRegistry(t *testing.T) {
	resetBranding()
	Init("cola-registry")

	if got := Prefix(); got != "cola" {
		t.Errorf("Prefix() = %q, want %q", got, "cola")
	}
	if got := AppName(); got != "cola-registry" {
		t.Errorf("AppName() = %q, want %q", got, "cola-registry")
	}
	if got := EnvPrefix(); got != "COLA_REGISTRY" {
		t.Errorf("EnvPrefix() = %q, want %q", got, "COLA_REGISTRY")
	}
	if got := URLEnvVar(); got != "COLA_REGISTRY_URL" {
		t.Errorf("URLEnvVar() = %q, want %q", got, "COLA_REGISTRY_URL")
	}
	if got := TokenEnvVar(); got != "COLA_REGISTRY_TOKEN" {
		t.Errorf("TokenEnvVar() = %q, want %q", got, "COLA_REGISTRY_TOKEN")
	}
	if got := ConfigDir(); got != ".config/cola-registry" {
		t.Errorf("ConfigDir() = %q, want %q", got, ".config/cola-registry")
	}
	if got := KeychainService(); got != "cola-registry" {
		t.Errorf("KeychainService() = %q, want %q", got, "cola-registry")
	}
}

func TestInit_ColaRegctl(t *testing.T) {
	resetBranding()
	Init("cola-regctl")

	if got := Prefix(); got != "cola" {
		t.Errorf("Prefix() = %q, want %q", got, "cola")
	}
	if got := EnvPrefix(); got != "COLA_REGISTRY" {
		t.Errorf("EnvPrefix() = %q, want %q", got, "COLA_REGISTRY")
	}
}

func TestInit_CdtRegistry(t *testing.T) {
	resetBranding()
	Init("cdt-registry")

	if got := Prefix(); got != "cdt" {
		t.Errorf("Prefix() = %q, want %q", got, "cdt")
	}
	if got := AppName(); got != "cdt-registry" {
		t.Errorf("AppName() = %q, want %q", got, "cdt-registry")
	}
	if got := EnvPrefix(); got != "CDT_REGISTRY" {
		t.Errorf("EnvPrefix() = %q, want %q", got, "CDT_REGISTRY")
	}
	if got := URLEnvVar(); got != "CDT_REGISTRY_URL" {
		t.Errorf("URLEnvVar() = %q, want %q", got, "CDT_REGISTRY_URL")
	}
	if got := TokenEnvVar(); got != "CDT_REGISTRY_TOKEN" {
		t.Errorf("TokenEnvVar() = %q, want %q", got, "CDT_REGISTRY_TOKEN")
	}
	if got := ConfigDir(); got != ".config/cdt-registry" {
		t.Errorf("ConfigDir() = %q, want %q", got, ".config/cdt-registry")
	}
	if got := KeychainService(); got != "cdt-registry" {
		t.Errorf("KeychainService() = %q, want %q", got, "cdt-registry")
	}
}

func TestInit_CdtRegctl(t *testing.T) {
	resetBranding()
	Init("cdt-regctl")

	if got := Prefix(); got != "cdt" {
		t.Errorf("Prefix() = %q, want %q", got, "cdt")
	}
	if got := EnvPrefix(); got != "CDT_REGISTRY" {
		t.Errorf("EnvPrefix() = %q, want %q", got, "CDT_REGISTRY")
	}
}

func TestInit_EmptyString(t *testing.T) {
	resetBranding()
	Init("cola-registry") // set known state
	Init("")              // should be a no-op

	if got := Prefix(); got != "cola" {
		t.Errorf("Prefix() = %q, want %q after empty Init", got, "cola")
	}
}

package auth

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"

	"github.com/criteo/command-launcher-registry/internal/config"
)

// CustomJWTAuth implements custom JWT authentication via external script
type CustomJWTAuth struct {
	config config.CustomJWTConfig
	logger *slog.Logger
}

// NewCustomJWTAuth creates a new CustomJWT authenticator
func NewCustomJWTAuth(cfg config.CustomJWTConfig, logger *slog.Logger) (*CustomJWTAuth, error) {
	if cfg.Script == "" {
		return nil, fmt.Errorf("custom_jwt script is required")
	}
	if _, err := exec.LookPath(cfg.Script); err != nil {
		return nil, fmt.Errorf("custom_jwt script not found or not executable: %v", err)
	}

	logger.Info("CustomJWT auth initialized",
		"script", cfg.Script,
		"required_group", cfg.RequiredGroup)

	return &CustomJWTAuth{
		config: cfg,
		logger: logger,
	}, nil
}

// Authenticate validates Bearer token using external script
func (a *CustomJWTAuth) Authenticate(r *http.Request) (*User, error) {
	token, err := a.extractBearerToken(r)
	if err != nil {
		return nil, err
	}

	groups, username, err := a.executeScript(token)
	if err != nil {
		return nil, err
	}

	if a.config.RequiredGroup != "" {
		if !a.hasGroup(groups, a.config.RequiredGroup) {
			a.logger.Warn("User is not a member of required group",
				"required_group", a.config.RequiredGroup,
				"source_ip", r.RemoteAddr)
			return nil, ErrForbidden
		}
	}

	a.logger.Debug("CustomJWT authentication successful",
		"username", username,
		"groups_count", len(groups),
		"source_ip", r.RemoteAddr)

	return &User{Username: username, Groups: groups}, nil
}

// Middleware returns HTTP middleware for CustomJWT authentication
func (a *CustomJWTAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := a.Authenticate(r)
			if err != nil {
				if errors.Is(err, ErrForbidden) {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				if errors.Is(err, ErrUnauthorized) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractBearerToken extracts the Bearer token from Authorization header
func (a *CustomJWTAuth) extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrUnauthorized
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrUnauthorized
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrUnauthorized
	}

	return token, nil
}

// executeScript runs the JWT validation script and returns groups and username
func (a *CustomJWTAuth) executeScript(token string) ([]string, string, error) {
	cmd := exec.Command(a.config.Script, token)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			a.logger.Warn("Script failed",
				"exit_code", exitError.ExitCode(),
				"stderr", strings.TrimSpace(stderr.String()))
			return nil, "", ErrForbidden
		}
		a.logger.Error("Failed to execute script",
			"error", err)
		return nil, "", ErrInternal
	}

	groups, username := a.parseOutput(stdout.String())
	if username == "" {
		a.logger.Warn("Script returned empty username")
		return nil, "", ErrForbidden
	}
	return groups, username, nil
}

// parseOutput parses the script output to extract username and groups
// Expected format: "username:<value>" on first line, then one group per line
func (a *CustomJWTAuth) parseOutput(output string) ([]string, string) {
	var groups []string
	var username string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// First line with username: prefix sets the username
		if username == "" && strings.HasPrefix(line, "username:") {
			username = strings.TrimSpace(strings.TrimPrefix(line, "username:"))
			continue
		}

		groups = append(groups, line)
	}

	return groups, username
}

// hasGroup checks if the user has the required group
func (a *CustomJWTAuth) hasGroup(groups []string, requiredGroup string) bool {
	for _, group := range groups {
		if group == requiredGroup {
			return true
		}
	}
	return false
}

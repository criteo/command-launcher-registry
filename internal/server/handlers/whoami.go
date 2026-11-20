package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/principal"
)

// WhoamiHandler handles whoami requests
type WhoamiHandler struct {
	authenticator auth.Authenticator
	serverAdmins  []string
	logger        *slog.Logger
}

// NewWhoamiHandler creates a new whoami handler. serverAdmins is the
// server-admin list in force: nil or empty when role enforcement is off, so
// that nobody is reported as a server admin.
func NewWhoamiHandler(authenticator auth.Authenticator, serverAdmins []string, logger *slog.Logger) *WhoamiHandler {
	return &WhoamiHandler{
		authenticator: authenticator,
		serverAdmins:  serverAdmins,
		logger:        logger,
	}
}

// WhoamiResponse represents the whoami response
type WhoamiResponse struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups,omitempty"`
	// ServerAdmin is true when role enforcement is on and the caller's
	// username or one of its groups is a server admin.
	ServerAdmin bool `json:"server_admin"`
}

// GetWhoami handles GET /api/v1/whoami
// This endpoint requires authentication and returns the authenticated username
func (h *WhoamiHandler) GetWhoami(w http.ResponseWriter, r *http.Request) {
	// Authenticate the request
	user, err := h.authenticator.Authenticate(r)
	if err != nil {
		h.logger.Debug("Authentication failed for whoami", "error", err)
		w.Header().Set("WWW-Authenticate", `Basic realm="COLA Registry"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Return the authenticated user info
	response := WhoamiResponse{
		Username:    user.Username,
		Groups:      user.Groups,
		ServerAdmin: principal.Matches(h.serverAdmins, user.Username, user.Groups),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode whoami response", "error", err)
	}
}

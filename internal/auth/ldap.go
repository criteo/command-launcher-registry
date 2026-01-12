package auth

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/go-ldap/ldap/v3"
)

// LDAPAuth implements LDAP authentication
type LDAPAuth struct {
	config config.LDAPConfig
	logger *slog.Logger
}

// NewLDAPAuth creates a new LDAP authenticator
func NewLDAPAuth(cfg config.LDAPConfig, logger *slog.Logger) (*LDAPAuth, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("LDAP server is required")
	}
	if cfg.BindDN == "" {
		return nil, fmt.Errorf("LDAP bind DN is required")
	}
	if cfg.UserBaseDN == "" {
		return nil, fmt.Errorf("LDAP user base DN is required")
	}
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(uid=%s)"
	}
	if cfg.GroupFilter == "" {
		cfg.GroupFilter = "(member=%s)"
	}

	return &LDAPAuth{config: cfg, logger: logger}, nil
}

// Authenticate validates LDAP credentials from HTTP Basic Auth
func (l *LDAPAuth) Authenticate(r *http.Request) (*User, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, fmt.Errorf("missing credentials")
	}

	if username == "" || password == "" {
		return nil, fmt.Errorf("invalid credentials")
	}

	conn, err := ldap.DialURL(l.config.Server)
	if err != nil {
		l.logger.Error("Failed to connect to LDAP server",
			"error", err,
			"server", l.config.Server)
		return nil, fmt.Errorf("authentication failed")
	}
	defer conn.Close()

	err = conn.Bind(l.config.BindDN, l.config.BindPassword)
	if err != nil {
		l.logger.Error("Failed to bind to LDAP server",
			"error", err,
			"bind_dn", l.config.BindDN)
		return nil, fmt.Errorf("authentication failed")
	}

	searchRequest := ldap.NewSearchRequest(
		l.config.UserBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(l.config.UserFilter, ldap.EscapeFilter(username)),
		[]string{"dn"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		l.logger.Error("Failed to search for user",
			"error", err,
			"username", username)
		return nil, fmt.Errorf("authentication failed")
	}

	if len(sr.Entries) == 0 {
		l.logger.Warn("User not found in LDAP",
			"username", username,
			"source_ip", r.RemoteAddr)
		return nil, fmt.Errorf("authentication failed")
	}

	if len(sr.Entries) > 1 {
		l.logger.Warn("Multiple entries found for user",
			"username", username)
		return nil, fmt.Errorf("authentication failed")
	}

	userDN := sr.Entries[0].DN

	err = conn.Bind(userDN, password)
	if err != nil {
		l.logger.Warn("Authentication failed: invalid password",
			"username", username,
			"source_ip", r.RemoteAddr)
		return nil, fmt.Errorf("authentication failed")
	}

	if l.config.RequiredGroup != "" {
		err = conn.Bind(l.config.BindDN, l.config.BindPassword)
		if err != nil {
			l.logger.Error("Failed to rebind for group check",
				"error", err)
			return nil, fmt.Errorf("authentication failed")
		}

		groupSearchRequest := ldap.NewSearchRequest(
			l.config.GroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf("(&%s(cn=%s))", fmt.Sprintf(l.config.GroupFilter, ldap.EscapeFilter(userDN)), ldap.EscapeFilter(l.getGroupCN(l.config.RequiredGroup))),
			[]string{"cn"},
			nil,
		)

		gsr, err := conn.Search(groupSearchRequest)
		if err != nil {
			l.logger.Error("Failed to search for group membership",
				"error", err)
			return nil, fmt.Errorf("authentication failed")
		}

		if len(gsr.Entries) == 0 {
			l.logger.Warn("User is not a member of required group",
				"username", username,
				"required_group", l.config.RequiredGroup,
				"source_ip", r.RemoteAddr)
			return nil, fmt.Errorf("forbidden")
		}
	}

	l.logger.Debug("LDAP authentication successful",
		"username", username,
		"source_ip", r.RemoteAddr)

	return &User{Username: username}, nil
}

// Middleware returns HTTP middleware for LDAP authentication
func (l *LDAPAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := l.Authenticate(r)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="COLA Registry"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			_ = user

			next.ServeHTTP(w, r)
		})
	}
}

func (l *LDAPAuth) getGroupCN(groupDN string) string {
	entry, err := ldap.ParseDN(groupDN)
	if err != nil {
		return groupDN
	}

	for _, rdn := range entry.RDNs {
		for _, attr := range rdn.Attributes {
			if attr.Type == "cn" || attr.Type == "CN" {
				return attr.Value
			}
		}
	}

	return groupDN
}

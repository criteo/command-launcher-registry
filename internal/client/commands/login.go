package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/branding"
	"github.com/criteo/command-launcher-registry/internal/client"
	"github.com/criteo/command-launcher-registry/internal/client/auth"
	"github.com/criteo/command-launcher-registry/internal/client/config"
	"github.com/criteo/command-launcher-registry/internal/client/errors"
	"github.com/criteo/command-launcher-registry/internal/client/output"
	"github.com/criteo/command-launcher-registry/internal/client/prompts"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [server-url]",
	Short: "Authenticate with a registry server",
	Long: `Authenticate with a registry server and store credentials securely.

Server URL can be provided as an argument or via the URL environment variable.
If both are provided, the argument takes precedence.

Authentication methods:
- Username and password (basic authentication)
- JWT token (bearer authentication)

Credentials are stored:
- macOS: Token in Keychain, URL in config file
- Windows: Token in Credential Manager, URL in config file
- Linux: Both in config file with 0600 permissions

Only one server's credentials are stored at a time. Logging into a new server
replaces existing credentials.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runLogin,
}

func runLogin(cmd *cobra.Command, args []string) {
	var serverURL string

	// Resolve server URL: argument takes precedence over environment variable
	if len(args) > 0 {
		serverURL = args[0]
	} else {
		// Try to get URL from environment variable
		var err error
		serverURL, err = config.ResolveURL("")
		if err != nil {
			errors.ExitWithCode(errors.ExitInvalidArguments, fmt.Sprintf("no server URL specified. Provide server URL as argument or set %s environment variable", branding.URLEnvVar()))
		}
	}

	// Normalize URL (remove trailing slash)
	serverURL = config.NormalizeURL(serverURL)

	// Prompt for authentication method
	authMethod, err := prompts.PromptAuthMethod()
	if err != nil {
		errors.ExitWithError(err, "failed to read authentication method")
	}

	var token string
	var username string

	switch authMethod {
	case prompts.AuthMethodUsernamePassword:
		// Prompt for username and password
		username, err = prompts.PromptUsername()
		if err != nil {
			errors.ExitWithError(err, "failed to read username")
		}

		password, err := prompts.PromptPassword()
		if err != nil {
			errors.ExitWithError(err, "failed to read password")
		}

		// Validate inputs
		if username == "" {
			errors.ExitWithCode(errors.ExitInvalidArguments, "username cannot be empty")
		}
		if password == "" {
			errors.ExitWithCode(errors.ExitInvalidArguments, "password cannot be empty")
		}

		// Format token as "username:password" for basic auth
		token = fmt.Sprintf("%s:%s", username, password)

	case prompts.AuthMethodJWT:
		// Prompt for JWT token
		jwtToken, err := prompts.PromptJWTToken()
		if err != nil {
			errors.ExitWithError(err, "failed to read JWT token")
		}

		// Validate JWT token format
		if !auth.IsJWTToken(jwtToken) {
			errors.ExitWithCode(errors.ExitInvalidArguments, "invalid JWT token format (expected three dot-separated parts)")
		}

		token = jwtToken
		username = "(from JWT token)" // Placeholder - will get actual username from whoami response

	default:
		errors.ExitWithCode(errors.ExitInvalidArguments, "invalid authentication method")
	}

	// Prepare token for client (JWT as-is, basic auth base64-encoded)
	clientToken := auth.EncodeToken(token)

	// Test authentication by calling /api/v1/whoami
	c := client.NewClient(serverURL, clientToken, flagTimeout, flagVerbose)
	resp, err := c.Get("/api/v1/whoami")
	if err != nil {
		errors.ExitWithError(err, "failed to connect to server")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		errors.ExitWithCode(errors.ExitAuthError, "authentication failed: invalid credentials or token")
	}

	if resp.StatusCode != http.StatusOK {
		errors.HandleHTTPError(resp.StatusCode, fmt.Sprintf("server returned status %d", resp.StatusCode))
	}

	// For JWT tokens, extract username from whoami response
	if auth.IsJWTToken(token) {
		if body, err := io.ReadAll(resp.Body); err == nil {
			var whoamiResp map[string]any
			if json.Unmarshal(body, &whoamiResp) == nil {
				if user, ok := whoamiResp["username"].(string); ok && user != "" {
					username = user
				}
			}
		}
	}

	// Authentication successful - store credentials
	if err := auth.SaveCredentials(serverURL, token); err != nil {
		errors.ExitWithError(err, "failed to save credentials")
	}

	if flagJSON {
		output.OutputJSON(map[string]string{
			"server": serverURL,
			"user":   username,
		}, nil)
	} else {
		output.PrintSuccess(fmt.Sprintf("Logged in to %s as %s", serverURL, username))
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/criteo/command-launcher-registry/internal/client"
	"github.com/criteo/command-launcher-registry/internal/client/auth"
	"github.com/criteo/command-launcher-registry/internal/client/config"
	"github.com/criteo/command-launcher-registry/internal/client/errors"
	"github.com/criteo/command-launcher-registry/internal/client/output"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show authentication status and server information",
	Long: `Check authentication status by calling the server's /api/v1/whoami endpoint.

Resolves server URL and credentials using normal precedence:
- URL: --url flag > COLA_REGISTRY_URL env var > stored URL
- Token: --token flag > COLA_REGISTRY_TOKEN env var > stored token`,
	Args: cobra.NoArgs,
	Run:  runWhoami,
}

func runWhoami(cmd *cobra.Command, args []string) {
	// Resolve URL
	serverURL, err := config.ResolveURL(flagURL)
	if err != nil {
		errors.ExitWithCode(errors.ExitInvalidArguments, err.Error())
	}

	// Resolve token
	token, err := auth.ResolveToken(flagToken)
	if err != nil {
		errors.ExitWithError(err, "failed to resolve authentication token")
	}

	// Check authentication by calling /api/v1/whoami
	// Prepare token for client (JWT as-is, basic auth base64-encoded)
	clientToken := auth.EncodeToken(token)

	c := client.NewClient(serverURL, clientToken, flagTimeout, flagVerbose)
	resp, err := c.Get("/api/v1/whoami")
	if err != nil {
		errors.ExitWithError(err, "failed to connect to server")
	}
	defer resp.Body.Close()

	authenticated := resp.StatusCode == http.StatusOK
	username := "(username unknown)"

	// Extract username from server response
	if authenticated {
		if body, err := io.ReadAll(resp.Body); err == nil {
			var whoamiResp map[string]any
			if json.Unmarshal(body, &whoamiResp) == nil {
				if user, ok := whoamiResp["username"].(string); ok && user != "" {
					username = user
				}
			}
		}
	}

	if flagJSON {
		output.OutputJSON(map[string]interface{}{
			"server":        serverURL,
			"authenticated": authenticated,
			"username":      username,
		}, nil)
	} else {
		if authenticated {
			output.PrintSuccess(fmt.Sprintf("Authenticated to %s as %s", serverURL, username))
		} else if resp.StatusCode == http.StatusUnauthorized {
			output.PrintError(fmt.Sprintf("Not authenticated to %s", serverURL))
			fmt.Println("Run 'cola-regctl login' to authenticate")
		} else {
			output.PrintError(fmt.Sprintf("Server returned status %d", resp.StatusCode))
		}
	}

	if !authenticated {
		errors.ExitWithCode(errors.ExitAuthError, "")
	}
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

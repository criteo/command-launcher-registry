package commands

import (
	"fmt"

	"github.com/criteo/command-launcher-registry/internal/client/errors"
	"github.com/criteo/command-launcher-registry/internal/principal"
)

// parsePrincipalFlag turns the values of a repeatable --admin/--maintainer
// flag into a principal list: each value may hold several principals
// separated by commas and/or whitespace. A flag given only empty values is
// an invalid argument, not a request to clear the list (use --clear-* for that).
func parsePrincipalFlag(flag string, values []string) []string {
	principals := principal.ParseList(values...)
	if len(principals) == 0 {
		errors.ExitWithCode(errors.ExitInvalidArguments,
			fmt.Sprintf("--%s needs at least one user or group", flag))
	}
	return principals
}

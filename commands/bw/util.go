package main

import (
	"os"
	"os/user"

	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

// DisplayName for the user
func DisplayName() string {
	u := systemx.CurrentUserOrDefault(user.User{Username: stringsx.DefaultIfBlank(os.Getenv("BEARDED_WOOKIE_DEPLOYER"), "unknown")})
	return stringsx.DefaultIfBlank(u.Name, u.Username)
}

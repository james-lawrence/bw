package shell

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextVariableSubstitution(t *testing.T) {
	ctx1 := Context{
		User:      user.User{Username: "username", Uid: "1000", HomeDir: "/home/username"},
		Hostname:  "MyHost",
		MachineID: "MachineID",
		Domain:    "Domain",
		FQDN:      "FQDN",
	}

	for _, c := range []struct {
		name     string
		ctx      Context
		example  string
		expected string
	}{
		{"example - hostname", ctx1, `echo %H`, fmt.Sprintf("echo %s", ctx1.Hostname)},
		{"example - machine id", ctx1, `echo %m`, fmt.Sprintf("echo %s", ctx1.MachineID)},
		{"example - domain name", ctx1, `echo %d`, fmt.Sprintf("echo %s", ctx1.Domain)},
		{"example - FQDN", ctx1, `echo %f`, fmt.Sprintf("echo %s", ctx1.FQDN)},
		{"example - username", ctx1, `echo %u`, fmt.Sprintf("echo %s", ctx1.User.Username)},
		{"example - user id", ctx1, `echo %U`, fmt.Sprintf("echo %s", ctx1.User.Uid)},
		{"example - home directory", ctx1, `echo %h`, fmt.Sprintf("echo %s", ctx1.User.HomeDir)},
		{"example - percent", ctx1, `echo %%`, "echo %"},
		{"example 1", ctx1, `echo ${HELLO} ${WORLD} %m`, fmt.Sprintf("echo ${HELLO} ${WORLD} %s", ctx1.MachineID)},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, c.ctx.variableSubst(c.example))
		})
	}
}

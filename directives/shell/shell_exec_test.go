package shell

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/user"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShellExecute(t *testing.T) {
	ctx1 := Context{
		Shell:     os.Getenv("SHELL"),
		User:      user.User{Username: "username", Uid: "1000", HomeDir: "/home/username"},
		Hostname:  "MyHost",
		MachineID: "MachineID",
		Domain:    "Domain",
		FQDN:      "FQDN",
		Environ: append(
			os.Environ(),
			"FOO=BAR",
		),
		output: io.Discard,
	}

	for _, tc := range []struct {
		name   string
		ctx    Context
		err    error
		output string
		c      Exec
	}{
		{"times out", ctx1, errors.New("signal: killed"), "", Exec{Command: "sleep 0.5", Timeout: 200 * time.Millisecond}},
		{"complex command", ctx1, nil, "BAZ\n", Exec{Command: "echo ${FOO} | sed 's/BAR/BAZ/'", Timeout: 1 * time.Second}},
		{"allow failures", ctx1, nil, "command failed, ignoring false %m exit status 1\n", Exec{Command: "false %m", Lenient: true, Timeout: 1 * time.Second}},
		{"additional environment variables per command", ctx1, nil, "HELLO BAR", Exec{Command: "printf \"HELLO ${BAZZ}\"", Timeout: 1 * time.Second, Environ: "BAZZ=${FOO}"}},
		{"retries", ctx1, errors.New("signal: killed"), "command failed after 5 attempts sleep 0.01 signal: killed\n", Exec{Command: "sleep 0.01", Timeout: 2 * time.Millisecond, Retries: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx
			buf := bytes.NewBufferString("")
			ctx.output = buf

			if tc.err != nil {
				require.EqualError(t, tc.c.execute(context.Background(), ctx), tc.err.Error())
			} else {
				require.NoError(t, tc.c.execute(context.Background(), ctx))
			}

			require.Equal(t, tc.output, buf.String())
		})
	}
}

package shell

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/user"
	"time"

	"github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Shell", func() {
	ginkgo.Context("Execute", func() {
		var ctx1 = Context{
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
		ginkgo.DescribeTable("Execute functions", func(ctx Context, err error, output string, c Exec) {
			buf := bytes.NewBufferString("")
			ctx.output = buf

			if err != nil {
				Expect(c.execute(context.Background(), ctx)).To(MatchError(err.Error()))
			} else {
				Expect(c.execute(context.Background(), ctx)).ToNot(HaveOccurred())
			}

			Expect(buf.String()).To(Equal(output))
		},
			ginkgo.Entry("times out", ctx1, errors.New("signal: killed"), "", Exec{Command: "sleep 0.5", Timeout: 200 * time.Millisecond}),
			ginkgo.Entry("complex command", ctx1, nil, "BAZ\n", Exec{Command: "echo ${FOO} | sed 's/BAR/BAZ/'", Timeout: 1 * time.Second}),
			ginkgo.Entry("allow failures", ctx1, nil, "command failed, ignoring false %m exit status 1\n", Exec{Command: "false %m", Lenient: true, Timeout: 1 * time.Second}),
			ginkgo.Entry("additional environment variables per command", ctx1, nil, "HELLO BAR", Exec{Command: "printf \"HELLO ${BAZZ}\"", Timeout: 1 * time.Second, Environ: "BAZZ=${FOO}"}),
			ginkgo.Entry("retries", ctx1, errors.New("signal: killed"), "command failed after 5 attempts sleep 0.01 signal: killed\n", Exec{Command: "sleep 0.01", Timeout: 2 * time.Millisecond, Retries: 5}),
		)
	})
})

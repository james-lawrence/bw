package shell

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
			output: ioutil.Discard,
		}
		DescribeTable("Execute functions", func(ctx Context, err error, c Exec) {
			if err != nil {
				Expect(c.execute(context.Background(), ctx)).To(MatchError(err.Error()))
			} else {
				Expect(c.execute(context.Background(), ctx)).ToNot(HaveOccurred())
			}
		},
			Entry("times out", ctx1, errors.New("signal: killed"), Exec{Command: "sleep 0.5", Timeout: 200 * time.Millisecond}),
			Entry("complex command", ctx1, nil, Exec{Command: "echo ${FOO} | sed 's/BAR/BAZ/'", Timeout: 1 * time.Second}),
			Entry("allow failures", ctx1, nil, Exec{Command: "false %m", Lenient: true, Timeout: 1 * time.Second}),
		)
	})
})

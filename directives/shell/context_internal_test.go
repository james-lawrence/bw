package shell

import (
	"fmt"
	"os/user"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Context", func() {
	var ctx1 = Context{
		User:      user.User{Username: "username", Uid: "1000", HomeDir: "/home/username"},
		Hostname:  "MyHost",
		MachineID: "MachineID",
		Domain:    "Domain",
		FQDN:      "FQDN",
	}

	DescribeTable("variable substitution",
		func(ctx Context, example string, expected string) {
			Expect(ctx.variableSubst(example)).To(Equal(expected))
		},
		Entry("example - hostname", ctx1, `echo %H`, fmt.Sprintf("echo %s", ctx1.Hostname)),
		Entry("example - machine id", ctx1, `echo %m`, fmt.Sprintf("echo %s", ctx1.MachineID)),
		Entry("example - domain name", ctx1, `echo %d`, fmt.Sprintf("echo %s", ctx1.Domain)),
		Entry("example - FQDN", ctx1, `echo %f`, fmt.Sprintf("echo %s", ctx1.FQDN)),
		Entry("example - username", ctx1, `echo %u`, fmt.Sprintf("echo %s", ctx1.User.Username)),
		Entry("example - user id", ctx1, `echo %U`, fmt.Sprintf("echo %s", ctx1.User.Uid)),
		Entry("example - home directory", ctx1, `echo %h`, fmt.Sprintf("echo %s", ctx1.User.HomeDir)),
		Entry("example - percent", ctx1, `echo %%`, "echo %"),
		Entry("example 1", ctx1, `echo ${HELLO} ${WORLD} %m`, fmt.Sprintf("echo ${HELLO} ${WORLD} %s", ctx1.MachineID)),
	)
})

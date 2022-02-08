package shell

import (
	"fmt"
	"os/user"

	"github.com/onsi/ginkgo/v2"
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

	ginkgo.DescribeTable("variable substitution",
		func(ctx Context, example string, expected string) {
			Expect(ctx.variableSubst(example)).To(Equal(expected))
		},
		ginkgo.Entry("example - hostname", ctx1, `echo %H`, fmt.Sprintf("echo %s", ctx1.Hostname)),
		ginkgo.Entry("example - machine id", ctx1, `echo %m`, fmt.Sprintf("echo %s", ctx1.MachineID)),
		ginkgo.Entry("example - domain name", ctx1, `echo %d`, fmt.Sprintf("echo %s", ctx1.Domain)),
		ginkgo.Entry("example - FQDN", ctx1, `echo %f`, fmt.Sprintf("echo %s", ctx1.FQDN)),
		ginkgo.Entry("example - username", ctx1, `echo %u`, fmt.Sprintf("echo %s", ctx1.User.Username)),
		ginkgo.Entry("example - user id", ctx1, `echo %U`, fmt.Sprintf("echo %s", ctx1.User.Uid)),
		ginkgo.Entry("example - home directory", ctx1, `echo %h`, fmt.Sprintf("echo %s", ctx1.User.HomeDir)),
		ginkgo.Entry("example - percent", ctx1, `echo %%`, "echo %"),
		ginkgo.Entry("example 1", ctx1, `echo ${HELLO} ${WORLD} %m`, fmt.Sprintf("echo ${HELLO} ${WORLD} %s", ctx1.MachineID)),
	)
})

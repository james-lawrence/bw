package shell

import (
	"os/user"
	"time"

	g "github.com/onsi/ginkgo"
	gt "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var example1 = Context{
	WorkDirectory: "WORK DIRECTORY",
	User:          user.User{Username: "USERNAME", Uid: "USERID", HomeDir: "HOMEDIR"},
	Hostname:      "HOSTNAME",
	MachineID:     "MACHINEID",
	Domain:        "DOMAIN",
	FQDN:          "FQDN",
	Environ:       []string{"FOO=BAR"},
	dir:           "ROOT",
	timeout:       time.Second,
}

var _ = g.Describe("Context", func() {
	gt.DescribeTable("variable substitution",
		func(ctx Context, input, expected string) {
			result := ctx.variableSubst(input)
			Expect(result).To(Equal(expected))
		},
		gt.Entry("basic environment", example1, "%H %m %d %f %u %U %h %bwroot %bwcwd %%", "HOSTNAME MACHINEID DOMAIN FQDN USERNAME USERID HOMEDIR ROOT WORK DIRECTORY %"),
		gt.Entry("properly escape", example1, "git show -s --format=%ct-%%h", "git show -s --format=%ct-%h"),
	)

	gt.DescribeTable("environment variables",
		func(ctx Context, expected ...string) {
			result := ctx.environmentSubst()
			Expect(result).To(Equal(expected))
		},
		gt.Entry(
			"basic environment",
			example1,
			"FOO=BAR",
			"BW_ENVIRONMENT_HOST=HOSTNAME",
			"BW_ENVIRONMENT_MACHINE_ID=MACHINEID",
			"BW_ENVIRONMENT_DOMAIN=DOMAIN",
			"BW_ENVIRONMENT_FQDN=FQDN",
			"BW_ENVIRONMENT_USERNAME=USERNAME",
			"BW_ENVIRONMENT_USERID=USERID",
			"BW_ENVIRONMENT_USERHOME=HOMEDIR",
			"BW_ENVIRONMENT_ROOT=ROOT",
			"BW_ENVIRONMENT_WORK_DIRECTORY=WORK DIRECTORY",
		),
	)
})

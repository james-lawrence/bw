package shell

import (
	"os/user"
	"time"

	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var example1 = Context{
	deploymentID:  "deployment.id",
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
	g.DescribeTable("variable substitution",
		func(ctx Context, input, expected string) {
			result := ctx.variableSubst(input)
			Expect(result).To(Equal(expected))
		},
		g.Entry("basic environment", example1, "%H %m %d %f %u %U %h %bw.archive.directory% %bwcwd %bw.deploy.id% %%", "HOSTNAME MACHINEID DOMAIN FQDN USERNAME USERID HOMEDIR ROOT WORK DIRECTORY deployment.id %"),
		g.Entry("properly escape", example1, "git show -s --format=%ct-%%h", "git show -s --format=%ct-%h"),
		g.Entry("deprecated bwroot usage", example1, "%bwroot", "ROOT"),
	)

	g.DescribeTable("environment variables",
		func(ctx Context, expected ...string) {
			result := ctx.environmentSubst()
			Expect(result).To(Equal(expected))
		},
		g.Entry(
			"basic environment",
			example1,
			"FOO=BAR",
			"BW_ENVIRONMENT_DEPLOY_ID=deployment.id",
			"BW_ENVIRONMENT_HOST=HOSTNAME",
			"BW_ENVIRONMENT_MACHINE_ID=MACHINEID",
			"BW_ENVIRONMENT_DOMAIN=DOMAIN",
			"BW_ENVIRONMENT_FQDN=FQDN",
			"BW_ENVIRONMENT_USERNAME=USERNAME",
			"BW_ENVIRONMENT_USERID=USERID",
			"BW_ENVIRONMENT_USERHOME=HOMEDIR",
			"BW_ENVIRONMENT_ROOT=ROOT",
			"BW_ENVIRONMENT_ARCHIVE_DIRECTORY=ROOT",
			"BW_ENVIRONMENT_WORK_DIRECTORY=WORK DIRECTORY",
			"BW_ENVIRONMENT_TEMP_DIRECTORY=",
		),
	)
})

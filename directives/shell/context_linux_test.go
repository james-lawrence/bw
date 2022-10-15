package shell

import (
	"os/user"
	"time"

	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var example1 = Context{
	deploymentID:  "00000000-0000-0000-0000-000000000000",
	WorkDirectory: "WORKDIR",
	User:          user.User{Username: "USERNAME", Uid: "USERID", HomeDir: "HOMEDIR"},
	Hostname:      "HOSTNAME",
	MachineID:     "MACHINEID",
	Domain:        "DOMAIN",
	FQDN:          "FQDN",
	Environ:       []string{"FOO=BAR"},
	dir:           "ARCHIVEDIR",
	tmpdir:        "TEMPDIR",
	cachedir:      "CACHEDIR",
	timeout:       time.Second,
	commit:        "d9662c91bf8c4591ae311d853404ae8e",
}

var _ = g.Describe("Context", func() {
	g.DescribeTable("variable substitution",
		func(ctx Context, input, expected string) {
			result := ctx.variableSubst(input)
			Expect(result).To(Equal(expected))
		},
		g.Entry("basic environment deprecated", example1, "%H %m %d %f %u %U %h %bwroot %bwcwd %bw.deploy.id% %%", "HOSTNAME MACHINEID DOMAIN FQDN USERNAME USERID HOMEDIR ARCHIVEDIR WORKDIR 00000000-0000-0000-0000-000000000000 %"),
		g.Entry("basic environment", example1, "%H %m %d %f %u %U %h %bw.work.directory% %bw.archive.directory% %bw.cache.directory% %bw.temp.directory% %bw.deploy.id% %%", "HOSTNAME MACHINEID DOMAIN FQDN USERNAME USERID HOMEDIR WORKDIR ARCHIVEDIR CACHEDIR TEMPDIR 00000000-0000-0000-0000-000000000000 %"),
		g.Entry("properly escape", example1, "git show -s --format=%ct-%%h", "git show -s --format=%ct-%h"),
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
			"BW_ENVIRONMENT_DEPLOY_ID=00000000-0000-0000-0000-000000000000",
			"BW_ENVIRONMENT_DEPLOY_COMMIT=d9662c91bf8c4591ae311d853404ae8e",
			"BW_ENVIRONMENT_HOST=HOSTNAME",
			"BW_ENVIRONMENT_MACHINE_ID=MACHINEID",
			"BW_ENVIRONMENT_DOMAIN=DOMAIN",
			"BW_ENVIRONMENT_FQDN=FQDN",
			"BW_ENVIRONMENT_USERNAME=USERNAME",
			"BW_ENVIRONMENT_USERID=USERID",
			"BW_ENVIRONMENT_USERHOME=HOMEDIR",
			"BW_ENVIRONMENT_ROOT=ARCHIVEDIR",
			"BW_ENVIRONMENT_ARCHIVE_DIRECTORY=ARCHIVEDIR",
			"BW_ENVIRONMENT_WORK_DIRECTORY=WORKDIR",
			"BW_ENVIRONMENT_TEMP_DIRECTORY=TEMPDIR",
			"BW_ENVIRONMENT_CACHE_DIRECTORY=CACHEDIR",
		),
	)
})

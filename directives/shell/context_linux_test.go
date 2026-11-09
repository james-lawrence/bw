package shell

import (
	"os/user"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	latestdir:     "LATESTDIR",
	tmpdir:        "TEMPDIR",
	cachedir:      "CACHEDIR",
	timeout:       time.Second,
	commit:        "d9662c91bf8c4591ae311d853404ae8e",
	bwconfigdir:   "CONFIGDIR",
	initiator:     "DEPLOYER",
}

func TestContextLinuxVariableSubstitution(t *testing.T) {
	for _, c := range []struct {
		name     string
		ctx      Context
		input    string
		expected string
	}{
		{"basic environment deprecated", example1, "%H %m %d %f %u %U %h %bwroot %bwcwd %bw.deploy.id% %%", "HOSTNAME MACHINEID DOMAIN FQDN USERNAME USERID HOMEDIR ARCHIVEDIR WORKDIR 00000000-0000-0000-0000-000000000000 %"},
		{"basic environment", example1, "%H %m %d %f %u %U %h %bw.work.directory% %bw.archive.directory% %bw.cache.directory% %bw.temp.directory% %bw.deploy.id% %%", "HOSTNAME MACHINEID DOMAIN FQDN USERNAME USERID HOMEDIR WORKDIR ARCHIVEDIR CACHEDIR TEMPDIR 00000000-0000-0000-0000-000000000000 %"},
		{"archive latest", example1, "%bw.archive.latest%", "LATESTDIR"},
		{"properly escape", example1, "git show -s --format=%ct-%%h", "git show -s --format=%ct-%h"},
		{"deploy information", example1, "%bw.deploy.id% %bw.deploy.user.name% %bw.deploy.commit%", "00000000-0000-0000-0000-000000000000 DEPLOYER d9662c91bf8c4591ae311d853404ae8e"},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, c.ctx.variableSubst(c.input))
		})
	}
}

func TestContextLinuxEnvironmentVariables(t *testing.T) {
	t.Run("basic environment", func(t *testing.T) {
		require.Equal(t, []string{
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
			"BW_ENVIRONMENT_ARCHIVE_LATEST=LATESTDIR",
			"BW_ENVIRONMENT_WORK_DIRECTORY=WORKDIR",
			"BW_ENVIRONMENT_TEMP_DIRECTORY=TEMPDIR",
			"BW_ENVIRONMENT_CACHE_DIRECTORY=CACHEDIR",
			"BW_ENVIRONMENT_CONFIG_DIRECTORY=CONFIGDIR",
		}, example1.environmentSubst())
	})
}

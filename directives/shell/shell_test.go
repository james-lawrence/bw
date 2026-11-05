package shell_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/james-lawrence/bw/directives/shell"

	"github.com/stretchr/testify/require"
)

const yaml1 = `
- command: "echo hello world"
- command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"
- command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"
- command: "echo %H %h %m"
  lenient: true
  timeout: 10m
  retries: 10
  directory: /tmp
  environ: |
    FOO=BAR
    BIZZ=${BAZZ}
`

const yaml2 = `
- command: "echo hello world"
- command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"
- command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"
- command: "echo %H %h %m"
  lenient: true
  timeout: 10m
  retries: -1
  directory: "%bw.work.directory%/hello"
  environ: |
    FOO=BAR
    BIZZ=${BAZZ}
    CIN="%bw.work.directory%"
`

func TestParseYAML(t *testing.T) {
	for _, c := range []struct {
		name     string
		example  string
		expected []Exec
	}{
		{
			"example 1", yaml1,
			[]Exec{
				{Command: "echo hello world"},
				{Command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"},
				{Command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"},
				{
					Command: "echo %H %h %m",
					Lenient: true,
					Timeout: 10 * time.Minute,
					Retries: 10,
					WorkDir: "/tmp",
					Environ: "FOO=BAR\nBIZZ=${BAZZ}\n",
				},
			},
		},
		{
			"example 2", yaml2,
			[]Exec{
				{Command: "echo hello world"},
				{Command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"},
				{Command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"},
				{
					Command: "echo %H %h %m",
					Lenient: true,
					Timeout: 10 * time.Minute,
					Retries: -1,
					WorkDir: "%bw.work.directory%/hello",
					Environ: "FOO=BAR\nBIZZ=${BAZZ}\nCIN=\"%bw.work.directory%\"\n",
				},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			actual, err := ParseYAML(strings.NewReader(c.example))
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

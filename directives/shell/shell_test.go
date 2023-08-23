package shell_test

import (
	"strings"
	"time"

	. "github.com/james-lawrence/bw/directives/shell"

	"github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Shell", func() {
	const yaml1 = `
- command: "echo hello world"
- command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"
- command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"
- command: "echo %H %h %m"
  lenient: true
  timeout: 10m
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
  directory: "%bw.work.directory%/hello"
  environ: |
    FOO=BAR
    BIZZ=${BAZZ}
`
	ginkgo.DescribeTable("ParseYAML",
		func(example string, expected ...Exec) {
			Expect(ParseYAML(strings.NewReader(example))).To(Equal(expected))
		},
		ginkgo.Entry(
			"example 1", yaml1,
			Exec{Command: "echo hello world"},
			Exec{Command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"},
			Exec{Command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"},
			Exec{
				Command: "echo %H %h %m",
				Lenient: true,
				Timeout: 10 * time.Minute,
				WorkDir: "/tmp",
				Environ: "FOO=BAR\nBIZZ=${BAZZ}\n",
			},
		),
		ginkgo.Entry(
			"example 2", yaml2,
			Exec{Command: "echo hello world"},
			Exec{Command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"},
			Exec{Command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"},
			Exec{
				Command: "echo %H %h %m",
				Lenient: true,
				Timeout: 10 * time.Minute,
				WorkDir: "%bw.work.directory%/hello",
				Environ: "FOO=BAR\nBIZZ=${BAZZ}\n",
			},
		),
	)
})

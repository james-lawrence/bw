package shell_test

import (
	"strings"
	"time"

	. "bitbucket.org/jatone/bearded-wookie/directives/shell"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
`
	DescribeTable("ParseYAML",
		func(example string, expected ...Exec) {
			Expect(ParseYAML(strings.NewReader(example))).To(Equal(expected))
		},
		Entry(
			"example", yaml1,
			Exec{Command: "echo hello world"},
			Exec{Command: "cat .filesystems/rsyslog/00_templates.conf | envsubst '${LOGGLY_TOKEN} ${DOMAIN_NAME} ${ENVIRONMENT_NAME}' >> .filesystems/rsyslog/00_templates.conf"},
			Exec{Command: "aws s3 cp ${GAPPS_CERTIFICATE_S3_PATH} .credentials/google.json"},
			Exec{
				Command: "echo %H %h %m",
				Lenient: true,
				Timeout: 10 * time.Minute,
			},
		),
	)
})

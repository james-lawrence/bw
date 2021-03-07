package httputilx_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHttputilx(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(filepath.Join(os.Getenv("JUNIT_DIR"), "junit_routes.xml"))
	RunSpecsWithDefaultAndCustomReporters(t, "httputilx Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})

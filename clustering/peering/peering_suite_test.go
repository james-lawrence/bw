package peering_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestPeering(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(filepath.Join(os.Getenv("JUNIT_DIR"), "junit_peering.xml"))
	RunSpecsWithDefaultAndCustomReporters(t, "Peering Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})

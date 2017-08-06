package rendezvous_test

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

func TestRendezvous(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(filepath.Join(os.Getenv("JUNIT_DIR"), "junit_rendezvous.xml"))
	RunSpecsWithDefaultAndCustomReporters(t, "Rendezvous Suite", []Reporter{junitReporter})
}

func TestMailman(t *testing.T) {
	RegisterFailHandler(Fail)
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})

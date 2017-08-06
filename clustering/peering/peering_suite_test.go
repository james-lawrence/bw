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

var (
// TX     *sql.Tx
// DB     *sql.DB
// dbname string
)

func TestPeering(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(filepath.Join(os.Getenv("JUNIT_DIR"), "junit_peering.xml"))
	RunSpecsWithDefaultAndCustomReporters(t, "Peering Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})

// var _ = BeforeEach(func() {
// 	var err error
// 	dbname, DB = NewPostgresql(TemplateDatabaseName)
// 	Expect(err).ToNot(HaveOccurred())
// 	TX, err = DB.Begin()
// 	Expect(err).ToNot(HaveOccurred())
// })
//
// var _ = AfterEach(func() {
// 	Expect(TX.Rollback()).ToNot(HaveOccurred())
// 	Expect(DB.Close()).ToNot(HaveOccurred())
// 	DestroyPostgresql(TemplateDatabaseName, dbname)
// })

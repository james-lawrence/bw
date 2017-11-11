package agentutil_test

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	. "github.com/james-lawrence/bw/agentutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agentutil", func() {
	Context("directory cleanup", func() {
		var (
			root string
		)

		makedir := func(root string) string {
			d, err := ioutil.TempDir(root, "")
			Expect(err).ToNot(HaveOccurred())
			return d
		}

		BeforeEach(func() {
			var (
				err error
			)
			root, err = ioutil.TempDir(".", "dircleanup-test")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			defer os.RemoveAll(root)
		})

		Describe("KeepNewestN", func() {
			DescribeTable("should keep the n newest directories",
				func(m, n int) {
					if n > m {
						n = m
					}
					dirs := make([]string, 0, n)
					for i := 0; i < m; i++ {
						dirs = append(dirs, makedir(root))
						time.Sleep(30 * time.Millisecond)
					}

					MaybeClean(KeepNewestN(n))(Dirs(root))

					for _, d := range dirs[:len(dirs)-n] {
						Expect(d).ToNot(BeAnExistingFile())
					}
					for _, d := range dirs[len(dirs)-n:] {
						Expect(d).To(BeAnExistingFile())
					}
				},
				Entry("example 1", 5, 1),
				Entry("example 2", 5, 2),
				Entry("example 3", 5, 3),
				Entry("example 4", 5, 4),
				Entry("example 5", 5, 5),
				Entry("example 6", 5, 6),
			)
			It("should keep the n newest directories", func() {
				d1 := makedir(root)
				time.Sleep(30 * time.Millisecond)
				d2 := makedir(root)
				time.Sleep(30 * time.Millisecond)
				d3 := makedir(root)
				log.Println(d1, d2, d3)
				MaybeClean(KeepNewestN(1))(Dirs(root))
				Expect(d1).ToNot(BeAnExistingFile())
				Expect(d2).ToNot(BeAnExistingFile())
				Expect(d3).To(BeAnExistingFile())
			})
		})

		Describe("KeepOldestN", func() {
			DescribeTable("should keep the n oldest directories",
				func(m, n int) {
					if n > m {
						n = m
					}
					dirs := make([]string, 0, n)
					for i := 0; i < m; i++ {
						dirs = append(dirs, makedir(root))
						// necessary to ensure order of dirs by creation date.
						time.Sleep(30 * time.Millisecond)
					}

					MaybeClean(KeepOldestN(n))(Dirs(root))

					for _, d := range dirs[:n] {
						Expect(d).To(BeAnExistingFile())
					}
					for _, d := range dirs[n:] {
						Expect(d).ToNot(BeAnExistingFile())
					}
				},
				Entry("example 1", 5, 1),
				Entry("example 2", 5, 2),
				Entry("example 3", 5, 3),
				Entry("example 4", 5, 4),
				Entry("example 5", 5, 5),
				Entry("example 6", 5, 6),
			)
			It("should keep the n oldest directories", func() {
				d1 := makedir(root)
				time.Sleep(30 * time.Millisecond)
				d2 := makedir(root)
				time.Sleep(30 * time.Millisecond)
				d3 := makedir(root)

				MaybeClean(KeepOldestN(1))(Dirs(root))
				Expect(d1).To(BeAnExistingFile())
				Expect(d2).ToNot(BeAnExistingFile())
				Expect(d3).ToNot(BeAnExistingFile())
			})
		})
	})
})

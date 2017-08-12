package packages

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	DescribeTable("parsing",
		func(example string, expected Package) {
			Expect(parse(example)).To(Equal(expected))
		},
		Entry("example 1", `package`, Package{Name: "package"}),
		Entry("example 2", `package;`, Package{Name: "package"}),
		Entry("example 3", `;version;`, Package{Version: "version"}),
		Entry("example 4", `'package'`, Package{Name: "package"}),
		Entry("example 5", `'package';`, Package{Name: "package"}),
		Entry("example 6", `package;version`, Package{Name: "package", Version: "version"}),
		Entry("example 7", `package;version;`, Package{Name: "package", Version: "version"}),
		Entry("example 8", `package;version;`, Package{Name: "package", Version: "version"}),
		Entry("example 9", `package;version;arch`, Package{Name: "package", Version: "version", Architecture: "arch"}),
		Entry("example 10", `package;version;arch;repo`, Package{Name: "package", Version: "version", Architecture: "arch", Repository: "repo"}),
		Entry("example 11", `'package';'version';'arch';'repo'`, Package{Name: "package", Version: "version", Architecture: "arch", Repository: "repo"}),
	)
})

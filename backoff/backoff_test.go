package backoff

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func testBackoff(attempts int, s Strategy, expected ...time.Duration) {
	for i := 0; i < attempts; i++ {
		Expect(s.Backoff(i)).To(Equal(expected[i]))
	}
}

var _ = Describe("Backoff", func() {
	DescribeTable("Explicit",
		testBackoff,
		Entry("more attempts than delays", 5, Explicit(1*time.Second, 2*time.Second, 3*time.Second), 1*time.Second, 2*time.Second, 3*time.Second, 1*time.Second, 2*time.Second),
	)
	DescribeTable("Exponential",
		testBackoff,
		Entry("should double each time", 5, Exponential(1*time.Second), 1*time.Second, 2*time.Second, 4*time.Second, 8*time.Second, 16*time.Second),
	)
	DescribeTable("Constant",
		testBackoff,
		Entry("should remain constant", 5, Constant(1*time.Second), 1*time.Second, 1*time.Second, 1*time.Second, 1*time.Second, 1*time.Second),
	)
})

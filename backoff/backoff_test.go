package backoff

import (
	"math"
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

func expectedDurationTest(attempt int, s Strategy, expected time.Duration) {
	Expect(s.Backoff(attempt)).To(Equal(expected))
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

	DescribeTable("Exponential Backoff",
		expectedDurationTest,
		Entry("attempt 0", 0, Exponential(1*time.Second), time.Duration(1*time.Second)),
		Entry("attempt 1", 1, Exponential(1*time.Second), time.Duration(2*time.Second)),
		Entry("attempt 2", 2, Exponential(1*time.Second), time.Duration(4*time.Second)),
		Entry("attempt 3", 3, Exponential(1*time.Second), time.Duration(8*time.Second)),
		Entry("with scaling - attempt 0", 0, Exponential(500*time.Millisecond), time.Duration(500*time.Millisecond)),
		Entry("with scaling - attempt 1", 1, Exponential(500*time.Millisecond), time.Duration(1*time.Second)),
		Entry("with scaling - attempt 2", 2, Exponential(500*time.Millisecond), time.Duration(2*time.Second)),
		Entry("with scaling - attempt 3", 3, Exponential(500*time.Millisecond), time.Duration(4*time.Second)),
		Entry("max attempt value", math.MaxInt64, Exponential(1*time.Second), time.Duration(math.MaxInt64)),
	)
})

package agent_test

import (
	"bytes"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var epoch = time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)

var _ = Describe("Cursor", func() {
	segment := uuid.Must(uuid.FromString(journal.Segment(epoch)[:36]))

	It("should round trip the segment and offset", func() {
		c := agent.NewCursor(segment, 123456789)
		Expect(c.Segment()).To(Equal(segment))
		Expect(c.Offset()).To(Equal(int64(123456789)))
	})

	It("should be the segment id at offset zero and remain a valid v7", func() {
		c := agent.NewCursor(segment, 0)
		Expect(uuid.UUID(c)).To(Equal(segment))
		Expect(uuid.UUID(c).Version()).To(Equal(uuid.V7))
		Expect(uuid.UUID(c).Variant()).To(Equal(uuid.VariantRFC9562))
	})

	It("should keep the version and variant when holding a maximum offset", func() {
		c := agent.NewCursor(segment, 1<<62-1)
		Expect(c.Offset()).To(Equal(int64(1<<62 - 1)))
		Expect(uuid.UUID(c).Version()).To(Equal(uuid.V7))
		Expect(uuid.UUID(c).Variant()).To(Equal(uuid.VariantRFC9562))
		Expect(c.Segment()).To(Equal(segment))
	})

	It("should order by segment then offset", func() {
		next := uuid.Must(uuid.FromString(journal.Segment(epoch.Add(15 * time.Second))[:36]))
		a, b, c := agent.NewCursor(segment, 5), agent.NewCursor(segment, 900), agent.NewCursor(next, 1)
		Expect(bytes.Compare(a[:], b[:])).To(Equal(-1))
		Expect(bytes.Compare(b[:], c[:])).To(Equal(-1))
	})
})

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

var _ = Describe("Cursor from messages", func() {
	It("should be the start of the window the message belongs to", func() {
		m := agent.LogInfo(agent.NewPeer("node"), "message")
		c := agent.CursorOf(m)
		Expect(c).To(Equal(agent.WindowCursor(agent.MessageTime(m))))
		Expect(c.Offset()).To(BeZero())
		Expect(agent.MessageTime(m).Sub(time.UnixMilli(agent.MessageTime(m).UnixMilli()))).To(BeZero())
	})

	It("should place v4 messages by their timestamp", func() {
		m := agent.LogInfo(agent.NewPeer("node"), "message")
		m.Id = uuid.Must(uuid.NewV4()).String()
		m.Ts = epoch.Add(7 * time.Second).Unix()
		Expect(agent.CursorOf(m)).To(Equal(agent.WindowCursor(epoch)))
	})

	It("should be zero for messages without a time", func() {
		Expect(agent.CursorOf(&agent.Message{})).To(Equal(agent.Cursor{}))
	})

	It("should round trip through bytes and treat anything else as zero", func() {
		c := agent.NewCursor(agent.WindowCursor(epoch).Segment(), 42)
		Expect(agent.CursorFromBytes(c.Bytes())).To(Equal(c))
		Expect(agent.Cursor{}.Bytes()).To(BeNil())
		Expect(agent.CursorFromBytes(nil)).To(Equal(agent.Cursor{}))
		Expect(agent.CursorFromBytes([]byte{1, 2, 3})).To(Equal(agent.Cursor{}))
	})
})

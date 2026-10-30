package agentutil

import (
	"bytes"

	"github.com/james-lawrence/bw/agent"
)

// resume tracks the messages delivered from the newest window so that a watch resumed from its
// cursor, which starts by replaying that whole window, doesn't deliver them twice.
type resume struct {
	cursor agent.Cursor        // start of the newest window delivered, the zero cursor watches live messages.
	seen   map[string]struct{} // ids delivered from that window.
}

// accept reports whether the message should be delivered, recording it.
func (t *resume) accept(m *agent.Message) bool {
	c := agent.CursorOf(m)
	switch {
	case c == (agent.Cursor{}):
		return true // no time to place it in a window, nothing to deduplicate against.
	case bytes.Compare(c[:], t.cursor[:]) > 0:
		t.cursor, t.seen = c, make(map[string]struct{}) // newer window.
	case c != t.cursor:
		return true // older window, a replay never starts before the newest one.
	}

	if _, dup := t.seen[m.GetId()]; dup {
		return false
	}

	t.seen[m.GetId()] = struct{}{}
	return true
}

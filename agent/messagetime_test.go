package agent_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
)

func TestMessageTime(t *testing.T) {
	at := time.Date(2026, 10, 30, 16, 16, 32, 123456789, time.UTC)

	for name, tc := range map[string]struct {
		id       string
		ts       int64
		expected time.Time
	}{
		"v7 uses the embedded timestamp": {
			id:       uuid.Must(uuid.NewV7AtTime(at)).String(),
			ts:       at.Add(time.Hour).UnixNano(),
			expected: at.Truncate(time.Millisecond),
		},
		"v7 ignores the seconds interpretation of ts": {
			id:       uuid.Must(uuid.NewV7AtTime(at)).String(),
			expected: at.Truncate(time.Millisecond),
		},
		"v4 uses ts in seconds": {
			id:       uuid.Must(uuid.NewV4()).String(),
			ts:       at.Unix(),
			expected: at.Truncate(time.Second),
		},
		"v4 without ts is zero": {
			id: uuid.Must(uuid.NewV4()).String(),
		},
		"no id and no ts is zero": {},
	} {
		t.Run(name, func(t *testing.T) {
			got := agent.MessageTime(&agent.Message{Id: tc.id, Ts: tc.ts})
			if !got.Equal(tc.expected) {
				t.Fatalf("expected %v got %v", tc.expected, got)
			}
		})
	}
}

func TestNewMessagesCarryTheirOwnTime(t *testing.T) {
	before := time.Now().Add(-time.Second)
	m := agent.LogEvent(agent.NewPeer("node"), "message")
	if ts := agent.MessageTime(m); ts.Before(before) || ts.After(time.Now().Add(time.Second)) {
		t.Fatalf("unexpected time %v", ts)
	}
	if uuid.FromStringOrNil(m.Id).Version() != uuid.V7 {
		t.Fatalf("expected a v7 id got %s", m.Id)
	}
	if time.Unix(0, m.Ts).Before(before) {
		t.Fatalf("expected ts in nanoseconds got %d", m.Ts)
	}
}

package agentutil

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
)

func at(text string, ts time.Time) *agent.Message {
	m := agent.LogInfo(agent.NewPeer("node"), text)
	m.Id = uuid.Must(uuid.NewV7AtTime(ts)).String()
	m.Ts = ts.UnixNano()
	return m
}

func TestResumeDeduplicatesTheReplayedWindow(t *testing.T) {
	epoch := time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)
	m1, m2 := at("1", epoch), at("2", epoch.Add(time.Second))
	m3 := at("3", epoch.Add(2*time.Second))
	next := at("4", epoch.Add(20*time.Second))

	var r resume
	for _, m := range []*agent.Message{m1, m2} {
		if !r.accept(m) {
			t.Fatalf("first delivery of %s was dropped", m.GetLog().GetLog())
		}
	}

	if r.cursor != agent.WindowCursor(epoch) {
		t.Fatalf("expected the cursor to be the start of the newest window")
	}

	// the replay starts at the beginning of the window.
	for _, m := range []*agent.Message{m1, m2} {
		if r.accept(m) {
			t.Fatalf("replayed message %s was delivered again", m.GetLog().GetLog())
		}
	}

	if !r.accept(m3) || !r.accept(next) {
		t.Fatal("new messages must be delivered")
	}

	if r.cursor != agent.WindowCursor(epoch.Add(20*time.Second)) {
		t.Fatal("expected the cursor to advance to the new window")
	}

	if !r.accept(at("old", epoch)) {
		t.Fatal("a message from an older window can't be a replay")
	}
}

func TestResumeAcceptsMessagesWithoutATime(t *testing.T) {
	var r resume
	m := agent.LogInfo(agent.NewPeer("node"), "untimed")
	m.Id, m.Ts = "", 0
	if !r.accept(m) || !r.accept(m) {
		t.Fatal("messages without a time are never deduplicated")
	}
}

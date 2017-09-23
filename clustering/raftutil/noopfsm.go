package raftutil

import (
	"io"
	"log"
	"sync"

	"github.com/hashicorp/raft"
)

type noopFSM struct {
	sync.Mutex
}

func (m *noopFSM) Apply(alog *raft.Log) interface{} {
	m.Lock()
	defer m.Unlock()
	log.Println("applying", alog.Type, alog.Term, alog.Index)
	return 0
}

func (m *noopFSM) Snapshot() (raft.FSMSnapshot, error) {
	m.Lock()
	defer m.Unlock()
	return &noopSnapshot{}, nil
}

func (m *noopFSM) Restore(inp io.ReadCloser) error {
	m.Lock()
	defer m.Unlock()
	defer inp.Close()
	return nil
}

type noopSnapshot struct{}

func (m *noopSnapshot) Persist(sink raft.SnapshotSink) error {
	sink.Close()
	return nil
}

func (m *noopSnapshot) Release() {
}

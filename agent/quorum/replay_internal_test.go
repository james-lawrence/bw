package quorum

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/jsonl"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// replays recorded messages, see `actl history --format-jsonl`, through the write ahead log to make sure
// the history of the quorum doesn't regress. add captures as .xfixtures/*.jsonl, they are all replayed.

func loadMessages(t *testing.T, path string) []*agent.Message {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var (
		msgs []*agent.Message
		d    = jsonl.NewDecoder(bytes.NewReader(raw))
	)

	for {
		m := &agent.Message{}
		if err = d.Decode(m); errors.Is(err, io.EOF) {
			return msgs
		}
		require.NoError(t, err)
		msgs = append(msgs, m)
	}
}

// replay applies the messages to the log in order. duplicates are rejected by the log, anything
// else is a failure.
func replay(t *testing.T, w *WAL, msgs []*agent.Message) {
	t.Helper()

	for _, m := range msgs {
		data, err := proto.Marshal(m)
		require.NoError(t, err)

		if _, err = w.ApplyLog(&raft.Log{Type: raft.LogCommand, Data: data}); !agent.IsDuplicate(err) {
			require.NoError(t, err)
		}
	}
}

// firsts returns the ids of the messages in order, each id once.
func firsts(msgs []*agent.Message) (ids []string) {
	seen := map[string]bool{}
	for _, m := range msgs {
		if !seen[m.Id] {
			seen[m.Id] = true
			ids = append(ids, m.Id)
		}
	}
	return ids
}

func ids(msgs []*agent.Message) (ids []string) {
	for _, m := range msgs {
		ids = append(ids, m.Id)
	}
	return ids
}

func TestReplayJSONL(t *testing.T) {
	files, err := filepath.Glob(".xfixtures/*.jsonl")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		msgs := loadMessages(t, file)
		require.NotEmpty(t, msgs, file)

		// the capture is only interesting when it is not unique already.
		t.Run(filepath.Base(file), func(t *testing.T) {
			expected := firsts(msgs)

			t.Run("records each message once, in order", func(t *testing.T) {
				h := NewHistory()
				w := NewWAL(NewTranscoder(h))
				replay(t, &w, msgs)
				require.Equal(t, expected, ids(h.Snapshot()))
			})

			t.Run("replaying again records nothing", func(t *testing.T) {
				h := NewHistory()
				w := NewWAL(NewTranscoder(h))
				replay(t, &w, msgs)
				replay(t, &w, msgs)
				require.Equal(t, expected, ids(h.Snapshot()))
			})

			t.Run("a restored snapshot rebuilds the history from the snapshot", func(t *testing.T) {
				h := NewHistory()
				w := NewWAL(NewTranscoder(h))
				replay(t, &w, msgs)

				// snapshots keep the messages from the last deploy command onward.
				unique := []*agent.Message{}
				seen := map[string]bool{}
				for _, m := range msgs {
					if !seen[m.Id] {
						seen[m.Id] = true
						unique = append(unique, m)
					}
				}
				last := 0
				for i, m := range unique {
					if m.Type == agent.Message_DeployCommandEvent {
						last = i
					}
				}

				var buff bytes.Buffer
				require.NoError(t, snapshotRestore(w, &buff))
				require.Equal(t, ids(unique[last:]), ids(h.Snapshot()))
			})
		})
	}
}

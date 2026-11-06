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
	"github.com/james-lawrence/bw/agent/journal"
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

// newJournaledWAL builds a write ahead log recording to a journal, the same stack the quorum uses.
func newJournaledWAL(t *testing.T) (*journal.Journal, WAL) {
	t.Helper()

	j, err := journal.New(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { j.Close() })

	return j, NewWAL(NewTranscoder(NewObserver(j)))
}

// latestHistory the history of the latest deploy the messages should produce: each message once, without
// heartbeats, from the last begin.
func latestHistory(msgs []*agent.Message) (result []*agent.Message) {
	seen := map[string]bool{}
	for _, m := range msgs {
		if seen[m.Id] || m.Type == agent.Message_DeployHeartbeat {
			continue
		}
		seen[m.Id] = true

		if m.Type == agent.Message_DeployCommandEvent && m.GetDeployCommand().GetCommand() == agent.DeployCommand_Begin {
			result = result[:0]
		}

		result = append(result, m)
	}

	return result
}

func TestReplayJSONL(t *testing.T) {
	files, err := filepath.Glob(".xfixtures/*.jsonl")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	history := func(t *testing.T, j *journal.Journal) []*agent.Message {
		t.Helper()
		state, err := scanLatest(t.Context(), j)
		require.NoError(t, err)
		return state.history
	}

	for _, file := range files {
		msgs := loadMessages(t, file)
		require.NotEmpty(t, msgs, file)

		// the capture is only interesting when it is not unique already.
		t.Run(filepath.Base(file), func(t *testing.T) {
			expected := ids(latestHistory(msgs))

			t.Run("records each message once, in order", func(t *testing.T) {
				j, w := newJournaledWAL(t)
				replay(t, &w, msgs)
				require.Equal(t, expected, ids(history(t, j)))
			})

			t.Run("replaying again records nothing", func(t *testing.T) {
				j, w := newJournaledWAL(t)
				replay(t, &w, msgs)
				replay(t, &w, msgs)
				require.Equal(t, expected, ids(history(t, j)))
			})

			t.Run("a restored snapshot rebuilds the history from the snapshot", func(t *testing.T) {
				j, w := newJournaledWAL(t)
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
				require.Equal(t, ids(latestHistory(unique[last:])), ids(history(t, j)))
			})
		})
	}
}

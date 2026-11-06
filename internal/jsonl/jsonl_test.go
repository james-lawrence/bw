package jsonl_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/jsonl"
	"github.com/stretchr/testify/require"
)

func TestEncodeOneLinePerMessage(t *testing.T) {
	var out bytes.Buffer
	local := agent.NewPeerZero()
	require.NoError(t, jsonl.NewEncoder(&out).Encode(agent.LogInfo(local, "one\ntwo"), agent.LogInfo(local, "three")))

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	require.Len(t, lines, 2)
	require.Contains(t, lines[0], "one\\ntwo")
	require.Contains(t, lines[1], "three")
}

func TestDecodeRoundTrip(t *testing.T) {
	var out bytes.Buffer
	local := agent.NewPeerZero()
	in := []*agent.Message{agent.LogInfo(local, "one\ntwo"), agent.LogInfo(local, "three")}
	require.NoError(t, jsonl.NewEncoder(&out).Encode(in[0], in[1]))
	out.WriteString("\n") // trailing blank lines are skipped.

	d := jsonl.NewDecoder(&out)
	for _, expected := range in {
		var m agent.Message
		require.NoError(t, d.Decode(&m))
		require.Equal(t, expected.Id, m.Id)
		require.Equal(t, expected.GetLog().Log, m.GetLog().Log)
	}

	var m agent.Message
	require.ErrorIs(t, d.Decode(&m), io.EOF)
}

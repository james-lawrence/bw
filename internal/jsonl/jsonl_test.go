package jsonl_test

import (
	"bytes"
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

package agent_test

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

func TestDecodePeerID(t *testing.T) {
	t.Run("decode old base32 id", func(t *testing.T) {
		c := agent.DecodePeerID("59063fd62fc42d1a711768bc21a053802641593bed61d36346c58f5fc325aa16")
		require.Equal(t, "59063fd6-2fc4-2d1a-7117-68bc21a05380", c)
	})
}

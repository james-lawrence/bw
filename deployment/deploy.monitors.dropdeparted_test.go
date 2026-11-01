package deployment

import (
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

func TestDropDeparted(t *testing.T) {
	var (
		stays    = newPending(agent.NewPeer("stays"), time.Second)
		departed = newPending(agent.NewPeer("departed"), time.Second)
	)

	t.Run("keeps everything when no node departed", func(t *testing.T) {
		require.Equal(t, []*pending{stays, departed}, dropDeparted(nil, stays, departed))
	})

	t.Run("releases the nodes that departed", func(t *testing.T) {
		remaining := dropDeparted(map[string]*agent.Peer{"departed": departed.Peer}, stays, departed)

		require.Equal(t, []*pending{stays}, remaining)

		_, open := <-departed.done
		require.False(t, open)

		select {
		case <-stays.done:
			t.Fatal("expected the remaining node to stay pending")
		default:
		}
	})
}

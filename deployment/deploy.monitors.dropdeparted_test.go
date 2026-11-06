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
		require.Equal(t, []*pending{stays, departed}, dropDeparted(nil, func(*agent.Peer) { t.Fatal("nothing departed") }, stays, departed))
	})

	t.Run("releases the nodes that departed", func(t *testing.T) {
		remaining := dropDeparted(map[string]*agent.Peer{"departed": departed.Peer}, func(*agent.Peer) {}, stays, departed)

		require.Equal(t, []*pending{stays}, remaining)

		_, open := <-departed.done
		require.False(t, open)

		select {
		case <-stays.done:
			t.Fatal("expected the remaining node to stay pending")
		default:
		}
	})

	t.Run("acknowledges each departed node and does not fail it", func(t *testing.T) {
		var acknowledged []*agent.Peer
		d := newPending(agent.NewPeer("departed"), time.Second)

		dropDeparted(map[string]*agent.Peer{"departed": d.Peer}, func(p *agent.Peer) {
			acknowledged = append(acknowledged, p)
		}, stays, d)

		require.Equal(t, []*agent.Peer{d.Peer}, acknowledged)

		// closed without an error, the deploy treats the node as no longer part of it rather than as failed.
		err, open := <-d.done
		require.False(t, open)
		require.NoError(t, err)
	})
}

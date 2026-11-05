package raftutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPassiveStateStable(t *testing.T) {
	t.Run("should not modify the current state", func(t *testing.T) {
		p := passive{
			failures: 10,
			protocol: &Protocol{},
			sgroup:   &sync.WaitGroup{},
		}
		updated := p.stable()
		require.Equal(t, uint64(10), p.failures)
		require.Equal(t, uint64(0), updated.next.(passive).failures)
	})
}

func TestPassiveStateUnstable(t *testing.T) {
	t.Run("should not modify the current state", func(t *testing.T) {
		p := passive{
			failures: 10,
			protocol: &Protocol{},
			sgroup:   &sync.WaitGroup{},
		}
		updated := p.unstable()
		require.Equal(t, uint64(10), p.failures)
		require.Equal(t, uint64(11), updated.next.(passive).failures)
	})
}

package agentcmd_test

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"
)

// output captures what is printed with pterm, restoring it when the test completes.
type output struct {
	m   sync.Mutex
	buf bytes.Buffer
}

func (t *output) Write(p []byte) (int, error) {
	t.m.Lock()
	defer t.m.Unlock()
	return t.buf.Write(p)
}

func (t *output) String() string {
	t.m.Lock()
	defer t.m.Unlock()
	return t.buf.String()
}

func capture(t *testing.T) *output {
	t.Helper()

	out := &output{}
	pterm.SetDefaultOutput(out)
	t.Cleanup(func() { pterm.SetDefaultOutput(os.Stdout) })

	return out
}

func TestNodes(t *testing.T) {
	boot(t)
	out := capture(t)
	require.NoError(t, actl(t).Run("nodes", agenttestutil.Environment, "--insecure"))

	rendered := out.String()
	require.Contains(t, rendered, "Leader    :")
	require.Contains(t, rendered, "address")
}

// watch prints the nodes when it starts, and again only when they change.
func TestWatchPrintsNodesOnChange(t *testing.T) {
	boot(t)
	out := capture(t)

	h := actl(t)
	result := make(chan error, 1)
	go func() { result <- h.Run("watch", agenttestutil.Environment, "--insecure", "--refresh=100ms") }()

	require.Eventually(t, func() bool { return strings.Contains(out.String(), "Leader    :") }, 10*time.Second, 50*time.Millisecond)

	// plenty of refreshes, nothing changed.
	time.Sleep(time.Second)
	require.Equal(t, 1, strings.Count(out.String(), "Leader    :"), "unchanged nodes were printed again:\n%s", out.String())

	select {
	case err := <-result:
		t.Fatalf("watch returned early: %v", err)
	default:
	}
}

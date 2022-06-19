package ux

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"google.golang.org/grpc"
)

type dialer interface {
	Defaults(...grpc.DialOption) []grpc.DialOption
}

// gathers up failures
type failure struct {
	failures map[string]*agent.Message
	cState
}

func (t failure) Consume(m *agent.Message) consumer {
	switch m.Type {
	case agent.Message_DeployCommandEvent:
		t.logs()
		t.cState.printDeployCommand(m)
		return nil // done.
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		switch d.Stage {
		case agent.Deploy_Failed:
			digest := md5.Sum([]byte(d.Error))
			t.failures[hex.EncodeToString(digest[:])] = m
			return failure{failures: t.failures, cState: t.cState}
		}
	}

	return t
}

func (t failure) logs() {
	for _, m := range t.failures {
		t.cState.FailureDisplay.Display(t.cState, m)
	}
}

type failureDisplay interface {
	Display(cState, *agent.Message)
}

// FailureDisplayNoop - ignores failures
type FailureDisplayNoop struct{}

// Display does nothing
func (t FailureDisplayNoop) Display(cState, *agent.Message) {}

// NewFailureDisplayPrint ...
func NewFailureDisplayPrint(local *agent.Peer, d dialers.Defaults) FailureDisplayPrint {
	return FailureDisplayPrint{
		local: local,
		n:     new(int64),
		d:     d,
	}
}

// FailureDisplayPrint prints the logs for each unique error encountered
type FailureDisplayPrint struct {
	local *agent.Peer
	n     *int64
	d     dialers.Defaults
}

// Display prints the logs for each message
func (t FailureDisplayPrint) Display(s cState, m *agent.Message) {
	var (
		err    error
		client agent.DeployConn
		conn   *grpc.ClientConn
	)

	if m.Peer == nil {
		log.Println("unexpected nil peer skipping", spew.Sdump(m))
		return
	}

	if m.Peer.Name == t.local.Name {
		return
	}

	if atomic.LoadInt64(t.n) > 0 {
		os.Stderr.WriteString("\n\n")
	}

	d := dialers.NewDirect(agent.RPCAddress(m.Peer), t.d.Defaults()...)
	b, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()

	if conn, err = d.DialContext(b); err != nil {
		log.Println("unable to dial failed peer", err, "\n", spew.Sdump(m))
		return
	}
	defer conn.Close()

	client = agent.NewDeployConn(conn)
	s.Logger.Println(s.au.Brown(fmt.Sprint("BEGIN LOGS:", messagePrefix(m))))
	logx.MaybeLog(
		errorsx.Compact(
			agentutil.PrintLogs(b, client, m.Peer, m.GetDeploy().Archive.DeploymentID, os.Stderr),
			client.Close(),
		),
	)
	s.Logger.Println(s.au.Brown(fmt.Sprint("CEASE LOGS:", messagePrefix(m))))

	atomic.AddInt64(t.n, 1)
}

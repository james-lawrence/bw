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
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

// gathers up failures
type failure struct {
	failures map[string]agent.Message
	cState
}

func (t failure) Consume(m agent.Message) consumer {
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
	Display(cState, agent.Message)
}

// FailureDisplayNoop - ignores failures
type FailureDisplayNoop struct{}

// Display does nothing
func (t FailureDisplayNoop) Display(cState, agent.Message) {}

// NewFailureDisplayPrint ...
func NewFailureDisplayPrint(d dialer) FailureDisplayPrint {
	return FailureDisplayPrint{
		n:      new(int64),
		Dialer: d,
	}
}

// FailureDisplayPrint prints the logs for each unique error encountered
type FailureDisplayPrint struct {
	n      *int64
	Dialer dialer
}

// Display prints the logs for each message
func (t FailureDisplayPrint) Display(s cState, m agent.Message) {
	var (
		err error
		c   agent.Client
	)

	if m.Peer == nil {
		log.Println("unexpected nil peer skipping", spew.Sdump(m))
		return
	}

	if c, err = t.Dialer.Dial(*m.Peer); err != nil {
		log.Println(errors.Wrapf(err, "unable to retrieve logs, failed to connect to peer: %s", spew.Sdump(m.Peer)))
		return
	}

	if atomic.LoadInt64(t.n) > 0 {
		os.Stderr.WriteString("\n\n")
	}

	s.Logger.Println(s.au.Brown(fmt.Sprint("BEGIN LOGS:", messagePrefix(m))))
	b, done := context.WithTimeout(context.Background(), 20*time.Second)
	logx.MaybeLog(
		errorsx.Compact(
			agentutil.PrintLogs(b, m.GetDeploy().Archive.DeploymentID, os.Stderr)(c),
			c.Close(),
		),
	)
	done()
	s.Logger.Println(s.au.Brown(fmt.Sprint("CEASE LOGS:", messagePrefix(m))))

	atomic.AddInt64(t.n, 1)
}

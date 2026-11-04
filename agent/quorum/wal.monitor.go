package quorum

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/asynccompute"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/slogx"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// applyWarnThreshold is how long a single Apply may take before it is reported.
// Apply runs on the raft FSM goroutine, a slow apply throttles the entire log.
const applyWarnThreshold = time.Second

// monitoredWAL wraps the WAL, timing every apply. snapshots and restores pass straight through.
type monitoredWAL struct {
	*WAL
	monitor *applyMonitor
}

func newMonitoredWAL(w *WAL, m *applyMonitor) monitoredWAL {
	return monitoredWAL{WAL: w, monitor: m}
}

// Apply the log to the WAL, observing how long it takes.
func (t monitoredWAL) Apply(l *raft.Log) interface{} {
	start := time.Now()
	m, err := t.WAL.ApplyLog(l)
	elapsed := time.Since(start)

	// the begin of a deploy resets the tracking, its own apply counts toward the new deploy.
	t.monitor.begin(m)
	t.monitor.observe(l, elapsed)
	t.monitor.end(m)

	if err != nil {
		return err
	}

	return nil
}

// newApplyMonitor reports to the dispatcher in the background, the monitor is invoked from the raft
// FSM goroutine and a replicated write waits on that same goroutine.
func newApplyMonitor(d agent.Dispatcher, local func() *agent.Peer) *applyMonitor {
	return &applyMonitor{
		reports: asynccompute.New(
			func(ctx context.Context, m *agent.Message) error {
				return errors.Wrap(d.Dispatch(ctx, m), "unable to report fsm apply")
			},
			asynccompute.Workers[*agent.Message](1),
			asynccompute.Backlog[*agent.Message](32),
		),
		local:     local,
		threshold: applyWarnThreshold,
		warnrate:  rate.NewLimiter(slogx.DefaultWarnRate, 1),
	}
}

// applyMonitor tracks how long the FSM takes to apply messages, per deploy. slow applies are
// logged, the first one of a deploy and the maximum duration at its end are reported to the
// cluster through the dispatcher. local returns nil when not the leader, followers only log.
type applyMonitor struct {
	local     func() *agent.Peer
	threshold time.Duration
	warnrate  *rate.Limiter // a stalled fsm would otherwise log on every apply.
	reports   *asynccompute.Pool[*agent.Message]

	// only touched by the observe/begin/end which raft serializes.
	maxApply time.Duration
	warned   bool
}

// Close waits for the queued reports to be sent.
func (t *applyMonitor) Close() error {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	return asynccompute.Shutdown(ctx, t.reports)
}

// observe the duration of an apply, warning when it is slow.
func (t *applyMonitor) observe(l *raft.Log, elapsed time.Duration) {
	if t == nil {
		return
	}

	t.maxApply = max(t.maxApply, elapsed)

	if elapsed <= t.threshold {
		return
	}

	slogx.With(log.Default()).Limit(t.warnrate).Printf("WARN - fsm apply took %s index=%d term=%d\n", elapsed, l.Index, l.Term)

	if t.warned {
		return
	}

	t.warned = true
	if p := t.local(); p != nil {
		errorsx.Log(t.reports.Run(context.Background(), agent.LogEvent(p, fmt.Sprintf("WARN - fsm apply exceeded %s, took %s", t.threshold, elapsed))))
	}
}

// begin resets the tracking when a deploy begins.
func (t *applyMonitor) begin(m *agent.Message) {
	if t == nil {
		return
	}

	if dc := m.GetDeployCommand(); dc != nil && dc.Command == agent.DeployCommand_Begin {
		t.maxApply, t.warned = 0, false
	}
}

// end reports the maximum apply duration once a deploy ends. must run after the apply of the
// final message is observed so it is included.
func (t *applyMonitor) end(m *agent.Message) {
	if t == nil {
		return
	}

	dc := m.GetDeployCommand()
	if dc == nil || dc.Command == agent.DeployCommand_Begin {
		return
	}

	if p := t.local(); p != nil {
		errorsx.Log(t.reports.Run(context.Background(), agent.LogEvent(p, fmt.Sprintf("fsm apply max duration %s", t.maxApply))))
	}
	t.maxApply, t.warned = 0, false
}

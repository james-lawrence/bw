package cstate

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/james-lawrence/torrent/internal/asyncx"
)

type logger interface {
	Println(v ...any)
	Printf(format string, v ...any)
	Print(v ...any)
}
type Shared struct {
	done context.CancelCauseFunc
	log  logger
}

type T interface {
	Update(context.Context, *Shared) T
}

// Idle creates an idler that wakes when wake is broadcast to. wake remembers broadcasts that
// happen while the state machine is not idle, so a wakeup is never lost between checking
// state and going idle.
//
// signals are shared conds (they have other waiters, so they cannot be replaced by a semaphore),
// each is relayed into wake by a goroutine that lives until Stop is called.
func Idle(ctx context.Context, wake *asyncx.Semaphore, signals ...*sync.Cond) *Idler {
	tt := time.NewTimer(time.Hour)
	tt.Stop()

	_ctx, done := context.WithCancel(ctx)
	return (&Idler{
		timeout: tt,
		target:  wake,
		signals: signals,
		stop:    done,
	}).monitor(_ctx)
}

type Idler struct {
	timeout *time.Timer
	target  *asyncx.Semaphore
	signals []*sync.Cond
	stop    context.CancelFunc
}

func (t *Idler) Stop() {
	t.stop()
	for _, s := range t.signals {
		s.Broadcast()
	}
}

func (t *Idler) Idle(next T, d time.Duration) idle {
	if d > 0 {
		t.timeout.Reset(d)
	}
	return idle{Idler: t, next: next}
}

func (t *Idler) monitor(ctx context.Context) *Idler {
	for _, s := range t.signals {
		go func(_s *sync.Cond) {
			for {
				_s.L.Lock()
				_s.Wait()
				_s.L.Unlock()
				t.target.Signal()
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}(s)
	}

	return t
}

type idle struct {
	*Idler
	next T
}

func (t idle) Update(ctx context.Context, c *Shared) T {
	defer t.Idler.timeout.Stop()
	select {
	case <-t.target.Wait():
	case <-t.Idler.timeout.C:
	case <-ctx.Done():
	}
	return t.next
}
func (t idle) String() string {
	return fmt.Sprintf("%T - %s - idle", t.next, t.next)
}

func Failure(cause error) failed {
	return failed{cause: cause}
}

type failed struct {
	cause error
}

func (t failed) Update(ctx context.Context, c *Shared) T {
	c.done(t.cause)
	return nil
}

func (t failed) String() string {
	return fmt.Sprintf("%T - %s", t, t.cause)
}

func Warning(next T, cause error) warning {
	return warning{next: next, cause: cause}
}

type warning struct {
	cause error
	next  T
}

func (t warning) Update(ctx context.Context, c *Shared) T {
	c.log.Println("[warning]", t.cause)
	return t.next
}

func (t warning) String() string {
	return fmt.Sprintf("%T - %T", t, t.cause)
}

func Halt() halt {
	return halt{}
}

type halt struct{}

func (t halt) Update(ctx context.Context, c *Shared) T {
	return nil
}
func Fn(fn fn) fn {
	return fn
}

type fn func(context.Context, *Shared) T

func (t fn) Update(ctx context.Context, s *Shared) T {
	return t(ctx, s)
}

func (t fn) String() string {
	pc := reflect.ValueOf(t).Pointer()
	info := runtime.FuncForPC(pc)
	fname, line := info.FileLine(pc)
	return fmt.Sprintf("%s:%d", fname, line)
}

func Run(ctx context.Context, s T, l logger) error {
	ctx, cancelled := context.WithCancelCause(ctx)
	var (
		m = Shared{
			done: cancelled,
			log:  l,
		}
	)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
			l.Printf("%s - %T\n", s, s)
			s = s.Update(ctx, &m)
		}

		if s == nil {
			return context.Cause(ctx)
		}

		runtime.Gosched()
	}
}

package observers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/akutz/memconn"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// NewMemory see Directory
func NewMemory() (cache Memory, err error) {
	cache = Memory{
		initialize: &sync.Once{},
		m:          &sync.RWMutex{},
		observers:  map[string]Conn{},
	}

	return cache, nil
}

// Memory observes a directory for sockets to write messages into.
type Memory struct {
	initialize *sync.Once
	m          *sync.RWMutex
	observers  map[string]Conn
}

func (t Memory) genid() (ids string, err error) {
	var (
		id bw.RandomID
	)

	if id, err = bw.SimpleGenerateID(); err != nil {
		return ids, err
	}

	return fmt.Sprintf("obs-%s", id.String()), nil
}

// Connect subscribe to events.
func (t Memory) Connect(b chan *agent.Message) (l net.Listener, s *grpc.Server, err error) {
	var (
		id string
	)

	if id, err = t.genid(); err != nil {
		return l, s, err
	}

	if l, err = memconn.Listen("memu", id); err != nil {
		return l, s, err
	}

	s = grpc.NewServer(
		// grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		// grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    10 * time.Second,
			Timeout: 30 * time.Second,
		}),
	)

	New(b).Bind(s)

	go func() {
		if err = s.Serve(l); err != nil {
			log.Println("client observer completed", err)
		}

		log.Println("observer disconnecting", id, len(t.observers))

		if err = t.disconnect(id); err != nil {
			log.Println("client observer disconnect failed", err)
		}
	}()

	log.Println("observer connecting", id, len(t.observers))

	return l, s, t.connect(id)
}

// Dispatch messages to the observers.
func (t Memory) Dispatch(ctx context.Context, messages ...*agent.Message) error {
	t.m.RLock()
	cpy := make(map[string]Conn, len(t.observers))
	for id, obs := range t.observers {
		cpy[id] = obs
	}
	t.m.RUnlock()

	if len(cpy) == 0 {
		return nil
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("observer dispatch initiated", len(cpy))
		defer log.Println("observer dispatch completed", len(cpy))
	}

	for id, conn := range cpy {
		if err := t.dispatch(ctx, conn, messages...); err != nil {
			log.Println(errors.Wrapf(err, "failed to deliver messages: %s", id))
		}
	}

	return nil
}

func (t Memory) dispatch(ctx context.Context, conn Conn, messages ...*agent.Message) error {
	ctx, done := context.WithTimeout(ctx, time.Second)
	defer done()
	return conn.Dispatch(ctx, messages...)
}

func (t Memory) connect(id string) (err error) {
	var (
		conn *grpc.ClientConn
	)

	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()

	if conn, err = grpc.DialContext(ctx, id, grpcx.DialInmem(), grpc.WithInsecure(), grpc.WithBlock()); err != nil {
		return errors.Wrap(err, "failed to connect observer")
	}

	t.m.Lock()
	t.observers[id] = NewConn(conn)
	t.m.Unlock()

	return err
}

func (t *Memory) disconnect(id string) (err error) {
	var (
		ok   bool
		conn Conn
	)

	log.Println("lost observer", id)

	t.m.Lock()
	defer t.m.Unlock()

	if conn, ok = t.observers[id]; !ok {
		return errorsx.String("disconnect for a non-existant observer")
	}

	if err = errors.Wrap(conn.conn.Close(), "failed to disconnect"); err != nil {
		return err
	}

	delete(t.observers, id)

	return err
}

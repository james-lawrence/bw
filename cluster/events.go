package cluster

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// LoggingEventHandler ...
type LoggingEventHandler struct{}

// NotifyJoin logs when a peer joins.
func (t LoggingEventHandler) NotifyJoin(peer *memberlist.Node) {
	log.Println("NotifyJoin", peer.Name)
}

// NotifyLeave logs when a peer leaves.
func (t LoggingEventHandler) NotifyLeave(peer *memberlist.Node) {
	log.Println("NotifyLeave", peer.Name)
}

// NotifyUpdate logs when a peer updates.
func (t LoggingEventHandler) NotifyUpdate(peer *memberlist.Node) {
	log.Println("NotifyUpdate", peer.Name)
}

// LogingSubscription logs out the events as they arrive.
func LoggingSubscription(ctx context.Context, evt *agent.ClusterWatchEvents) error {
	log.Println("LoggingSubscription", evt.Event.String(), evt.Node.Ip, evt.Node.Name, len(evt.Node.PublicKey))
	return nil
}

type EventsSubscriber func(ctx context.Context, evt *agent.ClusterWatchEvents) error

func NewEventsSubscription(ctx context.Context, conn *grpc.ClientConn, s EventsSubscriber) (err error) {
	var (
		client agent.Cluster_WatchClient
	)

	if client, err = agent.NewClusterClient(conn).Watch(ctx, &agent.ClusterWatchRequest{}); err != nil {
		return err
	}

	enqueue := func(evt *agent.ClusterWatchEvents) error {
		ctx, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		return errorsx.Compact(s(ctx, evt), ctx.Err())
	}

	go func() {
		var (
			err error
			evt *agent.ClusterWatchEvents
		)

		defer log.Println("event subscription done")
		defer client.CloseSend()
		for {
			if evt, err = client.Recv(); err != nil {
				log.Printf("%T - %s\n", s, errors.Wrap(err, "unable to receive event"))
				return
			}

			if err = enqueue(evt); err != nil {
				log.Println("unable to handle event", err)
				continue
			}
		}
	}()

	return nil
}

func NewEventsQueue(l *Local) *EventsQueue {
	return &EventsQueue{
		local:       l,
		connections: make(map[int64]agent.Cluster_WatchServer),
		queue:       make(chan *agent.ClusterWatchEvents, 100),
	}
}

type EventsQueue struct {
	agent.UnimplementedClusterServer
	local       *Local
	idx         int64
	connections map[int64]agent.Cluster_WatchServer
	queue       chan *agent.ClusterWatchEvents
	qm          sync.Mutex
	m           sync.RWMutex
}

func (t *EventsQueue) Bind(srv *grpc.Server) {
	agent.RegisterClusterServer(srv, t)
	go t.background()
}

func (t *EventsQueue) background() {
	for evt := range t.queue {
		t.dispatch(evt)
	}
}

func (t *EventsQueue) dispatch(evt *agent.ClusterWatchEvents) {
	t.m.RLock()
	defer t.m.RUnlock()

	for _, conn := range t.connections {
		if err := conn.Send(evt); err != nil {
			log.Println("unable to dispatch to subscription", err)
			continue
		}
	}
}

func (t *EventsQueue) Watch(req *agent.ClusterWatchRequest, stream agent.Cluster_WatchServer) (err error) {
	id := atomic.AddInt64(&t.idx, 1)
	t.m.Lock()
	t.connections[id] = stream
	t.m.Unlock()

	ctx := stream.Context()
	<-ctx.Done()

	t.m.Lock()
	delete(t.connections, id)
	t.m.Unlock()

	return ctx.Err()
}

// ensures ordering
func (t *EventsQueue) enqueueNode(typ agent.ClusterWatchEvents_Event, n *memberlist.Node) {
	var (
		err error
		p   *agent.Peer
	)

	if p, err = agent.NodeToPeer(n); err != nil {
		log.Println("unable to convert node to peer", err)
		return
	}

	t.enqueue(typ, p)
}

// ensures ordering
func (t *EventsQueue) enqueue(typ agent.ClusterWatchEvents_Event, n *agent.Peer) {
	t.qm.Lock()
	defer t.qm.Unlock()

	t.queue <- &agent.ClusterWatchEvents{
		Event: typ,
		Node:  n,
	}
}

// NotifyJoin logs when a peer joins.
func (t *EventsQueue) NotifyJoin(n *memberlist.Node) {
	t.enqueueNode(agent.ClusterWatchEvents_Joined, n)
}

// NotifyLeave logs when a peer leaves.
func (t *EventsQueue) NotifyLeave(n *memberlist.Node) {
	t.enqueueNode(agent.ClusterWatchEvents_Depart, n)
}

// NotifyUpdate logs when a peer updates.
func (t *EventsQueue) NotifyUpdate(n *memberlist.Node) {
	t.enqueueNode(agent.ClusterWatchEvents_Update, n)
}

// NodeMeta provides the metadata about the node.
func (t *EventsQueue) NodeMeta(limit int) []byte {
	if t.local == nil {
		return []byte(nil)
	}

	log.Println("NodeMeta invoked limit:", limit, len(t.local.metadata))
	if limit < len(t.local.metadata) {
		log.Println("insufficient room to send metadata")
		return []byte(nil)
	}

	return t.local.metadata
}

// LocalState local state of the node.
func (t *EventsQueue) LocalState(join bool) []byte {
	if t.local == nil {
		return []byte(nil)
	}
	return t.local.encoded
}

// GetBroadcasts ...
func (t *EventsQueue) GetBroadcasts(overhead, limit int) [][]byte {
	// if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
	// 	log.Printf("GetBroadcasts overhead(%d) limit(%d)\n", overhead, limit)
	// }
	return [][]byte(nil)
}

// MergeRemoteState ...
func (t *EventsQueue) MergeRemoteState(buf []byte, join bool) {
	var (
		p agent.Peer
	)

	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Printf("MergeRemoteState join(%t) len(%d)\n", join, len(buf))
	}

	if err := proto.Unmarshal(buf, &p); err != nil {
		log.Println("MergeRemoteState: failed", err)
		return
	}

	t.enqueue(agent.ClusterWatchEvents_Update, &p)
}

// NotifyMsg ...
func (t *EventsQueue) NotifyMsg(buf []byte) {
	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("NotifyMsg string(buf):", string(buf))
	}
}

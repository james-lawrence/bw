package notary

import (
	"bytes"
	"log"

	"github.com/james-lawrence/bw/internal/md5x"
	"github.com/james-lawrence/bw/internal/x/iox"
	"github.com/pkg/errors"
	"github.com/willf/bloom"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewSyncRequest(b *bloom.BloomFilter) (r *SyncRequest, err error) {
	var (
		buf bytes.Buffer
	)

	if _, err = b.WriteTo(&buf); err != nil {
		return nil, err
	}

	return &SyncRequest{Bloom: buf.Bytes()}, nil
}

// Sync from a client connection
func Sync(stream Sync_StreamClient, b Bloomy, s storage) (err error) {
	for {
		var (
			event *SyncStream
		)

		if event, err = stream.Recv(); err != nil {
			err = iox.IgnoreEOF(err)
			break
		}

		switch evt := event.Events.(type) {
		case *SyncStream_Chunk:
			for _, g := range evt.Chunk.Grants {
				log.Println("retrieved", g.Fingerprint)
				if _, err := s.Insert(g); err != nil {
					return err
				}

				b.Add([]byte(g.Fingerprint))
			}
		}
	}

	return err
}

// NewSyncService ...
func NewSyncService(a auth, s SyncStorage) *SyncService {
	return &SyncService{
		auth:        a,
		SyncStorage: s,
	}
}

type SyncService struct {
	UnimplementedSyncServer
	auth
	SyncStorage
}

// Bind the service to the given grpc server.
func (t SyncService) Bind(s *grpc.Server, options ...option) {
	RegisterSyncServer(s, t)
}

func (t SyncService) Stream(r *SyncRequest, s Sync_StreamServer) (err error) {
	grantevent := func(grants []*Grant) *SyncStream {
		return &SyncStream{
			Events: &SyncStream_Chunk{
				Chunk: &SyncGrants{
					Grants: grants,
				},
			},
		}
	}

	if p := t.auth.Authorize(s.Context()); !p.Sync {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}

	b := bloom.New(1, 1)
	if _, err = b.ReadFrom(bytes.NewReader(r.Bloom)); err != nil {
		log.Println(errors.Wrap(err, "unable to generate bloom filter"))
		return status.Error(codes.InvalidArgument, "unable to generate filter")
	}

	out := make(chan *Grant, 200)
	errc := make(chan error)
	go func() {
		errc <- t.SyncStorage.Sync(s.Context(), b, out)
		close(out)
	}()

	batch := make([]*Grant, 0, 100)

	for {
		select {
		case g, ok := <-out:
			if !ok {
				if err = s.Send(grantevent(batch)); err != nil {
					log.Println(errors.Wrap(err, "failed to send event"))
					return status.Error(codes.Internal, "failed to send event")
				}
				return nil
			}

			if batch = append(batch, g); len(batch) < cap(batch) {
				continue
			}

			if err = s.Send(grantevent(batch)); err != nil {
				log.Println(errors.Wrap(err, "failed to send event"))
				return status.Error(codes.Internal, "failed to send event")
			}

			batch = batch[:0]
		case err := <-errc:
			if err == nil {
				continue
			}

			if cause := s.Send(grantevent(batch)); cause != nil {
				log.Println(errors.Wrap(cause, "failed to send event"))
			}

			return status.Error(codes.Internal, "failed to send event")
		}
	}
}

func NewEntropyBloom(entropy []byte, b *bloom.BloomFilter) EntropyBloom {
	return EntropyBloom{
		entropy: entropy,
		b:       b,
	}
}

type EntropyBloom struct {
	entropy []byte
	b       *bloom.BloomFilter
}

func (t EntropyBloom) Test(k []byte) bool {
	buf := make([]byte, 0, len(t.entropy)+len(k))
	copy(buf, t.entropy)
	copy(buf, k)

	return t.b.Test(md5x.DigestX(buf))
}

func (t EntropyBloom) Add(k []byte) EntropyBloom {
	buf := make([]byte, 0, len(t.entropy)+len(k))
	copy(buf, t.entropy)
	copy(buf, k)

	t.b.Add(buf)
	return t
}

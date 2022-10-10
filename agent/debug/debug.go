package debug

import (
	"bytes"
	context "context"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/bytesx"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/internal/md5x"
	"github.com/james-lawrence/bw/internal/profilex"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/james-lawrence/bw/internal/systemx"
	"github.com/pkg/errors"
	"github.com/pkg/profile"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type auth interface {
	Deploy(ctx context.Context) error
}

func NewService(a auth) *Service {
	return &Service{
		auth:      a,
		profiling: &atomic.Bool{},
		stoppable: profilex.Noop(),
		tmpdir:    path.Join(stringsx.DefaultIfBlank(os.Getenv("CACHE_DIR"), os.TempDir()), "profiling"),
	}
}

type Service struct {
	UnimplementedDebugServer
	auth      auth
	profiling *atomic.Bool
	stoppable profilex.Stoppable
	tmpdir    string
}

// Bind to a grpc server.
func (t *Service) Bind(srv *grpc.Server) *Service {
	RegisterDebugServer(srv, t)
	return t
}

func (t Service) Stacktrace(ctx context.Context, _ *StacktraceRequest) (_ *StacktraceResponse, err error) {
	if err = t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 16*bytesx.KiB))

	if err = debugx.DumpRoutinesInto(iox.WriteNopCloser(buf)); err != nil {
		log.Println(errors.Wrap(err, "unable to generate stack trace"))
		return nil, status.Error(codes.Internal, "unable to generate stack trace")
	}

	return &StacktraceResponse{
		Trace: buf.Bytes(),
	}, nil
}

func (t *Service) CPU(ctx context.Context, req *ProfileRequest) (_ *ProfileResponse, err error) {
	return t.profile(ctx, req, profile.CPUProfile)
}

func (t *Service) Memory(ctx context.Context, req *ProfileRequest) (_ *ProfileResponse, err error) {
	return t.profile(ctx, req, profile.MemProfile)
}

func (t *Service) Heap(ctx context.Context, req *ProfileRequest) (_ *ProfileResponse, err error) {
	return t.profile(ctx, req, profile.MemProfileHeap)
}

func (t *Service) Download(ctx context.Context, req *DownloadRequest) (_ *DownloadResponse, err error) {
	var (
		src *os.File
	)

	if active := t.profiling.Load(); active {
		return nil, status.Errorf(codes.Unavailable, "profile has not completed")
	}

	if err = t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	p := path.Join(t.tmpdir, "profile.pprof")
	if !systemx.FileExists(p) {
		return nil, status.Errorf(codes.FailedPrecondition, "there is no profile to download")
	}

	buf := bytes.NewBuffer(nil)

	if src, err = os.Open(p); err != nil {
		return nil, err
	}
	defer src.Close()

	if _, err = io.Copy(buf, src); err != nil {
		return nil, err
	}

	return &DownloadResponse{
		Profile: buf.Bytes(),
	}, nil
}

func (t *Service) Cancel(ctx context.Context, req *CancelRequest) (_ *CancelResponse, err error) {
	if err = t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	t.stoppable.Stop()

	return &CancelResponse{}, nil
}

func (t *Service) profile(ctx context.Context, req *ProfileRequest, strategy func(*profile.Profile)) (_ *ProfileResponse, err error) {
	if swapped := t.profiling.CompareAndSwap(false, true); !swapped {
		return nil, status.Errorf(codes.AlreadyExists, "a profile is already in progress")
	}

	if err = t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	if err = os.RemoveAll(t.tmpdir); err != nil {
		log.Println("unable to clear profiling directory", err)
		return nil, status.Errorf(codes.FailedPrecondition, "unable to start profiling - storage unavailable")
	}

	if err = os.MkdirAll(t.tmpdir, 0700); err != nil {
		log.Println("unable to create profiling directory", err)
		return nil, status.Errorf(codes.FailedPrecondition, "unable to start profiling - storage unavailable")
	}

	tmpdir, err := os.MkdirTemp(t.tmpdir, strings.ReplaceAll("{}.*.profile", "{}", md5x.DigestString(req.Id)))
	if err != nil {
		log.Println("unable to create profiling directory", err)
		return nil, status.Errorf(codes.FailedPrecondition, "unable to start profiling - storage unavailable")
	}

	dctx, done := context.WithTimeout(context.Background(), time.Duration(req.Duration))
	p := profile.Start(
		strategy,
		profile.NoShutdownHook,
		profile.ProfilePath(tmpdir),
	)

	t.stoppable = profilex.StopFunc(func() {
		defer t.profiling.Store(false)
		defer done()
		p.Stop()
		errorsx.MaybeLog(errors.Wrap(t.clone(tmpdir), "unable to finalize profile"))
		t.stoppable = profilex.Noop()
	})

	go func() {
		errorsx.MaybeLog(errors.Wrap(profilex.Run(dctx, t.stoppable), ""))
	}()

	return &ProfileResponse{}, nil
}

func (t *Service) clone(dir string) (err error) {
	var (
		dst, src *os.File
	)

	location := bw.LocateFirstInDir(
		dir,
		"cpu.pprof",
		"mem.pprof",
		"mutex.pprof",
		"block.pprof",
		"threadcreation.pprof",
	)

	if dst, err = os.Create(path.Join(t.tmpdir, "profile.pprof")); err != nil {
		return errors.Wrap(err, "copy failed")
	}
	defer dst.Close()

	if src, err = os.Open(location); err != nil {
		return errors.Wrap(err, "copy failed")
	}
	defer src.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copy failed")
	}

	return nil
}

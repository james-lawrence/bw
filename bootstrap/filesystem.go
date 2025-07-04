package bootstrap

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/fsx"
	"github.com/james-lawrence/bw/internal/iox"
)

// NewFilesystem consumes a configuration and generates a bootstrap socket
// for the agent.
func NewFilesystem(a agent.Config, c cluster, d dialer) Filesystem {
	return Filesystem{
		a:        a,
		c:        c,
		d:        d,
		current:  filepath.Join(a.Bootstrap.ArchiveDirectory, "current.tar.gz"),
		metadata: filepath.Join(a.Bootstrap.ArchiveDirectory, "current.meta"),
		uploaded: filepath.Join(a.Bootstrap.ArchiveDirectory, "current.uploaded"),
	}
}

// Filesystem bootstrap service will monitor the cluster and write the last
// successful deployment to the filesystem and return that deployment when queried.
// this is useful for storing a backup copy that can be treated as bootstrappable archive.
type Filesystem struct {
	agent.UnimplementedBootstrapServer
	a        agent.Config
	c        cluster
	d        dialer
	current  string
	metadata string
	uploaded string
}

// Bind the bootstrap service to the provided socket.
func (t Filesystem) Bind(ctx context.Context, socket string, options ...grpc.ServerOption) (err error) {
	if t.a.Bootstrap.ArchiveDirectory == "" {
		log.Println("filesystem bootstrap: disabled")
		return nil
	}

	if err = t.init(); err != nil {
		log.Println("filesystem bootstrap: disabled", err)
		return nil
	}

	if !t.a.Bootstrap.ReadOnly {
		go t.monitor()
	}

	if err = t.upload(); err != nil {
		return err
	}

	log.Println("filesystem bootstrap: enabled", t.a.Bootstrap.ArchiveDirectory)

	return Run(ctx, socket, t, options...)
}

// Archive - implements the bootstrap service.
func (t Filesystem) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	var (
		dc *agent.DeployCommand
	)

	if _, err = os.Stat(t.uploaded); err != nil {
		return nil, status.Error(codes.NotFound, "filesystem: found no deployments")
	}

	if dc, err = agent.ReadMetadata(t.uploaded); err != nil {
		log.Println(errors.Wrap(err, "filesystem: upload failed"))
		return nil, status.Error(codes.NotFound, "filesystem: upload failed")
	}

	return &agent.ArchiveResponse{
		Deploy: &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: dc.Archive,
			Options: dc.Options,
		},
	}, nil
}

func (t Filesystem) monitor() {
	var (
		events = make(chan *agent.Message, 5)
	)

	d := dialers.NewProxy(
		dialers.NewDirect(agent.RPCAddress(t.c.Local()), t.d.Defaults()...),
	)

	go agentutil.WatchEvents(context.Background(), t.c.Local(), d, events)

	for m := range events {
		if m.Hidden {
			continue
		}

		switch event := m.GetEvent().(type) {
		case *agent.Message_DeployCommand:
			dc := event.DeployCommand
			if dc.Command == agent.DeployCommand_Done && dc.Archive != nil {
				go func() {
					if cause := errors.Wrap(t.clone(dc), "clone failed"); cause == nil {
						log.Println("clone successful")
					} else {
						log.Println(cause)
					}
				}()
			}
		default:
			// log.Println("FILESYSTEM EVENT", spew.Sdump(event))
			// ignore other commands.
		}
	}
}

func (t Filesystem) init() error {
	return errors.WithStack(os.MkdirAll(t.a.Bootstrap.ArchiveDirectory, 0744))
}

func (t Filesystem) clone(a *agent.DeployCommand) (err error) {
	var (
		d        *os.File
		archive  *os.File
		metadata = t.metadata + ".tmp"
	)

	log.Println("cloning successful deploy")

	if err = t.init(); err != nil {
		return err
	}

	if err = agent.WriteMetadata(metadata, a); err != nil {
		return errors.WithStack(err)
	}

	if archive, err = os.Open(filepath.Join(bw.DeployDir(t.a.Root), bw.RandomID(a.Archive.DeploymentID).String(), bw.ArchiveFile)); err != nil {
		return errors.WithStack(err)
	}

	if d, err = os.CreateTemp(t.a.Bootstrap.ArchiveDirectory, "download-*.bin"); err != nil {
		return errors.WithStack(err)
	}
	defer d.Close()

	if _, err = io.Copy(d, archive); err != nil {
		return errors.WithStack(err)
	}

	if err = errorsx.Compact(d.Sync(), d.Close()); err != nil {
		return errors.WithStack(err)
	}

	if err = os.Rename(d.Name(), t.current); err != nil {
		return errors.WithStack(err)
	}

	if err = os.Rename(metadata, t.metadata); err != nil {
		return errors.WithStack(err)
	}

	// persist into the archive directory.
	if p := filepath.Join(t.a.Credentials.Directory, certificatecache.DefaultTLSCertCA); fsx.IsRegularFile(p) {
		if err = iox.Copy(p, filepath.Join(t.a.Bootstrap.ArchiveDirectory, certificatecache.DefaultTLSCertCA)); err != nil {
			log.Printf("failed to copy file: %s - %v", p, err)
			// return errors.WithStack(err)
		}
	}

	if p := filepath.Join(t.a.Credentials.Directory, certificatecache.DefaultTLSCertServer); fsx.IsRegularFile(p) {
		if err = iox.Copy(p, filepath.Join(t.a.Bootstrap.ArchiveDirectory, certificatecache.DefaultTLSCertServer)); err != nil {
			log.Printf("failed to copy file: %s - %v", p, err)
			// return errors.WithStack(err)
		}
	}

	if p := filepath.Join(t.a.Credentials.Directory, certificatecache.DefaultTLSKeyServer); fsx.IsRegularFile(p) {
		if err = iox.Copy(p, filepath.Join(t.a.Bootstrap.ArchiveDirectory, certificatecache.DefaultTLSKeyServer)); err != nil {
			log.Printf("failed to copy file: %s - %v", p, err)
			// return errors.WithStack(err)
		}
	}

	return nil
}

func (t Filesystem) upload() (err error) {
	var (
		conn *grpc.ClientConn
		i    os.FileInfo
		src  *os.File
		dc   *agent.DeployCommand
	)

	if i, err = os.Stat(t.current); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return errors.WithStack(err)
	}

	if src, err = os.Open(t.current); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return errors.WithStack(err)
	}

	if dc, err = agent.ReadMetadata(t.metadata); err != nil {
		return err
	}

	if conn, err = t.d.DialContext(context.Background(), grpc.WithBlock()); err != nil {
		return err
	}
	defer conn.Close()

	meta := &agent.UploadMetadata{
		Bytes:     uint64(i.Size()),
		Vcscommit: dc.Archive.Commit,
	}

	if dc.Archive, err = agent.NewConn(conn).Upload(context.Background(), meta, src); err != nil {
		return err
	}

	if err = agent.WriteMetadata(t.uploaded, dc); err != nil {
		return err
	}

	return nil
}

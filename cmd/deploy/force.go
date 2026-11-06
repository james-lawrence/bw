package deploy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Force replaces the bootstrap archive of the cluster without running a deploy. The deployspace is
// packed and uploaded, the upload is verified, and then the archive is announced as a successful
// deploy so nodes which archive deployments store it as their bootstrap archive. Finally
// the nodes are restarted one at a time (unless ctx.NoRestart is set).
func Force(gctx *Context) (err error) {
	var (
		d         dialers.Defaults
		config    agent.ConfigClient
		ss        notary.Signer
		darchive  *agent.Archive
		commitish string
	)

	if config, err = commandutils.LoadConfiguration(gctx.Context, gctx.Environment, agent.CCOptionInsecure(gctx.Insecure)); err != nil {
		return errors.Wrap(err, "unable to load configuration")
	}

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())

	if ss, err = notary.NewAutoSigner(displayname); err != nil {
		return errors.Wrap(err, "unable to setup authorization")
	}

	if commitish, err = commandutils.RunLocalDirectives(gctx.Context, config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	local := agent.NewPeerZero()

	var (
		debugopt1 grpc.DialOption = grpc.EmptyDialOption{}
		debugopt2 grpc.DialOption = grpc.EmptyDialOption{}
	)

	d, c, err := daemons.ConnectClientUntilSuccess(gctx.Context, config, ss, debugopt1, debugopt2, grpc.WithPerRPCCredentials(ss))
	if err != nil {
		return errors.Wrap(err, "unable to connect to cluster")
	}

	// nothing archives deployments, forcing an archive would accomplish nothing.
	archivers := archivingPeers(agent.NodesToPeers(c.Members()...)...)
	if len(archivers) == 0 {
		return errorsx.String("no nodes in the cluster archive deployments (bootstrap.archiveDirectory is unset or readonly), refusing to force an archive")
	}

	// the upload is sent through the quorum.
	qd := dialers.NewQuorum(c, d.Defaults()...)
	dispatcher := agentutil.NewDispatcher(qd)

	conn := grpcx.UntilSuccess(gctx.Context, func(ictx context.Context) (*grpc.ClientConn, error) {
		return qd.DialContext(ictx)
	})
	defer func() { errorsx.Log(errors.Wrap(conn.Close(), "failed to close connection")) }()

	deployspace := config.Deployspace()
	if err = os.WriteFile(filepath.Join(deployspace, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(deployspace, bw.AuthKeysFile)); err != nil {
			return err
		}
	}

	dst, err := os.CreateTemp("", "bwarchive")
	if err != nil {
		return errors.Wrap(err, "archive creation failed")
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = archive.Pack(dst, deployspace); err != nil {
		return errors.Wrap(err, "archive creation failed")
	}

	size, sum, err := fingerprint(dst)
	if err != nil {
		return errors.Wrap(err, "archive creation failed")
	}

	// announce what is about to happen, who is doing it, and why before anything is changed.
	if err = agentutil.ReliableDispatch(gctx.Context, dispatcher, agent.LogWarning(
		local,
		fmt.Sprintf(
			"FORCING BOOTSTRAP ARCHIVE: by(%s) - commit(%s) - this bypasses a deploy and overrides the archive new and restarting nodes bootstrap from; archiving nodes(%d), restart(%t)",
			displayname, commitish, len(archivers), !gctx.NoRestart,
		),
	)); err != nil {
		return errors.Wrap(err, "unable to announce forced archive")
	}

	client := agent.NewDeployConn(conn)
	err = grpcx.Retry(gctx.Context, func() error {
		if _, err = dst.Seek(0, io.SeekStart); err != nil {
			return err
		}

		meta := agent.UploadMetadata{
			Bytes:     uint64(size),
			Vcscommit: commitish,
		}

		if darchive, err = client.Upload(gctx.Context, &meta, dst); errors.Is(err, io.EOF) {
			return status.Error(codes.Unavailable, err.Error())
		}

		return err
	}, codes.Unavailable)

	if err != nil {
		return errors.Wrap(err, "archive upload failed")
	}

	if err = verifyUpload(darchive, sum, commitish); err != nil {
		errorsx.Log(agentutil.ReliableDispatch(gctx.Context, dispatcher, agent.LogError(local, errors.Wrap(err, "forced archive upload failed verification, nothing was changed"))))
		return err
	}

	log.Printf("archive upload completed: location(%s) id(%s)\n", darchive.Location, bw.RandomID(darchive.DeploymentID))

	dopts := agent.DeployOptions{
		Timeout:           int64(config.Deployment.Timeout),
		Heartbeat:         int64(gctx.Heartbeat),
		SilenceDeployLogs: gctx.Silent,
	}

	if err = agentutil.ReliableDispatch(
		gctx.Context,
		dispatcher,
		agent.NewDeployCommand(local, agent.DeployCommandDone(displayname, darchive.DeployOption, dopts.DeployOption)),
	); err != nil {
		return errors.Wrap(err, "unable to mark the forced archive as deployed")
	}

	if gctx.NoRestart {
		return nil
	}

	peers := agent.NodesToPeers(c.Members()...)
	if gctx.Canary {
		peers = agent.NodesToPeers(c.Get(rendezvous.Auto()))
	}

	if gctx.Filter != nil {
		peers = deployment.ApplyFilter(gctx.Filter, peers...)
	}

	// shutdown visits one node at a time, stopping at the first failure.
	return errors.Wrap(agentutil.Shutdown(gctx.Context, agentutil.PeerSet(peers), d), "restart failed")
}

// archivingPeers filters the peers down to those that archive deployments.
func archivingPeers(peers ...*agent.Peer) (archivers []*agent.Peer) {
	for _, p := range peers {
		if p.Archive {
			archivers = append(archivers, p)
		}
	}

	return archivers
}

// fingerprint returns the size and sha256 of the file.
func fingerprint(f *os.File) (size int64, sum []byte, err error) {
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return 0, nil, err
	}

	h := sha256.New()
	if size, err = io.Copy(h, f); err != nil {
		return 0, nil, err
	}

	return size, h.Sum(nil), nil
}

// verifyUpload asserts the archive returned by storage matches what was uploaded.
func verifyUpload(a *agent.Archive, sum []byte, commit string) error {
	switch {
	case a == nil:
		return errorsx.String("storage returned no archive")
	case a.Location == "":
		return errorsx.String("storage returned an archive without a location")
	case !bytes.Equal(a.Checksum, sum):
		return errorsx.String("storage archive checksum does not match the uploaded archive")
	case !bytes.Equal(a.DeploymentID, a.Checksum):
		return errorsx.String("storage archive deployment id does not match its checksum")
	case a.Commit != commit:
		return fmt.Errorf("storage archive commit mismatch: expected(%s) actual(%s)", commit, a.Commit)
	}

	return nil
}

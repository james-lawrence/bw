package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"bitbucket.org/jatone/bearded-wookie/agent"
	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type agentCmd struct {
	*global
	network     *net.TCPAddr
	server      *grpc.Server
	listener    net.Listener
	credentials string
	upnpEnabled bool
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(
		parent,
		clusterCmdOptionBind(
			&net.TCPAddr{
				IP:   t.global.systemIP,
				Port: 2001,
			},
		),
	)

	parent.Flag("upnp-enabled", "enable upnp forwarding for the agent").Default(strconv.FormatBool(t.upnpEnabled)).Hidden().BoolVar(&t.upnpEnabled)
	parent.Flag("agent-bind", "network interface to listen on").Default(t.network.String()).TCPVar(&t.network)
	parent.Flag("credentials", "credentials to use").StringVar(&t.credentials)
	t.operatingSystemSpecificConfiguration(parent)
}

func (t *agentCmd) bind(a agent.Server) error {
	var (
		err   error
		c     clustering.Cluster
		creds credentials.TransportCredentials
	)

	log.Println("initiated binding rpc server", t.network.String())
	defer log.Println("completed binding rpc server", t.network.String())

	if t.listener, err = net.Listen("tcp", t.network.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.network)
	}

	rootdir := filepath.Join(t.global.user.HomeDir, credentialsDirDefault, stringsx.DefaultIfBlank(t.credentials, credentialsDefault))
	if creds, err = buildTLSServer(filepath.Join(rootdir, tlsserverKeyDefault), filepath.Join(rootdir, tlsserverCertDefault), filepath.Join(rootdir, tlscaCertDefault)); err != nil {
		return errors.WithStack(err)
	}

	t.server = grpc.NewServer(
		grpc.Creds(creds),
	)

	agent.RegisterServer(
		t.server,
		a,
	)

	options := []clustering.Option{
		clustering.OptionDelegate(cp.NewLocal([]byte{})),
		clustering.OptionLogger(os.Stderr),
	}

	if c, err = t.global.cluster.Join(options...); err != nil {
		return errors.Wrap(err, "failed to join cluster")
	}

	go t.server.Serve(t.listener)
	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()

		log.Println("left cluster", c.Shutdown())
		log.Println("agent shutdown", t.listener.Close())
	}()

	return nil
}

func buildTLSServer(keyp, certp, cap string) (creds credentials.TransportCredentials, err error) {
	var (
		cert tls.Certificate
		ca   []byte
	)

	if cert, err = tls.LoadX509KeyPair(certp, keyp); err != nil {
		return nil, errors.WithStack(err)
	}

	pool := x509.NewCertPool()
	if ca, err = ioutil.ReadFile(cap); err != nil {
		return nil, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	creds = credentials.NewTLS(
		&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{cert},
			ClientCAs:    pool,
		},
	)

	return creds, nil
}

func buildTLSClient(servername, keyp, certp, cap string) (creds credentials.TransportCredentials, err error) {
	var (
		cert tls.Certificate
		ca   []byte
	)

	log.Println("loading client cert", certp)
	log.Println("loading client key", keyp)
	log.Println("loading authority cert", cap)
	log.Println("using server name", servername)
	if cert, err = tls.LoadX509KeyPair(certp, keyp); err != nil {
		return nil, errors.WithStack(err)
	}

	pool := x509.NewCertPool()
	if ca, err = ioutil.ReadFile(cap); err != nil {
		return nil, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	creds = credentials.NewTLS(
		&tls.Config{
			ServerName:   servername,
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
		},
	)

	return creds, nil
}

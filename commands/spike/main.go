package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

func main() {
	var (
		root    string
		address = &net.TCPAddr{
			IP:   systemx.HostIP(systemx.HostnameOrLocalhost()),
			Port: 2000,
		}
	)
	app := kingpin.New("spike", "spike command line for testing functionality")

	server := app.Command("server", "server").Action(server(&root, address))
	server.Flag("bind", "bind address").TCPVar(&address)
	server.Flag("config", "configuration file").StringVar(&root)
	client := app.Command("client", "client").Action(client(&root, address))
	client.Flag("config", "configuration file").StringVar(&root)
	client.Flag("dial", "dial").TCPVar(&address)
	// _ = app.Command("agent", "agent server").Action(agentx).Default()
	// _ = app.Command("client", "client cli").Action(deploy)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		log.Println("boom", err)
	}
}

func client(root *string, address *net.TCPAddr) func(*kingpin.ParseContext) error {
	return func(*kingpin.ParseContext) (err error) {
		var (
			config = agent.NewConfigClient()
			cs     *tls.Config
			conn   net.Conn
		)
		path := stringsx.DefaultIfBlank(*root, filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), bw.DefaultEnvironmentName))
		if err = bw.ExpandAndDecodeFile(path, &config); err != nil {
			return errors.Wrap(err, "failed to decode config")
		}

		if cs, err = config.TLSConfig.BuildClient(); err != nil {
			return errors.WithStack(err)
		}

		if conn, err = tls.Dial("tcp", address.String(), cs); err != nil {
			return errors.WithStack(err)
		}
		defer conn.Close()
		for t := range time.Tick(time.Second) {
			if _, err = conn.Write([]byte(fmt.Sprintf("%s: hello world\n%s: bizz bazz\n", t, t))); err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}
}

func server(root *string, address *net.TCPAddr) func(*kingpin.ParseContext) error {
	return func(*kingpin.ParseContext) (err error) {
		var (
			config agent.Config = agent.NewConfig()
			cs     *tls.Config
			l      net.Listener
		)

		path := stringsx.DefaultIfBlank(*root, bw.DefaultLocation(bw.DefaultAgentConfig, ""))
		if err = bw.ExpandAndDecodeFile(path, &config); err != nil {
			return err
		}

		if cs, err = config.TLSConfig.BuildServer(); err != nil {
			return errors.WithStack(err)
		}

		if l, err = net.ListenTCP(address.Network(), address); err != nil {
			return errors.WithStack(err)
		}

		l = tls.NewListener(l, cs)

		for {
			var (
				conn net.Conn
			)

			if conn, err = l.Accept(); err != nil {
				return errors.Wrap(err, "failed to accept connection")
			}

			go func(c net.Conn) {
				s := bufio.NewScanner(c)
				for s.Scan() {
					log.Println("read", s.Text())
				}
			}(conn)
		}

		return nil
	}
}

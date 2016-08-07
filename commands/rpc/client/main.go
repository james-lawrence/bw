package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"

	"github.com/kballard/go-shellquote"
	"gopkg.in/alecthomas/kingpin.v2"

	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
)

//
// var (
// 	serf_server_ip_address = flag.String("serf-ipaddress", "127.0.0.1", "rpc server ip address to connect to. defaults to localhost.")
// 	serf_server_port       = flag.String("serf-port", "5000", "rpc server ip address to connect to. defaults to localhost.")
//
// 	rpc_server_ip_address = flag.String("rpc-ipaddress", "127.0.0.1", "rpc server ip address to connect to. defaults to localhost.")
// 	rpc_server_port       = flag.String("rpc-port", "2000", "rpc server port to connect to. defaults to 2000.")
//
// 	client_port = flag.Int("port", 5001, "port for client")
// )

func main() {
	var (
		address         = &net.TCPAddr{}
		clusterAddress  = &net.TCPAddr{}
		local           = &net.TCPAddr{}
		installPackages []string
		args            []string
		// cachestore      bool
	)

	app := kingpin.New("spike", "spike command line for testing functionality")
	app.Flag("cluster-address", "cluster server address").Default("127.0.0.1:5000").TCPVar(&clusterAddress)
	app.Flag("rpc-address", "rpc server address").Default("127.0.0.1:2000").TCPVar(&address)
	app.Flag("cluster-local-address", "local cluster network to bind").Default("localhost:5001").TCPVar(&local)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		log.Fatalln("failed to parse initialization arguments:", err)
	}

	commands := kingpin.New("commands", "commands")
	checksum := commands.Command("checksum", "retrieve system checksum")
	install := commands.Command("install", "install package")
	install.Arg("packages", "packages to install").StringsVar(&installPackages)

	quit := commands.Command("quit", "quit the application")

	fmt.Println("creating serf client")
	serfClient, err := serfdom.NewDefault("client", local.IP.String(), local.Port)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if &serfClient != nil {
			fmt.Println("leaving serf cluster")
			serfClient.Leave()
			fmt.Println("shutting down serf client connection")
			serfClient.Shutdown()
		}
	}()

	fmt.Println("joining serf cluster")

	n, err := serfClient.Join([]string{clusterAddress.String()}, true)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("Joined a cluster of size", n)

	rpcClient, err := rpc.Dial("tcp", address.String())
	if err != nil {
		log.Panic(err)
	}
	deployClient := adapters.DeploymentClient{Client: rpcClient}

	instances, err := serfClient.Instances()
	fmt.Println("Instances:", instances, "error:", err)
	instances, err = serfClient.Filter(cluster.NeverMatch)
	fmt.Println("Never Match Filter: Instances:", instances, "error:", err)
	instances, err = serfClient.Filter(cluster.AlwaysMatch)
	fmt.Println("Always Match Filter: Instances:", instances, "error:", err)

	copy(args, os.Args[1:])
	reader := bufio.NewReader(os.Stdin)
	for {
		var (
			command string
			input   string
			err     error
		)
		installPackages = []string{}
		args = []string{}

		fmt.Print(">")
		if input, err = reader.ReadString('\n'); err != nil {
			fmt.Println("Scan Error:", err)
			continue
		}

		if args, err = shellquote.Split(input); err != nil {
			fmt.Println("Input Error:", err)
			continue
		}

		if command, err = commands.Parse(args); err != nil {
			continue
		}

		switch command {
		case install.FullCommand():
			if err = deployClient.InstallPackages(installPackages...); err != nil {
				log.Println("failed to install: ", err, installPackages)
			}
		case checksum.FullCommand():
			var (
				hash []byte
			)
			log.Println("Retrieving checksum")
			hash, err = deployClient.SystemStateChecksum()
			if err != nil {
				log.Println("failed to retrieve checksum:", err)
				continue
			}

			log.Println("system checksum", hex.EncodeToString(hash))
		case quit.FullCommand():
			return
		}
	}
}

package main

import (
	"context"
	"log"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bitbucket.org/jatone/bearded-wookie/x/debug"
	"bitbucket.org/jatone/bearded-wookie/x/netx"

	"gopkg.in/alecthomas/kingpin.v2"
)

type core struct {
	Agent    *agent
	Deployer *deployer
}

func main() {
	var (
		err             error
		cleanup, cancel = context.WithCancel(context.Background())
		c               = &cluster{}
		system          = core{
			Agent: &agent{
				cluster:  c,
				listener: netx.NewNoopListener(),
				server:   rpc.NewServer(),
			},
			Deployer: &deployer{
				cluster: c,
				cancel:  cancel,
			},
		}
	)

	app := kingpin.New("bearded-wookie", "deployment system")
	system.Agent.configure(app.Command("agent", "agent that manages deployments").Default())
	system.Deployer.configure(app.Command("deploy", "deploys the application"))

	go signals(cancel)

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalln("failed to parse initialization arguments:", err)
	}

	<-cleanup.Done()
	log.Println("left cluster", c.memberlist.Leave(5*time.Second))
	log.Println("cluster shutdown", c.memberlist.Shutdown())
	log.Println("agent shutdown", system.Agent.listener.Close())
}

func signals(shutdown context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Kill, os.Interrupt, syscall.SIGUSR2)

	for s := range signals {
		switch s {
		case os.Kill, os.Interrupt:
			log.Println("shutdown request received")
			shutdown()
			signal.Stop(signals)
			close(signals)
		case syscall.SIGUSR2:
			var (
				err  error
				path string
			)

			if path, err = debug.DumpRoutines(); err != nil {
				log.Println("failed to dump routines:", err)
			} else {
				log.Println("dump located at:", path)
			}
		}
	}
}

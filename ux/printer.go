package ux

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/deployment"
)

// Logging based ux
func Logging() *deployment.Events {
	events := deployment.NewEvents()
	go func() {
		for {
			select {
			case nodesFound := <-events.NodesFound:
				log.Println("nodes found", nodesFound)
			case _ = <-events.NodesCompleted:
			case stage := <-events.StageUpdate:
				switch stage {
				case deployment.StageWaitingForReady:
					log.Println("waiting for all nodes to become ready")
				case deployment.StageDeploying:
					log.Println("deploying to nodes")
				case deployment.StageDone:
					log.Println("completed")
					return
				}
				if stage == deployment.StageDone {
					return
				}
			case e := <-events.Status:
				log.Println(e.Peer.Name, e.Status)
			}
		}
	}()

	return events
}

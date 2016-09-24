package deployment

import (
	"log"
	"time"

	"github.com/hashicorp/memberlist"
)

type cluster interface {
	Members() []*memberlist.Node
}

// Deploy - trigger a deploy
func Deploy(c cluster, deployer, status func(peer *memberlist.Node) error) {
	nodes := c.Members()
	log.Println("waiting for cluster to enter ready state")
	awaitCompletion(status, nodes)
	log.Println("everything is gtg, deploying")
	for _, partition := range partition(len(nodes), len(nodes)/2) {
		deployTo(deployer, nodes[partition.Min:partition.Max])
		log.Println("awaiting completion")
		awaitCompletion(status, nodes)
	}

	log.Println("completed")
}

func partition(length, partitionSize int) []struct{ Min, Max int } {
	numFullPartitions := length / partitionSize
	partitions := make([]struct{ Min, Max int }, 0, numFullPartitions+1)
	var i int
	for ; i < numFullPartitions; i++ {
		partitions = append(partitions, struct{ Min, Max int }{Min: i * partitionSize, Max: (i + 1) * partitionSize})
	}

	if length%partitionSize != 0 { // left over
		partitions = append(partitions, struct{ Min, Max int }{Min: i * partitionSize, Max: length})
	}

	return partitions
}

func deployTo(deployer func(peer *memberlist.Node) error, nodes []*memberlist.Node) error {
	for _, peer := range nodes {
		if err := deployer(peer); err != nil {
			log.Println("failed to deploy to", peer.Name, err)
		}
	}

	return nil
}

func awaitCompletion(status func(*memberlist.Node) error, nodes []*memberlist.Node) {
	start := time.Now()
	for len(nodes) > 0 {
		remaining := make([]*memberlist.Node, 0, len(nodes))
		for _, peer := range nodes {
			s := status(peer)

			if IsReady(s) {
				continue
			}

			if IsFailed(s) {
				log.Println(peer.Name, "failed", s)
				continue
			}

			remaining = append(remaining, peer)
		}

		log.Printf(
			"%3s waiting for %d node(s) to finish\n",
			time.Now().Sub(start),
			len(remaining),
		)

		nodes = remaining
		time.Sleep(time.Second)
	}
}

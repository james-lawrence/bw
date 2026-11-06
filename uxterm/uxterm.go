package uxterm

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/pterm/pterm"
)

func PrintNode(nodeinfo *agent.StatusResponse) error {
	var deployments = pterm.TableData{
		{"time", "deployment id", "stage"},
	}

	for _, d := range nodeinfo.Deployments {
		deployments = append(deployments, []string{
			time.Unix(d.Archive.Ts, 0).UTC().String(),
			bw.RandomID(d.Archive.DeploymentID).String(),
			d.Stage.String(),
		})
	}

	pterm.Printfln("Node: %s", PeerString(nodeinfo.Peer))
	if err := pterm.DefaultTable.WithHasHeader().WithData(deployments).Render(); err != nil {
		return err
	}
	pterm.Println()

	return nil
}

func PrintQuorum(quoruminfo *agent.InfoResponse) error {
	var quorum = pterm.TableData{
		{"name", "address", "port"},
	}

	for _, p := range quoruminfo.Quorum {
		quorum = append(quorum, []string{
			p.Name, p.Ip, strconv.Itoa(int(p.P2PPort)),
		})
	}

	pterm.Printfln("Leader    : %s", PeerString(quoruminfo.Leader))
	pterm.Printfln("Latest    : %s", DeploymentString(quoruminfo.Deployed))
	pterm.Printfln("Deploying : %s", DeploymentString(quoruminfo.Deploying))
	return pterm.DefaultTable.WithHasHeader().WithData(quorum).Render()
}

// NodeName shortens the name of a node to the uuid format when the name is a hex encoded identifier.
func NodeName(name string) string {
	if raw, err := hex.DecodeString(name); err == nil && len(raw) >= 16 {
		return bw.RandomID(raw).String()
	}

	return name
}

// NodeMode describes the role of the peer within the cluster: leader, quorum, follower or peer.
// follower is a quorum member that isn't the leader, quorum is a quorum member while there is no leader.
func NodeMode(info *agent.InfoResponse, p *agent.Peer) string {
	if info.GetLeader().GetName() == p.Name {
		return "leader"
	}

	for _, q := range info.GetQuorum() {
		if q.Name == p.Name {
			if info.GetLeader() == nil {
				return "quorum"
			}

			return "follower"
		}
	}

	return "peer"
}

// PrintNodes prints the nodes of the cluster along with the deployment state of the quorum.
func PrintNodes(info *agent.InfoResponse, peers []*agent.Peer) error {
	var nodes = pterm.TableData{
		{"archive", "mode", "name", "address", "port"},
	}

	for _, p := range peers {
		nodes = append(nodes, []string{
			strconv.FormatBool(p.Archive), NodeMode(info, p), NodeName(p.Name), p.Ip, strconv.Itoa(int(p.P2PPort)),
		})
	}

	pterm.Printfln("Leader    : %s", PeerString(info.Leader))
	pterm.Printfln("Latest    : %s", DeploymentString(info.Deployed))
	pterm.Printfln("Deploying : %s", DeploymentString(info.Deploying))
	return pterm.DefaultTable.WithHasHeader().WithData(nodes).Render()
}

func PeerString(p *agent.Peer) string {
	if p == nil {
		return "None"
	}

	return fmt.Sprintf("%s - %s - %s:%d", p.Name, p.Status, p.Ip, p.P2PPort)
}

func DeploymentString(c *agent.DeployCommand) string {
	if c == nil || c.Archive == nil {
		return "None"
	}

	return fmt.Sprintf("%s - %s - %s", bw.RandomID(c.Archive.DeploymentID), c.Initiator, c.Command.String())
}

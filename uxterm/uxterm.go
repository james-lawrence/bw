package uxterm

import (
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
	pterm.DefaultTable.WithHasHeader().WithData(deployments).Render()
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
	pterm.DefaultTable.WithHasHeader().WithData(quorum).Render()

	return nil
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

	return fmt.Sprintf("%s - %s - %s", bw.RandomID(c.Archive.DeploymentID), c.Archive.Initiator, c.Command.String())
}

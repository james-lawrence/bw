package uxterm

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/pterm/pterm"
)

type rendezvous interface {
	GetN(int, []byte) []*memberlist.Node
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

	pterm.Printfln("Latest : %s", DeploymentString(quoruminfo.Deployed))
	pterm.Printfln("Ongoing: %s", DeploymentString(quoruminfo.Deploying))
	pterm.Printfln("Leader : %s", PeerString(quoruminfo.Leader))
	pterm.DefaultTable.WithHasHeader().WithData(quorum).Render()

	return nil
}

func PeerString(p *agent.Peer) string {
	if p == nil {
		return "None"
	}

	return fmt.Sprintf("%s - %s:%d", p.Name, p.Ip, p.P2PPort)
}

func DeploymentString(c *agent.DeployCommand) string {
	if c == nil || c.Archive == nil {
		return "None"
	}

	return fmt.Sprintf("%s - %s - %s", bw.RandomID(c.Archive.DeploymentID), c.Archive.Initiator, c.Command.String())
}

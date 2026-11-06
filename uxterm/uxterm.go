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

// ModeString describes the role of the node within the cluster.
func ModeString(m agent.PeerMode) string {
	switch m {
	case agent.PeerMode_PeerModeLeader:
		return "leader"
	case agent.PeerMode_PeerModeFollower:
		return "follower"
	default:
		return "peer"
	}
}

// PrintNodes prints the nodes of the cluster along with the deployment state of the quorum.
func PrintNodes(info *agent.InfoResponse, peers []*agent.Peer) error {
	rendered, err := RenderNodes(info, peers)
	if err != nil {
		return err
	}

	pterm.Println(rendered)
	return nil
}

// RenderNodes renders the nodes of the cluster along with the deployment state of the quorum.
func RenderNodes(info *agent.InfoResponse, peers []*agent.Peer) (string, error) {
	var nodes = pterm.TableData{
		{"archive", "mode", "name", "address", "port", "deployed"},
	}

	for _, p := range peers {
		nodes = append(nodes, []string{
			strconv.FormatBool(p.Archive), ModeString(p.Mode), NodeName(p.Name), p.Ip, strconv.Itoa(int(p.P2PPort)), NodeDeployed(info, p),
		})
	}

	table, err := pterm.DefaultTable.WithHasHeader().WithData(nodes).Srender()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Leader    : %s\nLatest    : %s\nDeploying : %s\n%s",
		PeerString(info.Leader), DeploymentString(info.Deployed), DeploymentString(info.Deploying), table,
	), nil
}

// NodeDeployed describes the archive the quorum last saw the node complete a deployment of, "-" when the
// node hasn't reported one.
func NodeDeployed(info *agent.InfoResponse, p *agent.Peer) string {
	for _, m := range info.GetCompleted() {
		if m.GetPeer().GetName() != p.Name {
			continue
		}

		if id := m.GetDeploy().GetArchive().GetDeploymentID(); len(id) > 0 {
			return bw.RandomID(id).String()
		}
	}

	return "-"
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

package agentcmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/pkg/errors"
)

// Detected describes the cloud environment of the host.
type Detected struct {
	AWS    bool
	GCloud bool
}

// Probes are the checks used to detect the environment, injectable for testing.
type Probes struct {
	AWS    func(context.Context) bool
	GCloud func(context.Context) bool
}

// DefaultProbes query the instance metadata services of the cloud providers.
func DefaultProbes() Probes {
	return Probes{
		AWS: func(ctx context.Context) bool {
			sess, err := session.NewSession()
			if err != nil {
				return false
			}
			return ec2metadata.New(sess).AvailableWithContext(ctx)
		},
		GCloud: func(context.Context) bool {
			return metadata.OnGCE()
		},
	}
}

// Detect which cloud providers the host is running within.
func Detect(ctx context.Context, p Probes) Detected {
	ctx, done := context.WithTimeout(ctx, 2*time.Second)
	defer done()

	return Detected{
		AWS:    p.AWS(ctx),
		GCloud: p.GCloud(ctx),
	}
}

// Environment returns the environment settings for the detected providers.
func (t Detected) Environment() (r []string) {
	if t.AWS {
		r = append(r, bw.EnvAgentClusterEnableAWSAutoscaling+"=true")
	}
	if t.GCloud {
		r = append(r, bw.EnvAgentClusterEnableGoogleCloudPool+"=true")
	}
	return r
}

// MergeEnvironment sets the provided KEY=VALUE settings within the existing environment file contents.
// existing settings for the same keys are replaced in place, everything else is preserved, new keys are appended.
func MergeEnvironment(existing string, settings ...string) string {
	lines := strings.Split(strings.TrimRight(existing, "\n"), "\n")
	if existing == "" {
		lines = nil
	}

	for _, setting := range settings {
		key, _, _ := strings.Cut(setting, "=")
		replaced := false
		for i, l := range lines {
			if k, _, ok := strings.Cut(l, "="); ok && strings.TrimSpace(k) == key {
				lines[i], replaced = setting, true
			}
		}
		if !replaced {
			lines = append(lines, setting)
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

// CmdBootstrapEnv auto detects the peering configuration of the host and writes it to the agent environment file.
type CmdBootstrapEnv struct {
	Alternates []*net.TCPAddr `name:"agent-address-bindings" alias:"agent-p2p-alternates" help:"alternate addresses for the agent to bind" default:"127.0.0.1:2000" env:"${env_bw_agent_bind_secondary}"`
	P2PPort    int            `name:"p2p-port" help:"p2p discovery port to write, not written when unset" env:"${env_bw_agent_cluster_p2p_discovery_port}"`
	Output     string         `name:"output" help:"environment file to write, existing settings are preserved" default:"${vars_bw_default_agent_environment}"`
}

func (t CmdBootstrapEnv) Run(ctx *cmdopts.Global) error {
	existing, err := os.ReadFile(t.Output)
	if err != nil && !os.IsNotExist(err) {
		return errors.WithStack(err)
	}

	settings := Detect(ctx.Context, DefaultProbes()).Environment()

	// the agent binds to the hostname's address by default (e.g. the 10.x.x.x address on gcp), the alternates
	// default to a loopback address so local services such as nginx can always reach the agent.
	addrs := make([]string, 0, len(t.Alternates))
	for _, a := range t.Alternates {
		addrs = append(addrs, a.String())
	}
	settings = append(settings, fmt.Sprintf("%s=%s", bw.EnvAgentP2PAlternatesBind, strings.Join(addrs, ",")))

	if t.P2PPort > 0 {
		settings = append(settings, fmt.Sprintf("%s=%d", bw.EnvAgentClusterP2PDiscoveryPort, t.P2PPort))
	}

	if len(settings) == 0 {
		log.Println("nothing to write")
		return nil
	}

	if err = os.MkdirAll(filepath.Dir(t.Output), 0755); err != nil {
		return errors.WithStack(err)
	}

	if err = os.WriteFile(t.Output, []byte(MergeEnvironment(string(existing), settings...)), 0640); err != nil {
		return errors.WithStack(err)
	}

	log.Println("wrote", t.Output, strings.Join(settings, " "))
	return nil
}

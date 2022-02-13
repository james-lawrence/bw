package main

import (
	"net"
	"os"
	"regexp"

	"github.com/alecthomas/kingpin"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/deployment"
)

type deployCmd struct {
	global         *global
	environment    string
	deploymentID   string
	filteredIP     []net.IP
	filteredRegex  []*regexp.Regexp
	snapshotOutput *os.File
	concurrency    int64
	canary         bool
	debug          bool
	ignoreFailures bool
	silenceLogs    bool
	insecure       bool
}

func (t *deployCmd) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	tls := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("insecure", "skips verifying tls host").Default("false").BoolVar(&t.insecure)
		return cmd
	}

	deployOptions := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("ignoreFailures", "ignore when an agent fails its deploy").Default("false").BoolVar(&t.ignoreFailures)
		cmd.Flag("silenceLogs", "prevents the logs from being written for a deploy").Default("false").BoolVar(&t.silenceLogs)
		return cmd
	}

	t.snapshotCmd(common(parent.Command("snapshot", "generate a deployment archive without uploading it anywhere")))
	t.deployCmd(deployOptions(tls(common(parent.Command("deploy", "deploy to nodes within the cluster").Default()))))
	t.redeployCmd(deployOptions(tls(common(parent.Command("archive", "redeploy an archive to nodes within the cluster")))))
	t.localCmd(common(parent.Command("local", "deploy to the local system")))
	t.cancelCmd(tls(common(parent.Command("cancel", "cancel any current deploy"))))
}

func (t *deployCmd) localCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("debug", "leaves artifacts on the filesystem for debugging").BoolVar(&t.debug)
	return parent.Action(func(*kingpin.ParseContext) error {
		return deploy.Locally(&deploy.Context{
			Environment: t.environment,
			Insecure:    t.insecure,
			Context:     t.global.ctx,
			CancelFunc:  t.global.shutdown,
			WaitGroup:   t.global.cleanup,
			Concurrency: t.concurrency,
			Canary:      t.canary,
		}, t.debug)
	})
}

func (t *deployCmd) cancelCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(func(*kingpin.ParseContext) error {
		return deploy.Cancel(&deploy.Context{
			Context:     t.global.ctx,
			CancelFunc:  t.global.shutdown,
			WaitGroup:   t.global.cleanup,
			Environment: t.environment,
			Insecure:    t.insecure,
			Concurrency: t.concurrency,
			Canary:      t.canary,
		})
	})
}

func (t *deployCmd) deployCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("canary", "deploy only to the canary server - this option is used to consistent select a single server for deployments without having to manually filter").BoolVar(&t.canary)
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	parent.Flag("concurrency", "control how many nodes are deployed to simultaneously").Int64Var(&t.concurrency)
	return parent.Action(func(ctx *kingpin.ParseContext) error {
		filters := make([]deployment.Filter, 0, len(t.filteredRegex))
		for _, n := range t.filteredRegex {
			filters = append(filters, deployment.Named(n))
		}

		for _, n := range t.filteredIP {
			filters = append(filters, deployment.IP(n))
		}

		// need a filter to be present for the canary to work.
		if t.canary {
			filters = append(filters, deployment.AlwaysMatch)
		}

		return deploy.Into(&deploy.Context{
			Context:        t.global.ctx,
			CancelFunc:     t.global.shutdown,
			WaitGroup:      t.global.cleanup,
			Environment:    t.environment,
			Insecure:       t.insecure,
			IgnoreFailures: t.ignoreFailures,
			SilenceLogs:    t.silenceLogs,
			Concurrency:    t.concurrency,
			Canary:         t.canary,
			Filter:         deployment.Or(filters...),
			AllowEmpty:     len(filters) == 0,
		})
	})
}

func (t *deployCmd) redeployCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("canary", "deploy only to the canary server - this option is used to consistent select a single server for deployments without having to manually filter").BoolVar(&t.canary)
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	parent.Arg("archive", "deployment ID to redeploy").StringVar(&t.deploymentID)
	return parent.Action(func(ctx *kingpin.ParseContext) error {
		filters := make([]deployment.Filter, 0, len(t.filteredRegex))
		for _, n := range t.filteredRegex {
			filters = append(filters, deployment.Named(n))
		}

		for _, n := range t.filteredIP {
			filters = append(filters, deployment.IP(n))
		}

		// need a filter to be present for the canary to work.
		if t.canary {
			filters = append(filters, deployment.AlwaysMatch)
		}

		return deploy.Redeploy(&deploy.Context{
			Context:     t.global.ctx,
			CancelFunc:  t.global.shutdown,
			WaitGroup:   t.global.cleanup,
			Concurrency: t.concurrency,
			Canary:      t.canary,
			Filter:      deployment.Or(filters...),
			AllowEmpty:  len(filters) == 0,
		}, t.deploymentID)
	})
}

func (t *deployCmd) snapshotCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("output", "file to write to. by default archive is written to stdout").Short('o').OpenFileVar(&t.snapshotOutput, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	return parent.Action(func(*kingpin.ParseContext) error {
		if t.snapshotOutput == nil {
			t.snapshotOutput = os.Stdout
		}

		return deploy.Snapshot(deploy.Context{
			Environment: t.environment,
			Insecure:    t.insecure,
			Context:     t.global.ctx,
			CancelFunc:  t.global.shutdown,
			WaitGroup:   t.global.cleanup,
			Concurrency: t.concurrency,
			Canary:      t.canary,
		}, t.snapshotOutput)
	})
}

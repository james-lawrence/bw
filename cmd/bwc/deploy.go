package main

import (
	"net"
	"regexp"
)

type deployCmd struct {
	Cluster  cmdDeployEnvironment `cmd:"env"`
	Locally  cmdDeployLocal       `cmd:"locally"`
	Snapshot cmdDeploySnapshot    `cmd:"snapshot"`
	Previous cmdDeployRedeploy    `cmd:"previous"`
	Cancel   cmdDeployCancel      `cmd:"cancel"`
}

type cmdDeployEnvironment struct {
	Canary      bool             `name:"canary" help:"deploy to the canary server" default:"false"`
	Names       []*regexp.Regexp `name:"name" help:"regex to match names against"`
	IPs         []net.IP         `name:"ip" help:"match against the provided IP addresses"`
	Concurrency int64            `name:"concurrency" help:"number of nodes allowed to deploy simultaneously"`
	Environment string           `arg:"" name:"environment" predictor:"bw.environment"`
}

func (t cmdDeployEnvironment) Run(ctx *Global) error {
	return nil
}

type cmdDeployLocal struct{}

func (t cmdDeployLocal) Run(ctx *Global) error {
	return nil
}

type cmdDeploySnapshot struct{}

func (t cmdDeploySnapshot) Run(ctx *Global) error {
	return nil
}

type cmdDeployRedeploy struct{}

func (t cmdDeployRedeploy) Run(ctx *Global) error {
	return nil
}

type cmdDeployCancel struct{}

func (t cmdDeployCancel) Run(ctx *Global) error {
	return nil
}

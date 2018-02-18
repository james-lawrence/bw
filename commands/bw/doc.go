// Copyright James Lawrence
// All Rights Reserved

// Package bw provides commands to run agents and clients to perform deploys.
//
// Concepts
// bw is built around four concepts, a cluster, a quorum, agents and clients.
//
// cluster - a set of agents.
//
// quorum - a dynamic subset of the agents within the cluster responsible for
// coordinating deploys.
//
// agent - a system that will receive a deploy.
//
// client - some process that interacts with a cluster. usually to initiate deploys
// or add some functionality to the cluster, such as notifications, health checks, etc.
//
// Commandline Interface
// commandline settings have the following precedence: environment, commandline argument, configuration file.
// e.g.) BEARDED_WOOKIE_EXAMPLE=foo bw agent example=bar; exampe would take on the value of foo.
//
// Examples
//  bw agent --cluster=example.com:2001
//  bw agent --agent-name="node1" --agent-bind=:2000 --cluster-bind=:2001 --cluster-bind-raft=:2002 --agent-torrent=:2003
//  bw agent --agent-config=".bw/agent.config"
package main

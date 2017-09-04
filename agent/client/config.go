package client

import "bitbucket.org/jatone/bearded-wookie/agent"

// Config ...
type Config struct {
	Address string
	agent.TLSConfig
}

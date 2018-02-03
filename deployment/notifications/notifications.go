package notifications

import "github.com/james-lawrence/bw/agent"

type creator func() alerts.Notifier

var Plugins = map[string]creator{}

func Add(name string, creator creator) {
	Plugins[name] = creator
}

type Notifier interface {
	Notify(a agent.Archive)
}

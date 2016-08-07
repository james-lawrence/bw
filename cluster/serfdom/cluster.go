package serfdom

import "net"
import "github.com/hashicorp/serf/serf"
import "bitbucket.org/jatone/bearded-wookie/cluster"

func NewDefault(name, addr string, port int) (serfdom, error) {
	var s *serf.Serf
	var err error
	var config *serf.Config = serf.DefaultConfig()
	config.NodeName = name
	config.EventCh = make(chan serf.Event, 64)
	config.MemberlistConfig.BindAddr = addr
	config.MemberlistConfig.BindPort = port

	s, err = serf.Create(config)

	return New(s), err
}

func New(s *serf.Serf) serfdom {
	return serfdom{s}
}

type serfdom struct {
	*serf.Serf
}

// Filters instances within the cluster by the specified filter
func (t serfdom) Filter(filter cluster.Filter) ([]cluster.Instance, error) {
	return t.members(filter), nil
}

// Special case that returns every instance in the cluster
func (t serfdom) Instances() ([]cluster.Instance, error) {
	return t.members(cluster.AlwaysMatch), nil
}

func (t serfdom) members(filter cluster.Filter) []cluster.Instance {
	instances := make([]cluster.Instance, 0, 200)
	for _, member := range t.Serf.Members() {
		instance := Instance{member}
		if filter.Match(instance) {
			instances = append(instances, instance)
		}
	}
	return instances
}

type Instance struct {
	serf.Member
}

func (t Instance) Name() string {
	return t.Member.Name
}

func (t Instance) IP() net.IP {
	return t.Member.Addr
}

func (t Instance) Tags() map[string]string {
	return t.Member.Tags
}

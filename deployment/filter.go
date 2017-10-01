package deployment

import (
	"net"
	"regexp"

	"bitbucket.org/jatone/bearded-wookie/agent"
)

// And(Role("app"), Named("host1"))
// OR(Role("app"), Named("host1"))
// XOR(Role("app"), Named("host1"))
// And(And(Role("app"), Named("host1")), NOT(Role("app"), Named("host1"))

// Filter - Matches against Instances, returns true if the agent.Peer matches the filter,
// false otherwise
// matches := Role("app").Match(agent.Peer)
// matches := Named("host1").Match(agent.Peer)
// type Filter interface {
// 	// Returns true if the agent.Peer matches the criteria, false otherwise
// 	Match(agent.Peer) bool
// }

// Filter determines if a node should be deployed to based on some conditions.
type Filter interface {
	Match(agent.Peer) bool
}

// Named matches an agent.Peer by name.
func Named(r *regexp.Regexp) Filter {
	return FilterFunc(func(i agent.Peer) bool {
		return r.MatchString(i.Name)
	})
}

// IP matches an agent.Peer by ip.
func IP(ip net.IP) Filter {
	return FilterFunc(func(i agent.Peer) bool {
		return ip.Equal(net.ParseIP(i.Ip))
	})
}

// FilterFunc - func that matches against Instances
type FilterFunc func(agent.Peer) bool

// Match - See Filter.Match
func (t FilterFunc) Match(i agent.Peer) bool {
	return t(i)
}

// Implement the FilterFunc interface
func always(agent.Peer) bool {
	return true
}

// AlwaysMatch - Always returns true
var AlwaysMatch = FilterFunc(always)

// Implement the FilterFunc interface
func never(agent.Peer) bool {
	return false
}

// NeverMatch - Always returns false
var NeverMatch = FilterFunc(never)

// FilterSet - a slice of related filters.
type FilterSet struct {
	Filters []Filter
}

// Composite filters

// And - A composite filter, returns true iff all the component
// filters match a given agent.Peer.
func And(filters ...Filter) Filter {
	return and{filters}
}

// Or - A composite filter, returns false iff all the component filters
// match a given agent.Peer.
func Or(filters ...Filter) Filter {
	return or{filters}
}

type and FilterSet

func (t and) Match(i agent.Peer) bool {
	for _, filter := range t.Filters {
		if !filter.Match(i) {
			return false
		}
	}

	return true
}

type or FilterSet

func (t or) Match(i agent.Peer) bool {
	for _, filter := range t.Filters {
		if filter.Match(i) {
			return true
		}
	}

	return false
}

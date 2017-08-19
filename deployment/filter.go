package deployment

import (
	"net"
	"regexp"

	"github.com/hashicorp/memberlist"
)

// And(Role("app"), Named("host1"))
// OR(Role("app"), Named("host1"))
// XOR(Role("app"), Named("host1"))
// And(And(Role("app"), Named("host1")), NOT(Role("app"), Named("host1"))

// Filter - Matches against Instances, returns true if the *memberlist.Node matches the filter,
// false otherwise
// matches := Role("app").Match(*memberlist.Node)
// matches := Named("host1").Match(*memberlist.Node)
// type Filter interface {
// 	// Returns true if the *memberlist.Node matches the criteria, false otherwise
// 	Match(*memberlist.Node) bool
// }

// Filter determines if a node should be deployed to based on some conditions.
type Filter interface {
	Match(*memberlist.Node) bool
}

// Named matches an *memberlist.Node by name.
func Named(r *regexp.Regexp) Filter {
	return FilterFunc(func(i *memberlist.Node) bool {
		return r.MatchString(i.Name)
	})
}

// IP matches an *memberlist.Node by ip.
func IP(ip net.IP) Filter {
	return FilterFunc(func(i *memberlist.Node) bool {
		return ip.Equal(i.Addr)
	})
}

// FilterFunc - func that matches against Instances
type FilterFunc func(*memberlist.Node) bool

// Match - See Filter.Match
func (t FilterFunc) Match(i *memberlist.Node) bool {
	return t(i)
}

// Implement the FilterFunc interface
func always(*memberlist.Node) bool {
	return true
}

// AlwaysMatch - Always returns true
var AlwaysMatch = FilterFunc(always)

// Implement the FilterFunc interface
func never(*memberlist.Node) bool {
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
// filters match a given *memberlist.Node.
func And(filters ...Filter) Filter {
	return and{filters}
}

// Or - A composite filter, returns false iff all the component filters
// match a given *memberlist.Node.
func Or(filters ...Filter) Filter {
	return or{filters}
}

type and FilterSet

func (t and) Match(i *memberlist.Node) bool {
	for _, filter := range t.Filters {
		if !filter.Match(i) {
			return false
		}
	}

	return true
}

type or FilterSet

func (t or) Match(i *memberlist.Node) bool {
	for _, filter := range t.Filters {
		if filter.Match(i) {
			return true
		}
	}

	return false
}

type nodeInstance struct {
	n *memberlist.Node
}

func (t nodeInstance) Name() string {
	return t.n.Name
}
func (t nodeInstance) IP() net.IP {
	return t.n.Addr
}

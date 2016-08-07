package cluster

import "net"

// And(Role("app"), Named("host1"))
// OR(Role("app"), Named("host1"))
// XOR(Role("app"), Named("host1"))
// And(And(Role("app"), Named("host1")), NOT(Role("app"), Named("host1"))

// Filter - Matches against Instances, returns true if the instance matches the filter,
// false otherwise
// matches := Role("app").Match(instance)
// matches := Named("host1").Match(instance)
type Filter interface {
	// Returns true if the instance matches the criteria, false otherwise
	Match(Instance) bool
}

// Instance - Instance represents a server
type Instance interface {
	Name() string
	IP() net.IP
}

// TaggedInstance - Tagged instance.
type TaggedInstance interface {
	Instance
	Tags() map[string]string
}

// Interface - Answers questions about what servers are within the cluster.
// Also provides filtering based on conditions about the servers.
type Interface interface {
	// Filters instances within the cluster by the specified filter
	Filter(filters Filter) ([]TaggedInstance, error)
	// Special case that returns every instance in the cluster
	Instances() ([]TaggedInstance, error)
}

// FilterFunc - func that matches against Instances
type FilterFunc func(Instance) bool

// Match - See Filter.Match
func (t FilterFunc) Match(instance Instance) bool {
	return t(instance)
}

// Implement the FilterFunc interface
func always(Instance) bool {
	return true
}

// AlwaysMatch - Always returns true
var AlwaysMatch FilterFunc = FilterFunc(always)

// Implement the FilterFunc interface
func never(Instance) bool {
	return false
}

// NeverMatch - Always returns false
var NeverMatch FilterFunc = FilterFunc(never)

// FilterSet - a slice of related filters.
type FilterSet struct {
	Filters []Filter
}

// Composite filters

// And - A composite filter, returns true iff all the component
// filters match a given instance.
func And(filters ...Filter) Filter {
	return and{filters}
}

// Or - A composite filter, returns false iff all the component filters
// match a given instance.
func Or(filters ...Filter) Filter {
	return or{filters}
}

type and FilterSet

func (t and) Match(instance Instance) bool {
	for _, filter := range t.Filters {
		if !filter.Match(instance) {
			return false
		}
	}

	return true
}

type or FilterSet

func (t or) Match(instance Instance) bool {
	for _, filter := range t.Filters {
		if filter.Match(instance) {
			return true
		}
	}

	return false
}

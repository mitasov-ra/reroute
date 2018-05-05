package reroute

import "net/http"

// Filter is an interface for router filters,
// each filter must contain Run method with FilterChain parameter,
// which should be run when filtration is successful
type Filter interface {
	Run(http.ResponseWriter, *http.Request, *FilterChain) bool
}

// FilterFunc is a function with the same signature, as the Run method
// of Filter interface
type FilterFunc func(http.ResponseWriter, *http.Request, *FilterChain) bool

func (f FilterFunc) Run(w http.ResponseWriter, r *http.Request, c *FilterChain) bool {
	return f(w, r, c)
}

// FilterChain is a list of filters, added to route
type FilterChain struct {
	count   int
	current int
	filters []Filter
}

// Runs all filters in a row until the first filter failure.
// If all filters succeed, returns true
func (c *FilterChain) Run(w http.ResponseWriter, req *http.Request) bool {
	if c.count == 0 || c.current >= c.count {
		return true
	}

	filter := c.filters[c.current]
	c.current++

	defer func() { c.current = 0 }()
	return filter.Run(w, req, c)
}

// Adds filter to filter chain
func (c *FilterChain) Add(filter Filter) {
	c.filters = append(c.filters, filter)
	c.count++
}

// Adds function as a filter to filter chain
func (c *FilterChain) AddFunc(f func(http.ResponseWriter, *http.Request, *FilterChain) bool) {
	c.Add(FilterFunc(f))
}

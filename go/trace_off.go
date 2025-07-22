//go:build !snet_trace
// +build !snet_trace

package snet

// isTraceEnabled returns false when snet_trace build tag is not present
func isTraceEnabled() bool {
	return false
}

func (l *Listener) trace(format string, args ...interface{}) {
}

func (c *Conn) trace(format string, args ...interface{}) {
}

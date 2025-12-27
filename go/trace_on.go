//go:build snet_trace
// +build snet_trace

package snet

import (
	"fmt"

	"github.com/sandwich-go/logbus"
)

// isTraceEnabled returns true when snet_trace build tag is present
func isTraceEnabled() bool {
	return true
}

func (l *Listener) trace(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	logbus.Debug("Listener: "+message,
		logbus.String("component", "listener"),
		logbus.String("addr", l.Addr().String()))
}

func (c *Conn) trace(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	// Add connection details for better debugging
	fields := []logbus.Field{
		logbus.Uint64("conn_id", c.id),
		logbus.Uint64("read_count", c.readCount),
		logbus.Uint64("write_count", c.writeCount),
	}

	// Add remote address if available
	if c.base != nil {
		if remoteAddr := c.base.RemoteAddr(); remoteAddr != nil {
			fields = append(fields, logbus.String("remote_addr", remoteAddr.String()))
		}
	}

	if c.listener == nil {
		// Client connection
		logbus.Debug("Client conn: "+message, append(fields,
			logbus.String("component", "client_conn"))...)
	} else {
		// Server connection
		logbus.Debug("Server conn: "+message, append(fields,
			logbus.String("component", "server_conn"))...)
	}
}

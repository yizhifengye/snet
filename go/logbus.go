package snet

import (
	"sync"

	"github.com/sandwich-go/logbus"
	"go.uber.org/zap/zapcore"
)

var logbusInitOnce sync.Once

// InitLogbus initializes logbus for the snet package
// This can be called multiple times safely due to sync.Once
func InitLogbus() {
	logbusInitOnce.Do(func() {
		// Use Debug level when trace is enabled, Info level otherwise
		var logLevel zapcore.Level
		var devMode bool

		// Check if trace is enabled via build tag
		if isTraceEnabled() {
			logLevel = zapcore.DebugLevel
			devMode = true
		} else {
			logLevel = zapcore.InfoLevel
			devMode = false
		}

		logbus.Init(logbus.NewConf(
			logbus.WithCallerSkip(2),
			logbus.WithDev(devMode),
			logbus.WithLogLevel(logLevel),
		))
	})
}

// SetLogLevel allows runtime adjustment of log level
func SetLogLevel(level zapcore.Level) {
	logbus.Init(logbus.NewConf(
		logbus.WithCallerSkip(2),
		logbus.WithDev(level <= zapcore.DebugLevel),
		logbus.WithLogLevel(level),
	))
}

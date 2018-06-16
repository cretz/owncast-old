package util

import "log"

type Logger func(format string, v ...interface{})

var noopLogger = func(format string, v ...interface{}) {}
var NoopLogger Logger = noopLogger

var SimpleLogger Logger = log.Printf

var LogDebug Logger = NoopLogger

func LogDebugEnabled() bool { return &LogDebug != &NoopLogger }

var LogInfo Logger = SimpleLogger

func LogInfoEnabled() bool { return &LogInfo != &NoopLogger }

package log

import "log"

type Logf func(format string, v ...interface{})

var noopLogf = func(format string, v ...interface{}) {}
var NoopLogf Logf = noopLogf

var SimpleLogf Logf = log.Printf

var Debugf Logf = NoopLogf

func DebugEnabled() bool { return &Debugf != &NoopLogf }

var Infof Logf = SimpleLogf

func InfoEnabled() bool { return &Infof != &NoopLogf }

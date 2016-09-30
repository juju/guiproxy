package logger

import (
	"fmt"
	"log"
)

// Interface holds the logger interface used to log string messages.
type Interface interface {
	// Print logs messages.
	Print(string)
}

// New creates and returns a new API logger implementing Interface.
// The resulting logger will process messages using the given process function,
// and each message will be prefixed with prefix.
func New(prefix string, process func(string) string) Interface {
	if process == nil {
		process = func(msg string) string {
			return msg
		}
	}
	return &apiLogger{
		prefix:  prefix,
		process: process,
	}
}

// apiLogger implements Interface by logging API messages.
type apiLogger struct {
	prefix  string
	process func(msg string) string
}

// Print implements Interface and logs string messages.
func (l *apiLogger) Print(msg string) {
	msg = fmt.Sprintf("%s: %s", l.prefix, msg)
	logPrintln(l.process(msg))
}

// logPrintln is defined as a variable for testing purposes.
var logPrintln = func(v ...interface{}) {
	log.Println(v...)
}

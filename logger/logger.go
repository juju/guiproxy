package logger

import "log"

// Interface holds the logger interface used to log string messages.
type Interface interface {
	// Print logs messages.
	Print(string)
}

// New creates and returns a new API logger implementing Interface.
// The resulting logger will process messages using the given message modifier
// functions.
func New(modifiers ...func(string) string) Interface {
	return &apiLogger{
		modifiers: modifiers,
	}
}

// apiLogger implements Interface by logging API messages.
type apiLogger struct {
	modifiers []func(msg string) string
}

// Print implements Interface and logs string messages.
func (l *apiLogger) Print(msg string) {
	for _, modifier := range l.modifiers {
		msg = modifier(msg)
	}
	logPrintln(msg)
}

// logPrintln is defined as a variable for testing purposes.
var logPrintln = func(v ...interface{}) {
	log.Println(v...)
}

// AddPrefix returns an apiLogger message modifier that adds the given prefix
// to the message.
func AddPrefix(prefix string) func(string) string {
	return func(msg string) string {
		return prefix + ": " + msg
	}
}

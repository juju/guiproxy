package logger_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/frankban/guiproxy/logger"
)

func TestPrint(t *testing.T) {
	restore, getOutput := patchLogPrintln()
	defer restore()
	l := logger.New("my prefix", strings.ToUpper)
	l.Print("these are the voyages")
	assertEqual(t, getOutput(), "MY PREFIX: THESE ARE THE VOYAGES\n")
}

// patchLogPrintln patches the logger.LogPrintln variable so that it is
// possible to collect logs.
func patchLogPrintln() (restore func(), getOutput func() string) {
	original := *logger.LogPrintln
	var output string
	*logger.LogPrintln = func(v ...interface{}) {
		output = fmt.Sprintln(v...)
	}
	restore = func() {
		*logger.LogPrintln = original
	}
	getOutput = func() string {
		return output
	}
	return restore, getOutput
}

// assertEqual fails if the given strings are not equal.
func assertEqual(t *testing.T, obtained, expected string) {
	if obtained != expected {
		t.Fatalf("\n%q !=\n%q", obtained, expected)
	}
}

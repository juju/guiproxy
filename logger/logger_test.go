package logger_test

import (
	"fmt"
	"strings"
	"testing"

	it "github.com/frankban/guiproxy/internal/testing"
	"github.com/frankban/guiproxy/logger"
)

func TestPrint(t *testing.T) {
	restore, getOutput := patchLogPrintln()
	defer restore()
	l := logger.New()
	l.Print("these are the voyages")
	it.AssertString(t, getOutput(), "these are the voyages\n")
}

func TestModifiers(t *testing.T) {
	restore, getOutput := patchLogPrintln()
	defer restore()
	l := logger.New(logger.AddPrefix("my prefix"), strings.ToUpper)
	l.Print("of the starship enterprise")
	it.AssertString(t, getOutput(), "MY PREFIX: OF THE STARSHIP ENTERPRISE\n")
}

func TestAddPrefix(t *testing.T) {
	f := logger.AddPrefix(">>> answer")
	it.AssertString(t, f("42"), ">>> answer: 42")
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

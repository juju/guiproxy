package logger_test

import (
	"fmt"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/juju/guiproxy/logger"
)

func TestPrint(t *testing.T) {
	c := qt.New(t)
	defer c.Cleanup()
	getOutput := patchLogPrintln(c)
	l := logger.New()
	l.Print("these are the voyages")
	c.Assert(getOutput(), qt.Equals, "these are the voyages\n")
}

func TestModifiers(t *testing.T) {
	c := qt.New(t)
	defer c.Cleanup()
	getOutput := patchLogPrintln(c)
	l := logger.New(logger.AddPrefix("my prefix"), strings.ToUpper)
	l.Print("of the starship enterprise")
	c.Assert(getOutput(), qt.Equals, "MY PREFIX: OF THE STARSHIP ENTERPRISE\n")
}

func TestNilModifiers(t *testing.T) {
	c := qt.New(t)
	defer c.Cleanup()
	getOutput := patchLogPrintln(c)
	l := logger.New(nil, nil)
	l.Print("exterminate")
	c.Assert(getOutput(), qt.Equals, "exterminate\n")
}

func TestAddPrefix(t *testing.T) {
	c := qt.New(t)
	f := logger.AddPrefix(">>> answer")
	c.Assert(f("42"), qt.Equals, ">>> answer: 42")
}

// patchLogPrintln patches the logger.LogPrintln variable so that it is
// possible to collect logs. The returned function is used to retrieve logs.
func patchLogPrintln(c *qt.C) (getOutput func() string) {
	var output string
	c.Patch(logger.LogPrintln, func(v ...interface{}) {
		output = fmt.Sprintln(v...)
	})
	return func() string {
		return output
	}
}

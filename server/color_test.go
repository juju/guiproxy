package server_test

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/juju/guiproxy/server"
)

func TestMkColor(t *testing.T) {
	c := qt.New(t)
	f := server.MkColor(42)
	msg := f("these are the voyages")
	c.Assert(msg, qt.Equals, "\033[38;5;42mthese are the voyages\033[00m")
	f = server.MkColor(47)
	msg = f("of the starship enterprise")
	c.Assert(msg, qt.Equals, "\033[38;5;47mof the starship enterprise\033[00m")
}

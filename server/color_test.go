package server_test

import (
	"testing"

	it "github.com/juju/guiproxy/internal/testing"
	"github.com/juju/guiproxy/server"
)

func TestMkColor(t *testing.T) {
	f := server.MkColor(42)
	msg := f("these are the voyages")
	it.AssertString(t, msg, "\033[38;5;42mthese are the voyages\033[00m")
	f = server.MkColor(47)
	msg = f("of the starship enterprise")
	it.AssertString(t, msg, "\033[38;5;47mof the starship enterprise\033[00m")
}

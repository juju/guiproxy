package proxy_test

import (
	"testing"

	"github.com/frankban/guiproxy/proxy"
)

func TestMkColor(t *testing.T) {
	f := proxy.MkColor(42)
	msg := f("these are the voyages")
	assertEqual(t, msg, "\033[38;5;42mthese are the voyages\033[00m")
	f = proxy.MkColor(47)
	msg = f("of the starship enterprise")
	assertEqual(t, msg, "\033[38;5;47mof the starship enterprise\033[00m")
}

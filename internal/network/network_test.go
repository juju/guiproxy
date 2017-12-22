package network_test

import (
	"errors"
	"net"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/juju/guiproxy/internal/network"
)

var addressesTests = []struct {
	about         string
	addrs         []string
	err           error
	expectedAddrs []string
}{{
	about: "error",
	err:   errors.New("bad wolf"),
}, {
	about: "no addresses",
}, {
	about:         "only ipv4 addresses",
	addrs:         []string{"1.2.3.4", "4.3.2.1"},
	expectedAddrs: []string{"1.2.3.4", "4.3.2.1"},
}, {
	about:         "only ipv6 addresses",
	addrs:         []string{"fdf8:f53b:82e4::53", "fe80::200:5aee:feaa:20a2"},
	expectedAddrs: []string{"fdf8:f53b:82e4::53", "fe80::200:5aee:feaa:20a2"},
}, {
	about:         "both ipv4 and ipv6 addresses",
	addrs:         []string{"fdf8:f53b:82e4::53", "1.2.3.4", "fe80::200:5aee:feaa:20a2"},
	expectedAddrs: []string{"1.2.3.4"},
}}

func TestAddresses(t *testing.T) {
	c := qt.New(t)
	for _, test := range addressesTests {
		c.Run(test.about, func(c *qt.C) {
			patchAddresses(c, test.addrs, test.err)
			addrs, err := network.Addresses()
			if test.err != nil {
				c.Assert(err.Error(), qt.Equals, test.err.Error())
				c.Assert(addrs, qt.IsNil)
				return
			}
			c.Assert(err, qt.Equals, nil)
			c.Assert(addrs, qt.DeepEquals, test.expectedAddrs)
		})
	}
}

// patchAddresses patches the netInterfaceAddrs variable so that it is possible
// to simulate network interfaces for the local machine.
func patchAddresses(c *qt.C, strAddrs []string, err error) {
	c.Patch(network.NetInterfaceAddrs, func() ([]net.Addr, error) {
		addrs := make([]net.Addr, len(strAddrs))
		for i, strAddr := range strAddrs {
			addrs[i] = &net.IPNet{
				IP: net.ParseIP(strAddr),
			}
		}
		return addrs, err
	})
}

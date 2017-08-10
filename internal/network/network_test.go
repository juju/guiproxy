package network_test

import (
	"errors"
	"net"
	"testing"

	"github.com/juju/guiproxy/internal/network"
	it "github.com/juju/guiproxy/internal/testing"
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
	for _, test := range addressesTests {
		t.Run(test.about, func(t *testing.T) {
			restore := patchAddresses(test.addrs, test.err)
			defer restore()
			addrs, err := network.Addresses()
			it.AssertError(t, err, test.err)
			it.AssertStringSlice(t, addrs, test.expectedAddrs)
		})
	}
}

// patchAddresses patches the netInterfaceAddrs variable so that it is possible
// to simulate network interfaces for the local machine.
func patchAddresses(strAddrs []string, err error) (restore func()) {
	original := *network.NetInterfaceAddrs
	*network.NetInterfaceAddrs = func() ([]net.Addr, error) {
		addrs := make([]net.Addr, len(strAddrs))
		for i, strAddr := range strAddrs {
			ip := net.ParseIP(strAddr)
			addrs[i] = &net.IPNet{
				IP: ip,
			}
		}
		return addrs, err
	}
	return func() {
		*network.NetInterfaceAddrs = original
	}
}

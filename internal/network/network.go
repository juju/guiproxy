package network

import "net"

// Addresses returns the list of addresses for the local machine.
// If ipv4 addresses are available, those are preferred over ipv6 ones.
func Addresses() ([]string, error) {
	addrs, err := netInterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var v4Addrs []string
	var v6Addrs []string
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if isIPv4(ip) {
			v4Addrs = append(v4Addrs, ip.String())
		} else {
			v6Addrs = append(v6Addrs, ip.String())
		}
	}
	if len(v4Addrs) != 0 {
		return v4Addrs, nil
	}
	return v6Addrs, nil
}

// netInterfaceAddrs is defined as a variable for testing purposes.
var netInterfaceAddrs = net.InterfaceAddrs

// isIPv4 reports whether the given ip is an ipv4 address.
func isIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

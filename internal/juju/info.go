package juju

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Info returns the Juju controller address be used for the proxy. If the given
// controllerAddr is empty, then the current controller address is returned.
// Otherwise the given controllerAddr is validated to be properly listening.
func Info(controllerAddr string) (string, error) {
	if controllerAddr != "" {
		controllerAddr, err := chooseAddress([]string{controllerAddr})
		if err != nil {
			return "", fmt.Errorf("cannot connect to the Juju controller: %s", err)
		}
		return controllerAddr, nil
	}

	// Retrieve Juju info from the CLI.
	out, err := execCommand("juju", "show-controller", "--format", "json")
	if err != nil {
		return "", fmt.Errorf("cannot retrieve controller info: %s", err)
	}
	var infos map[string]*controllerInfo
	err = json.Unmarshal(out, &infos)
	if err != nil || len(infos) != 1 {
		return "", fmt.Errorf("invalid controller info returned by juju: %q", out)
	}
	info := flattenInfo(infos)

	// Retrieve the controller address.
	if info.Details == nil || len(info.Details.Addrs) == 0 {
		return "", fmt.Errorf("no addresses found in controller info: %q", out)
	}
	controllerAddr, err = chooseAddress(info.Details.Addrs)
	if err != nil {
		return "", fmt.Errorf("cannot connect to the Juju controller: %s", err)
	}
	return controllerAddr, nil
}

// execCommand is defined as a variable for testing purposes.
var execCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// controllerInfo is used to unmarshal the output of "juju show-controller".
type controllerInfo struct {
	Details *struct {
		Addrs []string `json:"api-endpoints"`
	} `json:"details"`
}

// flattenInfo flattens the given controller info. The given map is assumed to
// include at least one entry.
func flattenInfo(infos map[string]*controllerInfo) *controllerInfo {
	for _, info := range infos {
		return info
	}
	panic("unreachable")
}

// dialTimeout holds the timeout for TCP connections to the Juju controller.
const dialTimeout = 10 * time.Second

// chooseAddress returns the first address in addrs that successfully accepts
// TCP connections.
func chooseAddress(addrs []string) (string, error) {
	numAddrs := len(addrs)
	addrCh := make(chan string, numAddrs)
	errCh := make(chan error, numAddrs)
	for _, addr := range addrs {
		go func(addr string) {
			conn, err := net.DialTimeout("tcp", addr, dialTimeout)
			if err != nil {
				errCh <- err
				return
			}
			conn.Close()
			addrCh <- addr
		}(addr)
	}
	errs := make([]string, 0, numAddrs)
	for {
		select {
		case addr := <-addrCh:
			return addr, nil
		case err := <-errCh:
			errs = append(errs, err.Error())
			if len(errs) == numAddrs {
				return "", fmt.Errorf(strings.Join(errs, "; "))
			}
		}
	}
	panic("unreachable")
}

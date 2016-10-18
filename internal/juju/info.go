package juju

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Info returns the Juju controller address and the model unique identifier to
// be used for the proxy. If controllerAddr is empty, then the current
// controller address is returned. If modelUUID is empty, then the uuid of the
// current active model is returned.
func Info(controllerAddr, modelUUID string) (string, string, error) {
	if controllerAddr != "" {
		return controllerAddr, modelUUID, nil
	}

	// Retrieve Juju info from the CLI.
	out, err := execCommand("juju", "show-controller", "--format", "json")
	if err != nil {
		return "", "", fmt.Errorf("cannot retrieve controller info: %s", err)
	}
	var infos map[string]*controllerInfo
	err = json.Unmarshal(out, &infos)
	if err != nil || len(infos) != 1 {
		return "", "", fmt.Errorf("invalid controller info returned by juju: %q", out)
	}
	info := flattenInfo(infos)

	// Retrieve the controller address.
	if info.Details == nil || len(info.Details.Addrs) == 0 {
		return "", "", fmt.Errorf("no addresses found in controller info: %q", out)
	}
	controllerAddr, err = chooseAddress(info.Details.Addrs)
	if err != nil {
		return "", "", fmt.Errorf("cannot connect to the Juju controller: %s", err)
	}

	// Retrieve the model unique identifier.
	if modelUUID == "" {
		parts := strings.SplitN(info.Current, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid model name in controller info: %q", out)
		}
		modelName := parts[1]
		modelInfo := info.Models[modelName]
		if modelInfo == nil || modelInfo.UUID == "" {
			return "", "", fmt.Errorf("no uuid found for model %q: %q", modelName, out)
		}
		modelUUID = modelInfo.UUID
	}

	return controllerAddr, modelUUID, nil
}

// execCommand is defined as a variable for testing purposes.
var execCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type controllerInfo struct {
	Details *struct {
		Addrs []string `json:"api-endpoints"`
	} `json:"details"`
	Models map[string]*struct {
		UUID string `json:"uuid"`
	}
	Current string `json:"current-model"`
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

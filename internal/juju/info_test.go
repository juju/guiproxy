package juju_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/juju/guiproxy/internal/juju"
	it "github.com/juju/guiproxy/internal/testing"
)

func TestInfo(t *testing.T) {
	// Set up a test server.
	ts := httptest.NewServer(newJujuServer())
	serverURL := it.MustParseURL(t, ts.URL)

	// Define the tests.
	tests := []struct {
		about                  string
		commandOut             string
		commandErr             error
		controllerAddr         string
		expectedControllerAddr string
		expectedError          error
	}{{
		about:         "command error",
		commandErr:    errors.New("bad wolf"),
		expectedError: errors.New("cannot retrieve controller info: bad wolf"),
	}, {
		about:         "invalid command output",
		commandOut:    "invalid",
		expectedError: errors.New(`invalid controller info returned by juju: "invalid"`),
	}, {
		about:         "empty command output",
		commandOut:    "{}",
		expectedError: errors.New(`invalid controller info returned by juju: "{}"`),
	}, {
		about:         "no addresses",
		commandOut:    makeControllerInfo(nil),
		expectedError: errors.New("no addresses found in controller info:"),
	}, {
		about:         "invalid addresses",
		commandOut:    makeControllerInfo([]string{":::"}),
		expectedError: errors.New("cannot connect to the Juju controller: dial tcp:"),
	}, {
		about:                  "success from juju",
		commandOut:             makeControllerInfo([]string{serverURL.Host}),
		expectedControllerAddr: serverURL.Host,
	}, {
		about:                  "success from juju: multiple addresses",
		commandOut:             makeControllerInfo([]string{"::::", serverURL.Host, ":::"}),
		expectedControllerAddr: serverURL.Host,
	}, {
		about:                  "success from juju: multiple valid addresses",
		commandOut:             makeControllerInfo([]string{serverURL.Host, serverURL.Host, serverURL.Host}),
		expectedControllerAddr: serverURL.Host,
	}, {
		about:          "invalid address from input",
		controllerAddr: ":::",
		expectedError:  errors.New("cannot connect to the Juju controller: dial tcp:"),
	}, {
		about:                  "success from input",
		controllerAddr:         serverURL.Host,
		expectedControllerAddr: serverURL.Host,
	}}

	// Run the tests.
	for _, test := range tests {
		t.Run(test.about, func(t *testing.T) {
			restore := patchCommand(t, []byte(test.commandOut), test.commandErr)
			defer restore()
			controllerAddr, err := juju.Info(test.controllerAddr)
			it.AssertError(t, err, test.expectedError)
			it.AssertString(t, controllerAddr, test.expectedControllerAddr)
		})
	}

	// Tear down the test server.
	ts.Close()
}

// newJujuServer creates and returns a new test server simulating that a remote
// Juju controller exists.
func newJujuServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})
}

// patchCommand patches the juju.ExecCommand variable so that it is possible
// to simulate different output and error scenarios.
func patchCommand(t *testing.T, out []byte, err error) (restore func()) {
	original := *juju.ExecCommand
	*juju.ExecCommand = func(name string, args ...string) ([]byte, error) {
		it.AssertString(t, name, "juju")
		it.AssertString(t, strings.Join(args, " "), "show-controller --format json")
		return out, err
	}
	return func() {
		*juju.ExecCommand = original
	}
}

// makeControllerInfo creates and returns a controller info output with the
// given addrs.
func makeControllerInfo(addrs []string) string {
	if addrs == nil {
		addrs = make([]string, 0)
	}
	out := map[string]interface{}{
		"controller-name": map[string]interface{}{
			"details": map[string]interface{}{
				"api-endpoints": addrs,
			},
		},
	}
	b, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}
	return string(b)
}

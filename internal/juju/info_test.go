package juju_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/juju/guiproxy/internal/juju"
	it "github.com/juju/guiproxy/internal/testing"
)

func TestInfo(t *testing.T) {
	c := qt.New(t)

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
		expectedError          string
	}{{
		about:         "command error",
		commandErr:    errors.New("bad wolf"),
		expectedError: "cannot retrieve controller info: bad wolf",
	}, {
		about:         "invalid command output",
		commandOut:    "invalid",
		expectedError: `invalid controller info returned by juju: "invalid"`,
	}, {
		about:         "empty command output",
		commandOut:    "{}",
		expectedError: `invalid controller info returned by juju: "{}"`,
	}, {
		about:         "no addresses",
		commandOut:    makeControllerInfo(nil),
		expectedError: "no addresses found in controller info: .*",
	}, {
		about:         "invalid addresses",
		commandOut:    makeControllerInfo([]string{":::"}),
		expectedError: "cannot connect to the Juju controller: dial tcp: .*",
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
		expectedError:  "cannot connect to the Juju controller: dial tcp: .*",
	}, {
		about:                  "success from input",
		controllerAddr:         serverURL.Host,
		expectedControllerAddr: serverURL.Host,
	}}

	// Run the tests.
	for _, test := range tests {
		c.Run(test.about, func(c *qt.C) {
			patchCommand(c, []byte(test.commandOut), test.commandErr)
			controllerAddr, err := juju.Info(test.controllerAddr)
			if test.expectedError != "" {
				c.Assert(err, qt.ErrorMatches, test.expectedError)
				c.Assert(controllerAddr, qt.Equals, "")
				return
			}
			c.Assert(err, qt.Equals, nil)
			c.Assert(controllerAddr, qt.Equals, test.expectedControllerAddr)
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
func patchCommand(c *qt.C, out []byte, err error) {
	c.Patch(juju.ExecCommand, func(name string, args ...string) ([]byte, error) {
		c.Assert(name, qt.Equals, "juju")
		c.Assert(args, qt.DeepEquals, []string{"show-controller", "--format", "json"})
		return out, err
	})
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

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
	infoTests := []struct {
		about                  string
		commandOut             string
		commandErr             error
		controllerAddr         string
		modelUUID              string
		expectedControllerAddr string
		expectedModelUUID      string
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
		commandOut:    makeControllerInfo(nil, nil, ""),
		expectedError: errors.New("no addresses found in controller info:"),
	}, {
		about:         "invalid addresses",
		commandOut:    makeControllerInfo([]string{":::"}, nil, ""),
		expectedError: errors.New("cannot connect to the Juju controller: dial tcp:"),
	}, {
		about:         "invalid current model name",
		commandOut:    makeControllerInfo([]string{serverURL.Host}, nil, "invalid"),
		expectedError: errors.New("invalid model name in controller info:"),
	}, {
		about:         "no models available",
		commandOut:    makeControllerInfo([]string{serverURL.Host}, nil, "admin@local/model47"),
		expectedError: errors.New(`no uuid found for model "model47"`),
	}, {
		about: "specific model not found",
		commandOut: makeControllerInfo([]string{serverURL.Host}, map[string]string{
			"model42": "uuid42",
		}, "admin@local/model47"),
		expectedError: errors.New(`no uuid found for model "model47"`),
	}, {
		about: "success from juju",
		commandOut: makeControllerInfo([]string{serverURL.Host}, map[string]string{
			"model42": "uuid42",
			"model47": "uuid47",
		}, "admin@local/model47"),
		expectedControllerAddr: serverURL.Host,
		expectedModelUUID:      "uuid47",
	}, {
		about: "success from juju: no current model",
		commandOut: makeControllerInfo([]string{serverURL.Host}, map[string]string{
			"model42": "uuid42",
			"model47": "uuid47",
		}, ""),
		expectedControllerAddr: serverURL.Host,
	}, {
		about: "success from juju: multiple addresses",
		commandOut: makeControllerInfo([]string{"::::", serverURL.Host, ":::"}, map[string]string{
			"model42": "uuid42",
			"model47": "uuid47",
		}, "admin@local/model42"),
		expectedControllerAddr: serverURL.Host,
		expectedModelUUID:      "uuid42",
	}, {
		about: "success from juju: multiple valid addresses",
		commandOut: makeControllerInfo([]string{serverURL.Host, serverURL.Host, serverURL.Host}, map[string]string{
			"model42": "uuid42",
			"model47": "uuid47",
		}, "admin@local/model42"),
		expectedControllerAddr: serverURL.Host,
		expectedModelUUID:      "uuid42",
	}, {
		about:          "invalid address from input",
		controllerAddr: ":::",
		expectedError:  errors.New("cannot connect to the Juju controller: dial tcp:"),
	}, {
		about:                  "success from input",
		controllerAddr:         serverURL.Host,
		modelUUID:              "uuid42",
		expectedControllerAddr: serverURL.Host,
		expectedModelUUID:      "uuid42",
	}, {
		about:                  "success from input: no model uuid",
		controllerAddr:         serverURL.Host,
		expectedControllerAddr: serverURL.Host,
	}}

	// Run the tests.
	for _, test := range infoTests {
		t.Run(test.about, func(t *testing.T) {
			restore := patchCommand(t, []byte(test.commandOut), test.commandErr)
			defer restore()
			controllerAddr, modelUUID, err := juju.Info(test.controllerAddr, test.modelUUID)
			it.AssertError(t, err, test.expectedError)
			it.AssertString(t, controllerAddr, test.expectedControllerAddr)
			it.AssertString(t, modelUUID, test.expectedModelUUID)
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
// given addrs, models and current model.
func makeControllerInfo(addrs []string, models map[string]string, current string) string {
	if addrs == nil {
		addrs = make([]string, 0)
	}
	ms := make(map[string]interface{}, len(models))
	for m, uuid := range models {
		ms[m] = map[string]string{"uuid": uuid}
	}
	out := map[string]interface{}{
		"controller-name": map[string]interface{}{
			"details": map[string]interface{}{
				"api-endpoints": addrs,
			},
			"models":        ms,
			"current-model": current,
		},
	}
	b, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}
	return string(b)
}

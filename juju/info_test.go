package juju_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/frankban/guiproxy/juju"
)

func TestInfo(t *testing.T) {
	// Set up a test server.
	ts := httptest.NewServer(newJujuServer())
	serverURL := mustParseURL(ts.URL)

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
		about: "success from juju: multiple addresses",
		commandOut: makeControllerInfo([]string{"::::", serverURL.Host, ":::"}, map[string]string{
			"model42": "uuid42",
			"model47": "uuid47",
		}, "admin@local/model42"),
		expectedControllerAddr: serverURL.Host,
		expectedModelUUID:      "uuid42",
	}, {
		about:                  "success from input",
		controllerAddr:         "1.2.3.4:4242",
		modelUUID:              "uuid42",
		expectedControllerAddr: "1.2.3.4:4242",
		expectedModelUUID:      "uuid42",
	}, {
		about:                  "success from input: no model uuid",
		controllerAddr:         "1.2.3.4:4242",
		expectedControllerAddr: "1.2.3.4:4242",
	}}

	// Run the tests.
	for _, test := range infoTests {
		t.Run(test.about, func(t *testing.T) {
			restore := patchCommand(t, []byte(test.commandOut), test.commandErr)
			defer restore()
			controllerAddr, modelUUID, err := juju.Info(test.controllerAddr, test.modelUUID)
			assertError(t, err, test.expectedError)
			assertEqual(t, controllerAddr, test.expectedControllerAddr)
			assertEqual(t, modelUUID, test.expectedModelUUID)
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
		assertEqual(t, name, "juju")
		assertEqual(t, strings.Join(args, " "), "show-controller --format json")
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

// assertEqual fails if the given strings are not equal.
func assertEqual(t *testing.T, obtained, expected string) {
	if obtained != expected {
		t.Fatalf("\n%q !=\n%q", obtained, expected)
	}
}

// assertError fails if the given errors are not equal.
func assertError(t *testing.T, obtained, expected error) {
	if obtained == nil && expected == nil {
		return
	}
	if obtained == nil || expected == nil {
		t.Fatalf("\n%v !=\n%v", obtained, expected)
	}
	if !strings.HasPrefix(obtained.Error(), expected.Error()) {
		t.Fatalf("\n%v !=\n%v", obtained, expected)
	}
}

// mustParseURL parses the given URL, and panics if it is not parsable.
func mustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

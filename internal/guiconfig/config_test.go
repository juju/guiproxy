package guiconfig_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/juju/guiproxy/internal/guiconfig"
	it "github.com/juju/guiproxy/internal/testing"
)

var newTests = []struct {
	about             string
	ctx               guiconfig.Context
	overrides         map[string]interface{}
	expectedFragments []string
}{{
	about: "without overrides",
	ctx: guiconfig.Context{
		Address:            "1.2.3.4",
		JujuVersion:        "42.47.0",
		ControllerTemplate: "wss://$server:$port/api",
		ModelTemplate:      "wss://$server:$port/model/$uuid/api",
	},
	expectedFragments: []string{
		`"apiAddress": "1.2.3.4"`,
		`"jujuCoreVersion": "42.47.0"`,
		`"jujuEnvUUID": ""`,
		`"controllerSocketTemplate": "wss://$server:$port/api"`,
		`"socketTemplate": "wss://$server:$port/model/$uuid/api"`,
		fmt.Sprintf(`"baseUrl": "%s"`, guiconfig.DefaultBaseURL),
		`"gisf": false`,
		`"socket_protocol": "ws"`,
	},
}, {
	about: "with overrides",
	ctx: guiconfig.Context{
		Address:            "wss://1.2.3.4",
		JujuVersion:        "2.0.0",
		ControllerTemplate: "/api",
		ModelTemplate:      "/model/$uuid/api",
	},
	overrides: map[string]interface{}{
		"answer":          42,
		"baseUrl":         "/base",
		"gisf":            true,
		"jujuEnvUUID":     "example-override",
		"socket_protocol": "ws",
	},
	expectedFragments: []string{
		`"answer": 42`,
		`"apiAddress": "wss://1.2.3.4"`,
		`"jujuCoreVersion": "2.0.0"`,
		`"jujuEnvUUID": "example-override"`,
		`"controllerSocketTemplate": "/api"`,
		`"socketTemplate": "/model/$uuid/api"`,
		`"baseUrl": "/base"`,
		`"gisf": true`,
		`"socket_protocol": "ws"`,
	},
}}

func TestNew(t *testing.T) {
	for _, test := range newTests {
		t.Run(test.about, func(t *testing.T) {
			cfg := guiconfig.New(test.ctx, test.overrides)
			for _, fragment := range test.expectedFragments {
				if !strings.Contains(cfg, fragment) {
					t.Fatalf("%q not included in %q", fragment, cfg)
				}
			}
			if !strings.HasPrefix(cfg, "var juju_config = {") {
				t.Fatalf("invalid prefix for config: %q", cfg)
			}
			if !strings.HasSuffix(cfg, "};") {
				t.Fatalf("invalid suffix for config: %q", cfg)
			}
		})
	}
}

var parseOverridesForEnvTests = []struct {
	about             string
	input             string
	env               string
	expectedOverrides map[string]interface{}
	expectedError     error
}{{
	about: "no overrides",
}, {
	about: "with production",
	env:   "production",
	expectedOverrides: map[string]interface{}{
		"bundleServiceURL": "https://api.jujucharms.com/bundleservice/",
		"charmstoreURL":    "https://api.jujucharms.com/charmstore/",
		"identityURL":      "https://api.jujucharms.com/identity/",
		"paymentURL":       "https://api.jujucharms.com/payment/",
		"plansURL":         "https://api.jujucharms.com/plans/",
		"termsURL":         "https://api.jujucharms.com/terms/",
		"gisf":             true,
		"baseUrl":          "/",
	},
}, {
	about: "with staging",
	env:   "staging",
	expectedOverrides: map[string]interface{}{
		"bundleServiceURL": "https://api.staging.jujucharms.com/bundleservice/",
		"charmstoreURL":    "https://api.staging.jujucharms.com/charmstore/",
		"identityURL":      "https://api.staging.jujucharms.com/identity/",
		"paymentURL":       "https://api.staging.jujucharms.com/payment/",
		"plansURL":         "https://api.staging.jujucharms.com/plans/",
		"termsURL":         "https://api.staging.jujucharms.com/terms/",
		"gisf":             true,
		"baseUrl":          "/",
	},
}, {
	about: "with qa",
	env:   "qa",
	expectedOverrides: map[string]interface{}{
		"bundleServiceURL": "https://www.jujugui.org/bundleservice/",
		"charmstoreURL":    "https://www.jujugui.org/charmstore/",
		"identityURL":      "https://www.jujugui.org/identity/",
		"paymentURL":       "https://www.jujugui.org/payment/",
		"plansURL":         "https://www.jujugui.org/plans/",
		"termsURL":         "https://www.jujugui.org/terms/",
		"gisf":             true,
		"baseUrl":          "/",
	},
}, {
	about: "success: single bool",
	input: "gisf: true",
	expectedOverrides: map[string]interface{}{
		"gisf": true,
	},
}, {
	about: "success: single text",
	input: `charmstoreURL: "https://1.2.3.4/cs/"`,
	expectedOverrides: map[string]interface{}{
		"charmstoreURL": "https://1.2.3.4/cs/",
	},
}, {
	about: "success: multiple",
	input: `answer: 42, socketTemplate: "/model-api", gisf: false`,
	expectedOverrides: map[string]interface{}{
		"answer":         42,
		"socketTemplate": "/model-api",
		"gisf":           false,
	},
}, {
	about: "success: trim spaces",
	input: ` apiAddress : "1.2.3.4" , gisf  :  true `,
	expectedOverrides: map[string]interface{}{
		"apiAddress": "1.2.3.4",
		"gisf":       true,
	},
}, {}, {
	about:         "failure: invalid environment",
	env:           "bad-wolf",
	expectedError: errors.New(`invalid environment: "bad-wolf"`),
}, {
	about:         "failure: invalid pairs",
	input:         "bad, wolf",
	expectedError: errors.New(`invalid key/value pair "bad"`),
}, {
	about:         "failure: empty overrides",
	input:         "    ",
	expectedError: errors.New(`invalid key/value pair ""`),
}, {
	about:         "failure: invalid JSON",
	input:         "gisf: bad-wolf",
	expectedError: errors.New("invalid value for key gisf: invalid character"),
}}

func TestParseOverridesForEnv(t *testing.T) {
	for _, test := range parseOverridesForEnvTests {
		t.Run(test.about, func(t *testing.T) {
			overrides, err := guiconfig.ParseOverridesForEnv(test.env, test.input)
			assertMap(t, overrides, test.expectedOverrides)
			it.AssertError(t, err, test.expectedError)
		})
	}
}

func TestBaseURL(t *testing.T) {
	invalidRawMessage := json.RawMessage([]byte("bad wolf"))

	tests := []struct {
		about         string
		overrides     map[string]interface{}
		expectedURL   string
		expectedError error
	}{{
		about:       "no overrides",
		expectedURL: guiconfig.DefaultBaseURL,
	}, {
		about: "no relevant overrides",
		overrides: map[string]interface{}{
			"gisf": true,
		},
		expectedURL: guiconfig.DefaultBaseURL,
	}, {
		about: "string",
		overrides: map[string]interface{}{
			"baseUrl": "/base/",
		},
		expectedURL: "/base/",
	}, {
		about: "string no trailing slash",
		overrides: map[string]interface{}{
			"baseUrl": "/base",
		},
		expectedURL: "/base/",
	}, {
		about: "string slash",
		overrides: map[string]interface{}{
			"baseUrl": "/",
		},
		expectedURL: "/",
	}, {
		about: "raw message",
		overrides: map[string]interface{}{
			"baseUrl": rawMessage(t, "/raw/"),
		},
		expectedURL: "/raw/",
	}, {
		about: "raw message no trailing slash",
		overrides: map[string]interface{}{
			"baseUrl": rawMessage(t, "/raw"),
		},
		expectedURL: "/raw/",
	}, {
		about: "raw message slash",
		overrides: map[string]interface{}{
			"baseUrl": rawMessage(t, "/"),
		},
		expectedURL: "/",
	}, {
		about: "failure: empty string",
		overrides: map[string]interface{}{
			"baseUrl": "",
		},
		expectedError: errors.New(`invalid base URL "": must be a path starting with "/"`),
	}, {
		about: "failure: invalid string",
		overrides: map[string]interface{}{
			"baseUrl": "bad wolf",
		},
		expectedError: errors.New(`invalid base URL "bad wolf": must be a path starting with "/"`),
	}, {
		about: "failure: empty raw message",
		overrides: map[string]interface{}{
			"baseUrl": rawMessage(t, ""),
		},
		expectedError: errors.New(`invalid base URL "": must be a path starting with "/"`),
	}, {
		about: "failure: invalid raw message",
		overrides: map[string]interface{}{
			"baseUrl": rawMessage(t, "bad wolf"),
		},
		expectedError: errors.New(`invalid base URL "bad wolf": must be a path starting with "/"`),
	}, {
		about: "failure: raw message not a JSON",
		overrides: map[string]interface{}{
			"baseUrl": &invalidRawMessage,
		},
		expectedError: errors.New(`cannot unmarshal base URL "bad wolf"`),
	}, {
		about: "failure: invalid type",
		overrides: map[string]interface{}{
			"baseUrl": 42,
		},
		expectedError: errors.New("invalid base URL: unexpected type int"),
	}, {
		about: "failure: nil",
		overrides: map[string]interface{}{
			"baseUrl": nil,
		},
		expectedError: errors.New("invalid base URL: unexpected type <nil>"),
	}}

	for _, test := range tests {
		t.Run(test.about, func(t *testing.T) {
			baseURL, err := guiconfig.BaseURL(test.overrides)
			if test.expectedError != nil {
				it.AssertError(t, err, test.expectedError)
				it.AssertString(t, baseURL, "")
				return
			}
			it.AssertError(t, err, nil)
			it.AssertString(t, baseURL, test.expectedURL)
		})
	}
}

func assertMap(t *testing.T, obtained, expected map[string]interface{}) {
	o, err := json.Marshal(obtained)
	if err != nil {
		t.Fatalf("cannot marshal obtained overrides: %s", err)
	}
	e, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("cannot marshal expected overrides: %s", err)
	}
	it.AssertString(t, string(o), string(e))
}

func rawMessage(t *testing.T, s string) *json.RawMessage {
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("cannot marshal string %q: %s", s, err)
	}
	msg := json.RawMessage(b)
	return &msg
}

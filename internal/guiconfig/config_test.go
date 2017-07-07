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
		fmt.Sprintf(`"baseUrl": "%s"`, guiconfig.BaseURL),
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
	about: "with staging",
	env:   "staging",
	expectedOverrides: map[string]interface{}{
		"bundleServiceURL": "https://api.staging.jujucharms.com/bundleservice/",
		"charmstoreURL":    "https://api.staging.jujucharms.com/charmstore/",
		"plansURL":         "https://api.staging.jujucharms.com/plans/",
		"termsURL":         "https://api.staging.jujucharms.com/terms/",
		"identityURL":      "https://api.staging.jujucharms.com/identity/",
		"gisf":             true,
	},
}, {
	about: "with qa",
	env:   "qa",
	expectedOverrides: map[string]interface{}{
		"bundleServiceURL": "https://www.jujugui.org/bundleservice/",
		"charmstoreURL":    "https://www.jujugui.org/charmstore/",
		"plansURL":         "https://www.jujugui.org/plans/",
		"termsURL":         "https://www.jujugui.org/terms/",
		"identityURL":      "https://www.jujugui.org/identity/",
		"gisf":             true,
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
		if test.env == "" {
			test.env = "production"
		}
		t.Run(test.about, func(t *testing.T) {
			overrides, err := guiconfig.ParseOverridesForEnv(test.env, test.input)
			assertMap(t, overrides, test.expectedOverrides)
			it.AssertError(t, err, test.expectedError)
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

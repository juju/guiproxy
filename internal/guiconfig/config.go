package guiconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// BaseURL is the base URL from which the GUI is served.
	BaseURL = "/gui/"

	productionBaseURL = "https://api.jujucharms.com"
	prefix            = "var juju_config = "
	suffix            = ";"
	separator         = ","
)

// New generates and returns the Juju GUI configuration file as a string, based
// on the given context. The overrides argument can be used to override or
// extend the predefined configuration with user defined values.
func New(ctx Context, overrides map[string]interface{}) string {
	cfg := map[string]interface{}{
		"jujuCoreVersion":          ctx.JujuVersion,
		"apiAddress":               ctx.Address,
		"controllerSocketTemplate": ctx.ControllerTemplate,
		"socketTemplate":           ctx.ModelTemplate,
		"baseUrl":                  BaseURL,
		"jujuEnvUUID":              "",
		"gisf":                     false,
		"socket_protocol":          "ws",
		"interactiveLogin":         true,
		"html5":                    true,
		"container":                "#main",
		"viewContainer":            "#main",
		"consoleEnabled":           true,
		"serverRouting":            false,
	}
	for k, v := range envOverrides(productionBaseURL) {
		if _, found := cfg[k]; !found {
			cfg[k] = v
		}
	}
	for k, v := range overrides {
		cfg[k] = v
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		// This should never happen.
		panic(err)
	}
	return prefix + string(b) + suffix
}

// Context holds the context used to render the Juju GUI configuration file.
type Context struct {
	// Address holds the address of the Juju controller WebSocket server.
	Address string

	// JujuVersion holds the current Juju version.
	JujuVersion string

	// ControllerTemplate holds the controller WebSocket template.
	ControllerTemplate string

	// ModelTemplate holds the model WebSocket template.
	ModelTemplate string
}

// ParseOverridesForEnv generates overrides from the given string, populating
// URLs for a given environment. Accepted strings are like the following:
// `gisf: true; charmstoreURL: "https://1.2.3.4/cs"`.
func ParseOverridesForEnv(envName, v string) (map[string]interface{}, error) {
	envPairs, err := envPairs(envName)
	if err != nil {
		return nil, err
	}
	pairs := strings.Split(v, separator)
	overrides := make(map[string]interface{}, len(pairs)+len(envPairs))
	for k, v := range envPairs {
		overrides[k] = v
	}
	if v == "" {
		if len(overrides) == 0 {
			return nil, nil
		}
		return overrides, nil
	}
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		keyVal := strings.SplitN(pair, ":", 2)
		if len(keyVal) != 2 {
			return nil, fmt.Errorf("invalid key/value pair %q", pair)
		}
		key := strings.TrimSpace(keyVal[0])
		val := strings.TrimSpace(keyVal[1])
		var value json.RawMessage
		if err := json.Unmarshal([]byte(val), &value); err != nil {
			return nil, fmt.Errorf("invalid value for key %s: %v", key, err)
		}
		overrides[key] = &value
	}
	return overrides, nil
}

// Environments holds a map of environment names to their corresponding info.
var Environments = map[string]env{
	"production": {
		ControllerAddr: "jimm.jujucharms.com:443",
		overrides:      envOverrides(productionBaseURL),
	},
	"staging": {
		ControllerAddr: "jimm.staging.jujucharms.com:443",
		overrides:      envOverrides("https://api.staging.jujucharms.com"),
	},
	"qa": {
		ControllerAddr: "jimm.jujugui.org:443",
		overrides:      envOverrides("https://www.jujugui.org"),
	},
}

// env holds information about an environment in which the GUI can be run,
// for example staging or production.
type env struct {
	// ControllerAddr holds the controller address for this environment.
	ControllerAddr string

	overrides map[string]interface{}
}

// envOverrides appends URL paths to the base URL provided, resulting in a map
// that can be used to override the default configuration.
func envOverrides(url string) map[string]interface{} {
	url = strings.TrimRight(url, "/")
	return map[string]interface{}{
		"bundleServiceURL": url + "/bundleservice/",
		"charmstoreURL":    url + "/charmstore/",
		"identityURL":      url + "/identity/",
		"plansURL":         url + "/plans/",
		"termsURL":         url + "/terms/",
		// In all main GUI scenarios we can assume gisf to be true.
		"gisf": true,
	}
}

// envPairs returns override pairs for the given environment name, which can be
// empty.
func envPairs(envName string) (map[string]interface{}, error) {
	if envName == "" {
		return nil, nil
	}
	env, found := Environments[envName]
	if !found {
		return nil, fmt.Errorf("invalid environment: %q", envName)
	}
	return env.overrides, nil
}

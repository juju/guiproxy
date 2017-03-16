package guiconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	prefix    = "var juju_config = "
	suffix    = ";"
	separator = ","
)

var environments = map[string]map[string]string{
	"production": map[string]string{
		"charmstoreURL":    "https://api.jujucharms.com/charmstore/",
		"bundleServiceURL": "https://api.jujucharms.com/bundleservice/",
		"plansURL":         "https://api.jujucharms.com/omnibus/",
		"termsURL":         "https://api.jujucharms.com/terms/",
	},
	"staging": map[string]string{
		"charmstoreURL":    "https://api.staging.jujucharms.com/charmstore/",
		"bundleServiceURL": "https://api.staging.jujucharms.com/bundleservice/",
		"plansURL":         "https://api.staging.jujucharms.com/omnibus/",
		"termsURL":         "https://api.staging.jujucharms.com/terms/",
	},
	"qa": map[string]string{
		"charmstoreURL":    "https://api.jujugui.org/charmstore/",
		"bundleServiceURL": "https://api.jujugui.org/bundleservice/",
		"plansURL":         "https://api.jujugui.org/omnibus/",
		"termsURL":         "https://api.jujugui.org/terms/",
	},
}

// New generates and returns the Juju GUI configuration file as a string, based
// on the given context. The overrides argument can be used to override or
// extend the predefined configuration with user defined values.
func New(ctx Context, overrides map[string]interface{}) string {
	cfg := map[string]interface{}{
		"jujuCoreVersion":          ctx.JujuVersion,
		"jujuEnvUUID":              ctx.UUID,
		"apiAddress":               ctx.Address,
		"controllerSocketTemplate": ctx.ControllerTemplate,
		"socketTemplate":           ctx.ModelTemplate,
		"baseUrl":                  "/",
		"gisf":                     false,
		"socket_protocol":          "ws",
		"charmstoreURL":            "https://api.jujucharms.com/charmstore/",
		"bundleServiceURL":         "https://api.jujucharms.com/bundleservice/",
		"plansURL":                 "https://api.jujucharms.com/omnibus/",
		"termsURL":                 "https://api.jujucharms.com/terms/",
		"interactiveLogin":         true,
		"html5":                    true,
		"container":                "#main",
		"viewContainer":            "#main",
		"consoleEnabled":           true,
		"serverRouting":            false,
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

	// UUID optionally holds a Juju model unique identifier.
	UUID string

	// JujuVersion holds the current Juju version.
	JujuVersion string

	// ControllerTemplate holds the controller WebSocket template.
	ControllerTemplate string

	// ModelTemplate holds the model WebSocket template.
	ModelTemplate string
}

// ParseOverridesForEnv generates overrides from the given string, populating URLs for a given environment.
// Accepted strings are like the following:
// `gisf: true; charmstoreURL: "https://1.2.3.4/cs"`.
func ParseOverridesForEnv(env, v string) (map[string]interface{}, error) {
	pairs := strings.Split(v, separator)
	overrides := make(map[string]interface{}, len(pairs)+4)
	for envKey, envValue := range environments[env] {
		overrides[envKey] = envValue
	}
	if v != "" {
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
	}
	return overrides, nil
}

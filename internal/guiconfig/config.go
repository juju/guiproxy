package guiconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	baseURLKey        = "baseUrl"
	defaultBaseURL    = "/gui/"
	productionBaseURL = "https://api.jujucharms.com"
	prefix            = "var juju_config = "
	suffix            = ";"
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
		baseURLKey:                 defaultBaseURL,
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

// Overrides generates and returns overrides from the given GUI environment
// (for instance the production or qa env), the given GUI feature flags
// (provided as a slice of strings), and the given GUI configuration. If there
// is an overlap between parameters, the GUI configuration overrides flags, and
// flags override the environment options.
func Overrides(env Environment, flags []string, config map[string]interface{}) map[string]interface{} {
	numOverrides := len(env.overrides) + len(config)
	numFlags := len(flags)
	if numFlags > 0 {
		numOverrides += 1
	} else if numOverrides == 0 {
		return nil
	}
	overrides := make(map[string]interface{}, numOverrides)
	// Handle environment specific overrides.
	for k, v := range env.overrides {
		overrides[k] = v
	}
	// Handle feature flags.
	if numFlags != 0 {
		fs := make(map[string]bool, numFlags)
		for _, flag := range flags {
			fs[flag] = true
		}
		overrides["flags"] = fs
	}
	// Handle provided configuration options.
	for k, v := range config {
		overrides[k] = v
	}
	return overrides
}

// GetEnvironment returns the environment with the given name.
func GetEnvironment(name string) (Environment, error) {
	if name == "" {
		return Environment{}, nil
	}
	for _, env := range Environments {
		if env.Name == name {
			return env, nil
		}
		for _, alias := range env.aliases {
			if alias == name {
				return env, nil
			}
		}
	}
	return Environment{}, fmt.Errorf("environment %q not found", name)
}

// Environments holds the list of environments the GUI can be run into.
var Environments = []Environment{{
	Name:           "production",
	ControllerAddr: "jimm.jujucharms.com:443",
	aliases:        []string{"prod"},
	overrides:      envOverrides(productionBaseURL),
}, {
	Name:           "staging",
	ControllerAddr: "jimm.staging.jujucharms.com:443",
	aliases:        []string{"stage"},
	overrides:      envOverrides("https://api.staging.jujucharms.com"),
}, {
	Name:           "qa",
	ControllerAddr: "jimm.jujugui.org:443",
	aliases:        []string{"brian", "bruce"},
	overrides:      envOverrides("https://www.jujugui.org"),
}}

// Environment holds information about an environment in which the GUI can be
// run, for example staging or production.
type Environment struct {
	// Name holds the name of this environment.
	Name string

	// ControllerAddr holds the controller address for this environment.
	ControllerAddr string

	aliases   []string
	overrides map[string]interface{}
}

// String implements the Stringer interface for the environment.
func (env Environment) String() string {
	aliases := ""
	if len(env.aliases) != 0 {
		aliases = fmt.Sprintf(" (aliases: %s)", strings.Join(env.aliases, ", "))
	}
	return env.Name + aliases
}

// envOverrides appends URL paths to the base URL provided, resulting in a map
// that can be used to override the default configuration.
func envOverrides(url string) map[string]interface{} {
	url = strings.TrimRight(url, "/")
	return map[string]interface{}{
		"bundleServiceURL": url + "/bundleservice/",
		"charmstoreURL":    url + "/charmstore/",
		"identityURL":      url + "/identity/",
		"paymentURL":       url + "/payment/",
		"plansURL":         url + "/omnibus/",
		"ratesURL":         url + "/omnibus/",
		"termsURL":         url + "/terms/",
		baseURLKey:         "/",
		// In all main GUI scenarios we can assume gisf to be true.
		"gisf": true,
	}
}

// BaseURL returns the base URL from which the GUI is served by the proxy.
// The given overrides are used to retrieve the URL. Otherwise, a default
// base URL is returned.
func BaseURL(overrides map[string]interface{}) (string, error) {
	value, found := overrides[baseURLKey]
	if !found {
		return defaultBaseURL, nil
	}
	var u string
	switch v := value.(type) {
	case string:
		// The value is probably an env override.
		u = v
	case *json.RawMessage:
		// The value has been passed as a -config CLI parameter.
		if err := json.Unmarshal([]byte(*v), &u); err != nil {
			return "", fmt.Errorf("cannot unmarshal base URL %q: %s", *v, err)
		}
	default:
		return "", fmt.Errorf(`invalid base URL: unexpected type %T`, v)
	}
	if !strings.HasPrefix(u, "/") {
		return "", fmt.Errorf(`invalid base URL %q: must be a path starting with "/"`, u)
	}
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u, nil
}

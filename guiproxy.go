package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juju/guiproxy/internal/guiconfig"
	"github.com/juju/guiproxy/internal/juju"
	"github.com/juju/guiproxy/internal/network"
	"github.com/juju/guiproxy/server"
	"github.com/juju/guiproxy/stringflag"
)

// version holds the guiproxy program version.
const version = "0.7.1"

var program = filepath.Base(os.Args[0])

// main starts the proxy server.
func main() {
	// Retrieve information from flags and from Juju itself (if required).
	options, err := parseOptions()
	if err != nil {
		log.Fatalf("cannot parse configuration options: %s", err)
	}
	log.Printf("%s %s\n", program, version)
	if options.showVersion {
		return
	}
	log.Println("configuring the server")
	controllerAddr, err := juju.Info(options.controllerAddr)
	if err != nil {
		log.Fatalf("cannot retrieve Juju URLs: %s", err)
	}
	log.Printf("GUI sandbox: %s\n", options.guiURL)
	log.Printf("controller: %s\n", controllerAddr)
	if options.legacyJuju {
		log.Println("using Juju 1")
	}
	if options.envName != "" {
		log.Printf("environment: %s\n", options.envName)
	}
	if len(options.guiConfig) != 0 {
		log.Println("GUI config has been customized")
	}

	// Set up the HTTP server.
	srv := server.New(server.Params{
		ControllerAddr: controllerAddr,
		GUIURL:         options.guiURL,
		GUIConfig:      options.guiConfig,
		BaseURL:        options.baseURL,
		LegacyJuju:     options.legacyJuju,
		NoColor:        options.noColor,
	})

	// Start the GUI proxy server.
	log.Println("starting the server\n")
	printAddresses(options.port, options.baseURL)
	if err := http.ListenAndServe(":"+strconv.Itoa(options.port), srv); err != nil {
		log.Fatalf("cannot start server: %s", err)
	}
}

// parseOptions returns the GUI proxy server configuration options.
func parseOptions() (*config, error) {
	flag.Usage = usage
	port := flag.Int("port", defaultPort, "GUI proxy server port")
	guiAddr := flag.String("gui", defaultGUIAddr, "address on which the GUI in sandbox mode is listening")
	controllerAddr := flag.String("controller", "", `controller address (defaults to the address of the current controller), for instance:
		-controller jimm.jujucharms.com:443`)
	guiConfig := stringflag.Map("config", nil, `override or extend GUI options with a JSON key/value string, with or without enclosing braces, for instance:
		-config '{"gisf": true}'
		-config '"gisf": true, "charmstoreURL": "https://1.2.3.4/cs"'
		-config '"flags": {"exterminate": true}'`)
	envName := flag.String("env", "", "select a predefined environment to run against between the following:\n"+envChoices())
	flags := stringflag.Slice("flags", nil, `a comma separated list of GUI feature flags to activate, for instance:
		- flags profile,status`)
	legacyJuju := flag.Bool("juju1", false, "connect to a Juju 1 model")
	noColor := flag.Bool("nocolor", false, "do not use colors")
	showVersion := flag.Bool("version", false, "show application version and exit")
	flag.Parse()

	if !strings.HasPrefix(*guiAddr, "http") {
		*guiAddr = "http://" + *guiAddr
	}
	guiURL, err := url.Parse(*guiAddr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse GUI address: %s", err)
	}
	env, err := guiconfig.GetEnvironment(*envName)
	if err != nil {
		return nil, fmt.Errorf("cannot get the environment: %s", err)
	}
	overrides := guiconfig.Overrides(env, *flags, *guiConfig)
	baseURL, err := guiconfig.BaseURL(overrides)
	if err != nil {
		return nil, fmt.Errorf("cannot parse base URL in config: %s", err)
	}

	if *controllerAddr == "" && env.ControllerAddr != "" {
		*controllerAddr = env.ControllerAddr
	}
	return &config{
		port:           *port,
		guiURL:         guiURL,
		controllerAddr: *controllerAddr,
		envName:        env.Name,
		guiConfig:      overrides,
		baseURL:        baseURL,
		legacyJuju:     *legacyJuju,
		noColor:        *noColor,
		showVersion:    *showVersion,
	}, nil
}

const (
	defaultPort    = 8042
	defaultGUIAddr = "http://localhost:6543"
)

// config holds the GUI proxy server configuration options.
type config struct {
	port           int
	guiURL         *url.URL
	controllerAddr string
	envName        string
	guiConfig      map[string]interface{}
	baseURL        string
	legacyJuju     bool
	noColor        bool
	showVersion    bool
}

// usage provides the command help and usage information.
func usage() {
	fmt.Fprintf(os.Stderr, "The %s command proxies WebSocket requests from the GUI sandbox to a Juju controller.\n", program)
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", program)
	flag.PrintDefaults()
}

// envChoices pretty formats GUI environment choices.
func envChoices() string {
	texts := make([]string, 0, len(guiconfig.Environments))
	for _, env := range guiconfig.Environments {
		texts = append(texts, fmt.Sprintf("		- %s", env))
	}
	return strings.Join(texts, "\n")
}

// printAddresses prints the URL addresses from which is possible to reach the
// GUI as served by guiproxy.
func printAddresses(port int, base string) {
	addrs, err := network.Addresses()
	if err != nil || len(addrs) == 0 {
		log.Printf("visit the GUI at http://localhost:%d%s\n", port, base)
		return
	}
	urls := make([]string, len(addrs))
	for i, addr := range addrs {
		urls[i] = fmt.Sprintf("  http://%s:%d%s\n", addr, port, base)
	}
	log.Printf("visit the GUI at any of the following addresses:\n%s\n", strings.Join(urls, ""))
}

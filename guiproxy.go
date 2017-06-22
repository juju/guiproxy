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

	"github.com/frankban/guiproxy/internal/guiconfig"
	"github.com/frankban/guiproxy/internal/juju"
	"github.com/frankban/guiproxy/server"
)

// version holds the guiproxy program version.
const version = "0.3.2"

var program = filepath.Base(os.Args[0])

// main starts the proxy server.
func main() {
	// Retrieve information from flags and from Juju itself (if required).
	options, err := parseOptions()
	if err != nil {
		log.Fatalf("cannot parse configuration options: %s", err)
	}
	log.Printf("%s %s\n", program, version)
	log.Println("configuring the server")
	listenAddr := ":" + strconv.Itoa(options.port)
	controllerAddr, modelUUID, err := juju.Info(options.controllerAddr, options.modelUUID)
	if err != nil {
		log.Fatalf("cannot retrieve Juju URLs: %s", err)
	}
	log.Printf("GUI sandbox: %s\n", options.guiURL)
	log.Printf("controller: %s\n", controllerAddr)
	log.Printf("environment: %s\n", options.environment)
	if modelUUID != "" {
		log.Printf("model: %s\n", modelUUID)
	}
	if options.legacyJuju {
		log.Println("using Juju 1")
	}
	if options.hasCustomConfig {
		log.Println("GUI config has been customized")
	}

	// Set up the HTTP server.
	srv := server.New(server.Params{
		ControllerAddr: controllerAddr,
		ModelUUID:      modelUUID,
		OriginAddr:     "http://0.0.0.0" + listenAddr,
		GUIURL:         options.guiURL,
		GUIConfig:      options.guiConfig,
		LegacyJuju:     options.legacyJuju,
		NoColor:        options.noColor,
	})

	// Start the GUI proxy server.
	log.Println("starting the server\n")
	log.Printf("visit the GUI at http://0.0.0.0:%d/\n", options.port)
	if err := http.ListenAndServe(listenAddr, srv); err != nil {
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
	modelUUID := flag.String("uuid", "", "model uuid (defaults to the uuid of the current model)")
	guiConfig := flag.String("config", "", `override or extend fields in the GUI configuration, for instance:
		-config gisf:true
		-config 'gisf: true, charmstoreURL: "https://1.2.3.4/cs"'`)
	environment := flag.String("env", "production", `select a pre-defined environment to run against.
		valid options:
		- 'production' (default)
		- 'staging'
		- 'qa'
		- 'none' (explicitly empty values)`)
	legacyJuju := flag.Bool("juju1", false, "connect to a Juju 1 model")
	noColor := flag.Bool("nocolor", false, "do not use colors")
	flag.Parse()
	if !strings.HasPrefix(*guiAddr, "http") {
		*guiAddr = "http://" + *guiAddr
	}
	guiURL, err := url.Parse(*guiAddr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse GUI address: %s", err)
	}
	overrides, err := guiconfig.ParseOverridesForEnv(*environment, *guiConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot parse GUI config: %s", err)
	}
	return &config{
		port:            *port,
		guiURL:          guiURL,
		controllerAddr:  *controllerAddr,
		modelUUID:       *modelUUID,
		environment:     *environment,
		guiConfig:       overrides,
		hasCustomConfig: len(*guiConfig) != 0,
		legacyJuju:      *legacyJuju,
		noColor:         *noColor,
	}, nil
}

const (
	defaultPort    = 8042
	defaultGUIAddr = "http://localhost:6543"
)

// config holds the GUI proxy server configuration options.
type config struct {
	port            int
	guiURL          *url.URL
	controllerAddr  string
	modelUUID       string
	environment     string
	guiConfig       map[string]interface{}
	hasCustomConfig bool
	legacyJuju      bool
	noColor         bool
}

// usage provides the command help and usage information.
func usage() {
	fmt.Fprintf(os.Stderr, "The %s command proxies WebSocket requests from the GUI sandbox to a Juju controller.\n", program)
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", program)
	flag.PrintDefaults()
}

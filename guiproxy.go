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

	"github.com/frankban/guiproxy/internal/juju"
	"github.com/frankban/guiproxy/server"
)

// version holds the guiproxy program version.
const version = "0.1.0"

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
	if modelUUID != "" {
		log.Printf("model: %s\n", modelUUID)
	}
	if options.legacyJuju {
		log.Println("using Juju 1")
	}

	// Set up the HTTP server.
	srv := server.New(server.Params{
		ControllerAddr: controllerAddr,
		ModelUUID:      modelUUID,
		OriginAddr:     "http://localhost" + listenAddr,
		Port:           options.port,
		GUIURL:         options.guiURL,
		LegacyJuju:     options.legacyJuju,
		NoColor:        options.noColor,
	})

	// Start the GUI proxy server.
	log.Println("starting the server\n")
	log.Printf("visit the GUI at http://localhost:%d/\n", options.port)
	if err := http.ListenAndServe(listenAddr, srv); err != nil {
		log.Fatalf("cannot start server: %s", err)
	}
}

// parseOptions returns the GUI proxy server configuration options.
func parseOptions() (*config, error) {
	flag.Usage = usage
	port := flag.Int("port", defaultPort, "GUI proxy server port")
	guiAddr := flag.String("gui", defaultGUIAddr, "address on which the GUI in sandbox mode is listening")
	controllerAddr := flag.String("controller", "", "controller address (defaults to the address of the current controller)")
	modelUUID := flag.String("uuid", "", "model uuid (defaults to the uuid of the current model)")
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
	return &config{
		port:           *port,
		guiURL:         guiURL,
		controllerAddr: *controllerAddr,
		modelUUID:      *modelUUID,
		legacyJuju:     *legacyJuju,
		noColor:        *noColor,
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
	modelUUID      string
	legacyJuju     bool
	noColor        bool
}

// usage provides the command help and usage information.
func usage() {
	fmt.Fprintf(os.Stderr, "The %s command proxies WebSocket requests from the GUI sandbox to a Juju controller.\n", program)
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", program)
	flag.PrintDefaults()
}

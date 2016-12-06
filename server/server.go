package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/net/websocket"

	"github.com/frankban/guiproxy/logger"
	"github.com/frankban/guiproxy/wsproxy"
)

const (
	// DisconnectedUUID holds the model unique identifier provided to the GUI
	// when disconnected mode is enabled.
	DisconnectedUUID = "disconnected"

	// controllerSrcTemplate, controllerDstTemplate, modelSrcTemplate and
	// modelDstTemplate hold templates to be provided and used by the Juju GUI
	// in order to establish WebSocket connections.
	controllerSrcTemplate = "/controller/$server/$port/controller-api"
	controllerDstTemplate = "wss://$server:$port/api"
	modelSrcTemplate      = "/model/$server/$port/$uuid/model-api"
	modelDstTemplate      = "wss://$server:$port/model/$uuid/api"

	// legacyModelSrcTemplate and legacyModelDstTemplate hold templates to be
	// provided and used by the Juju GUI in order to establish WebSocket
	// connections to Juju 1 models.
	legacyModelSrcTemplate = "/model/$server/$port/model-api"
	legacyModelDstTemplate = "wss://$server:$port/"

	// jujuVersion and legacyJujuVersion hold the Juju versions declared in the
	// dynamically generated Juju GUI configuration file.
	jujuVersion       = "2.0.1"
	legacyJujuVersion = "1.25.7"
)

// New creates and returns a new GUI proxy server.
func New(p Params) http.Handler {
	mux := http.NewServeMux()
	var serveModel func(*websocket.Conn)
	if p.LegacyJuju {
		serveModel = newWebSocketProxy(legacyModelDstTemplate, legacyModelSrcTemplate, p.OriginAddr, p.NoColor)
	} else {
		serveController := newWebSocketProxy(controllerDstTemplate, controllerSrcTemplate, p.OriginAddr, p.NoColor)
		mux.Handle("/controller/", websocket.Handler(serveController))
		serveModel = newWebSocketProxy(modelDstTemplate, modelSrcTemplate, p.OriginAddr, p.NoColor)
	}
	mux.Handle("/model/", websocket.Handler(serveModel))
	mux.Handle("/juju-core/", http.StripPrefix("/juju-core/", newTLSReverseProxy(p.ControllerAddr)))
	mux.HandleFunc("/config.js", serveConfig(p.ControllerAddr, p.ModelUUID, p.Port, p.LegacyJuju))
	mux.Handle("/", httputil.NewSingleHostReverseProxy(p.GUIURL))
	return mux
}

// Params holds parameters for creating a GUI proxy server.
type Params struct {
	// ControllerAddr holds the address of the remote Juju controller.
	ControllerAddr string

	// ModelUUID optionally holds the unique identifier of the target model.
	ModelUUID string

	// OriginAddr holds the address from which the WebSocket request is made.
	OriginAddr string

	// Port holds the port number on which the server will be listening.
	Port int

	// GUIURL holds the URL on which the GUI sandbox instance is listening.
	GUIURL *url.URL

	// LegacyJuju holds whether the proxy is connected to a Juju 1 model.
	LegacyJuju bool

	// NoColor holds whether to use colors in the log output.
	NoColor bool
}

// newTLSReverseProxy returns a new ReverseProxy that routes URLs to the given
// host using TLS protocol. The resulting proxy does not verify certificates.
func newTLSReverseProxy(host string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "https",
		Host:   host,
	})
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return proxy
}

// newWebSocketProxy returns a WebSocket handler that proxies the WebSocket
// frames from the Juju GUI to Juju and vice versa. WebSocket addresses are
// translated using the given source and destination templates.
func newWebSocketProxy(dstTemplate, srcTemplate, origin string, noColor bool) func(*websocket.Conn) {
	r := strings.NewReplacer(
		"$server", `(?P<server>.*)`,
		"$port", `(?P<port>\d+)`,
		"$uuid", `(?P<uuid>.*)`,
	)
	re := regexp.MustCompile(r.Replace(srcTemplate))
	return func(guiWS *websocket.Conn) {
		target := resolveWebSocketAddress(re, guiWS.Request().URL.Path, dstTemplate)

		// Open the WebSocket connection to the remote server.
		log.Printf("opening %s\n", target)
		targetWS, err := wsConnect(target, origin)
		if err != nil {
			log.Fatalf(err.Error())
		}

		// Start copying WebSocket messages back and forth.
		addr := targetWS.RemoteAddr().String()
		inColor, outColor := logColors(strings.HasPrefix(srcTemplate, "/model/"), noColor)
		err = wsproxy.Copy(
			targetWS,
			guiWS,
			logger.New(logger.AddPrefix("<-- "+addr), inColor),
			logger.New(logger.AddPrefix("--> "+addr), outColor),
		)
		log.Printf("closed %s: %s\n", target, err)
	}
}

// resolveWebSocketAddress returns a Juju WebSocket address based on the given
// regular expression, current request path and destination socket template.
func resolveWebSocketAddress(re *regexp.Regexp, path, dstTemplate string) string {
	match := re.FindStringSubmatch(path)
	oldnew := make([]string, 0, 6)
	for i, name := range re.SubexpNames() {
		if i != 0 {
			oldnew = append(oldnew, "$"+name, match[i])
		}
	}
	r := strings.NewReplacer(oldnew...)
	return r.Replace(dstTemplate)
}

// wsConnect opens a secure WebSocket client connection to the given address
// with the given origin. The TLS certificate verification is skipped.
func wsConnect(addr, origin string) (*websocket.Conn, error) {
	config, err := websocket.NewConfig(addr, origin)
	if err != nil {
		return nil, fmt.Errorf("cannot create ws config for %s: %s", addr, err)
	}
	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := websocket.DialConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot dial %s: %s", addr, err)
	}
	return conn, nil
}

// serveConfig returns an HTTP handler that serves the Juju GUI JavaScript
// configuration file. The configuration is dynamically generated using the
// given controller address, model UUID and guiproxy port.
func serveConfig(addr, uuid string, port int, legacyJuju bool) func(w http.ResponseWriter, req *http.Request) {
	controller, model := controllerSrcTemplate, modelSrcTemplate
	version := jujuVersion
	if legacyJuju {
		controller, model = "", legacyModelSrcTemplate
		version = legacyJujuVersion
	}
	ctx := map[string]interface{}{
		"addr":       addr,
		"controller": controller,
		"gisf":       uuid == DisconnectedUUID,
		"model":      model,
		"port":       port,
		"uuid":       uuid,
		"version":    version,
	}
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", jsMimeType)
		configTemplate.Execute(w, ctx)
	}
}

// jsMimeType holds the mime type used to serve the GUI configuration.
var jsMimeType = mime.TypeByExtension(".js")

// configTemplate holds the template used to render the GUI configuration.
var configTemplate = template.Must(template.New("config").Parse(`
var juju_config = {
    baseUrl: 'http://0.0.0.0:{{.port}}/',
    jujuCoreVersion: '{{.version}}',
    jujuEnvUUID: '{{.uuid}}',
    apiAddress: '{{.addr}}',
    controllerSocketTemplate: '{{.controller}}',
    socketTemplate: '{{.model}}',
    gisf: {{.gisf}},
    socket_protocol: 'ws',
    charmstoreURL: 'https://api.jujucharms.com/charmstore/',
    bundleServiceURL: 'https://api.jujucharms.com/bundleservice/',
    plansURL: 'https://api.jujucharms.com/omnibus/',
    termsURL: 'https://api.jujucharms.com/terms/',
    interactiveLogin: true,
    html5: true,
    container: '#main',
    viewContainer: '#main',
    consoleEnabled: true,
    serverRouting: false
};
`))

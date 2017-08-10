package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/juju/guiproxy/httpproxy"
	"github.com/juju/guiproxy/internal/guiconfig"
	"github.com/juju/guiproxy/logger"
	"github.com/juju/guiproxy/wsproxy"
)

const (
	// controllerSrcTemplate, controllerDstTemplate, modelSrcTemplate and
	// modelDstTemplate hold templates to be provided and used by the Juju GUI
	// in order to establish WebSocket connections.
	controllerSrcTemplate = "/controller/?controller=$server:$port"
	controllerDstTemplate = "wss://$controller/api"
	modelSrcTemplate      = "/model/?model=$server:$port&uuid=$uuid"
	modelDstTemplate      = "wss://$model/model/$uuid/api"

	// legacyModelSrcTemplate and legacyModelDstTemplate hold templates to be
	// provided and used by the Juju GUI in order to establish WebSocket
	// connections to Juju 1 models.
	legacyModelSrcTemplate = "/model/?model=$server:$port"
	legacyModelDstTemplate = "wss://$model/"

	// jujuVersion and legacyJujuVersion hold the Juju versions declared in the
	// dynamically generated Juju GUI configuration file.
	jujuVersion       = "2.2.0"
	legacyJujuVersion = "1.25.7"

	// webSocketBufferSize holds the frame size for WebSocket messages.
	webSocketBufferSize = 65536
)

// New creates and returns a new GUI proxy server.
func New(p Params) http.Handler {
	mux := http.NewServeMux()

	var serveModel http.Handler
	if p.LegacyJuju {
		serveModel = newWebSocketProxy(legacyModelDstTemplate, legacyModelSrcTemplate, p.NoColor)
	} else {
		serveController := newWebSocketProxy(controllerDstTemplate, controllerSrcTemplate, p.NoColor)
		mux.Handle("/controller/", serveController)
		serveModel = newWebSocketProxy(modelDstTemplate, modelSrcTemplate, p.NoColor)
	}
	mux.Handle("/model/", serveModel)

	configColor, jujuProxyColor, guiProxyColor := pink, orange, yellow
	if p.NoColor {
		configColor, jujuProxyColor, guiProxyColor = nil, nil, nil
	}
	mux.HandleFunc("/config.js", serveConfig(p.ControllerAddr, p.GUIConfig, p.LegacyJuju, logger.New(configColor)))
	mux.Handle("/juju-core/", http.StripPrefix("/juju-core/", httpproxy.NewTLSReverseProxy(p.ControllerAddr, logger.New(jujuProxyColor))))
	mux.Handle("/", httpproxy.NewRedirectHandler(p.BaseURL, p.GUIURL, logger.New(guiProxyColor)))
	return mux
}

// Params holds parameters for creating a GUI proxy server.
type Params struct {
	// ControllerAddr holds the address of the remote Juju controller.
	ControllerAddr string

	// GUIURL holds the URL on which the GUI sandbox instance is listening.
	GUIURL *url.URL

	// GUIConfig holds the key/value pairs used to optionally override the
	// predefined Juju GUI configuration file.
	GUIConfig map[string]interface{}

	// BaseURL holds the base URL from which the GUI is served by the proxy.
	BaseURL string

	// LegacyJuju holds whether the proxy is connected to a Juju 1 model.
	LegacyJuju bool

	// NoColor holds whether to use colors in the log output.
	NoColor bool
}

// newWebSocketProxy returns a WebSocket handler that proxies the WebSocket
// frames from the Juju GUI to Juju and vice versa. WebSocket addresses are
// translated using the given source and destination templates.
func newWebSocketProxy(dstTemplate, srcTemplate string, noColor bool) http.Handler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  webSocketBufferSize,
		WriteBufferSize: webSocketBufferSize,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Upgrade the HTTP connection.
		log.Printf("upgrading %s\n", req.URL)
		guiConn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Printf("cannot upgrade %s: %s", req.URL, err)
			return
		}
		defer guiConn.Close()

		// Open the WebSocket connection to the remote server.
		target := resolveWebSocketAddress(req.URL, dstTemplate)
		log.Printf("opening %s\n", target)
		targetConn, err := wsDial(target)
		if err != nil {
			log.Printf("cannot dial %s: %s", target, err)
			return
		}
		defer targetConn.Close()

		// Start copying WebSocket messages back and forth.
		addr := targetConn.RemoteAddr().String()
		inColor, outColor := logColors(strings.HasPrefix(srcTemplate, "/model/"), noColor)
		err = wsproxy.Copy(
			targetConn,
			guiConn,
			logger.New(logger.AddPrefix("<-- "+addr), inColor),
			logger.New(logger.AddPrefix("--> "+addr), outColor),
		)
		log.Printf("closed %s: %s\n", target, err)
	})
}

// resolveWebSocketAddress returns a Juju WebSocket address based on the given
// regular expression, current request path and destination socket template.
func resolveWebSocketAddress(u *url.URL, dstTemplate string) string {
	query := u.Query()
	fields := []string{"controller", "model", "uuid"}
	oldnew := make([]string, 0, len(fields)*2)
	for _, field := range fields {
		if !strings.Contains(dstTemplate, "$"+field) {
			continue
		}
		value := query.Get(field)
		if value == "" {
			log.Fatalf("invalid WebSocket URL %q: %q query not present", u, field)
		}
		oldnew = append(oldnew, "$"+field, value)
	}
	r := strings.NewReplacer(oldnew...)
	return r.Replace(dstTemplate)
}

// wsDial opens a secure WebSocket client connection to the given address. The
// TLS certificate verification is skipped. The returned connection must be
// closed by callers.
func wsDial(addr string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ReadBufferSize:  webSocketBufferSize,
		WriteBufferSize: webSocketBufferSize,
	}
	conn, _, err := dialer.Dial(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot dial %s: %s", addr, err)
	}
	return conn, nil
}

// serveConfig returns an HTTP handler that serves the Juju GUI JavaScript
// configuration file. The configuration is dynamically generated using the
// given controller address, configuration overrides and whether a legacy Juju
// is in use.
func serveConfig(addr string, configOverrides map[string]interface{}, legacyJuju bool, log logger.Interface) func(w http.ResponseWriter, req *http.Request) {
	controller, model := controllerSrcTemplate, modelSrcTemplate
	version := jujuVersion
	if legacyJuju {
		controller, model = "", legacyModelSrcTemplate
		version = legacyJujuVersion
	}
	cfg := guiconfig.New(guiconfig.Context{
		Address:            addr,
		JujuVersion:        version,
		ControllerTemplate: controller,
		ModelTemplate:      model,
	}, configOverrides)
	return func(w http.ResponseWriter, req *http.Request) {
		log.Print(fmt.Sprintf("%s %s: %d OK\n%s", req.Method, req.URL, http.StatusOK, cfg))
		w.Header().Set("Content-Type", jsMimeType)
		fmt.Fprint(w, cfg)
	}
}

// jsMimeType holds the mime type used to serve the GUI configuration.
var jsMimeType = mime.TypeByExtension(".js")

package proxy

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"text/template"

	"golang.org/x/net/websocket"

	"github.com/frankban/guiproxy/logger"
)

// New creates and returns a new GUI proxy server.
func New(p Params) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api", websocket.Handler(serveWebSocket(p.ControllerAddr, p.OriginAddr, p.NoColor)))
	mux.Handle("/model/", websocket.Handler(serveWebSocket(p.ControllerAddr, p.OriginAddr, p.NoColor)))
	mux.Handle("/juju-core/", http.StripPrefix("/juju-core/", newTLSReverseProxy(p.ControllerAddr)))
	mux.HandleFunc("/config.js", serveConfig(p.Port, p.ModelUUID))
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

// serveWebSocket returns a WebSocket handler that proxies the WebSocket frames
// to and from the given address.
func serveWebSocket(addr, origin string, noColor bool) func(*websocket.Conn) {
	return func(guiWS *websocket.Conn) {
		// Set up the WebSocket client configuration.
		path := guiWS.Request().URL.Path
		target := "wss://" + addr + path
		config, err := websocket.NewConfig(target, origin)
		if err != nil {
			log.Fatalf("cannot create ws config for %s: %s", target, err)
		}
		config.TlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}

		// Open the WebSocket connection to the remote server.
		log.Printf("opening %s\n", target)
		targetWS, err := websocket.DialConfig(config)
		if err != nil {
			log.Fatalf("cannot dial %s: %s", target, err)
		}

		// Start copying WebSocket messages back and forth.
		errCh := make(chan error, 2)
		wsAddr := targetWS.RemoteAddr().String()
		inColor, outColor := logColors(path, noColor)
		go cp(targetWS, guiWS, errCh, logger.New("--> "+wsAddr, outColor))
		go cp(guiWS, targetWS, errCh, logger.New("<-- "+wsAddr, inColor))
		err = <-errCh
		log.Printf("closing %s: %s\n", target, err)
	}
}

// cp copies all frames sent from the src WebSocket connection to the dst one,
// and sends errors to the given error channel. The content of each frame is
// also logged using the given logger.
func cp(dst, src *websocket.Conn, errCh chan error, apiLog logger.Interface) {
	var msg string
	var err error
	for {
		msg, err = copyJSON(dst, src)
		if err != nil {
			errCh <- err
			return
		}
		apiLog.Print(msg)
	}
}

// copyJSON copies a single JSON frame sent by src to dst. Note that a simple
// io.Copy would not work here, as copying without a specific codec would
// result in truncated frames.
func copyJSON(dst, src *websocket.Conn) (string, error) {
	var m *json.RawMessage
	if err := websocket.JSON.Receive(src, &m); err != nil {
		return "", err
	}
	if err := websocket.JSON.Send(dst, m); err != nil {
		return "", err
	}
	return string(*m), nil
}

// serveConfig returns an HTTP handler that serves the Juju GUI JavaScript
// configuration file.
func serveConfig(port int, uuid string) func(w http.ResponseWriter, req *http.Request) {
	ctx := map[string]interface{}{
		"port": port,
		"uuid": uuid,
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
    "baseUrl": "http://0.0.0.0:{{.port}}/",
    "jujuCoreVersion": "2.0.0",
    "jujuEnvUUID": "{{.uuid}}",
    "apiAddress": "ws://localhost:{{.port}}",
    "controllerSocketTemplate": "/api",
    "socketTemplate": "/model/$uuid/api",
    "socket_protocol": "ws",
    "charmstoreURL": "https://api.jujucharms.com/charmstore/",
    "bundleServiceURL": "https://api.jujucharms.com/bundleservice/",
    "plansURL": "https://api.jujucharms.com/omnibus/",
    "termsURL": "https://api.jujucharms.com/terms/",
    "interactiveLogin": true,
    "html5": true,
    "container": "#main",
    "viewContainer": "#main",
    "consoleEnabled": true,
    "serverRouting": false
};
`))

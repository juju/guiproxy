package proxy_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/net/websocket"

	"github.com/frankban/guiproxy/proxy"
)

func TestNew(t *testing.T) {
	// Set up test servers.
	gui := httptest.NewServer(newGUIServer())
	juju := httptest.NewTLSServer(newJujuServer())
	jujuURL := mustParseURL(juju.URL)
	ts := httptest.NewServer(proxy.New(proxy.Params{
		ControllerAddr: jujuURL.Host,
		ModelUUID:      "example-uuid",
		OriginAddr:     "http://1.2.3.4:4242",
		Port:           4242,
		GUIURL:         mustParseURL(gui.URL),
	}))
	serverURL := mustParseURL(ts.URL)

	// Run the tests.
	t.Run("testJujuWebSocketController", testJujuWebSocket(serverURL, "/api"))
	t.Run("testJujuWebSocketModel1", testJujuWebSocket(serverURL, "/model/uuid/api"))
	t.Run("testJujuWebSocketModel2", testJujuWebSocket(serverURL, "/model/another-uuid/api"))
	t.Run("testJujuHTTPS", testJujuHTTPS(serverURL))
	t.Run("testGUIConfig", testGUIConfig(serverURL))
	t.Run("testGUIStaticFiles", testGUIStaticFiles(serverURL))

	// Tear down test servers.
	ts.Close()
	juju.Close()
	gui.Close()
}

func testJujuWebSocket(serverURL *url.URL, path string) func(t *testing.T) {
	origin := "http://localhost/"
	u := *serverURL
	u.Scheme = "ws"
	socketURL := u.String() + path
	return func(t *testing.T) {
		// Connect to the remote WebSocket.
		ws, err := websocket.Dial(socketURL, "", origin)
		if err != nil {
			t.Fatalf("cannot dial WebSocket at %s: %v", socketURL, err)
		}
		defer ws.Close()
		// Send a message.
		msg := jsonMessage{
			Request: "my api request",
		}
		if err = websocket.JSON.Send(ws, msg); err != nil {
			t.Fatalf("cannot send message to %s: %v", socketURL, err)
		}
		// Retrieve the response from the WebSocket server.
		if err = websocket.JSON.Receive(ws, &msg); err != nil {
			t.Fatalf("cannot retrieve response from %s: %v", socketURL, err)
		}
		assertEqual(t, msg.Request, "my api request")
		assertEqual(t, msg.Response, path)
	}
}

func testJujuHTTPS(serverURL *url.URL) func(t *testing.T) {
	return func(t *testing.T) {
		// Make the HTTP request to retrieve a Juju HTTPS API endpoint.
		resp, err := http.Get(serverURL.String() + "/juju-core/api/path")
		if err != nil {
			t.Fatalf("cannot send the request to get Juju endpoint: %v", err)
		}
		defer resp.Body.Close()
		// The request succeeded.
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("invalid response code from Juju endpoint: %v", resp.StatusCode)
		}
		// The response body includes the expected content.
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("cannot read Juju endpoint response body: %v", err)
		}
		assertEqual(t, string(b), "juju: /api/path")
	}
}

func testGUIConfig(serverURL *url.URL) func(t *testing.T) {
	return func(t *testing.T) {
		// Make the HTTP request to retrieve the GUI configuration file.
		resp, err := http.Get(serverURL.String() + "/config.js")
		if err != nil {
			t.Fatalf("cannot send the request to get config.js: %v", err)
		}
		defer resp.Body.Close()
		// The request succeeded.
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("invalid response code from config.js: %v", resp.StatusCode)
		}
		// The response body includes the GUI configuration.
		var expected bytes.Buffer
		err = proxy.ConfigTemplate.Execute(&expected, map[string]interface{}{
			"port": 4242,
			"uuid": "example-uuid",
		})
		if err != nil {
			t.Fatalf("cannot render the configuration template: %v", err)
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("cannot read config.js response body: %v", err)
		}
		assertEqual(t, string(b), expected.String())
	}
}

func testGUIStaticFiles(serverURL *url.URL) func(t *testing.T) {
	return func(t *testing.T) {
		// Make the HTTP request to retrieve a GUI static file.
		resp, err := http.Get(serverURL.String() + "/my/path")
		if err != nil {
			t.Fatalf("cannot send the request to get a GUI static file: %v", err)
		}
		defer resp.Body.Close()
		// The request succeeded.
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("invalid response code from GUI static file: %v", resp.StatusCode)
		}
		// The response body includes the expected content.
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("cannot read static file response body: %v", err)
		}
		assertEqual(t, string(b), "gui: /my/path")
	}
}

// newGUIServer creates and returns a new test server simulating a remote Juju
// GUI run in sandbox mode.
func newGUIServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "gui: "+req.URL.Path)
	})
	return mux
}

// newTestServer creates and returns a new test server simulating a remote Juju
// controller and model.
func newJujuServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api", websocket.Handler(echoHandler))
	mux.Handle("/model/", websocket.Handler(echoHandler))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "juju: "+req.URL.Path)
	})
	return mux
}

// echoHandler is a WebSocket handler repeating what it receives.
func echoHandler(ws *websocket.Conn) {
	path := ws.Request().URL.Path
	var msg jsonMessage
	var err error
	for {
		err = websocket.JSON.Receive(ws, &msg)
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		msg.Response = path
		if err = websocket.JSON.Send(ws, msg); err != nil {
			panic(err)
		}
	}
}

// jsonMessage holds messages used for testing the WebSocket handlers.
type jsonMessage struct {
	Request  string
	Response string
}

// assertEqual fails if the given strings are not equal.
func assertEqual(t *testing.T, obtained, expected string) {
	if obtained != expected {
		t.Fatalf("\n%q !=\n%q", obtained, expected)
	}
}

// mustParseURL parses the given URL, and panics if it is not parsable.
func mustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}

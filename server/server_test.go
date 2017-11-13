package server_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gorilla/websocket"

	it "github.com/juju/guiproxy/internal/testing"
	"github.com/juju/guiproxy/server"
)

func TestNew(t *testing.T) {
	c := qt.New(t)
	// Set up test servers.
	gui := httptest.NewServer(newGUIServer())
	defer gui.Close()
	guiURL := it.MustParseURL(t, gui.URL)

	juju := httptest.NewTLSServer(newJujuServer())
	defer juju.Close()
	jujuURL := it.MustParseURL(t, juju.URL)

	legacyJuju := httptest.NewTLSServer(newLegacyJujuServer())
	defer legacyJuju.Close()
	legacyJujuURL := it.MustParseURL(t, legacyJuju.URL)

	proxy := httptest.NewServer(server.New(server.Params{
		ControllerAddr: jujuURL.Host,
		GUIURL:         guiURL,
		BaseURL:        "/base/",
	}))
	defer proxy.Close()
	serverURL := it.MustParseURL(t, proxy.URL)

	legacyProxy := httptest.NewServer(server.New(server.Params{
		ControllerAddr: legacyJujuURL.Host,
		GUIURL:         guiURL,
		BaseURL:        "/base-legacy/",
		LegacyJuju:     true,
	}))
	defer proxy.Close()
	legacyServerURL := it.MustParseURL(t, legacyProxy.URL)

	customConfigProxy := httptest.NewServer(server.New(server.Params{
		ControllerAddr: jujuURL.Host,
		GUIURL:         guiURL,
		BaseURL:        "/",
		GUIConfig: map[string]interface{}{
			"answer":          42,
			"baseUrl":         "/",
			"container":       "#different-one",
			"gisf":            true,
			"jujuCoreVersion": "42.47.0",
		},
	}))
	defer customConfigProxy.Close()
	customConfigServerURL := it.MustParseURL(t, customConfigProxy.URL)

	controllerPath := fmt.Sprintf("/controller/?controller=%s", jujuURL.Host)
	modelPath1 := fmt.Sprintf("/model/?model=%s&uuid=uuid", jujuURL.Host)
	modelPath2 := fmt.Sprintf("/model/?model=%s&uuid=another-uuid", jujuURL.Host)
	legacyModelPath := fmt.Sprintf("/model/?model=%s", legacyJujuURL.Host)

	c.Run("testJujuWebSocket Controller", testJujuWebSocket(serverURL, "/api", controllerPath))
	c.Run("testJujuWebSocket Model1", testJujuWebSocket(serverURL, "/model/uuid/api", modelPath1))
	c.Run("testJujuWebSocket Model2", testJujuWebSocket(serverURL, "/model/another-uuid/api", modelPath2))
	c.Run("testJujuWebSocket Legacy", testJujuWebSocket(legacyServerURL, "/", legacyModelPath))

	c.Run("testJujuHTTPS", testJujuHTTPS(serverURL))
	c.Run("testJujuHTTPS Legacy", testJujuHTTPS(legacyServerURL))

	c.Run("testGUIConfig", testGUIConfig(
		serverURL,
		fmt.Sprintf(`"controllerSocketTemplate": %s`, jsonMarshalString(server.ControllerSrcTemplate)),
		fmt.Sprintf(`"socketTemplate": %s`, jsonMarshalString(server.ModelSrcTemplate)),
		fmt.Sprintf(`"apiAddress": "%s"`, jujuURL.Host),
		fmt.Sprintf(`"jujuCoreVersion": "%s"`, server.JujuVersion),
		`"jujuEnvUUID": ""`,
		`"gisf": false`,
	))
	c.Run("testGUIConfig Legacy", testGUIConfig(
		legacyServerURL,
		`"controllerSocketTemplate": ""`,
		fmt.Sprintf(`"socketTemplate": %s`, jsonMarshalString(server.LegacyModelSrcTemplate)),
		fmt.Sprintf(`"apiAddress": "%s"`, legacyJujuURL.Host),
		fmt.Sprintf(`"jujuCoreVersion": "%s"`, server.LegacyJujuVersion),
		`"jujuEnvUUID": ""`,
	))
	c.Run("testGUIConfig Customized", testGUIConfig(
		customConfigServerURL,
		fmt.Sprintf(`"controllerSocketTemplate": %s`, jsonMarshalString(server.ControllerSrcTemplate)),
		fmt.Sprintf(`"socketTemplate": %s`, jsonMarshalString(server.ModelSrcTemplate)),
		fmt.Sprintf(`"apiAddress": "%s"`, jujuURL.Host),
		`"answer": 42`,
		`"baseUrl": "/"`,
		`"container": "#different-one"`,
		`"gisf": true`,
		`"jujuCoreVersion": "42.47.0"`,
		`"jujuEnvUUID": ""`,
	))

	c.Run("testGUIStaticFiles", testGUIStaticFiles(serverURL))
	c.Run("testGUIStaticFiles Legacy", testGUIStaticFiles(legacyServerURL))

	c.Run("testGUIRedirect", testGUIRedirect(serverURL, "/base/"))
	c.Run("testGUIRedirect Legacy", testGUIRedirect(legacyServerURL, "/base-legacy/"))
	c.Run("testGUIRedirect Customized", testGUIRedirect(customConfigServerURL, "/"))
}

func testJujuWebSocket(serverURL *url.URL, dstPath, srcPath string) func(c *qt.C) {
	u := *serverURL
	u.Scheme = "ws"
	socketURL := u.String() + srcPath
	return func(c *qt.C) {
		// Connect to the remote WebSocket.
		conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
		c.Assert(err, qt.Equals, nil)
		defer conn.Close()
		// Send a message.
		msg := jsonMessage{
			Request: "my api request",
		}
		err = conn.WriteJSON(msg)
		c.Assert(err, qt.Equals, nil)
		// Retrieve the response from the WebSocket server.
		err = conn.ReadJSON(&msg)
		c.Assert(err, qt.Equals, nil)
		c.Assert(msg.Request, qt.Equals, "my api request")
		c.Assert(msg.Response, qt.Equals, dstPath)
	}
}

func testJujuHTTPS(serverURL *url.URL) func(c *qt.C) {
	return func(c *qt.C) {
		// Make the HTTP request to retrieve a Juju HTTPS API endpoint.
		resp, err := http.Get(serverURL.String() + "/juju-core/api/path")
		c.Assert(err, qt.Equals, nil)
		defer resp.Body.Close()
		// The request succeeded.
		c.Assert(resp.StatusCode, qt.Equals, http.StatusOK)
		// The response body includes the expected content.
		b, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, qt.Equals, nil)
		c.Assert(string(b), qt.Equals, "juju: /api/path")
	}
}

func testGUIConfig(serverURL *url.URL, fragments ...string) func(c *qt.C) {
	return func(c *qt.C) {
		// Make the HTTP request to retrieve the GUI configuration file.
		resp, err := http.Get(serverURL.String() + "/config.js")
		c.Assert(err, qt.Equals, nil)
		defer resp.Body.Close()
		// The request succeeded.
		c.Assert(resp.StatusCode, qt.Equals, http.StatusOK)
		// The response body includes all the provided fragments.
		b, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, qt.Equals, nil)
		cfg := string(b)
		for _, fragment := range fragments {
			if !strings.Contains(cfg, fragment) {
				c.Fatalf("invalid GUI config: %q not included in %q", fragment, cfg)
			}
		}
	}
}

func testGUIStaticFiles(serverURL *url.URL) func(c *qt.C) {
	return func(c *qt.C) {
		// Make the HTTP request to retrieve a GUI static file.
		resp, err := http.Get(serverURL.String() + "/my/path")
		c.Assert(err, qt.Equals, nil)
		defer resp.Body.Close()
		// The request succeeded.
		c.Assert(resp.StatusCode, qt.Equals, http.StatusOK)
		// The response body includes the expected content.
		b, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, qt.Equals, nil)
		c.Assert(string(b), qt.Equals, "gui: /my/path")
	}
}

func testGUIRedirect(serverURL *url.URL, baseURL string) func(c *qt.C) {
	return func(c *qt.C) {
		// Make the HTTP request to retrieve the GUI root path.
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := client.Get(serverURL.String() + "/")
		c.Assert(err, qt.Equals, nil)
		defer resp.Body.Close()
		// The request succeeded.
		if baseURL == "/" {
			c.Assert(resp.StatusCode, qt.Equals, http.StatusOK)
			return
		}
		c.Assert(resp.StatusCode, qt.Equals, http.StatusMovedPermanently)
		// The response body includes the expected location.
		b, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, qt.Equals, nil)
		content := string(b)
		fragment := fmt.Sprintf("<a href=\"%s\">Moved Permanently</a>", baseURL)
		c.Assert(strings.Contains(content, fragment), qt.Equals, true, qt.Commentf("content: %q", content))
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

// newJujuServer creates and returns a new test server simulating a remote Juju
// controller and model.
func newJujuServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api", http.HandlerFunc(echoHandler))
	mux.Handle("/model/", http.HandlerFunc(echoHandler))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "juju: "+req.URL.Path)
	})
	return mux
}

// newLegacyJujuServer creates and returns a new test server simulating a
// remote Juju 1 model.
func newLegacyJujuServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(echoHandler))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "juju: "+req.URL.Path)
	})
	return mux
}

// echoHandler is a WebSocket handler repeating what it receives.
func echoHandler(w http.ResponseWriter, req *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	var msg jsonMessage
	for {
		err = conn.ReadJSON(&msg)
		if websocket.IsUnexpectedCloseError(err) {
			return
		}
		if err != nil {
			panic(err)
		}
		msg.Response = req.URL.Path
		if err = conn.WriteJSON(msg); err != nil {
			panic(err)
		}
	}
}

// jsonMessage holds messages used for testing the WebSocket handlers.
type jsonMessage struct {
	Request  string
	Response string
}

func jsonMarshalString(s interface{}) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

package httpproxy_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/juju/guiproxy/httpproxy"
	it "github.com/juju/guiproxy/internal/testing"
	"github.com/juju/guiproxy/logger"
)

func TestNewTLSReverseProxy(t *testing.T) {
	t.Run("testTLSReverseProxy with logger", testTLSReverseProxy("/my/path", &logCollector{}))
	t.Run("testTLSReverseProxy without logger", testTLSReverseProxy("/another/path", nil))
}

func testTLSReverseProxy(path string, log logger.Interface) func(t *testing.T) {
	return func(t *testing.T) {
		// Set up a target HTTP server.
		target := httptest.NewTLSServer(targetHndler)
		defer target.Close()
		targetURL := it.MustParseURL(t, target.URL)

		// Set up a reverse proxy pointing to the target server.
		proxy := httptest.NewServer(httpproxy.NewTLSReverseProxy(targetURL.Host, log))
		defer proxy.Close()

		// Send a request to the proxy.
		resp, err := http.Get(proxy.URL + path)
		it.AssertError(t, err, nil)
		defer resp.Body.Close()

		// The response is correct.
		b, err := ioutil.ReadAll(resp.Body)
		it.AssertError(t, err, nil)
		it.AssertString(t, string(b), "target: "+path)

		// Logs are printed.
		if log, ok := log.(*logCollector); ok {
			it.AssertInt(t, len(log.messages), 1)
			it.AssertString(t, log.messages[0], fmt.Sprintf("GET %s/my/path: 200 OK", target.URL))
		}
	}
}

var newRedirectHandlerTests = []struct {
	about        string
	to           string
	path         string
	expectedPath string
}{{
	about:        "redirect root",
	to:           "/redirect/",
	path:         "/",
	expectedPath: "/redirect/",
}, {
	about:        "redirect add slash",
	to:           "/to/",
	path:         "/to",
	expectedPath: "/to/",
}, {
	about:        "redirect root no slash",
	to:           "/redirect",
	path:         "/",
	expectedPath: "/redirect/",
}, {
	about:        "redirect add slash no slash",
	to:           "/to",
	path:         "/to",
	expectedPath: "/to/",
}, {
	about:        "no redirect short",
	to:           "/gui/",
	path:         "/g",
	expectedPath: "/g",
}, {
	about:        "no redirect long",
	to:           "/gui/",
	path:         "/gui/my/path",
	expectedPath: "/gui/my/path",
}, {
	about:        "no redirect root",
	to:           "/",
	path:         "/",
	expectedPath: "/",
}, {
	about:        "no redirect root no slash",
	to:           "",
	path:         "/",
	expectedPath: "/",
}}

func TestNewRedirectHandler(t *testing.T) {
	for _, test := range newRedirectHandlerTests {
		t.Run(fmt.Sprintf("testRedirectHandler %s", test.about), testRedirectHandler(
			test.path, test.expectedPath, test.to, &logCollector{}))
		t.Run(fmt.Sprintf("testRedirectHandler %s without logger", test.about), testRedirectHandler(
			test.path, test.expectedPath, test.to, nil))
	}
}

func testRedirectHandler(path, expectedPath, to string, log logger.Interface) func(t *testing.T) {
	return func(t *testing.T) {
		// Set up a target HTTP server.
		target := httptest.NewServer(targetHndler)
		defer target.Close()
		targetURL := it.MustParseURL(t, target.URL)

		// Set up a redirect handler pointing to the target server.
		handler := httptest.NewServer(httpproxy.NewRedirectHandler(to, targetURL, log))
		defer handler.Close()

		// Send a request to the handler.
		resp, err := http.Get(handler.URL + path)
		it.AssertError(t, err, nil)
		defer resp.Body.Close()

		// The response is correct.
		b, err := ioutil.ReadAll(resp.Body)
		it.AssertError(t, err, nil)
		it.AssertString(t, string(b), "target: "+expectedPath)

		// Logs are printed.
		if log, ok := log.(*logCollector); ok {
			it.AssertInt(t, len(log.messages), 1)
			it.AssertString(t, log.messages[0], fmt.Sprintf("GET %s%s: 200 OK", target.URL, expectedPath))
		}
	}
}

// logCollector is a logger used for collecting log messages.
type logCollector struct {
	messages []string
}

// Print implements logger.Interface.Print.
func (l *logCollector) Print(msg string) {
	l.messages = append(l.messages, msg)
}

var targetHndler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "target: "+req.URL.Path)
})

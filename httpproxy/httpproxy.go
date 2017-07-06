package httpproxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/juju/guiproxy/logger"
)

// NewTLSReverseProxy returns a new ReverseProxy that routes URLs to the given
// host using TLS protocol. The resulting proxy does not verify certificates. A
// logger can be optionally provided to log requests and response statues.
func NewTLSReverseProxy(host string, log logger.Interface) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "https",
		Host:   host,
	})
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	if log != nil {
		addLogging(proxy, log)
	}
	return proxy
}

// NewRedirectHandler redirects all requests to "/" to the given path. All
// other requests are reverse proxied to the given target URL. A logger can
// be optionally provided to log requests and response statues.
func NewRedirectHandler(to string, target *url.URL, log logger.Interface) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	if log != nil {
		addLogging(proxy, log)
	}
	if !strings.HasSuffix(to, "/") {
		to += "/"
	}
	return &redirectHandler{
		to:      to,
		handler: proxy,
	}
}

// redirectHandler redirects all requests to "/" to the given path. All other
// requests are handled by the stored handler.
type redirectHandler struct {
	to      string
	handler http.Handler
}

// ServeHTTP implements http.Handler.ServeHTTP.
func (h *redirectHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != h.to && (req.URL.Path == "/" || req.URL.Path == strings.TrimSuffix(h.to, "/")) {
		http.Redirect(w, req, h.to, http.StatusMovedPermanently)
		return
	}
	h.handler.ServeHTTP(w, req)
}

func addLogging(proxy *httputil.ReverseProxy, log logger.Interface) {
	transport := proxy.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	proxy.Transport = &loggingTransport{
		RoundTripper: transport,
		log:          log,
	}
}

// loggingTransport is a default transport with logging ability.
type loggingTransport struct {
	http.RoundTripper
	log logger.Interface
}

// RoundTrip implements http.RoundTripper.RoundTrip.
func (t *loggingTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	t.log.Print(fmt.Sprintf("%s %s: %s", req.Method, req.URL, resp.Status))
	return resp, nil
}

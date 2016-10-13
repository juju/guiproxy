package wsproxy_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/websocket"

	it "github.com/frankban/guiproxy/internal/testing"
	"github.com/frankban/guiproxy/wsproxy"
)

func TestCopy(t *testing.T) {
	// Set up a target WebSocket server.
	ping := httptest.NewServer(websocket.Handler(pingHandler))
	defer ping.Close()

	// Set up the WebSocket proxy that copies the messages back and forth.
	conn1Log, conn2Log := &logStorage{}, &logStorage{}
	proxy := httptest.NewServer(websocket.Handler(newProxyHandler(wsURL(ping.URL), conn1Log, conn2Log)))
	defer proxy.Close()

	// Connect to the proxy.
	conn, err := websocket.Dial(wsURL(proxy.URL), "", "http://localhost")
	it.AssertError(t, err, nil)
	defer conn.Close()

	// Send messages and check that ping responses are properly received.
	send := func(content string) string {
		msg := jsonMessage{
			Content: content,
		}
		err = websocket.JSON.Send(conn, msg)
		it.AssertError(t, err, nil)
		err = websocket.JSON.Receive(conn, &msg)
		it.AssertError(t, err, nil)
		return msg.Content
	}
	it.AssertString(t, send("ping"), "ping pong")
	it.AssertString(t, send("bad wolf"), "bad wolf pong")

	// Incoming and outgoing WebSocket traffic has been logged.
	assertLogs := func(ls *logStorage, expected ...string) {
		messages := make([]string, len(expected))
		for i, content := range expected {
			b, err := json.Marshal(jsonMessage{
				Content: content,
			})
			it.AssertError(t, err, nil)
			messages[i] = string(b)
		}
		errMessage := fmt.Sprintf("\n%v !=\n%v", ls.messages, messages)
		if len(ls.messages) != len(messages) {
			t.Fatal(errMessage)
		}
		for i, msg := range ls.messages {
			it.AssertString(t, msg, messages[i])
		}
	}
	assertLogs(conn1Log, "ping", "bad wolf")
	assertLogs(conn2Log, "ping pong", "bad wolf pong")
}

// pingHandler is a WebSocket handler responding to pings.
func pingHandler(ws *websocket.Conn) {
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
		msg.Content += " pong"
		if err = websocket.JSON.Send(ws, msg); err != nil {
			panic(err)
		}
	}
}

// newCopyHandler returns a WebSocket handler copying from the given WebSocket
// server.
func newProxyHandler(srvURL string, conn1Log, conn2Log *logStorage) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		conn, err := websocket.Dial(srvURL, "", "http://localhost")
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		if err := wsproxy.Copy(ws, conn, conn1Log, conn2Log); err != nil {
			panic(err)
		}
	}
}

// logStorage is a logger.Interface used for testing purposes.
type logStorage struct {
	messages []string
}

// Print implements logger.Interface and stores log messages.
func (ls *logStorage) Print(msg string) {
	ls.messages = append(ls.messages, msg)
}

// wsURL returns a WebSocket URL from the given HTTP URL.
func wsURL(u string) string {
	return strings.Replace(u, "http://", "ws://", 1)
}

// jsonMessage holds messages used for testing the WebSocket handlers.
type jsonMessage struct {
	Content string
}

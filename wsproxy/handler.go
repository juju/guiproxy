package wsproxy

import (
	"encoding/json"

	"golang.org/x/net/websocket"

	"github.com/frankban/guiproxy/logger"
)

// Copy copies messages back and forth between the provided WebSocket
// connections. JSON encoded traffic is logged via the given loggers.
func Copy(conn1, conn2 *websocket.Conn, conn1Log, conn2Log logger.Interface) error {
	// Start copying WebSocket messages back and forth.
	errCh := make(chan error, 2)
	go cp(conn1, conn2, errCh, conn2Log)
	go cp(conn2, conn1, errCh, conn1Log)
	return <-errCh
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

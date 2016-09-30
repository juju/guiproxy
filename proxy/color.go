package proxy

import (
	"fmt"
	"strings"
)

// mkColor is used to create color functions.
func mkColor(color int) func(string) string {
	return func(msg string) string {
		return fmt.Sprintf("\033[38;5;%dm%s\033[00m", color, msg)
	}
}

var (
	green      = mkColor(28)
	lightGreen = mkColor(40)
	blue       = mkColor(27)
	lightBlue  = mkColor(39)
)

// logColors returns the color functions to use for incoming and outgoing API
// logger messages. It receives the path to which the connection has been made
// and a boolean flag used to disable colors.
func logColors(path string, noColor bool) (inColor func(string) string, outColor func(string) string) {
	if noColor {
		return nil, nil
	}
	if strings.HasPrefix(path, "/model/") {
		// This is a model connection.
		return lightGreen, green
	}
	// This is a controller connection.
	return lightBlue, blue
}

package server

import "fmt"

// colorFunc is a function that colorizes the given string.
type colorFunc func(string) string

// mkColor is used to create color functions.
func mkColor(color int) colorFunc {
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
// logger messages. It receives whether the connection has been made to a model
// and a boolean flag used to disable colors.
func logColors(isModel, noColor bool) (inColor, outColor colorFunc) {
	if noColor {
		return nil, nil
	}
	if isModel {
		return lightGreen, green
	}
	return lightBlue, blue
}

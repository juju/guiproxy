# GUIProxy: a Juju GUI development tool

The GUIProxy server proxies WebSocket requests from a Juju GUI sandbox instance
to the currently active Juju controller/model. This way it is really easy and
fast to iterate between coding and then immediately trying the GUI changes on a
real Juju controller.

## Installation

Run `go get github.com/juju/guiproxy`.

## Usage

- Start a development Juju GUI branch in sandbox mode, by running `make run`.
- Switch to the Juju controller you want to connect the GUI to.
- Run `guiproxy` and point your browser to the link suggested by the server.
- Enjoy!

Run `guiproxy -h` for instructions on how to customize the GUI proxy server.

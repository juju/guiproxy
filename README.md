[![GoDoc](https://godoc.org/github.com/juju/guiproxy?status.svg)](https://godoc.org/github.com/juju/guiproxy)
[![Build Status](https://travis-ci.org/juju/guiproxy.svg?branch=master)](https://travis-ci.org/juju/guiproxy)

# GUIProxy: a Juju GUI development tool

The GUIProxy server proxies WebSocket requests from a Juju GUI sandbox instance
to the currently active Juju controller/model. This way it is really easy and
fast to iterate between coding and then immediately trying the GUI changes on a
real Juju controller.

## Installation

Run `go get github.com/juju/guiproxy`.

## Usage

- Start a development Juju GUI branch in sandbox mode, by running `make run` in
  the GUI branch you want to use.
- Switch to the Juju controller you want to connect the GUI to.
- In another terminal tab, run `guiproxy`: this command will start the GUIProxy
  server and will output a list of URLs that can be used to access the GUI.
- Point your browser to one of the URLs above (from the `guiproxy` output).
- Enjoy!

Run `guiproxy -h` for instructions on how to customize the GUI proxy server.
For instance it is possible to point GUIProxy to JAAS by running
`guiproxy -env prod`, in which case you don't need to bootstrap any additional
controllers. Also, the `-flags` parameter can be used to enable feature flags.

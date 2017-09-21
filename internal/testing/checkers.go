package testing

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
)

// MustParseURL parses the given URL, and panics if it is not parsable.
func MustParseURL(t *testing.T, rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("%scannot parse %q: %s", caller(), rawurl, err)
	}
	return u
}

func caller() string {
	_, file, line, _ := runtime.Caller(2)
	return fmt.Sprintf("\n%s:%d:\n", filepath.Base(file), line)
}

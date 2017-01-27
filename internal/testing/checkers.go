package testing

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// AssertString fails if the given strings are not equal.
func AssertString(t *testing.T, obtained, expected string) {
	if obtained != expected {
		t.Fatalf("%s%q !=\n%q", caller(), obtained, expected)
	}
}

// AssertError fails if the given errors are not equal.
func AssertError(t *testing.T, obtained, expected error) {
	if obtained == nil && expected == nil {
		return
	}
	if obtained == nil || expected == nil {
		t.Fatalf("%s%v !=\n%v", caller(), obtained, expected)
	}
	obtainedErr, expectedErr := obtained.Error(), expected.Error()
	if (obtainedErr != "" && expectedErr == "") || !strings.HasPrefix(obtainedErr, expectedErr) {
		t.Fatalf("%s%q !=\n%q", caller(), obtainedErr, expectedErr)
	}
}

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

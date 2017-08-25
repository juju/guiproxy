package testing

import (
	"encoding/json"
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

// AssertInt fails if the given integers are not equal.
func AssertInt(t *testing.T, obtained, expected int) {
	if obtained != expected {
		t.Fatalf("%s%d != %d", caller(), obtained, expected)
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

// AssertStringSlice fails if the given slices are not equal.
func AssertStringSlice(t *testing.T, obtained, expected []string) {
	if obtained == nil && expected == nil {
		return
	}
	if obtained == nil || expected == nil || len(obtained) != len(expected) {
		t.Fatalf("%s%#v !=\n%#v", caller(), obtained, expected)
	}
	for i := range obtained {
		if obtained[i] != expected[i] {
			t.Fatalf("%s%#v !=\n%#v", caller(), obtained, expected)
		}
	}
}

// AssertMap fails if the given maps are not equal.
func AssertMap(t *testing.T, obtained, expected map[string]interface{}) {
	obtainedBytes, err := json.Marshal(obtained)
	if err != nil {
		t.Fatalf("%scannot marshal obtained map: %s", caller(), err)
	}
	expectedBytes, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("%scannot marshal expected map: %s", caller(), err)
	}
	o, e := string(obtainedBytes), string(expectedBytes)
	if o != e {
		t.Fatalf("%s%q !=\n%q", caller(), o, e)
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

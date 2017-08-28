package stringflag_test

import (
	"errors"
	"flag"
	"testing"

	it "github.com/juju/guiproxy/internal/testing"
	"github.com/juju/guiproxy/stringflag"
)

// These assignments are used to ensure that flag.Value is implemented.
var _ flag.Value = (*stringflag.StringSlice)(nil)
var _ flag.Value = (*stringflag.StringMap)(nil)

var sliceTests = []struct {
	about               string
	name                string
	value               string
	defaultValue        []string
	expectedValue       []string
	expectedStringValue string
	expectedError       error
}{{
	about:               "single string",
	name:                "single",
	value:               "exterminate",
	expectedValue:       []string{"exterminate"},
	expectedStringValue: "exterminate",
}, {
	about:               "multiple strings",
	name:                "multiple",
	value:               "these,are,the,voyages",
	expectedValue:       []string{"these", "are", "the", "voyages"},
	expectedStringValue: "these,are,the,voyages",
}, {
	about:               "weird formatting",
	name:                "weird",
	value:               "  these , are,the,  voyages ",
	expectedValue:       []string{"these", "are", "the", "voyages"},
	expectedStringValue: "these,are,the,voyages",
}, {
	about:         "default value: with value",
	name:          "def1",
	value:         "exterminate",
	defaultValue:  []string{"default", "not", "used"},
	expectedValue: []string{"exterminate"},
}, {
	about:         "default value: without value",
	name:          "def2",
	defaultValue:  []string{"default", "used"},
	expectedValue: []string{"default", "used"},
}, {
	about:         "error: empty string",
	name:          "err",
	expectedError: errors.New("cannot include empty strings in the list"),
}, {
	about:         "error: multiple values with empty string",
	name:          "err",
	value:         ",bad,wolf",
	expectedError: errors.New("cannot include empty strings in the list"),
}}

func TestSlice(t *testing.T) {
	for _, test := range sliceTests {
		runIsolated(t, test.about, func(t *testing.T) {
			v := stringflag.Slice(test.name, test.defaultValue, "slice usage")
			if test.value != "" || test.defaultValue == nil {
				err := flag.Set(test.name, test.value)
				it.AssertError(t, err, test.expectedError)
			}
			it.AssertStringSlice(t, *v, test.expectedValue)
		})
	}
}

func TestSliceVar(t *testing.T) {
	for _, test := range sliceTests {
		runIsolated(t, test.about, func(t *testing.T) {
			var v stringflag.StringSlice
			stringflag.SliceVar(&v, test.name, test.defaultValue, "slice usage")
			if test.value != "" || test.defaultValue == nil {
				err := flag.Set(test.name, test.value)
				it.AssertError(t, err, test.expectedError)
			}
			it.AssertStringSlice(t, v, test.expectedValue)
		})
	}
}

func TestStringSliceSet(t *testing.T) {
	for _, test := range sliceTests {
		runIsolated(t, test.about, func(t *testing.T) {
			if test.defaultValue != nil {
				return
			}
			var v stringflag.StringSlice
			err := v.Set(test.value)
			it.AssertError(t, err, test.expectedError)
			it.AssertStringSlice(t, v, test.expectedValue)
		})
	}
}

func TestStringSliceString(t *testing.T) {
	for _, test := range sliceTests {
		runIsolated(t, test.about, func(t *testing.T) {
			if test.defaultValue != nil {
				return
			}
			var v stringflag.StringSlice
			v.Set(test.value)
			it.AssertString(t, v.String(), test.expectedStringValue)
		})
	}
}

var mapTests = []struct {
	about               string
	name                string
	value               string
	defaultValue        map[string]interface{}
	expectedValue       map[string]interface{}
	expectedStringValue string
	expectedError       error
}{{
	about: "single pair",
	name:  "single",
	value: `{"gisf": true}`,
	expectedValue: map[string]interface{}{
		"gisf": true,
	},
	expectedStringValue: `{"gisf":true}`,
}, {
	about: "multiple pairs",
	name:  "multiple",
	value: `{"gisf": true, "url": "https://1.2.3.4"}`,
	expectedValue: map[string]interface{}{
		"gisf": true,
		"url":  "https://1.2.3.4",
	},
	expectedStringValue: `{"gisf":true,"url":"https://1.2.3.4"}`,
}, {
	about: "nested map",
	name:  "nested",
	value: `{"gisf": true, "flags": {"profile": true, "status": true}}`,
	expectedValue: map[string]interface{}{
		"gisf": true,
		"flags": map[string]bool{
			"profile": true,
			"status":  true,
		},
	},
	expectedStringValue: `{"flags":{"profile":true,"status":true},"gisf":true}`,
}, {
	about: "weird formatting",
	name:  "weird",
	value: `  {  "gisf" :  true } `,
	expectedValue: map[string]interface{}{
		"gisf": true,
	},
	expectedStringValue: `{"gisf":true}`,
}, {
	about:               "empty object",
	name:                "empty",
	value:               `{}`,
	expectedValue:       map[string]interface{}{},
	expectedStringValue: "{}",
}, {
	about: "no braces: single pair",
	name:  "single",
	value: `"gisf": true`,
	expectedValue: map[string]interface{}{
		"gisf": true,
	},
	expectedStringValue: `{"gisf":true}`,
}, {
	about: "no braces: multiple pairs",
	name:  "multiple",
	value: `"gisf": true, "url": "https://1.2.3.4"`,
	expectedValue: map[string]interface{}{
		"gisf": true,
		"url":  "https://1.2.3.4",
	},
	expectedStringValue: `{"gisf":true,"url":"https://1.2.3.4"}`,
}, {
	about: "no braces: nested map",
	name:  "nested",
	value: `"gisf": true, "flags": {"profile": true, "status": true}`,
	expectedValue: map[string]interface{}{
		"gisf": true,
		"flags": map[string]bool{
			"profile": true,
			"status":  true,
		},
	},
	expectedStringValue: `{"flags":{"profile":true,"status":true},"gisf":true}`,
}, {
	about: "no braces: weird formatting",
	name:  "weird",
	value: `    "gisf" :  true  `,
	expectedValue: map[string]interface{}{
		"gisf": true,
	},
	expectedStringValue: `{"gisf":true}`,
}, {
	about: "default value: with value",
	name:  "single",
	value: `{"gisf": true}`,
	defaultValue: map[string]interface{}{
		"answer": 42,
	},
	expectedValue: map[string]interface{}{
		"gisf": true,
	},
}, {
	about: "default value: without value",
	name:  "single",
	defaultValue: map[string]interface{}{
		"answer": 42,
	},
	expectedValue: map[string]interface{}{
		"answer": 42,
	},
}, {
	about:               "empty string",
	name:                "empty",
	expectedValue:       map[string]interface{}{},
	expectedStringValue: "{}",
}, {
	about:               "error: not a map",
	name:                "err",
	value:               "42",
	expectedStringValue: "null",
	expectedError:       errors.New("cannot unmarshal JSON"),
}, {
	about:               "error: invalid JSON",
	name:                "err",
	value:               "!",
	expectedStringValue: "null",
	expectedError:       errors.New("cannot unmarshal JSON"),
}}

func TestMap(t *testing.T) {
	for _, test := range mapTests {
		runIsolated(t, test.about, func(t *testing.T) {
			v := stringflag.Map(test.name, test.defaultValue, "map usage")
			if test.value != "" || test.defaultValue == nil {
				err := flag.Set(test.name, test.value)
				it.AssertError(t, err, test.expectedError)
			}
			it.AssertMap(t, *v, test.expectedValue)
		})
	}
}

func TestMapVar(t *testing.T) {
	for _, test := range mapTests {
		runIsolated(t, test.about, func(t *testing.T) {
			var v stringflag.StringMap
			stringflag.MapVar(&v, test.name, test.defaultValue, "map usage")
			if test.value != "" || test.defaultValue == nil {
				err := flag.Set(test.name, test.value)
				it.AssertError(t, err, test.expectedError)
			}
			it.AssertMap(t, v, test.expectedValue)
		})
	}
}

func TestStringMapSet(t *testing.T) {
	for _, test := range mapTests {
		runIsolated(t, test.about, func(t *testing.T) {
			if test.defaultValue != nil {
				return
			}
			var v stringflag.StringMap
			err := v.Set(test.value)
			it.AssertMap(t, v, test.expectedValue)
			it.AssertError(t, err, test.expectedError)
		})
	}
}

func TestStringMapString(t *testing.T) {
	for _, test := range mapTests {
		runIsolated(t, test.about, func(t *testing.T) {
			if test.defaultValue != nil {
				return
			}
			var v stringflag.StringMap
			v.Set(test.value)
			it.AssertString(t, v.String(), test.expectedStringValue)
		})
	}
}

// runIsolated runs the given test function without clobbering global flags.
func runIsolated(t *testing.T, name string, f func(t *testing.T)) {
	restore := resetForTesting()
	defer restore()
	t.Run(name, f)
}

// resetForTesting creates a new flag set for the global command line and
// returns a function that restores the original global command line.
func resetForTesting() (restore func()) {
	original := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet("", flag.ContinueOnError)
	return func() {
		flag.CommandLine = original
	}
}

package errchkjson

import (
	"slices"
	"testing"
)

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		tag         string
		wantName    string
		wantOptions []string
	}{
		{tag: "", wantName: "", wantOptions: nil},
		{tag: "name", wantName: "name", wantOptions: nil},
		{tag: ",omitempty", wantName: "", wantOptions: []string{"omitempty"}},
		{tag: "name,omitempty,string", wantName: "name", wantOptions: []string{"omitempty", "string"}},
		{tag: "name,format:RFC3339", wantName: "name", wantOptions: []string{"format:RFC3339"}},
		{tag: "name,", wantName: "name", wantOptions: []string{""}},
		// Single-quoted names are not supported by encoding/json/v2 in Go 1.27,
		// the quotes are part of the name.
		{tag: "'a,b'", wantName: "'a", wantOptions: []string{"b'"}},
		{tag: "'a',string", wantName: "'a'", wantOptions: []string{"string"}},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			name, options := parseJSONTag(tc.tag)
			if name != tc.wantName {
				t.Errorf("parseJSONTag(%q) name = %q, want %q", tc.tag, name, tc.wantName)
			}
			if !slices.Equal(options, tc.wantOptions) {
				t.Errorf("parseJSONTag(%q) options = %q, want %q", tc.tag, options, tc.wantOptions)
			}
		})
	}
}

func TestFormatOptionWithValue(t *testing.T) {
	tests := []struct {
		tag        string
		wantOption string
		wantOK     bool
	}{
		{tag: "", wantOK: false},
		{tag: "format:x", wantOK: false}, // name, not an option
		{tag: "a,format:x", wantOption: "format:x", wantOK: true},
		{tag: "a,omitempty,format:RFC3339", wantOption: "format:RFC3339", wantOK: true},
		{tag: "a,format:x-y", wantOption: "format:x", wantOK: true},
		{tag: "a,format:", wantOK: false},
		{tag: "a,format", wantOK: false},
		{tag: "a,format:123", wantOK: false},
		{tag: "a,format:''", wantOK: false},
		{tag: "a,format:'abc", wantOK: false},
		{tag: "a,format:'abc,omitempty", wantOK: false},
		{tag: "a,format:'2006-01-02'", wantOption: "format:'2006-01-02'", wantOK: true},
		{tag: "a,format:'a,b'", wantOption: "format:'a,b'", wantOK: true},
		{tag: `a,format:'it\'s'`, wantOption: `format:'it\'s'`, wantOK: true},
		{tag: `a,format:'\z'`, wantOK: false}, // invalid escape sequence
		{tag: "a,format:'abc,format:x", wantOption: "format:x", wantOK: true},
		{tag: "a,'format:x'", wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			option, ok := formatOptionWithValue(tc.tag)
			if option != tc.wantOption || ok != tc.wantOK {
				t.Errorf("formatOptionWithValue(%q) = %q, %t, want %q, %t", tc.tag, option, ok, tc.wantOption, tc.wantOK)
			}
		})
	}
}

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

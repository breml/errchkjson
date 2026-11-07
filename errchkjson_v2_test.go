//go:build go1.27

// encoding/json/v2 is part of the standard library as of Go 1.27. The test
// data for encoding/json/v2 can therefore only be type checked with Go 1.27
// or newer.

package errchkjson_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/breml/errchkjson"
)

func TestJSONv2(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson.NewAnalyzer(), "jsonv2")
}

func TestJSONv2OmitSafeFlag(t *testing.T) {
	errchkjson := errchkjson.NewAnalyzer()
	err := errchkjson.Flags.Set("omit-safe", "true")
	if err != nil {
		t.Fatalf("error setting 'omit-safe' command line flag: %v", err)
	}

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson, "jsonv2nosafe")
}

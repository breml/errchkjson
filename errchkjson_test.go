package errchkjson_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/breml/errchkjson"
)

// Because of the global nature of the analyzers flags, the order of the test cases needs to be maintained.
// Otherwise flags from previous tests could affect subsequent tests.

func TestDefault(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson.Analyzer, "standard")
}

func TestNoSafeFlag(t *testing.T) {
	err := errchkjson.Analyzer.Flags.Set("omit-safe", "true")
	if err != nil {
		t.Fatalf("error setting 'no-safe' command line flag: %v", err)
	}
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson.Analyzer, "nosafe")
}

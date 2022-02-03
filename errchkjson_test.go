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
	analysistest.Run(t, testdata, errchkjson.NewAnalyzer(), "standard")
}

func TestOmitSafeFlag(t *testing.T) {
	errchkjson := errchkjson.NewAnalyzer()
	err := errchkjson.Flags.Set("omit-safe", "true")
	if err != nil {
		t.Fatalf("error setting 'omit-safe' command line flag: %v", err)
	}
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson, "nosafe")
}

func TestNoExportedField(t *testing.T) {
	errchkjson := errchkjson.NewAnalyzer()
	err := errchkjson.Flags.Set("omit-safe", "true")
	if err != nil {
		t.Fatalf("error setting 'omit-safe' command line flag: %v", err)
	}
	err = errchkjson.Flags.Set("report-no-exported", "true")
	if err != nil {
		t.Fatalf("error setting 'report-no-exported' command line flag: %v", err)
	}
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson, "noexport")
}

func TestLoop(t *testing.T) {
	errchkjson := errchkjson.NewAnalyzer()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errchkjson, "loop")
}

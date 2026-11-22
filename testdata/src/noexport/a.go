package example

import (
	"encoding/json"
)

// JSONMarshalStructWithoutExportedFields contains a struct without exported fields.
func JSONMarshalStructWithoutExportedFields() {
	var err error

	var withoutExportedFields struct {
		privateField            bool
		ExportedButOmittedField bool `json:"-"`
	}
	_, err = json.Marshal(withoutExportedFields) // want "Error argument passed to `encoding/json.Marshal` does not contain any exported field"
	_ = err
}

type embeddedUnexportedPrivate struct {
	privateField bool
}

// JSONMarshalStructWithEmbeddedUnexportedStruct contains a struct, which only
// exports fields through an embedded struct of an unexported type. The
// promoted fields are checked, but they do not count as exported fields of the
// struct for the report-no-exported flag (see
// https://github.com/breml/errchkjson/issues/93).
func JSONMarshalStructWithEmbeddedUnexportedStruct() {
	var err error

	_, err = json.Marshal(struct{ embeddedUnexportedPrivate }{}) // want "Error argument passed to `encoding/json.Marshal` does not contain any exported field"
	_ = err
}

// JSONMarshalStructWithoutExportedFields contains a struct without exported fields.
func JSONMarshalStructWithNestedStructWithoutExportedFields() {
	var err error

	var withNestedStructWithoutExportedFields struct {
		ExportedStruct struct {
			privatField bool
		}
	}
	_, err = json.Marshal(withNestedStructWithoutExportedFields)
	_ = err
}

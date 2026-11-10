package jsonv2noexport

import (
	json "encoding/json/v2"
)

type inner struct {
	X int
}

// JSONv2MarshalStructWithoutExportedFields contains test cases with the
// report-no-exported flag set.
func JSONv2MarshalStructWithoutExportedFields() {
	var err error

	// The report-no-exported flag reports structs without exported fields,
	// even if encoding/json/v2 accepts them (because of the json struct tag).
	var withoutExportedFields struct {
		privateField            bool
		ExportedButOmittedField bool `json:"-"`
	}
	_ = withoutExportedFields.privateField
	_, err = json.Marshal(withoutExportedFields) // want "Error argument passed to `encoding/json/v2.Marshal` does not contain any exported field"
	_ = err

	// The report-no-exported flag takes precedence over the encoding/json/v2
	// rule for structs without exported fields.
	var unexportedOnly struct {
		privateField bool
	}
	_ = unexportedOnly.privateField
	_, err = json.Marshal(unexportedOnly) // want "Error argument passed to `encoding/json/v2.Marshal` does not contain any exported field"
	_ = err
	_, _ = json.Marshal(unexportedOnly) // want "Error argument passed to `encoding/json/v2.Marshal` does not contain any exported field" "Error return value of `encoding/json/v2.Marshal` is not checked: struct does not export any field"

	// The report-no-exported flag only applies to the top level struct, nested
	// structs are checked with the encoding/json/v2 rule.
	var nested struct {
		Inner struct {
			privateField bool
		}
	}
	_ = nested.Inner.privateField
	_, err = json.Marshal(nested) // want "`encoding/json/v2.Marshal` for unsupported struct .* without exported fields found"
	_ = err
	_, _ = json.Marshal(nested) // want "`encoding/json/v2.Marshal` for unsupported struct .* without exported fields found"

	// Fields promoted from embedded structs of unexported types do not count
	// as exported fields for the report-no-exported flag (same as for
	// encoding/json).
	var embedded struct {
		inner
	}
	_, err = json.Marshal(embedded) // want "Error argument passed to `encoding/json/v2.Marshal` does not contain any exported field"
	_ = err

	var exported struct {
		Field bool
	}
	_, err = json.Marshal(exported) // safe, but checking is not reported with omit-safe
	_ = err
}

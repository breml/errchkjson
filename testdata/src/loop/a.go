package example

import (
	"encoding/json"
)

type bqSchemaEntry struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Mode        string   `json:"mode"`
	Description string   `json:"description"`
	Fields      bqSchema `json:"fields"`
}

type bqSchema []bqSchemaEntry

// JSONMarshalStructWithoutExportedFields contains a struct without exported fields.
func JSONMarshalStructWithLoop() {
	var err error

	var withoutExportedFields bqSchema
	_, err = json.Marshal(withoutExportedFields) // want "Error argument passed to `encoding/json.Marshal` does not contain any exported field"
	_ = err
}

package example

import (
	"encoding/json"
)

type entry struct {
	Name   string `json:"name"`
	Fields schema `json:"fields"`
}

type schema []entry

// JSONMarshalStructWithLoop contains a struct with a loop. Recursive types
// are unsafe, see https://github.com/breml/errchkjson/issues/94.
func JSONMarshalStructWithLoop() {
	var structWithLoop schema
	_, _ = json.Marshal(structWithLoop) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe recursive type `loop.entry` found"
}

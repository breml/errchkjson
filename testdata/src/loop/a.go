package example

import (
	"encoding/json"
)

type entry struct {
	Name   string `json:"name"`
	Fields schema `json:"fields"`
}

type schema []entry

// JSONMarshalStructWithLoop contains a struct with a loop.
func JSONMarshalStructWithLoop() {
	var structWithLoop schema
	_, _ = json.Marshal(structWithLoop)
}

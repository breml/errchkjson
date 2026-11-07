package jsonv2nosafe

import (
	"bytes"
	json "encoding/json/v2"
)

// JSONv2MarshalOmitSafe contains test cases with the omit-safe flag set.
func JSONv2MarshalOmitSafe() {
	var err error

	var b bool
	_, _ = json.Marshal(b)   // want "Error return value of `encoding/json/v2.Marshal` is not checked"
	_, err = json.Marshal(b) // safe, but checking is not reported with omit-safe
	_ = err

	var str string
	_, _ = json.Marshal(str)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"
	_, err = json.Marshal(str) // err is checked
	_ = err

	var buf bytes.Buffer
	_ = json.MarshalWrite(&buf, b)   // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked"
	err = json.MarshalWrite(&buf, b) // err is checked
	_ = err

	ch := make(chan int)
	_, _ = json.Marshal(ch) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
}

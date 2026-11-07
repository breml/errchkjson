package jsonv2

import (
	"bytes"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io"
	"time"
	"unsafe"
)

type marshalText struct{}

func (_ marshalText) MarshalText() ([]byte, error) {
	return []byte(`mt`), nil
}

type marshalJSONTo struct{}

func (_ marshalJSONTo) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String("mjt"))
}

type appendText struct{}

func (_ appendText) AppendText(b []byte) ([]byte, error) {
	return append(b, `at`...), nil
}

type ptrMarshalJSON struct{ A int }

func (_ *ptrMarshalJSON) MarshalJSON() ([]byte, error) {
	return []byte(`"pmj"`), nil
}

type ptrMarshalText struct{ A int }

func (_ *ptrMarshalText) MarshalText() ([]byte, error) {
	return []byte(`pmt`), nil
}

type ptrMarshalJSONTo struct{ A int }

func (_ *ptrMarshalJSONTo) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String("pmjt"))
}

type ptrAppendText struct{ A int }

func (_ *ptrAppendText) AppendText(b []byte) ([]byte, error) {
	return append(b, `pat`...), nil
}

type ptrTextKey int

func (_ *ptrTextKey) MarshalText() ([]byte, error) {
	return []byte(`ptk`), nil
}

type structKey struct{ id int }

type unexportedOnly struct {
	a int
}

type inner struct {
	X int
}

// JSONv2MarshalSafeTypes contains test cases for types, that are safe to
// marshal with encoding/json/v2.
func JSONv2MarshalSafeTypes() {
	var err error

	_, _ = json.Marshal(nil)   // nil is safe
	json.Marshal(nil)          // nil is safe
	_, err = json.Marshal(nil) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	fmt.Print(json.Marshal(nil)) // nil is safe, error is passed as argument

	var b bool
	_, _ = json.Marshal(b)   // bool is safe
	_, err = json.Marshal(b) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var i int
	_, _ = json.Marshal(i)    // int is safe
	_, err = json.Marshal(&i) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var byteSlice []byte
	_, _ = json.Marshal(byteSlice)   // []byte is safe (base64)
	_, err = json.Marshal(byteSlice) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var mapIntBool map[int]bool
	_, _ = json.Marshal(mapIntBool)   // map[int]bool is safe
	_, err = json.Marshal(mapIntBool) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var safeStruct struct {
		Bool     bool
		Int      int    `json:"int,omitzero"`
		IntStr   int    `json:"int_str,string"`
		IntPtr   *int   `json:"int_ptr,string"`
		Ints     []int  `json:"ints"`
		Inner    inner  `json:"inner"`
		Embedded inner  // embedded fields are not checked for duplicate names
		Omitted  string `json:"-"`
		ignored  string // unexported
	}
	_ = safeStruct.ignored
	_, err = json.Marshal(safeStruct) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var emptyStruct struct{}
	_, err = json.Marshal(emptyStruct) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var floatStr struct {
		F float64 `json:"f,string"`
	}
	_, err = json.Marshal(floatStr) // float is unsafe, but the string option is valid
	_ = err

	var buf bytes.Buffer
	_ = json.MarshalWrite(&buf, b)   // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked"
	json.MarshalWrite(&buf, b)       // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked"
	err = json.MarshalWrite(&buf, b) // safe, but the io.Writer may return an error
	_ = err

	enc := jsontext.NewEncoder(io.Discard)
	_ = json.MarshalEncode(enc, b)   // want "Error return value of `encoding/json/v2.MarshalEncode` is not checked"
	err = json.MarshalEncode(enc, b) // safe, but the jsontext.Encoder may return an error
	_ = err
}

// JSONv2MarshalUnsafeTypes contains test cases for types, that can lead to an
// error, if marshaled with encoding/json/v2.
func JSONv2MarshalUnsafeTypes() {
	var err error

	var str string
	json.Marshal(str)          // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"
	_, _ = json.Marshal(str)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"
	_, err = json.Marshal(str) // err is checked
	_ = err
	fmt.Print(json.Marshal(str)) // err is passed and therefore considered as checked

	var safeV1 struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	_, _ = json.Marshal(safeV1)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"
	_, err = json.Marshal(safeV1) // err is checked
	_ = err

	var mapStrInt map[string]int
	_, _ = json.Marshal(mapStrInt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` as map key found"

	var mapF64Int map[float64]int
	_, _ = json.Marshal(mapF64Int) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `float64` as map key found"

	var f64 float64
	_, _ = json.Marshal(f64) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `float64` found"

	var ei any
	_, _ = json.Marshal(ei) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `any` found"

	var d time.Duration
	_, _ = json.Marshal(d)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"
	_, err = json.Marshal(d) // err is checked
	_ = err

	var durations struct {
		TTL   time.Duration            `json:"ttl"`
		Ptr   *time.Duration           `json:"ptr"`
		Slice []time.Duration          `json:"slice"`
		Map   map[int]time.Duration    `json:"map"`
		Key   map[time.Duration]int    `json:"key"`
		Deep  map[int][]*time.Duration `json:"deep"`
	}
	_, _ = json.Marshal(durations)         // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"
	_, _ = json.Marshal(durations.Ptr)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"
	_, _ = json.Marshal(durations.Slice)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"
	_, _ = json.Marshal(durations.Map)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"
	_, _ = json.Marshal(durations.Key)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` as map key found"
	_, _ = json.Marshal(durations.Deep)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"
	_, err = json.Marshal(durations.Deep)  // err is checked
	_, _ = stdjson.Marshal(durations.Deep) // time.Duration is safe with encoding/json
	_ = err

	var formatTag struct {
		When time.Time `json:"when,format:RFC3339"`
	}
	_, _ = json.Marshal(formatTag) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `format:RFC3339` on field `When` found"

	var formatTagDuration struct {
		D time.Duration `json:"d,format:nano"`
	}
	_, _ = json.Marshal(formatTagDuration) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `format:nano` on field `D` found"

	var quotedName struct {
		I int `json:"'a,b',format:foo"`
	}
	_, _ = json.Marshal(quotedName) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `format:foo` on field `I` found"

	var stringOptionBool struct {
		B bool `json:"b,string"`
	}
	_, _ = json.Marshal(stringOptionBool) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `string` on field `B` found"

	var stringOptionSlice struct {
		S []int `json:",string"`
	}
	_, _ = json.Marshal(stringOptionSlice) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `string` on field `S` found"

	var duplicateNames struct {
		A int `json:"x"`
		B int `json:"x"`
	}
	_, _ = json.Marshal(duplicateNames) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `x` in struct `struct{A int \"json:\\\\\"x\\\\\"\"; B int \"json:\\\\\"x\\\\\"\"}` found"

	var duplicateFieldName struct {
		X int
		Y int `json:"X"`
	}
	_, _ = json.Marshal(duplicateFieldName) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `X` in struct .* found"

	var uo unexportedOnly
	_, _ = json.Marshal(uo)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct `jsonv2.unexportedOnly` without exported fields found"
	_, _ = stdjson.Marshal(uo) // safe with encoding/json

	var mt marshalText
	_, _ = json.Marshal(mt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.marshalText` found"

	var mjt marshalJSONTo
	_, _ = json.Marshal(mjt)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.marshalJSONTo` found"
	_, _ = stdjson.Marshal(mjt) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `jsonv2.marshalJSONTo` found"

	var mapMJTString map[marshalJSONTo]string
	_, _ = stdjson.Marshal(mapMJTString) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `jsonv2.marshalJSONTo` as map key found"

	var at appendText
	_, _ = json.Marshal(at) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.appendText` found"

	// encoding/json/v2 always calls marshal methods with a pointer receiver,
	// even if the value is not addressable.
	var pmj ptrMarshalJSON
	_, _ = json.Marshal(pmj)      // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalJSON` found"
	_, _ = json.Marshal(&pmj)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalJSON` found"
	_, err = json.Marshal(pmj)    // err is checked
	_, err = stdjson.Marshal(pmj) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var pmt ptrMarshalText
	_, _ = json.Marshal(pmt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalText` found"

	var pmjt ptrMarshalJSONTo
	_, _ = json.Marshal(pmjt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalJSONTo` found"

	var pat ptrAppendText
	_, _ = json.Marshal(pat) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrAppendText` found"

	var pmjSlice []ptrMarshalJSON
	_, _ = json.Marshal(pmjSlice) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalJSON` found"

	var pmjMap map[int]ptrMarshalJSON
	_, _ = json.Marshal(pmjMap) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalJSON` found"

	var pmjField struct {
		F ptrMarshalJSONTo `json:"f"`
	}
	_, _ = json.Marshal(pmjField) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrMarshalJSONTo` found"

	var ptkMap map[ptrTextKey]int
	_, _ = json.Marshal(ptkMap)      // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.ptrTextKey` as map key found"
	_, err = stdjson.Marshal(ptkMap) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var buf bytes.Buffer
	_ = json.MarshalWrite(&buf, str)   // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"
	err = json.MarshalWrite(&buf, str) // err is checked
	_ = err

	enc := jsontext.NewEncoder(io.Discard)
	_ = json.MarshalEncode(enc, str) // want "Error return value of `encoding/json/v2.MarshalEncode` is not checked: unsafe type `string` found"
}

// JSONv2MarshalWithOptions contains test cases, where options are passed. The
// options may change the encoding of any type, therefore these are always
// considered unsafe.
func JSONv2MarshalWithOptions() {
	var err error

	var b bool
	_, _ = json.Marshal(b, json.Deterministic(true))   // want "Error return value of `encoding/json/v2.Marshal` is not checked: options passed, safety of the encoding can not be determined"
	_, err = json.Marshal(b, json.Deterministic(true)) // err is checked, no report about a safe argument
	_ = err

	opts := []json.Options{json.Deterministic(true)}
	_, _ = json.Marshal(b, opts...)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: options passed, safety of the encoding can not be determined"
	_, err = json.Marshal(b, opts...) // err is checked
	_ = err

	ch := make(chan int)
	_, err = json.Marshal(ch, opts...) // no report about an unsupported type, options may provide a custom marshaler
	_ = err

	var buf bytes.Buffer
	_ = json.MarshalWrite(&buf, b, opts...) // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: options passed, safety of the encoding can not be determined"
}

// JSONv2MarshalInvalidTypes contains test cases for types, that are not
// supported by encoding/json/v2.
func JSONv2MarshalInvalidTypes() {
	var err error

	ch := make(chan int)
	_, _ = json.Marshal(ch)   // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_, err = json.Marshal(ch) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err
	_, _ = json.Marshal(ch) //nolint:errchkjson // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"

	var c128 complex128
	_, _ = json.Marshal(c128) // want "`encoding/json/v2.Marshal` for unsupported type `complex128` found"

	f := func() {}
	_, _ = json.Marshal(f) // want "`encoding/json/v2.Marshal` for unsupported type `func\\(\\)` found"

	var unsafePtr unsafe.Pointer
	_, _ = json.Marshal(unsafePtr) // want "`encoding/json/v2.Marshal` for unsupported type `unsafe.Pointer` found"

	var mapBoolStr map[bool]int
	_, _ = json.Marshal(mapBoolStr) // want "`encoding/json/v2.Marshal` for unsupported type `bool` as map key found"

	var mapStructInt map[structKey]int
	_, _ = json.Marshal(mapStructInt) // want "`encoding/json/v2.Marshal` for unsupported type `jsonv2.structKey` as map key found"

	var buf bytes.Buffer
	_ = json.MarshalWrite(&buf, ch) // want "`encoding/json/v2.MarshalWrite` for unsupported type `chan int` found"

	enc := jsontext.NewEncoder(io.Discard)
	_ = json.MarshalEncode(enc, ch) // want "`encoding/json/v2.MarshalEncode` for unsupported type `chan int` found"
}

// Returned errors are the caller's responsibility.
func returned() ([]byte, error) {
	var str string
	return json.Marshal(str)
}

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

	expjsontext "github.com/go-json-experiment/json/jsontext"
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

type innerString struct {
	S string
}

type innerFloat struct {
	F float64
}

type innerDuration struct {
	D time.Duration
}

type EmbeddedInt int

type embeddedInt int

type EmbeddedSlice []int

type EmbeddedIface interface{ M() }

type namedString string

type namedInt int

type namedDuration time.Duration

type aliasDuration = time.Duration

// The following types have a MarshalJSONTo method, which does not match the
// encoding/json/v2.MarshalerTo interface.
type notMarshalerToResult struct{ A int }

func (_ notMarshalerToResult) MarshalJSONTo(enc *jsontext.Encoder) ([]byte, error) {
	return nil, nil
}

type notMarshalerToByValue struct{ A int }

func (_ notMarshalerToByValue) MarshalJSONTo(enc jsontext.Encoder) error {
	return nil
}

type notMarshalerToOptions struct{ A int }

func (_ notMarshalerToOptions) MarshalJSONTo(enc *jsontext.Encoder, opts json.Options) error {
	return nil
}

type notMarshalerToExperiment struct{ A int }

func (_ notMarshalerToExperiment) MarshalJSONTo(enc *expjsontext.Encoder) error {
	return nil
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
		Bool    bool
		Int     int    `json:"int,omitzero"`
		IntStr  int    `json:"int_str,string"`
		IntPtr  *int   `json:"int_ptr,string"`
		Ints    []int  `json:"ints"`
		Inner   inner  `json:"inner"`
		inner          // embedded struct, its fields are promoted
		Omitted string `json:"-"`
		ignored string // unexported
	}
	_ = safeStruct.ignored
	_, err = json.Marshal(safeStruct) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var emptyStruct struct{}
	_, err = json.Marshal(emptyStruct) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// encoding/json/v2 matches JSON object names case sensitive.
	var caseDistinctNames struct {
		A int `json:"x"`
		B int `json:"X"`
	}
	_, err = json.Marshal(caseDistinctNames) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// Named types with time.Duration as underlying type do not have the
	// methods of time.Duration and are encoded as integer.
	var nd namedDuration
	_, err = json.Marshal(nd) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var mapND map[namedDuration]int
	_, err = json.Marshal(mapND) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var stringOptionNamedInt struct {
		N namedInt `json:",string"`
	}
	_, err = json.Marshal(stringOptionNamedInt) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// Methods named MarshalJSONTo, which do not match the
	// encoding/json/v2.MarshalerTo interface, are ignored.
	_, err = json.Marshal(notMarshalerToResult{}) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(notMarshalerToByValue{}) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(notMarshalerToOptions{}) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(notMarshalerToExperiment{}) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = stdjson.Marshal(notMarshalerToExperiment{}) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
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
// error, if marshaled with encoding/json/v2. Some of these types always fail
// to encode with all versions of encoding/json/v2 (e.g. time.Duration or
// duplicate JSON object names) and are therefore reported as unsupported.
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

	var ns namedString
	_, _ = json.Marshal(ns) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.namedString` found"

	var strs []string
	_, _ = json.Marshal(strs) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	var mapNSInt map[namedString]int
	_, _ = json.Marshal(mapNSInt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.namedString` as map key found"

	var mapIntStrs map[int][]string
	_, _ = json.Marshal(mapIntStrs) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	var ad aliasDuration
	_, _ = json.Marshal(ad) // want "`encoding/json/v2.Marshal` for unsupported type `jsonv2.aliasDuration` found"
	var pad *aliasDuration
	_, _ = json.Marshal(pad) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `jsonv2.aliasDuration` found"
	var mapADInt map[aliasDuration]int
	_, _ = json.Marshal(mapADInt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.aliasDuration` as map key found"

	var stringOptionString struct {
		S string `json:",string"`
	}
	_, _ = json.Marshal(stringOptionString) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `string` on field `S` found"

	var stringOptionStruct struct {
		I inner `json:",string"`
	}
	_, _ = json.Marshal(stringOptionStruct) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `string` on field `I` found"

	var stringOptionBytes struct {
		B []byte `json:",string"`
	}
	_, _ = json.Marshal(stringOptionBytes) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `string` on field `B` found"

	var mapF64Int map[float64]int
	_, _ = json.Marshal(mapF64Int) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `float64` as map key found"

	var f64 float64
	_, _ = json.Marshal(f64) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `float64` found"

	var ei any
	_, _ = json.Marshal(ei) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `any` found"

	var d time.Duration
	_, _ = json.Marshal(d)   // want "`encoding/json/v2.Marshal` for unsupported type `time.Duration` found"
	_, err = json.Marshal(d) // want "`encoding/json/v2.Marshal` for unsupported type `time.Duration` found"
	_ = err

	var durations struct {
		TTL   time.Duration            `json:"ttl"`
		Ptr   *time.Duration           `json:"ptr"`
		Slice []time.Duration          `json:"slice"`
		Map   map[int]time.Duration    `json:"map"`
		Key   map[time.Duration]int    `json:"key"`
		Deep  map[int][]*time.Duration `json:"deep"`
	}
	_, _ = json.Marshal(durations)         // want "`encoding/json/v2.Marshal` for unsupported type `time.Duration` found"
	_, _ = json.Marshal(durations.Ptr)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `time.Duration` found"
	_, _ = json.Marshal(durations.Slice)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `time.Duration` found"
	_, _ = json.Marshal(durations.Map)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `time.Duration` found"
	_, _ = json.Marshal(durations.Key)     // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` as map key found"
	_, _ = json.Marshal(durations.Deep)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `time.Duration` found"
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

	// encoding/json/v2 rejects JSON object names containing a backslash or
	// quote characters. Single-quoted names are only rejected as of Go 1.27,
	// encoding/json/v2 of Go 1.26 accepts them (e.g. `'a,b'` is the name
	// `a,b`), they are therefore unsafe.
	var quotedName struct {
		I int `json:"'a,b'"`
	}
	_, _ = json.Marshal(quotedName)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `'a` on field `I` found"
	_, err = json.Marshal(quotedName) // err is checked
	_ = err

	var quotedNameOnly struct {
		I int `json:"'a'"`
	}
	_, _ = json.Marshal(quotedNameOnly) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `'a'` on field `I` found"

	var quotedNameFormat struct {
		I int `json:"'a,b',format:foo"`
	}
	_, _ = json.Marshal(quotedNameFormat) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `'a` on field `I` found"

	var quoteInName struct {
		I int `json:"a'b,omitempty"`
	}
	_, _ = json.Marshal(quoteInName) // want "`encoding/json/v2.Marshal` for unsupported JSON object name `a'b` on field `I` found"

	var backslashInName struct {
		I int `json:"a\\b"`
	}
	_, _ = json.Marshal(backslashInName) // want "`encoding/json/v2.Marshal` for unsupported JSON object name `a.b` on field `I` found"

	var doubleQuoteInName struct {
		I int `json:"a\"b"`
	}
	_, _ = json.Marshal(doubleQuoteInName) // want "`encoding/json/v2.Marshal` for unsupported JSON object name `a.b` on field `I` found"

	var validNames struct {
		A int `json:"a-b c.d"`
		B int `json:"üñí"`
		C int `json:",omitempty"`
	}
	_, err = json.Marshal(validNames) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

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
	_, _ = json.Marshal(duplicateNames) // want "`encoding/json/v2.Marshal` for unsupported duplicate JSON object name `x` in struct .* found"

	var duplicateFieldName struct {
		X int
		Y int `json:"X"`
	}
	_, _ = json.Marshal(duplicateFieldName) // want "`encoding/json/v2.Marshal` for unsupported duplicate JSON object name `X` in struct .* found"

	var uo unexportedOnly
	_, _ = json.Marshal(uo)    // want "`encoding/json/v2.Marshal` for unsupported struct `jsonv2.unexportedOnly` without exported fields found"
	_, _ = stdjson.Marshal(uo) // safe with encoding/json

	// encoding/json/v2 rejects json struct tags (other than `json:"-"`) on
	// unexported fields.
	var taggedUnexported struct {
		A int
		b int `json:"b"`
	}
	_ = taggedUnexported.b
	_, _ = json.Marshal(taggedUnexported)   // want "`encoding/json/v2.Marshal` for unsupported struct tag on unexported field `b` found"
	_, err = json.Marshal(taggedUnexported) // want "`encoding/json/v2.Marshal` for unsupported struct tag on unexported field `b` found"
	_ = err
	_, err = stdjson.Marshal(taggedUnexported) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var taggedUnexportedEmpty struct {
		A int
		b int `json:""`
	}
	_ = taggedUnexportedEmpty.b
	_, _ = json.Marshal(taggedUnexportedEmpty) // want "`encoding/json/v2.Marshal` for unsupported struct tag on unexported field `b` found"

	var taggedUnexportedOption struct {
		A int
		b int `json:",omitempty"`
	}
	_ = taggedUnexportedOption.b
	_, _ = json.Marshal(taggedUnexportedOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag on unexported field `b` found"

	var ignoredUnexported struct {
		A int
		b int `json:"-"`
		c int `xml:"c"`
	}
	_, _ = ignoredUnexported.b, ignoredUnexported.c
	_, err = json.Marshal(ignoredUnexported) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// encoding/json/v2 only rejects structs without exported fields, if no
	// field has a json struct tag.
	var unexportedOnlyIgnored struct {
		a int `json:"-"`
	}
	_ = unexportedOnlyIgnored.a
	_, err = json.Marshal(unexportedOnlyIgnored) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var ignoredExportedAndUnexported struct {
		A int `json:"-"`
		b int
	}
	_ = ignoredExportedAndUnexported.b
	_, err = json.Marshal(ignoredExportedAndUnexported) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var ignoredExportedOnly struct {
		A int `json:"-"`
	}
	_, err = json.Marshal(ignoredExportedOnly) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var unexportedOtherTag struct {
		a int `xml:"a"`
	}
	_ = unexportedOtherTag.a
	_, _ = json.Marshal(unexportedOtherTag) // want "`encoding/json/v2.Marshal` for unsupported struct .* without exported fields found"

	// encoding/json/v2 encodes the exported fields of embedded structs of
	// unexported types.
	var embeddedSafe struct {
		inner
		Y int
	}
	_, err = json.Marshal(embeddedSafe) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedString struct {
		innerString
	}
	_, _ = json.Marshal(embeddedString)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"
	_, err = json.Marshal(embeddedString) // err is checked
	_ = err

	var embeddedFloatPtr struct {
		*innerFloat
	}
	_, _ = json.Marshal(embeddedFloatPtr) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `float64` found"

	var embeddedDuration struct {
		innerDuration
		B int
	}
	_, _ = json.Marshal(embeddedDuration) // want "`encoding/json/v2.Marshal` for unsupported type `time.Duration` found"

	var embeddedNamed struct {
		innerString `json:"inner"`
	}
	_, _ = json.Marshal(embeddedNamed) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	var embeddedUnexportedOnly struct {
		unexportedOnly
		B int
	}
	_, _ = json.Marshal(embeddedUnexportedOnly) // want "`encoding/json/v2.Marshal` for unsupported struct `jsonv2.unexportedOnly` without exported fields found"

	var embeddedNamedDup struct {
		inner `json:"x"`
		X     int `json:"x"`
	}
	_, _ = json.Marshal(embeddedNamedDup) // want "`encoding/json/v2.Marshal` for unsupported duplicate JSON object name `x` in struct .* found"

	var embeddedPromotedConflict struct {
		inner
		X int
	}
	_, err = json.Marshal(embeddedPromotedConflict) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedIgnored struct {
		innerString `json:"-"`
		B           int
	}
	_, err = json.Marshal(embeddedIgnored) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// encoding/json/v2 rejects embedded fields of non-struct types without a
	// JSON name and embedded fields of unexported non-struct types.
	var embeddedNonStruct struct {
		EmbeddedInt
	}
	_, _ = json.Marshal(embeddedNonStruct)   // want "`encoding/json/v2.Marshal` for unsupported embedded field `EmbeddedInt` of non-struct type without JSON name found"
	_, err = json.Marshal(embeddedNonStruct) // want "`encoding/json/v2.Marshal` for unsupported embedded field `EmbeddedInt` of non-struct type without JSON name found"
	_ = err
	_, err = stdjson.Marshal(embeddedNonStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedNonStructPtr struct {
		*EmbeddedInt
	}
	_, _ = json.Marshal(embeddedNonStructPtr) // want "`encoding/json/v2.Marshal` for unsupported embedded field `EmbeddedInt` of non-struct type without JSON name found"

	var embeddedNonStructOption struct {
		EmbeddedInt `json:",omitempty"`
	}
	_, _ = json.Marshal(embeddedNonStructOption) // want "`encoding/json/v2.Marshal` for unsupported embedded field `EmbeddedInt` of non-struct type without JSON name found"

	var embeddedSlice struct {
		EmbeddedSlice
	}
	_, _ = json.Marshal(embeddedSlice) // want "`encoding/json/v2.Marshal` for unsupported embedded field `EmbeddedSlice` of non-struct type without JSON name found"

	var embeddedIface struct {
		EmbeddedIface
	}
	_, _ = json.Marshal(embeddedIface) // want "`encoding/json/v2.Marshal` for unsupported embedded field `EmbeddedIface` of non-struct type without JSON name found"

	var embeddedNonStructNamed struct {
		EmbeddedInt `json:"i"`
	}
	_, err = json.Marshal(embeddedNonStructNamed) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedUnexportedNonStruct struct {
		embeddedInt
		B int
	}
	_, _ = json.Marshal(embeddedUnexportedNonStruct)      // want "`encoding/json/v2.Marshal` for unsupported embedded field `embeddedInt` of unexported non-struct type found"
	_, err = stdjson.Marshal(embeddedUnexportedNonStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedUnexportedNonStructNamed struct {
		embeddedInt `json:"i"`
	}
	_, _ = json.Marshal(embeddedUnexportedNonStructNamed) // want "`encoding/json/v2.Marshal` for unsupported embedded field `embeddedInt` of unexported non-struct type found"

	var embeddedUnexportedNonStructIgnored struct {
		embeddedInt `json:"-"`
		B           int
	}
	_, err = json.Marshal(embeddedUnexportedNonStructIgnored) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedError struct {
		error
	}
	_, _ = json.Marshal(embeddedError) // want "`encoding/json/v2.Marshal` for unsupported embedded field `error` of unexported non-struct type found"

	var nestedUnexportedOnly struct {
		Inner unexportedOnly `json:"inner"`
	}
	_, _ = json.Marshal(nestedUnexportedOnly) // want "`encoding/json/v2.Marshal` for unsupported struct `jsonv2.unexportedOnly` without exported fields found"

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

	// encoding/json/v2 encodes nil and empty maps without considering the
	// key type, maps with unsupported key types are therefore unsafe.
	var mapBoolStr map[bool]int
	_, _ = json.Marshal(mapBoolStr)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `bool` as map key found"
	_, err = json.Marshal(mapBoolStr) // err is checked
	_ = err

	var mapStructInt map[structKey]int
	_, _ = json.Marshal(mapStructInt)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `jsonv2.structKey` as map key found"
	_, err = json.Marshal(mapStructInt) // err is checked
	_ = err

	var buf bytes.Buffer
	_ = json.MarshalWrite(&buf, ch) // want "`encoding/json/v2.MarshalWrite` for unsupported type `chan int` found"

	// The options of the encoder (e.g. json.WithMarshalers) can change the
	// encoding of any type.
	enc := jsontext.NewEncoder(io.Discard)
	_ = json.MarshalEncode(enc, ch)   // want "Error return value of `encoding/json/v2.MarshalEncode` is not checked: unsupported type `chan int` found"
	err = json.MarshalEncode(enc, ch) // err is checked
	_ = err
}

func writerAndBool() (io.Writer, bool)          { return io.Discard, true }
func writerAndString() (io.Writer, string)      { return io.Discard, "" }
func writerAndChan() (io.Writer, chan int)      { return io.Discard, nil }
func encoderAndBool() (*jsontext.Encoder, bool) { return jsontext.NewEncoder(io.Discard), true }
func boolAndOptions() (bool, json.Options)      { return true, json.Deterministic(true) }
func writerBoolAndOptions() (io.Writer, bool, json.Options) {
	return io.Discard, true, json.Deterministic(true)
}

// JSONv2MarshalTagOptions contains test cases for json struct tag options,
// which are rejected by encoding/json/v2.
func JSONv2MarshalTagOptions() {
	var err error

	var trailingComma struct {
		A int `json:"a,"`
	}
	_, _ = json.Marshal(trailingComma) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `` on field `A` found: malformed option"
	var emptyOption struct {
		A int `json:"a,,omitempty"`
	}
	_, _ = json.Marshal(emptyOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `` on field `A` found: malformed option"
	var quotedOption struct {
		A int `json:",'omitempty'"`
	}
	_, _ = json.Marshal(quotedOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `'omitempty'` on field `A` found: malformed option"
	var optionWithValue struct {
		A int `json:",omitempty:x"`
	}
	_, _ = json.Marshal(optionWithValue) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `omitempty:x` on field `A` found: malformed option"
	var misspelledOption struct {
		A int `json:",omitEmpty"`
	}
	_, _ = json.Marshal(misspelledOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `omitEmpty` on field `A` found: invalid spelling of `omitempty`"
	var underscoreOption struct {
		A int `json:",omit_empty"`
	}
	_, _ = json.Marshal(underscoreOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `omit_empty` on field `A` found: invalid spelling of `omitempty`"
	var upperCaseOption struct {
		A int `json:",OMITZERO"`
	}
	_, _ = json.Marshal(upperCaseOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `OMITZERO` on field `A` found: invalid spelling of `omitzero`"
	var duplicateOption struct {
		A int `json:",omitempty,omitempty"`
	}
	_, _ = json.Marshal(duplicateOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `omitempty` on field `A` found: duplicate option"
	var caseUnknown struct {
		A int `json:",case:foo"`
	}
	_, _ = json.Marshal(caseUnknown) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `case:foo` on field `A` found: invalid value, only `case:ignore` and `case:strict` are supported"
	var caseMissing struct {
		A int `json:",case"`
	}
	_, _ = json.Marshal(caseMissing) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `case` on field `A` found: invalid value, only `case:ignore` and `case:strict` are supported"
	var caseBoth struct {
		A int `json:",case:ignore,case:strict"`
	}
	_, _ = json.Marshal(caseBoth) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `case:strict` on field `A` found: duplicate option"
	var formatMissing struct {
		A int `json:",format"`
	}
	_, _ = json.Marshal(formatMissing) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `format` on field `A` found: missing value for `format`"
	var embedInt struct {
		A int `json:",embed"`
	}
	_, _ = json.Marshal(embedInt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `A` found: type `int` must be a struct, a map with string key or jsontext.Value"
	var embedSlice struct {
		A []int `json:",embed"`
	}
	_, _ = json.Marshal(embedSlice) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `A` found: type `\\[\\]int` must be a struct, a map with string key or jsontext.Value"
	var embedOmitzero struct {
		F inner `json:",embed,omitzero"`
	}
	_, _ = json.Marshal(embedOmitzero) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `F` found: option `omitzero` is not allowed, only `embed` is supported"
	var embedNamed struct {
		inner `json:"x,embed"`
	}
	_, _ = json.Marshal(embedNamed) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `inner` found: JSON object name `x` is not allowed"
	var embeddedOmitempty struct {
		inner `json:",omitempty"`
	}
	_, _ = json.Marshal(embeddedOmitempty) // want "`encoding/json/v2.Marshal` for unsupported embedded field `inner` found: option `omitempty` is not allowed, only `embed` is supported"
	var embeddedCase struct {
		inner `json:",case:ignore"`
	}
	_, _ = json.Marshal(embeddedCase)    // want "`encoding/json/v2.Marshal` for unsupported embedded field `inner` found: option `case` is not allowed, only `embed` is supported"
	_, err = json.Marshal(trailingComma) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `` on field `A` found: malformed option"
	_ = err
	_, err = stdjson.Marshal(trailingComma) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = stdjson.Marshal(misspelledOption) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = stdjson.Marshal(embedInt) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// Unknown options are ignored by encoding/json/v2.
	var unknownOption struct {
		A int `json:",foo"`
	}
	_, err = json.Marshal(unknownOption) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var unknownOptionUnderscore struct {
		A int `json:",foo_bar"`
	}
	_, err = json.Marshal(unknownOptionUnderscore) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var caseIgnore struct {
		A int `json:",case:ignore"`
	}
	_, err = json.Marshal(caseIgnore) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var caseStrict struct {
		A int `json:"a,case:strict,omitempty"`
	}
	_, err = json.Marshal(caseStrict) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var omitemptyOmitzero struct {
		A int `json:",omitempty,omitzero"`
	}
	_, err = json.Marshal(omitemptyOmitzero) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var embedStruct struct {
		F inner `json:",embed"`
		G int   `json:"F"`
	}
	// The `embed` option is not known by encoding/json/v2 of Go 1.26, F is
	// therefore a regular field and conflicts with G.
	_, _ = json.Marshal(embedStruct)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `F` in struct .* found"
	_, err = json.Marshal(embedStruct) // err is checked
	_ = err
	var embedStructPtr struct {
		F *inner `json:",embed"`
	}
	_, err = json.Marshal(embedStructPtr) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var embeddedUnknownOption struct {
		inner `json:",foo"`
	}
	_, err = json.Marshal(embeddedUnknownOption) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	var embeddedEmbedOption struct {
		inner `json:",embed"`
	}
	_, err = json.Marshal(embeddedEmbedOption) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
}

// JSONv2MarshalPointerMapKey contains test cases for pointer map keys, which
// are encoded like the pointed to value by encoding/json/v2.
func JSONv2MarshalPointerMapKey() {
	var err error

	var mapPtrInt map[*int]int
	_, _ = json.Marshal(mapPtrInt)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `\\*int` as map key found"
	_, err = json.Marshal(mapPtrInt) // err is checked
	_ = err

	var mapPtrNamedString map[*namedString]int
	_, _ = json.Marshal(mapPtrNamedString) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `\\*jsonv2.namedString` as map key found"

	var mapPtrBool map[*bool]int
	_, err = json.Marshal(mapPtrBool) // err is checked, nil and empty maps are encoded without error
	_ = err

	var mapPtrStruct map[*structKey]int
	_, err = json.Marshal(mapPtrStruct) // err is checked, nil and empty maps are encoded without error
	_ = err

	_, err = stdjson.Marshal(mapPtrInt) // err is checked, nil and empty maps are encoded without error as of Go 1.27
	_ = err
}

// JSONv2MarshalErrorTargets contains test cases for the different ways the
// returned error is handled.
func JSONv2MarshalErrorTargets() {
	var b bool
	var str string
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(io.Discard)

	if err := json.MarshalWrite(&buf, str); err != nil { // err is checked
		_ = err
	}
	if err := json.MarshalWrite(&buf, b); err != nil { // safe, but the io.Writer may return an error
		_ = err
	}
	if _, err := json.Marshal(b); err != nil { // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
		_ = err
	}
	if _, err := json.Marshal(str); err != nil { // err is checked
		_ = err
	}

	go json.MarshalWrite(&buf, str)    // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"
	defer json.MarshalWrite(&buf, str) // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"

	fmt.Print(json.MarshalEncode(enc, str)) // err is passed as argument

	_ = json.MarshalEncode(enc, b, json.Deterministic(true))    // want "Error return value of `encoding/json/v2.MarshalEncode` is not checked: options passed, safety of the encoding can not be determined"
	err := json.MarshalEncode(enc, b, json.Deterministic(true)) // err is checked
	_ = err
}

// JSONv2MarshalMultiValueArgument contains test cases, where the arguments are
// passed as the result of a call returning multiple values.
func JSONv2MarshalMultiValueArgument() {
	var err error

	_ = json.MarshalWrite(writerAndBool())   // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked"
	json.MarshalWrite(writerAndBool())       // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked"
	err = json.MarshalWrite(writerAndBool()) // safe, but the io.Writer may return an error
	_ = err
	fmt.Print(json.MarshalWrite(writerAndBool())) // err is passed as argument

	_ = json.MarshalWrite(writerAndString())   // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"
	err = json.MarshalWrite(writerAndString()) // err is checked
	_ = err

	_ = json.MarshalWrite(writerAndChan()) // want "`encoding/json/v2.MarshalWrite` for unsupported type `chan int` found"

	_ = json.MarshalEncode(encoderAndBool()) // want "Error return value of `encoding/json/v2.MarshalEncode` is not checked"

	_, _ = json.Marshal(boolAndOptions())   // want "Error return value of `encoding/json/v2.Marshal` is not checked: options passed, safety of the encoding can not be determined"
	_, err = json.Marshal(boolAndOptions()) // err is checked, no report about a safe argument
	_ = err

	_ = json.MarshalWrite(writerBoolAndOptions()) // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: options passed, safety of the encoding can not be determined"
}

// Returned errors are the caller's responsibility.
func returned() ([]byte, error) {
	var str string
	return json.Marshal(str)
}

type unmarshalJSONBase struct{ ID int }

func (_ *unmarshalJSONBase) UnmarshalJSON([]byte) error { return nil }

type UnmarshalTextBase struct{ ID int }

func (_ *UnmarshalTextBase) UnmarshalText([]byte) error { return nil }

type UnmarshalJSONFromBase struct{ ID int }

func (_ *UnmarshalJSONFromBase) UnmarshalJSONFrom(dec *jsontext.Decoder) error { return nil }

type unmarshalTextKey string

func (_ *unmarshalTextKey) UnmarshalText([]byte) error { return nil }

type isZeroInner struct{ ID int }

func (_ isZeroInner) IsZero() bool { return false }

// JSONv2MarshalEmbedMethods contains test cases for embedded fields, which
// implement methods to customize the encoding or decoding.
func JSONv2MarshalEmbedMethods() {
	var err error

	var embeddedUnmarshalJSON struct {
		unmarshalJSONBase
	}
	_, _ = json.Marshal(embeddedUnmarshalJSON)   // want "`encoding/json/v2.Marshal` for unsupported embedded field `unmarshalJSONBase` found: type `jsonv2.unmarshalJSONBase` must not implement marshal or unmarshal methods"
	_, err = json.Marshal(embeddedUnmarshalJSON) // want "`encoding/json/v2.Marshal` for unsupported embedded field `unmarshalJSONBase` found: type `jsonv2.unmarshalJSONBase` must not implement marshal or unmarshal methods"
	_ = err

	var embeddedPtrUnmarshalText struct {
		*UnmarshalTextBase
	}
	_, _ = json.Marshal(embeddedPtrUnmarshalText) // want "`encoding/json/v2.Marshal` for unsupported embedded field `UnmarshalTextBase` found: type `jsonv2.UnmarshalTextBase` must not implement marshal or unmarshal methods"

	var embedUnmarshalJSONFrom struct {
		F UnmarshalJSONFromBase `json:",embed"`
	}
	_, _ = json.Marshal(embedUnmarshalJSONFrom) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `F` found: type `jsonv2.UnmarshalJSONFromBase` must not implement marshal or unmarshal methods"

	var embedMapKeyUnmarshalText struct {
		M map[unmarshalTextKey]int `json:",embed"`
	}
	_, _ = json.Marshal(embedMapKeyUnmarshalText) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `M` found: map key type `jsonv2.unmarshalTextKey` must not implement marshal or unmarshal methods"

	var embedValue struct {
		inner
		N jsontext.Value `json:",embed"`
	}
	_, _ = json.Marshal(embedValue) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `encoding/json/jsontext.Value` found"

	// Methods of unexported types can not be called by encoding/json/v2.
	var unexportedNamedUnmarshalJSON struct {
		unmarshalJSONBase `json:"base"`
	}
	_, _ = json.Marshal(unexportedNamedUnmarshalJSON) // want "`encoding/json/v2.Marshal` for unsupported field `unmarshalJSONBase` of unexported type with methods for JSON encoding found"

	var unexportedNamedIsZeroOmitzero struct {
		isZeroInner `json:"in,omitzero"`
	}
	_, _ = json.Marshal(unexportedNamedIsZeroOmitzero) // want "`encoding/json/v2.Marshal` for unsupported field `isZeroInner` of unexported type with methods for JSON encoding found"

	var unexportedNamedIsZero struct {
		isZeroInner `json:"in"`
	}
	_, err = json.Marshal(unexportedNamedIsZero) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var exportedNamedUnmarshalText struct {
		UnmarshalTextBase `json:"base"`
	}
	_, err = json.Marshal(exportedNamedUnmarshalText) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var fieldUnmarshalJSON struct {
		F unmarshalJSONBase
	}
	_, err = json.Marshal(fieldUnmarshalJSON) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
}

// JSONv2MarshalGo126Options contains test cases for the json struct tag
// options `inline` and `unknown`, which are only known by encoding/json/v2 of
// Go 1.26 (GOEXPERIMENT=jsonv2).
func JSONv2MarshalGo126Options() {
	var err error

	var inlineInt struct {
		X int `json:",inline"`
	}
	_, _ = json.Marshal(inlineInt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `X` found: type `int` must be a struct, a map with string key or jsontext.Value"

	var unknownStruct struct {
		X inner `json:",unknown"`
	}
	_, _ = json.Marshal(unknownStruct) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `X` found: type `jsonv2.inner` must be a map with string key or jsontext.Value for option `unknown`"

	var inlineUnknown struct {
		X map[string]int `json:",inline,unknown"`
	}
	_, _ = json.Marshal(inlineUnknown) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `X` found: options `inline` and `unknown` can not be combined"

	var inlineNamed struct {
		X inner `json:"x,inline"`
	}
	_, _ = json.Marshal(inlineNamed) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `X` found: JSON object name `x` is not allowed"

	var inlineMisspelled struct {
		X int `json:",Inline"`
	}
	_, _ = json.Marshal(inlineMisspelled) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `Inline` on field `X` found: invalid spelling of `inline`"

	var inlineStruct struct {
		X inner `json:",inline"`
	}
	_, err = json.Marshal(inlineStruct) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var unknownValue struct {
		X jsontext.Value `json:",unknown"`
	}
	_, _ = json.Marshal(unknownValue) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `encoding/json/jsontext.Value` found"

	var embedInline struct {
		X inner `json:",embed,inline"`
	}
	_, err = json.Marshal(embedInline) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
}

type byteArrayKey [2]byte

type namedByte byte

type recursiveMapV2 map[int]recursiveMapV2

type recursiveSliceV2 []recursiveSliceV2

// JSONv2MarshalMiscellaneous contains further test cases for encoding/json/v2.
func JSONv2MarshalMiscellaneous() {
	var err error

	// Unknown options are ignored, unless they appear more than once.
	var duplicateUnknownOption struct {
		A int `json:"a,foo,foo"`
	}
	_, _ = json.Marshal(duplicateUnknownOption) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `foo` on field `A` found: duplicate option"

	var invalidUTF8Name struct {
		A int "json:\"\\xff\""
	}
	_, _ = json.Marshal(invalidUTF8Name)      // want "`encoding/json/v2.Marshal` for unsupported JSON object name with invalid UTF-8 on field `A` found"
	_, err = stdjson.Marshal(invalidUTF8Name) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// Arrays of bytes are encoded as base64 encoded string.
	var mapByteArrayKey map[[2]byte]int
	_, err = json.Marshal(mapByteArrayKey) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var mapNamedByteArrayKey map[byteArrayKey]int
	_, err = json.Marshal(mapNamedByteArrayKey) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var mapPtrByteArrayKey map[*[2]byte]int
	_, _ = json.Marshal(mapPtrByteArrayKey) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `\\*\\[2\\]byte` as map key found"

	var mapNamedByteElemKey map[[2]namedByte]int
	_, err = json.Marshal(mapNamedByteElemKey) // err is checked, nil and empty maps are encoded without error
	_ = err

	var mapIntArrayKey map[[2]int]int
	_, err = json.Marshal(mapIntArrayKey) // err is checked, nil and empty maps are encoded without error
	_ = err

	_, err = stdjson.Marshal(mapByteArrayKey) // err is checked, nil and empty maps are encoded without error as of Go 1.27
	_ = err

	// Recursive types, see JSONv2MarshalRecursiveTypes.
	var recursiveMap recursiveMapV2
	_, _ = json.Marshal(recursiveMap)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.recursiveMapV2` found"
	_, err = json.Marshal(recursiveMap) // err is checked
	_ = err

	var recursiveSlice recursiveSliceV2
	_, _ = json.Marshal(recursiveSlice)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.recursiveSliceV2` found"
	_, err = json.Marshal(recursiveSlice) // err is checked
	_ = err
}

// JSONv2MarshalAssignments contains test cases for multi-value assignments
// and var declarations.
func JSONv2MarshalAssignments() {
	var b bool
	var str string
	var buf bytes.Buffer

	n, err := 0, json.MarshalWrite(&buf, str) // err is checked
	_, _ = n, err

	_, _ = 0, json.MarshalWrite(&buf, str) // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"

	errA, _ := json.MarshalWrite(&buf, b), json.MarshalWrite(&buf, str) // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"
	_ = errA

	var errVar = json.MarshalWrite(&buf, str) // err is checked
	_ = errVar

	var _ = json.MarshalWrite(&buf, str) // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked: unsafe type `string` found"

	var _, errSafe = json.Marshal(b) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = errSafe

	var data, errUnsafe = json.Marshal(str) // err is checked
	_, _ = data, errUnsafe

	var _, _ = json.Marshal(str) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"
}

type selfPtrKey *selfPtrKey

type selfPtr *selfPtr

// JSONv2MarshalSelfReferencingPointers contains test cases for named pointer
// types referencing themselves.
func JSONv2MarshalSelfReferencingPointers() {
	var err error

	var mapSelfPtrKey map[selfPtrKey]int
	_, _ = json.Marshal(mapSelfPtrKey)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.selfPtrKey` as map key found"
	_, err = json.Marshal(mapSelfPtrKey) // err is checked
	_ = err

	// The `string` option never fails for a self-referencing pointer type,
	// because every value ends in nil or a cycle. A cycle is not detected by
	// encoding/json/v2 and leads to a stack overflow (see
	// https://go.dev/issue/82947), the type is therefore unsafe as recursive
	// type.
	var stringSelfPtr struct {
		X selfPtr `json:",string"`
	}
	_, _ = json.Marshal(stringSelfPtr)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.selfPtr` found"
	_, err = json.Marshal(stringSelfPtr) // err is checked
	_ = err
}

// JSONv2MarshalDashName contains test cases for the JSON object name `-`,
// which is rejected by encoding/json/v2 of Go 1.26 (GOEXPERIMENT=jsonv2).
func JSONv2MarshalDashName() {
	var dashName struct {
		A int `json:"-,omitempty"`
	}
	_, _ = json.Marshal(dashName)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `-` on field `A` found"
	_, err := json.Marshal(dashName) // err is checked
	_ = err

	// encoding/json does not reject the JSON object name `-`.
	_, err = stdjson.Marshal(dashName) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

// JSONv2MarshalUnsupportedPrecedence contains test cases for structs with
// unsafe and unsupported fields, the unsupported field takes precedence
// independent of the order of the fields.
func JSONv2MarshalUnsupportedPrecedence() {
	var stringBeforeChan struct {
		Name string
		C    chan int
	}
	_, err := json.Marshal(stringBeforeChan) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var mapBeforeFunc struct {
		Labels map[string]string
		Fn     func()
	}
	_, err = json.Marshal(mapBeforeFunc) // want "`encoding/json/v2.Marshal` for unsupported type `func\\(\\)` found"
	_ = err

	// The `string` option on non-numeric types is only rejected by
	// encoding/json/v2 of Go 1.27 (unsafe).
	var tagBeforeChan struct {
		A string   `json:"a,string"`
		C chan int `json:"c,string"`
	}
	_, err = json.Marshal(tagBeforeChan) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	// encoding/json reports the first unsafe field (unchanged behavior, known
	// false negative).
	var floatBeforeChan struct {
		F float64
		C chan int
	}
	_, err = stdjson.Marshal(floatBeforeChan) // err is checked
	_ = err
}

// JSONv2MarshalUnsupportedNotEncoded contains test cases for unsupported
// types, which are not encoded by encoding/json/v2 for some values (e.g. nil or
// empty slices and maps, nil pointers or omitted zero values). These types
// are therefore unsafe instead of unsupported.
func JSONv2MarshalUnsupportedNotEncoded() {
	var err error

	var sliceChan []chan int
	_, _ = json.Marshal(sliceChan)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(sliceChan) // err is checked
	_ = err

	var mapChan map[int]chan int
	_, _ = json.Marshal(mapChan)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(mapChan) // err is checked
	_ = err

	var ptrChanField struct {
		P *chan int
	}
	_, _ = json.Marshal(ptrChanField)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(ptrChanField) // err is checked
	_ = err

	// Arrays of length 0 are encoded without considering the element type.
	var emptyArrayChan [0]chan int
	_, err = json.Marshal(emptyArrayChan) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var omitzeroFunc struct {
		F func() `json:",omitzero"`
	}
	_, _ = json.Marshal(omitzeroFunc)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `func\\(\\)` found"
	_, err = json.Marshal(omitzeroFunc) // err is checked
	_ = err

	// Always encoded, therefore unsupported.
	var arrayChan [1]chan int
	_, err = json.Marshal(arrayChan) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var omitemptyFunc struct {
		F func() `json:",omitempty"`
	}
	_, err = json.Marshal(omitemptyFunc) // want "`encoding/json/v2.Marshal` for unsupported type `func\\(\\)` found"
	_ = err

	// encoding/json also encodes nil and empty slices without error.
	_, err = stdjson.Marshal(sliceChan) // err is checked, nil and empty slices are encoded without error
	_ = err
}

// JSONv2MarshalEncodeEncoderOptions contains test cases for the options of
// the jsontext.Encoder passed to MarshalEncode.
func JSONv2MarshalEncodeEncoderOptions() {
	enc := jsontext.NewEncoder(io.Discard, json.WithMarshalers(json.MarshalFunc(func(c chan int) ([]byte, error) {
		return []byte(`"chan"`), nil
	})))
	err := json.MarshalEncode(enc, make(chan int)) // err is checked
	_ = err
}

type node struct {
	ID   int
	Next *node
}

type treeNode struct {
	Children []treeNode
}

type mutualA struct {
	B *mutualB
}

type mutualB struct {
	A []mutualA
}

type genericList[T any] struct {
	Value T
	Next  *genericList[T]
}

type notRecursive struct {
	A inner
	B inner
	C *inner
}

type recursiveIgnored struct {
	ID   int
	Next *recursiveIgnored `json:"-"`
	prev *recursiveIgnored
}

// JSONv2MarshalRecursiveTypes contains test cases for recursive types.
// encoding/json/v2 returns an error for cyclic values and for values with a
// nesting depth larger than 10000, recursive types are therefore unsafe (see
// https://github.com/breml/errchkjson/issues/94).
func JSONv2MarshalRecursiveTypes() {
	var err error

	var n node
	_, _ = json.Marshal(n)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.node` found"
	_, err = json.Marshal(n) // err is checked
	_ = err

	_, _ = json.Marshal(&n) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.node` found"

	var nodes []node
	_, _ = json.Marshal(nodes) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.node` found"

	var tree treeNode
	_, _ = json.Marshal(tree) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.treeNode` found"

	var mutual mutualA
	_, _ = json.Marshal(mutual) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.mutualA` found"

	var list genericList[int]
	_, _ = json.Marshal(list) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.genericList\\[int\\]` found"

	var p selfPtr
	_, _ = json.Marshal(p) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.selfPtr` found"

	// Types used more than once are not recursive.
	var nr notRecursive
	_, err = json.Marshal(nr) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// Fields, which are not encoded, are not considered.
	var ri recursiveIgnored
	_, err = json.Marshal(ri) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// encoding/json is unchanged (known false negative, see
	// https://github.com/breml/errchkjson/issues/94).
	_, err = stdjson.Marshal(n) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

type cacheChan chan int

type cacheStruct struct {
	C chan int
}

type shadowedChanInner struct {
	C chan int
	N int
}

type shadowedChanOuter struct {
	shadowedChanInner
	C int
}

type ambiguousChan1 struct {
	X chan int
}

type ambiguousChan2 struct {
	X chan int
}

type ambiguousChanOuter struct {
	ambiguousChan1
	ambiguousChan2
	N int
}

type shadowedStringInner struct {
	S string
	N int
}

type shadowedStringOuter struct {
	shadowedStringInner
	S int
}

type taggedChan struct {
	X chan int `json:"X"`
}

type untaggedInt struct {
	X int
}

type taggedInt struct {
	X int `json:"X"`
}

type untaggedChan struct {
	X chan int
}

type taggedChanOuter struct {
	taggedChan
	untaggedInt
}

type untaggedChanOuter struct {
	taggedInt
	untaggedChan
}

type embedSelf struct {
	*embedSelf
	X int
}

type embedMutual1 struct {
	*embedMutual2
	X int
}

type embedMutual2 struct {
	*embedMutual1
	Y int
}

type embedSelfNamed struct {
	*embedSelfNamed `json:"self"`
	X               int
}

type embedRecursiveField struct {
	*embedRecursiveFieldNext
}

type embedRecursiveFieldNext struct {
	Next *embedRecursiveField
}

type embeddedBroken struct {
	A int
	B int `json:"A"`
}

type embeddedChan struct {
	C chan int
}

// JSONv2MarshalFieldResolution contains test cases for the resolution of the
// fields, which are encoded by encoding/json/v2, and for the caching of the
// results of already checked types.
func JSONv2MarshalFieldResolution() {
	var err error

	// A type checked behind a slice, pointer or `omitzero` field is still
	// unsupported, if it is used directly.
	var sliceThenDirect struct {
		A []cacheChan
		B cacheChan
	}
	_, err = json.Marshal(sliceThenDirect) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var ptrThenDirect struct {
		P *cacheStruct
		V cacheStruct
	}
	_, err = json.Marshal(ptrThenDirect) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var omitzeroThenDirect struct {
		A cacheStruct `json:",omitzero"`
		B cacheStruct
	}
	_, err = json.Marshal(omitzeroThenDirect) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var sliceThenDirectStd struct {
		A []cacheChan
		B cacheChan
	}
	_, err = stdjson.Marshal(sliceThenDirectStd) // want "`encoding/json.Marshal` for unsupported type `chan int` found"
	_ = err

	// Fields shadowed by a less nested field or ambiguous fields at the same
	// depth are not encoded.
	var shadowedChan shadowedChanOuter
	_, err = json.Marshal(shadowedChan) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var ambiguousChan ambiguousChanOuter
	_, err = json.Marshal(ambiguousChan) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var shadowedString shadowedStringOuter
	_, err = json.Marshal(shadowedString) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// For fields at the same depth, a field with an explicit JSON name takes
	// precedence over a field without JSON name.
	var taggedChanWins taggedChanOuter
	_, err = json.Marshal(taggedChanWins) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var taggedIntWins untaggedChanOuter
	_, err = json.Marshal(taggedIntWins) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	// Embedding a struct in itself is not recursive, because an embedded
	// struct is only visited once.
	var self embedSelf
	_, err = json.Marshal(self) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var mutual embedMutual1
	_, err = json.Marshal(mutual) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var selfNamed embedSelfNamed
	_, _ = json.Marshal(selfNamed) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.embedSelfNamed` found"

	var recursiveField embedRecursiveField
	_, _ = json.Marshal(recursiveField) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe recursive type `jsonv2.embedRecursiveField` found"

	// The definition of an embedded struct is checked, even if it is
	// embedded through a nil pointer, but its fields are not encoded.
	var embedBrokenPtr struct {
		*embeddedBroken
	}
	_, err = json.Marshal(embedBrokenPtr) // want "`encoding/json/v2.Marshal` for unsupported duplicate JSON object name `A` in struct `jsonv2.embeddedBroken` found"
	_ = err

	var embedChanPtr struct {
		*embeddedChan
	}
	_, _ = json.Marshal(embedChanPtr)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(embedChanPtr) // err is checked
	_ = err

	var embedChan struct {
		embeddedChan
	}
	_, err = json.Marshal(embedChan) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	// A nil pointer is encoded as null, unless the pointer is known to be
	// non-nil.
	var chanPtr *chan int
	_, _ = json.Marshal(chanPtr)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(chanPtr) // err is checked
	_ = err

	ch := make(chan int)
	_, err = json.Marshal(&ch) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err
	_, err = json.Marshal((new(chan int))) // want "`encoding/json/v2.Marshal` for unsupported type `chan int` found"
	_ = err

	var durationPtr *time.Duration
	_, err = json.Marshal(durationPtr) // err is checked
	_ = err
}

// JSONv2MarshalAlwaysFailing contains test cases for struct definitions,
// which are only rejected by some versions of encoding/json/v2 (unsafe) or by
// all versions (unsupported).
func JSONv2MarshalAlwaysFailing() {
	var err error

	// encoding/json/v2 of Go 1.26 supports time.Duration with the `format`
	// option.
	var durationFormat struct {
		D time.Duration `json:",format:units"`
	}
	_, _ = json.Marshal(durationFormat)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `format:units` on field `D` found"
	_, err = json.Marshal(durationFormat) // err is checked
	_ = err

	// Conflicts with fields with the `embed` option are only rejected by
	// encoding/json/v2 of Go 1.26.
	var dupEmbedOption struct {
		Inner inner `json:",embed"`
		B     int   `json:"Inner"`
	}
	_, _ = json.Marshal(dupEmbedOption)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `Inner` in struct .* found"
	_, err = json.Marshal(dupEmbedOption) // err is checked
	_ = err

	// Single-quoted options are accepted by encoding/json/v2 of Go 1.26,
	// unless they only consist of letters and digits.
	var quotedOption struct {
		A int `json:"a,'x y'"`
	}
	_, _ = json.Marshal(quotedOption)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `'x y'` on field `A` found: malformed option"
	_, err = json.Marshal(quotedOption) // err is checked
	_ = err

	// Single-quoted names may contain commas for encoding/json/v2 of Go 1.26.
	var quotedNameOptions struct {
		A int `json:"'a,b',omitempty"`
	}
	_, _ = json.Marshal(quotedNameOptions)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `'a` on field `A` found"
	_, err = json.Marshal(quotedNameOptions) // err is checked
	_ = err

	// Misspellings of `embed` are only rejected by Go 1.27, misspellings of
	// `inline` and `unknown` only by Go 1.26.
	var misspelledEmbed struct {
		A int `json:"a,Embed"`
	}
	_, _ = json.Marshal(misspelledEmbed)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `Embed` on field `A` found: invalid spelling of `embed`"
	_, err = json.Marshal(misspelledEmbed) // err is checked
	_ = err

	var misspelledInline struct {
		B int `json:"b,Inline"`
	}
	_, _ = json.Marshal(misspelledInline)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `Inline` on field `B` found: invalid spelling of `inline`"
	_, err = json.Marshal(misspelledInline) // err is checked
	_ = err

	// In combination, struct tags rejected by different versions are rejected
	// by all versions.
	var misspelledEmbedAndInline struct {
		A int `json:"a,Embed"`
		B int `json:"b,Inline"`
	}
	_, err = json.Marshal(misspelledEmbedAndInline) // want "`encoding/json/v2.Marshal` for unsupported struct definition \\(rejected by all versions of encoding/json/v2\\) found: unsafe struct tag option `Embed` on field `A` found: invalid spelling of `embed`"
	_ = err

	var dashNameAndString struct {
		A string `json:"-,string"`
	}
	_, err = json.Marshal(dashNameAndString) // want "`encoding/json/v2.Marshal` for unsupported struct definition \\(rejected by all versions of encoding/json/v2\\) found: unsafe JSON object name `-` on field `A` found"
	_ = err

	// Behind a pointer, struct definitions rejected by all versions in
	// combination are unsafe.
	var misspelledPtr *struct {
		A int `json:"a,Embed"`
		B int `json:"b,Inline"`
	}
	_, _ = json.Marshal(misspelledPtr)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported struct definition"
	_, err = json.Marshal(misspelledPtr) // err is checked
	_ = err

	// The `unknown` option on an embedded struct is only rejected by
	// encoding/json/v2 of Go 1.26.
	var embeddedUnknown struct {
		inner `json:",unknown"`
	}
	_, _ = json.Marshal(embeddedUnknown)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `inner` found: type `jsonv2.inner` must be a map with string key or jsontext.Value for option `unknown`"
	_, err = json.Marshal(embeddedUnknown) // err is checked
	_ = err

	// The JSON object name `-` with a trailing comma is rejected by all
	// versions.
	var dashTrailingComma struct {
		A int `json:"-,"`
	}
	_, err = json.Marshal(dashTrailingComma) // want "`encoding/json/v2.Marshal` for unsupported struct tag option `` on field `A` found: malformed option"
	_ = err

	// Struct definitions rejected by all versions are unsafe behind slices,
	// pointers and maps, because these struct definitions are only checked,
	// if a value is encoded.
	var sliceDup []struct {
		A int
		B int `json:"A"`
	}
	_, _ = json.Marshal(sliceDup)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsupported duplicate JSON object name `A` in struct .* found"
	_, err = json.Marshal(sliceDup) // err is checked
	_ = err
}

type constString string

type constMarshalText string

func (constMarshalText) MarshalText() ([]byte, error) {
	return []byte(`mt`), nil
}

const (
	validString   constString      = "valid"
	invalidString constString      = "\xff"
	floatConst                     = 1.5
	marshalConst  constMarshalText = "mt"
)

// JSONv2MarshalConstants contains test cases for constant arguments. A string
// constant with valid UTF-8 and a float constant (constants are always finite)
// are always encoded without error.
func JSONv2MarshalConstants() {
	var err error
	var buf bytes.Buffer

	_, err = json.Marshal("hello") // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(1.5) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(("hello")) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(validString) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_, err = json.Marshal(float32(floatConst)) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err
	_ = json.MarshalWrite(&buf, "hello") // want "Error return value of `encoding/json/v2.MarshalWrite` is not checked"

	// A string constant with invalid UTF-8 fails.
	_, _ = json.Marshal(invalidString) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.constString` found"
	_, _ = json.Marshal("\xff")        // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	// The encoding of types with methods does not only depend on the value.
	_, _ = json.Marshal(marshalConst) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.constMarshalText` found"

	// time.Duration is always unsupported.
	_, err = json.Marshal(time.Second) // want "`encoding/json/v2.Marshal` for unsupported type `time.Duration` found"
	_ = err

	// Options may change the encoding of any value.
	_, _ = json.Marshal("hello", json.Deterministic(true)) // want "Error return value of `encoding/json/v2.Marshal` is not checked: options passed, safety of the encoding can not be determined"

	// For backwards compatibility, constants are not considered for
	// encoding/json.
	_, _ = stdjson.Marshal(1.5) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
}

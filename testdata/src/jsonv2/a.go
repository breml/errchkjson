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

	var ns namedString
	_, _ = json.Marshal(ns) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.namedString` found"

	var strs []string
	_, _ = json.Marshal(strs) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	var mapNSInt map[namedString]int
	_, _ = json.Marshal(mapNSInt) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.namedString` as map key found"

	var mapIntStrs map[int][]string
	_, _ = json.Marshal(mapIntStrs) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	var ad aliasDuration
	_, _ = json.Marshal(ad) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.aliasDuration` found"
	var pad *aliasDuration
	_, _ = json.Marshal(pad) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `jsonv2.aliasDuration` found"
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

	// encoding/json/v2 rejects JSON object names containing a backslash or
	// quote characters, including single-quoted names.
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
	_, _ = json.Marshal(quoteInName) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `a'b` on field `I` found"

	var backslashInName struct {
		I int `json:"a\\b"`
	}
	_, _ = json.Marshal(backslashInName) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `a.b` on field `I` found"

	var doubleQuoteInName struct {
		I int `json:"a\"b"`
	}
	_, _ = json.Marshal(doubleQuoteInName) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe JSON object name `a.b` on field `I` found"

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
	_, _ = json.Marshal(duplicateNames) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `x` in struct .* found"

	var duplicateFieldName struct {
		X int
		Y int `json:"X"`
	}
	_, _ = json.Marshal(duplicateFieldName) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `X` in struct .* found"

	var uo unexportedOnly
	_, _ = json.Marshal(uo)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct `jsonv2.unexportedOnly` without exported fields found"
	_, _ = stdjson.Marshal(uo) // safe with encoding/json

	// encoding/json/v2 rejects json struct tags (other than `json:"-"`) on
	// unexported fields.
	var taggedUnexported struct {
		A int
		b int `json:"b"`
	}
	_ = taggedUnexported.b
	_, _ = json.Marshal(taggedUnexported)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag on unexported field `b` found"
	_, err = json.Marshal(taggedUnexported) // err is checked
	_ = err
	_, err = stdjson.Marshal(taggedUnexported) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var taggedUnexportedEmpty struct {
		A int
		b int `json:""`
	}
	_ = taggedUnexportedEmpty.b
	_, _ = json.Marshal(taggedUnexportedEmpty) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag on unexported field `b` found"

	var taggedUnexportedOption struct {
		A int
		b int `json:",omitempty"`
	}
	_ = taggedUnexportedOption.b
	_, _ = json.Marshal(taggedUnexportedOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag on unexported field `b` found"

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
	_, _ = json.Marshal(unexportedOtherTag) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct .* without exported fields found"

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
	_, _ = json.Marshal(embeddedDuration) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `time.Duration` found"

	var embeddedNamed struct {
		innerString `json:"inner"`
	}
	_, _ = json.Marshal(embeddedNamed) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe type `string` found"

	var embeddedUnexportedOnly struct {
		unexportedOnly
		B int
	}
	_, _ = json.Marshal(embeddedUnexportedOnly) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct `jsonv2.unexportedOnly` without exported fields found"

	var embeddedNamedDup struct {
		inner `json:"x"`
		X     int `json:"x"`
	}
	_, _ = json.Marshal(embeddedNamedDup) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe duplicate JSON object name `x` in struct .* found"

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
	_, _ = json.Marshal(embeddedNonStruct)   // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `EmbeddedInt` of non-struct type without JSON name found"
	_, err = json.Marshal(embeddedNonStruct) // err is checked
	_ = err
	_, err = stdjson.Marshal(embeddedNonStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedNonStructPtr struct {
		*EmbeddedInt
	}
	_, _ = json.Marshal(embeddedNonStructPtr) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `EmbeddedInt` of non-struct type without JSON name found"

	var embeddedNonStructOption struct {
		EmbeddedInt `json:",omitempty"`
	}
	_, _ = json.Marshal(embeddedNonStructOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `EmbeddedInt` of non-struct type without JSON name found"

	var embeddedSlice struct {
		EmbeddedSlice
	}
	_, _ = json.Marshal(embeddedSlice) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `EmbeddedSlice` of non-struct type without JSON name found"

	var embeddedIface struct {
		EmbeddedIface
	}
	_, _ = json.Marshal(embeddedIface) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `EmbeddedIface` of non-struct type without JSON name found"

	var embeddedNonStructNamed struct {
		EmbeddedInt `json:"i"`
	}
	_, err = json.Marshal(embeddedNonStructNamed) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedUnexportedNonStruct struct {
		embeddedInt
		B int
	}
	_, _ = json.Marshal(embeddedUnexportedNonStruct)      // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `embeddedInt` of unexported non-struct type found"
	_, err = stdjson.Marshal(embeddedUnexportedNonStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedUnexportedNonStructNamed struct {
		embeddedInt `json:"i"`
	}
	_, _ = json.Marshal(embeddedUnexportedNonStructNamed) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `embeddedInt` of unexported non-struct type found"

	var embeddedUnexportedNonStructIgnored struct {
		embeddedInt `json:"-"`
		B           int
	}
	_, err = json.Marshal(embeddedUnexportedNonStructIgnored) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
	_ = err

	var embeddedError struct {
		error
	}
	_, _ = json.Marshal(embeddedError) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `error` of unexported non-struct type found"

	var nestedUnexportedOnly struct {
		Inner unexportedOnly `json:"inner"`
	}
	_, _ = json.Marshal(nestedUnexportedOnly) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct `jsonv2.unexportedOnly` without exported fields found"

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
	_, _ = json.Marshal(trailingComma) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `` on field `A` found: malformed option"
	var emptyOption struct {
		A int `json:"a,,omitempty"`
	}
	_, _ = json.Marshal(emptyOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `` on field `A` found: malformed option"
	var quotedOption struct {
		A int `json:",'omitempty'"`
	}
	_, _ = json.Marshal(quotedOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `'omitempty'` on field `A` found: malformed option"
	var optionWithValue struct {
		A int `json:",omitempty:x"`
	}
	_, _ = json.Marshal(optionWithValue) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `omitempty:x` on field `A` found: malformed option"
	var misspelledOption struct {
		A int `json:",omitEmpty"`
	}
	_, _ = json.Marshal(misspelledOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `omitEmpty` on field `A` found: invalid spelling of `omitempty`"
	var underscoreOption struct {
		A int `json:",omit_empty"`
	}
	_, _ = json.Marshal(underscoreOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `omit_empty` on field `A` found: invalid spelling of `omitempty`"
	var upperCaseOption struct {
		A int `json:",OMITZERO"`
	}
	_, _ = json.Marshal(upperCaseOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `OMITZERO` on field `A` found: invalid spelling of `omitzero`"
	var duplicateOption struct {
		A int `json:",omitempty,omitempty"`
	}
	_, _ = json.Marshal(duplicateOption) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `omitempty` on field `A` found: duplicate option"
	var caseUnknown struct {
		A int `json:",case:foo"`
	}
	_, _ = json.Marshal(caseUnknown) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `case:foo` on field `A` found: invalid value, only `case:ignore` and `case:strict` are supported"
	var caseMissing struct {
		A int `json:",case"`
	}
	_, _ = json.Marshal(caseMissing) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `case` on field `A` found: invalid value, only `case:ignore` and `case:strict` are supported"
	var caseBoth struct {
		A int `json:",case:ignore,case:strict"`
	}
	_, _ = json.Marshal(caseBoth) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `case:strict` on field `A` found: duplicate option"
	var formatMissing struct {
		A int `json:",format"`
	}
	_, _ = json.Marshal(formatMissing) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe struct tag option `format` on field `A` found: `format` is not supported"
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
	_, _ = json.Marshal(embeddedOmitempty) // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `inner` found: option `omitempty` is not allowed, only `embed` is supported"
	var embeddedCase struct {
		inner `json:",case:ignore"`
	}
	_, _ = json.Marshal(embeddedCase)    // want "Error return value of `encoding/json/v2.Marshal` is not checked: unsafe embedded field `inner` found: option `case` is not allowed, only `embed` is supported"
	_, err = json.Marshal(trailingComma) // err is checked
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
	_, err = json.Marshal(embedStruct) // want "Error return value of `encoding/json/v2.Marshal` is checked but passed argument is safe"
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
	_, err = json.Marshal(mapPtrBool) // want "`encoding/json/v2.Marshal` for unsupported type `bool` as map key found"
	_ = err

	var mapPtrStruct map[*structKey]int
	_, err = json.Marshal(mapPtrStruct) // want "`encoding/json/v2.Marshal` for unsupported type `jsonv2.structKey` as map key found"
	_ = err

	_, err = stdjson.Marshal(mapPtrInt) // want "`encoding/json.Marshal` for unsupported type `\\*int` as map key found"
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

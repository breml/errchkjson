package example

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
	"unsafe"
)

type marshalText struct{}

func (_ marshalText) MarshalText() ([]byte, error) {
	return []byte(`mt`), nil
}

var _ encoding.TextMarshaler = marshalText(struct{}{})

type marshalJSON struct{}

func (_ marshalJSON) MarshalJSON() ([]byte, error) {
	return []byte(`mj`), nil
}

var _ json.Marshaler = marshalJSON(struct{}{})

// JSONMarshalSafeTypes contains a multitude of test cases to marshal different combinations of types to JSON,
// that are safe, that is, they will never return an error, if these types are marshaled to JSON.
func JSONMarshalSafeTypes() {
	var err error

	_, _ = json.Marshal(nil)   // nil is safe
	json.Marshal(nil)          // nil is safe
	_, err = json.Marshal(nil) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	fmt.Print(json.Marshal(nil)) // nil is safe, error is passed as argument

	_, _ = json.MarshalIndent(nil, "", " ")   // nil is safe
	json.MarshalIndent(nil, "", " ")          // nil is safe
	_, err = json.MarshalIndent(nil, "", " ") // want "Error return value of `encoding/json.MarshalIndent` is checked but passed argument is safe"
	_ = err

	enc := json.NewEncoder(ioutil.Discard)
	_ = enc.Encode(nil)   // want "Error return value of `\\([*]encoding/json.Encoder\\).Encode` is not checked"
	enc.Encode(nil)       // want "Error return value of `\\([*]encoding/json.Encoder\\).Encode` is not checked"
	err = enc.Encode(nil) // nil is safe, but encoding/json.Encoder may return non json related errors
	_ = err

	var b bool
	_, _ = json.Marshal(b)   // bool is safe
	_, err = json.Marshal(b) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var i int
	_, _ = json.Marshal(i)   // int is safe
	_, err = json.Marshal(i) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var i8 int8
	_, _ = json.Marshal(i8)   // int8 is safe
	_, err = json.Marshal(i8) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var i16 int16
	_, _ = json.Marshal(i16)   // int16 is safe
	_, err = json.Marshal(i16) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var i32 int32
	_, _ = json.Marshal(i32)   // int32 / rune is safe
	_, err = json.Marshal(i32) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var i64 int64
	_, _ = json.Marshal(i64)   // int64 is safe
	_, err = json.Marshal(i64) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var ui uint
	_, _ = json.Marshal(ui)   // uint is safe
	_, err = json.Marshal(ui) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var ui8 uint8
	_, _ = json.Marshal(ui8)   // uint8 / byte is safe
	_, err = json.Marshal(ui8) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var ui16 uint16
	_, _ = json.Marshal(ui16)   // uint16 is safe
	_, err = json.Marshal(ui16) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var ui32 uint32
	_, _ = json.Marshal(ui32)   // uint32 / rune is safe
	_, err = json.Marshal(ui32) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var ui64 uint64
	_, _ = json.Marshal(ui64)   // uint64 is safe
	_, err = json.Marshal(ui64) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var uiptr uintptr
	_, _ = json.Marshal(uiptr)   // uintptr is safe
	_, err = json.Marshal(uiptr) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var str string
	_, _ = json.Marshal(str)   // string is safe
	_, err = json.Marshal(str) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var strSlice []string
	_, _ = json.Marshal(strSlice)   // []string is safe
	_, err = json.Marshal(strSlice) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var intSlice []int
	_, _ = json.Marshal(intSlice)   // []int is safe
	_, err = json.Marshal(intSlice) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var boolSlice []bool
	_, _ = json.Marshal(boolSlice)   // []bool is safe
	_, err = json.Marshal(boolSlice) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var strArray [10]string
	_, _ = json.Marshal(strArray)   // [10]string is safe
	_, err = json.Marshal(strArray) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var intArray [10]int
	_, _ = json.Marshal(intArray)   // [10]int is safe
	_, err = json.Marshal(intArray) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var boolArray [10]bool
	_, _ = json.Marshal(boolArray)   // [10]bool is safe
	_, err = json.Marshal(boolArray) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var basicStruct struct {
		Bool    bool
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32 // also rune
		Int64   int64
		Uint    uint
		Uint8   uint8 // also byte
		Uint16  uint16
		Uint32  uint32
		Uint64  uint64
		Uintptr uintptr
		String  string
	}
	_, _ = json.Marshal(basicStruct)   // struct containing only safe basic types is safe
	_, err = json.Marshal(basicStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var ptrStruct struct {
		Bool    *bool
		Int     *int
		Int8    *int8
		Int16   *int16
		Int32   *int32
		Int64   *int64
		Uint    *uint
		Uint8   *uint8
		Uint16  *uint16
		Uint32  *uint32
		Uint64  *uint64
		Uintptr *uintptr
		String  *string
	}
	_, _ = json.Marshal(ptrStruct)   // struct containing pointer to only safe basic types is safe
	_, err = json.Marshal(ptrStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var mapStrStr map[string]string
	_, _ = json.Marshal(mapStrStr)   // map[string]string is safe
	_, err = json.Marshal(mapStrStr) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var mapStrInt map[string]int
	_, _ = json.Marshal(mapStrInt)   // map[string]int is safe
	_, err = json.Marshal(mapStrInt) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var mapStrBool map[string]bool
	_, _ = json.Marshal(mapStrBool)   // map[string]bool is safe
	_, err = json.Marshal(mapStrBool) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var mapIntStr map[int]string
	_, _ = json.Marshal(mapIntStr)   // map[int]string is safe
	_, err = json.Marshal(mapIntStr) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var mapIntInt map[int]int
	_, _ = json.Marshal(mapIntInt)   // map[int]int is safe
	_, err = json.Marshal(mapIntInt) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var mapIntBool map[int]bool
	_, _ = json.Marshal(mapIntBool)   // map[int]bool is safe
	_, err = json.Marshal(mapIntBool) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	type innerStruct struct {
		Bool   bool
		Int    int
		String string

		StrSlice  []string
		IntSlice  []int
		BoolSlice []bool

		StrArray  [10]string
		IntArray  [10]int
		BoolArray [10]bool

		MapStrStr  map[string]string
		MapStrInt  map[string]int
		MapStrBool map[string]bool

		MapIntStr  map[int]string
		MapIntInt  map[int]int
		MapIntBool map[int]bool
	}
	var outerStruct struct {
		Bool   bool
		Int    int
		String string

		StrSlice  []string
		IntSlice  []int
		BoolSlice []bool

		StrArray  [10]string
		IntArray  [10]int
		BoolArray [10]bool

		MapStrStr  map[string]string
		MapStrInt  map[string]int
		MapStrBool map[string]bool

		MapIntStr  map[int]string
		MapIntInt  map[int]int
		MapIntBool map[int]bool

		InnerStruct innerStruct
	}
	_, _ = json.Marshal(outerStruct)   // struct with only safe types
	_, err = json.Marshal(outerStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

type (
	structKey                      struct{ id int }
	ExportedUnsafeAndInvalidStruct struct { // unsafe unexported but omitted
		F64                  float64
		F64Ptr               *float64
		F64Slice             []float64
		F64Array             [10]float64
		MapStrF64            map[string]float64
		MapEIStr             map[interface{}]string
		Number               json.Number
		NumberPtr            *json.Number
		NumberSlice          []json.Number
		MapNumberStr         map[json.Number]string
		Ei                   interface{}
		Stringer             fmt.Stringer
		Mt                   marshalText
		MapMarshalTextString map[marshalText]string
		Mj                   marshalJSON
		MapMarshalJSONString map[marshalJSON]string

		C128         complex128
		C128Ptr      *complex128
		C128Slice    []complex128
		C128Array    [10]complex128
		MapBoolStr   map[bool]string
		MapF64Str    map[float64]string
		F            func()
		Ch           chan struct{}
		UnsafePtr    unsafe.Pointer
		MapStructStr map[structKey]string
	}
)

// JSONMarshalSafeStructWithUnexportedFields contains a struct with unexported, unsafe fields.
func JSONMarshalSafeStructWithUnexportedFields() {
	var err error

	var unexportedInStruct struct {
		Bool bool // safe exported

		f64                  float64                         // unsafe unexported
		f64Ptr               *float64                        // unsafe unexported
		f64Slice             []float64                       // unsafe unexported
		f64Array             [10]float64                     // unsafe unexported
		mapStrF64            map[string]float64              // unsafe unexported
		mapEIStr             map[interface{}]string          // unsafe unexported
		number               json.Number                     // unsafe unexported
		numberPtr            *json.Number                    // unsafe unexported
		numberSlice          []json.Number                   // unsafe unexported
		mapNumberStr         map[json.Number]string          // unsafe unexported
		ei                   interface{}                     // unsafe unexported
		stringer             fmt.Stringer                    // unsafe unexported
		mt                   marshalText                     // unsafe unexported
		mapMarshalTextString map[marshalText]string          // unsafe unexported
		mj                   marshalJSON                     // unsafe unexported
		mapMarshalJSONString map[marshalJSON]string          // unsafe unexported
		unexportedStruct     ExportedUnsafeAndInvalidStruct  // unsafe unexported
		unexportedStructPtr  *ExportedUnsafeAndInvalidStruct // unsafe unexported

		c128         complex128           // invalid unexported
		c128Slice    []complex128         // invalid unexported
		c128Array    [10]complex128       // invalid unexported
		mapBoolStr   map[bool]string      // invalid unexported
		mapF64Str    map[float64]string   // invalid unexported
		f            func()               // invalid unexported
		ch           chan struct{}        // invalid unexported
		unsafePtr    unsafe.Pointer       // invalid unexported
		mapStructStr map[structKey]string // invalid unexported
	}
	_ = unexportedInStruct.f64
	_ = unexportedInStruct.f64Ptr
	_ = unexportedInStruct.f64Slice
	_ = unexportedInStruct.f64Array
	_ = unexportedInStruct.mapStrF64
	_ = unexportedInStruct.mapEIStr
	_ = unexportedInStruct.number
	_ = unexportedInStruct.numberPtr
	_ = unexportedInStruct.numberSlice
	_ = unexportedInStruct.mapNumberStr
	_ = unexportedInStruct.ei
	_ = unexportedInStruct.stringer
	_ = unexportedInStruct.mt
	_ = unexportedInStruct.mapMarshalTextString
	_ = unexportedInStruct.unexportedStruct
	_ = unexportedInStruct.unexportedStructPtr

	_ = unexportedInStruct.c128
	_ = unexportedInStruct.c128Slice
	_ = unexportedInStruct.c128Array
	_ = unexportedInStruct.mapBoolStr
	_ = unexportedInStruct.mapF64Str
	_ = unexportedInStruct.f
	_ = unexportedInStruct.ch
	_ = unexportedInStruct.unsafePtr
	_ = unexportedInStruct.mapStructStr[structKey{1}]
	_, _ = json.Marshal(unexportedInStruct)   // struct containing unsafe but unexported fields is safe
	_, err = json.Marshal(unexportedInStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

// JSONMarshalSafeStructWithOmittedFields contains a struct with omitted, unsafe fields.
func JSONMarshalSafeStructWithOmittedFields() {
	var err error

	var omitInStruct struct {
		Bool bool // safe exported

		F64                  float64                         `json:"-"` // unsafe exported but omitted
		F64Ptr               *float64                        `json:"-"` // unsafe exported but omitted
		F64Slice             []float64                       `json:"-"` // unsafe exported but omitted
		F64Array             [10]float64                     `json:"-"` // unsafe exported but omitted
		MapStrF64            map[string]float64              `json:"-"` // unsafe exported but omitted
		MapEIStr             map[interface{}]string          `json:"-"` // unsafe exported but omitted
		Number               json.Number                     `json:"-"` // unsafe exported but omitted
		NumberPtr            *json.Number                    `json:"-"` // unsafe exported but omitted
		NumberSlice          []json.Number                   `json:"-"` // unsafe exported but omitted
		MapNumberStr         map[json.Number]string          `json:"-"` // unsafe exported but omitted
		Ei                   interface{}                     `json:"-"` // unsafe exported but omitted
		Stringer             fmt.Stringer                    `json:"-"` // unsafe exported but omitted
		Mt                   marshalText                     `json:"-"` // unsafe exported but omitted
		MapMarshalTextString map[marshalText]string          `json:"-"` // unsafe exported but omitted
		Mj                   marshalJSON                     `json:"-"` // unsafe exported but omitted
		MapMarshalJSONString map[marshalJSON]string          `json:"-"` // unsafe exported but omitted
		ExportedStruct       ExportedUnsafeAndInvalidStruct  `json:"-"` // unsafe exported but omitted
		ExportedStructPtr    *ExportedUnsafeAndInvalidStruct `json:"-"` // unsafe exported but omitted

		C128         complex128           `json:"-"` // invalid exported but omitted
		C128Slice    []complex128         `json:"-"` // invalid exported but omitted
		C128Array    [10]complex128       `json:"-"` // invalid exported but omitted
		MapBoolStr   map[bool]string      `json:"-"` // invalid exported but omitted
		MapF64Str    map[float64]string   `json:"-"` // invalid exported but omitted
		F            func()               `json:"-"` // invalid exported but omitted
		Ch           chan struct{}        `json:"-"` // invalid exported but omitted
		UnsafePtr    unsafe.Pointer       `json:"-"` // invalid exported but omitted
		MapStructStr map[structKey]string `json:"-"` // invalid exported but omitted
	}
	_ = omitInStruct.MapStructStr[structKey{1}]
	_, _ = json.Marshal(omitInStruct)   // struct containing unsafe but omitted, exported fields is safe
	_, err = json.Marshal(omitInStruct) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

// JSONMarshalUnsafeTypes contains a multitude of test cases to marshal different combinations of types to JSON,
// that can potentially lead to json.Marshal returning an error.
func JSONMarshalUnsafeTypes() {
	var err error

	var f32 float32
	json.Marshal(f32)          // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float32` found"
	_, _ = json.Marshal(f32)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float32` found"
	_, err = json.Marshal(f32) // err is checked
	_ = err
	fmt.Print(json.Marshal(f32)) // err is passed and therefore considered as checked

	var f64 float64
	_, _ = json.Marshal(f64)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, err = json.Marshal(f64) // err is checked
	_ = err

	var f32Slice []float32
	_, _ = json.Marshal(f32Slice)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float32` found"
	_, err = json.Marshal(f32Slice) // err is checked
	_ = err

	var f64Slice []float64
	_, _ = json.Marshal(f64Slice)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, err = json.Marshal(f64Slice) // err is checked
	_ = err

	var f32Array [10]float32
	_, _ = json.Marshal(f32Array)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float32` found"
	_, err = json.Marshal(f32Array) // err is checked
	_ = err

	var f64Array [10]float64
	_, _ = json.Marshal(f64Array)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, err = json.Marshal(f64Array) // err is checked
	_ = err

	var structPtrF32 struct {
		F32 *float32
	}
	_, _ = json.Marshal(structPtrF32)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float32` found"
	_, err = json.Marshal(structPtrF32) // err is checked
	_ = err

	var structPtrF64 struct {
		F64 *float64
	}
	_, _ = json.Marshal(structPtrF64)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, err = json.Marshal(structPtrF64) // err is checked
	_ = err

	var mapStrF32 map[string]float32
	_, _ = json.Marshal(mapStrF32)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float32` found"
	_, err = json.Marshal(mapStrF32) // err is checked
	_ = err

	var mapStrF64 map[string]float64
	_, _ = json.Marshal(mapStrF64)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, err = json.Marshal(mapStrF64) // err is checked
	_ = err

	var mapEIStr map[interface{}]string
	_, _ = json.Marshal(mapEIStr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `interface{}` as map key found"
	_, err = json.Marshal(mapEIStr) // err is checked
	_ = err

	var mapStringerStr map[fmt.Stringer]string
	_, _ = json.Marshal(mapStringerStr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `fmt.Stringer` as map key found"
	_, err = json.Marshal(mapStringerStr) // err is checked
	_ = err

	var number json.Number
	_, _ = json.Marshal(number)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `encoding/json.Number` found"
	_, err = json.Marshal(number) // err is checked
	_ = err

	var numberSlice []json.Number
	_, _ = json.Marshal(numberSlice)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `encoding/json.Number` found"
	_, err = json.Marshal(numberSlice) // err is checked
	_ = err

	var mapNumberStr map[json.Number]string
	_, _ = json.Marshal(mapNumberStr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `encoding/json.Number` as map key found"
	_, err = json.Marshal(mapNumberStr) // err is checked
	_ = err

	var mapStrNumber map[string]json.Number
	_, _ = json.Marshal(mapStrNumber)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `encoding/json.Number` found"
	_, err = json.Marshal(mapStrNumber) // err is checked
	_ = err

	var ei interface{}
	_, _ = json.Marshal(ei)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `interface{}` found"
	_, err = json.Marshal(ei) // err is checked
	_ = err

	var eiptr *interface{}
	_, _ = json.Marshal(eiptr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `*interface{}` found"
	_, err = json.Marshal(eiptr) // err is checked
	_ = err

	var stringer fmt.Stringer
	_, _ = json.Marshal(stringer)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `fmt.Stringer` found"
	_, err = json.Marshal(stringer) // err is checked
	_ = err

	var structWithEmptyInterface struct {
		EmptyInterface interface{}
	}
	_, _ = json.Marshal(structWithEmptyInterface)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `interface{}` found"
	_, err = json.Marshal(structWithEmptyInterface) // err is checked
	_ = err

	var mt marshalText
	_, _ = json.Marshal(mt)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.marshalText` found"
	_, err = json.Marshal(mt) // err is checked
	_ = err

	var mapMarshalTextString map[marshalText]string
	_, _ = json.Marshal(mapMarshalTextString)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.marshalText` as map key found"
	_, err = json.Marshal(mapMarshalTextString) // err is checked
	_ = err

	var mj marshalJSON
	_, _ = json.Marshal(mj)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.marshalJSON` found"
	_, err = json.Marshal(mj) // err is checked
	_ = err

	var mapMarshalJSONString map[marshalJSON]string
	_, _ = json.Marshal(mapMarshalJSONString)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.marshalJSON` as map key found"
	_, err = json.Marshal(mapMarshalJSONString) // err is checked
	_ = err
}

// JSONMarshalInvalidTypes contains a multitude of test cases to marshal different combinations of types to JSON,
// that are invalid and not supported by json.Marshal. Most of them will always return an error, if these types are
// used with json.Marshal, others only fail for some values (e.g. non-empty slices).
func JSONMarshalInvalidTypes() {
	var err error

	var c64 complex64
	json.Marshal(c64)          // want "`encoding/json.Marshal` for unsupported type `complex64` found"
	_, _ = json.Marshal(c64)   // want "`encoding/json.Marshal` for unsupported type `complex64` found"
	_, err = json.Marshal(c64) // want "`encoding/json.Marshal` for unsupported type `complex64` found"
	_ = err
	fmt.Print(json.Marshal(c64)) // want "`encoding/json.Marshal` for unsupported type `complex64` found"

	var c128 complex128
	_, _ = json.Marshal(c128)   // want "`encoding/json.Marshal` for unsupported type `complex128` found"
	_, err = json.Marshal(c128) // want "`encoding/json.Marshal` for unsupported type `complex128` found"
	_ = err

	// nil and empty slices (and nil pointers) of unsupported types are encoded
	// without error. The same applies to nil and empty maps with keys of
	// unsupported types as of Go 1.27. These types are therefore unsafe.
	var sliceC64 []complex64
	_, _ = json.Marshal(sliceC64)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex64` found"
	_, err = json.Marshal(sliceC64) // err is checked
	_ = err

	var sliceC128 []complex128
	_, _ = json.Marshal(sliceC128)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex128` found"
	_, err = json.Marshal(sliceC128) // err is checked
	_ = err

	var arrayC64 []complex64
	_, _ = json.Marshal(arrayC64)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex64` found"
	_, err = json.Marshal(arrayC64) // err is checked
	_ = err

	var arrayC128 []complex128
	_, _ = json.Marshal(arrayC128)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex128` found"
	_, err = json.Marshal(arrayC128) // err is checked
	_ = err

	var structPtrC64 struct {
		C64 *complex64
	}
	_, _ = json.Marshal(structPtrC64)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex64` found"
	_, err = json.Marshal(structPtrC64) // err is checked
	_ = err

	var structPtrC128 struct {
		C128 *complex128
	}
	_, _ = json.Marshal(structPtrC128)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex128` found"
	_, err = json.Marshal(structPtrC128) // err is checked
	_ = err

	var mapBoolStr map[bool]string
	_, _ = json.Marshal(mapBoolStr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `bool` as map key found"
	_, err = json.Marshal(mapBoolStr) // err is checked
	_ = err

	var mapF32Str map[float32]string
	_, _ = json.Marshal(mapF32Str)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `float32` as map key found"
	_, err = json.Marshal(mapF32Str) // err is checked
	_ = err

	var mapF64Str map[float64]string
	_, _ = json.Marshal(mapF64Str)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `float64` as map key found"
	_, err = json.Marshal(mapF64Str) // err is checked
	_ = err

	var mapC64Str map[complex64]string
	_, _ = json.Marshal(mapC64Str)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex64` as map key found"
	_, err = json.Marshal(mapC64Str) // err is checked
	_ = err

	var mapC128Str map[complex128]string
	_, _ = json.Marshal(mapC128Str)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `complex128` as map key found"
	_, err = json.Marshal(mapC128Str) // err is checked
	_ = err

	mapStructStr := map[structKey]string{{1}: "str"}
	_, _ = json.Marshal(mapStructStr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `standard.structKey` as map key found"
	_, err = json.Marshal(mapStructStr) // err is checked
	_ = err

	f := func() {}
	_, _ = json.Marshal(f)   // want "`encoding/json.Marshal` for unsupported type `func\\(\\)` found"
	_, err = json.Marshal(f) // want "`encoding/json.Marshal` for unsupported type `func\\(\\)` found"
	_ = err

	var ch chan struct{} = make(chan struct{})
	_, _ = json.Marshal(ch)   // want "`encoding/json.Marshal` for unsupported type `chan struct{}` found"
	_, err = json.Marshal(ch) // want "`encoding/json.Marshal` for unsupported type `chan struct{}` found"
	_ = err

	var unsafePtr unsafe.Pointer
	_, _ = json.Marshal(unsafePtr)   // want "`encoding/json.Marshal` for unsupported type `unsafe.Pointer` found"
	_, err = json.Marshal(unsafePtr) // want "`encoding/json.Marshal` for unsupported type `unsafe.Pointer` found"
	_ = err
}

// NotJSONMarshal contains other go ast node types, that are not considered by errchkjson
func NotJSONMarshal() {
	s := fmt.Sprintln("I am not considered by errchkjson")
	_ = s
	f := func() bool { return false }
	_ = f()
}

// Issue 5
type T struct {
	f64 float64
}

func (t T) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.f64) // not an error because it is the caller's responsibility to check the error
}

type appendText struct{}

func (_ appendText) AppendText(b []byte) ([]byte, error) {
	return append(b, `at`...), nil
}

// JSONMarshalGo127 contains test cases for types, which can lead to an error
// when encoded with encoding/json as of Go 1.27, where encoding/json is
// implemented on top of encoding/json/v2.
func JSONMarshalGo127() {
	var err error

	var at appendText
	_, _ = json.Marshal(at)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.appendText` found"
	_, err = json.Marshal(at) // err is checked
	_ = err

	var mapAppendTextString map[appendText]string
	_, _ = json.Marshal(mapAppendTextString)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.appendText` as map key found"
	_, err = json.Marshal(mapAppendTextString) // err is checked
	_ = err

	var formatTag struct {
		Int int `json:"int,format:foo"`
	}
	_, _ = json.Marshal(formatTag)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:foo` on field `Int` found"
	_, err = json.Marshal(formatTag) // err is checked
	_ = err

	var formatTagOmitted struct {
		Int int `json:"-"`
	}
	_, err = json.Marshal(formatTagOmitted) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// encoding/json does not apply the encoding/json/v2 rules.
	var v2Only struct {
		Bool   bool   `json:"bool,string"`
		String string `json:"string,string"`
		A      int    `json:"x"`
		B      int    `json:"x"`
	}
	_, err = json.Marshal(v2Only) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var v2OnlyTags struct {
		Quoted    int `json:"'a,b'"`
		Backslash int `json:"a\\b"`
		tagged    int `json:"tagged"`
	}
	_ = v2OnlyTags.tagged
	_, err = json.Marshal(v2OnlyTags) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// The exported fields promoted from an embedded struct of an unexported
	// type are encoded by encoding/json (see
	// https://github.com/breml/errchkjson/issues/93).
	var v2OnlyEmbedded struct {
		embeddedUnexportedFloat
		B int
	}
	_, _ = json.Marshal(v2OnlyEmbedded)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, err = json.Marshal(v2OnlyEmbedded) // err is checked
	_ = err

	// Multi-value arguments are not expanded for encoding/json.
	_, _ = json.MarshalIndent(intAndIndent())   // want "Error return value of `encoding/json.MarshalIndent` is not checked: unsafe type `\\(int, string, string\\)` found"
	_, err = json.MarshalIndent(intAndIndent()) // err is checked
	_ = err

	// time.Duration is encoded as integer by encoding/json.
	var v2OnlyDuration struct {
		D   time.Duration         `json:"d"`
		Ptr *time.Duration        `json:"ptr"`
		Map map[time.Duration]int `json:"map"`
	}
	_, err = json.Marshal(v2OnlyDuration) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// encoding/json ignores malformed and unknown json struct tag options.
	var v2OnlyTagOptions struct {
		A int `json:"a,"`
		B int `json:",omitEmpty"`
		C int `json:",omitempty,omitempty"`
		D int `json:",case:foo"`
		E int `json:",format"`
		F int `json:",embed"`
	}
	_, err = json.Marshal(v2OnlyTagOptions) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// As of Go 1.27, nil and empty maps with pointer keys are encoded without
	// error.
	var v2OnlyPtrMapKey map[*int]int
	_, err = json.Marshal(v2OnlyPtrMapKey) // err is checked
	_ = err

	var v2OnlyNoExported struct {
		a int
	}
	_ = v2OnlyNoExported.a
	_, err = json.Marshal(v2OnlyNoExported) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

type embeddedUnexportedFloat struct {
	F float64
}

func intAndIndent() (int, string, string) { return 0, "", "  " }

type EmbeddedFormat struct {
	I int
}

// JSONMarshalFormatOption contains test cases for the `format` option, which
// is rejected by encoding/json as of Go 1.27.
func JSONMarshalFormatOption() {
	var err error

	// encoding/json ignores `format` options without a valid value and on
	// embedded structs without JSON name.
	var formatTagIgnored struct {
		A              int `json:"a,format:"`
		B              int `json:"b,format:123"`
		C              int `json:"c,format:''"`
		EmbeddedFormat `json:",format:x"`
	}
	_, err = json.Marshal(formatTagIgnored) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var formatTagQuoted struct {
		A int `json:"a,format:'2006-01-02'"`
	}
	_, _ = json.Marshal(formatTagQuoted) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:'2006-01-02'` on field `A` found"

	var formatTagEmbeddedNamed struct {
		EmbeddedFormat `json:"e,format:x"`
	}
	_, _ = json.Marshal(formatTagEmbeddedNamed) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:x` on field `EmbeddedFormat` found"

	// The `embed` option is not known by encoding/json of Go 1.26 with
	// GOEXPERIMENT=jsonv2, the field is therefore a regular field.
	var formatTagEmbedOption struct {
		X EmbeddedFormat `json:",embed,format:x"`
	}
	_, _ = json.Marshal(formatTagEmbedOption) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:x` on field `X` found"
}

type recursiveMap map[int]recursiveMap

type recursiveSlice []recursiveSlice

type recursivePtr *recursivePtr

// JSONMarshalRecursiveTypes contains test cases for recursive types, which
// are not structs.
//
// Recursive types are considered unsafe for encoding/json, because encoding
// fails for cyclic values and, as of Go 1.27, for values with a nesting depth
// larger than 10000 (see https://github.com/breml/errchkjson/issues/94). For
// recursivePtr, a cyclic value leads to a stack overflow as of Go 1.27 (see
// https://go.dev/issue/82947).
func JSONMarshalRecursiveTypes() {
	var err error

	var m recursiveMap
	_, _ = json.Marshal(m)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe recursive type `standard.recursiveMap` found"
	_, err = json.Marshal(m) // err is checked
	_ = err

	var s recursiveSlice
	_, _ = json.Marshal(s)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe recursive type `standard.recursiveSlice` found"
	_, err = json.Marshal(s) // err is checked
	_ = err

	var p recursivePtr
	_, _ = json.Marshal(p)   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe recursive type `standard.recursivePtr` found"
	_, err = json.Marshal(p) // err is checked
	_ = err
}

// JSONMarshalAssignments contains test cases for multi-value assignments and
// var declarations.
func JSONMarshalAssignments() {
	var f float64
	enc := json.NewEncoder(ioutil.Discard)

	n, err := 0, enc.Encode(f) // err is checked
	_, _ = n, err

	_, _ = 0, enc.Encode(f) // want "Error return value of `\\(\\*encoding/json.Encoder\\).Encode` is not checked: unsafe type `float64` found"

	var errVar = enc.Encode(f) // err is checked
	_ = errVar

	// For backwards compatibility, var declarations are not handled for
	// encoding/json.Marshal. The error is assigned to a variable, therefore
	// the call is not reported as not checked (see
	// https://github.com/breml/errchkjson/issues/96).
	var _, errMarshal = json.Marshal(f)
	_ = errMarshal
}

type embeddedUnexportedFormat struct {
	I int
}

type embeddedUnexportedFormatMethod struct {
	I int
}

func (_ *embeddedUnexportedFormatMethod) UnmarshalJSON(_ []byte) error {
	return nil
}

// JSONMarshalFormatOptionParsing contains test cases for the parsing of the
// value of the `format` option and the `format` option on embedded structs of
// unexported types.
func JSONMarshalFormatOptionParsing() {
	var err error

	// A single-quoted value, which is not terminated, is ignored.
	var formatTagNotTerminated struct {
		A int `json:"a,format:'abc"`
	}
	_, err = json.Marshal(formatTagNotTerminated) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// A single-quoted value may contain commas.
	var formatTagQuotedComma struct {
		A int `json:"a,format:'a,b'"`
	}
	_, _ = json.Marshal(formatTagQuotedComma) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:'a,b'` on field `A` found"

	// A malformed value is ignored, but a later valid one is not.
	var formatTagSecond struct {
		A int `json:"a,format:'abc,format:x"`
	}
	_, _ = json.Marshal(formatTagSecond) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:x` on field `A` found"

	// Embedded structs of unexported types with a JSON name.
	var formatTagEmbeddedUnexported struct {
		embeddedUnexportedFormat `json:"e,format:x"`
	}
	_, _ = json.Marshal(formatTagEmbeddedUnexported) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:x` on field `embeddedUnexportedFormat` found"

	var formatTagEmbeddedUnexportedPtr struct {
		*embeddedUnexportedFormat `json:"e,format:x"`
	}
	_, _ = json.Marshal(formatTagEmbeddedUnexportedPtr) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe struct tag option `format:x` on field `embeddedUnexportedFormat` found"

	// Embedded structs of unexported types without JSON name or with methods
	// are not affected.
	var formatTagEmbeddedUnexportedIgnored struct {
		embeddedUnexportedFormat `json:",format:x"`
	}
	_, err = json.Marshal(formatTagEmbeddedUnexportedIgnored) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var formatTagEmbeddedUnexportedMethod struct {
		embeddedUnexportedFormatMethod `json:"e,format:x"`
	}
	_, err = json.Marshal(formatTagEmbeddedUnexportedMethod) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

// JSONMarshalNilValues contains test cases for unsupported types, which are
// encoded without error for nil and empty values (e.g. nil slices, nil
// pointers, zero values of fields with the `omitzero` option and, as of
// Go 1.27, nil and empty maps with keys of unsupported types).
func JSONMarshalNilValues() {
	var err error

	var arrayChan0 [0]chan int
	_, err = json.Marshal(arrayChan0) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	var omitzeroChan struct {
		C chan int `json:",omitzero"`
	}
	_, _ = json.Marshal(omitzeroChan)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(omitzeroChan) // err is checked
	_ = err

	var mapBoolField struct {
		M map[bool]int
	}
	_, _ = json.Marshal(mapBoolField)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `bool` as map key found"
	_, err = json.Marshal(mapBoolField) // err is checked
	_ = err

	// A nil pointer is encoded as null, unless the pointer is known to be
	// non-nil.
	var chanPtr *chan int
	_, _ = json.Marshal(chanPtr)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(chanPtr) // err is checked
	_ = err

	ch := make(chan int)
	_, err = json.Marshal(&ch) // want "`encoding/json.Marshal` for unsupported type `chan int` found"
	_ = err

	// The remaining fields are checked for unsupported types, if a field is
	// only unsafe, because nil and empty values are encoded without error.
	var sliceBeforeChan struct {
		S []chan int
		C chan int
	}
	_, err = json.Marshal(sliceBeforeChan) // want "`encoding/json.Marshal` for unsupported type `chan int` found"
	_ = err

	var sliceBeforeFloatAndChan struct {
		S []chan int
		F float64
		C chan int
	}
	_, err = json.Marshal(sliceBeforeFloatAndChan) // want "`encoding/json.Marshal` for unsupported type `chan int` found"
	_ = err

	var sliceBeforeFloat struct {
		S []chan int
		F float64
	}
	_, _ = json.Marshal(sliceBeforeFloat)   // want "Error return value of `encoding/json.Marshal` is not checked: unsupported type `chan int` found"
	_, err = json.Marshal(sliceBeforeFloat) // err is checked
	_ = err

	// An unsupported type is reported, even if an earlier field is only
	// unsafe for some values.
	var floatBeforeSliceAndChan struct {
		F float64
		S []chan int
		C chan int
	}
	_, err = json.Marshal(floatBeforeSliceAndChan) // want "`encoding/json.Marshal` for unsupported type `chan int` found"
	_ = err

	// An unsupported type takes precedence over an unsafe struct tag.
	var formatChan struct {
		C chan int `json:"c,format:x"`
	}
	_, err = json.Marshal(formatChan) // want "`encoding/json.Marshal` for unsupported type `chan int` found"
	_ = err
}

// pointerReceiverJSON implements encoding/json.Marshaler with a pointer
// receiver. encoding/json only calls such methods for addressable values (see
// https://github.com/breml/errchkjson/issues/92).
type pointerReceiverJSON struct {
	A int
}

func (*pointerReceiverJSON) MarshalJSON() ([]byte, error) { return []byte(`"prj"`), nil }

// pointerReceiverText implements encoding.TextMarshaler with a pointer
// receiver.
type pointerReceiverText struct {
	A int
}

func (*pointerReceiverText) MarshalText() ([]byte, error) { return []byte(`prt`), nil }

// JSONMarshalAddressable contains test cases for marshaler methods declared on
// a pointer receiver, which encoding/json only calls for addressable values.
func JSONMarshalAddressable() {
	var err error

	var v pointerReceiverJSON
	// A value, which is not addressable, is encoded without calling the
	// method of the pointer receiver.
	_, err = json.Marshal(v) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	_, _ = json.Marshal(&v) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.pointerReceiverJSON` found"

	// The elements of a slice are always addressable.
	_, _ = json.Marshal([]pointerReceiverJSON{{}}) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.pointerReceiverJSON` found"

	// The elements of an array are addressable if the array can be addressed.
	_, err = json.Marshal([1]pointerReceiverJSON{}) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	_, _ = json.Marshal(&[1]pointerReceiverJSON{}) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.pointerReceiverJSON` found"

	// Map values can not be addressed.
	_, err = json.Marshal(map[int]pointerReceiverJSON{0: {}}) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// Struct fields are addressable if the struct can be addressed.
	var s struct{ F pointerReceiverJSON }
	_, err = json.Marshal(s) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	_, _ = json.Marshal(&s) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.pointerReceiverJSON` found"

	// A pointer to a value is addressable, regardless of the enclosing type.
	var p struct{ F *pointerReceiverJSON }
	_, _ = json.Marshal(p) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `\\*standard.pointerReceiverJSON` found"

	// Other marshaler interfaces declared on a pointer receiver behave the
	// same way.
	var t pointerReceiverText
	_, err = json.Marshal(t) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
	_, _ = json.Marshal(&t) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `standard.pointerReceiverText` found"
}

// runFunc calls the passed function.
func runFunc(f func()) { f() }

// returnedFuncLit returns a function literal.
func returnedFuncLit() func() {
	return func() {
		_, _ = json.Marshal(1.5) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	}
}

// JSONMarshalFuncLit contains test cases for JSON calls inside function
// literals, which are passed as arguments, returned or deferred (see
// https://github.com/breml/errchkjson/issues/95).
func JSONMarshalFuncLit() {
	runFunc(func() {
		_, _ = json.Marshal(1.5) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	})

	go func() {
		_, _ = json.Marshal(1.5) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	}()

	defer func() {
		_, _ = json.Marshal(1.5) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	}()

	_ = returnedFuncLit()

	_ = func() {
		_, _ = json.Marshal(1.5) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	}
}

// JSONMarshalErrorUsed contains test cases, where the error of a JSON call is
// used, but not assigned to a variable or passed as argument (see
// https://github.com/breml/errchkjson/issues/96).
func JSONMarshalErrorUsed() {
	enc := json.NewEncoder(ioutil.Discard)
	var errs []error
	ch := make(chan error, 1)

	if enc.Encode(1) != nil { // the error is compared
		return
	}
	switch enc.Encode(1) { // the error is used as switch tag
	case nil:
	}
	errs = append(errs, enc.Encode(1))       // the error is stored
	ch <- enc.Encode(1)                      // the error is sent
	_ = map[string]error{"a": enc.Encode(1)} // the error is stored in a map
	_ = errs

	// The error is discarded.
	enc.Encode(1)       // want "Error return value of `\\(\\*encoding/json.Encoder\\).Encode` is not checked"
	defer enc.Encode(1) // want "Error return value of `\\(\\*encoding/json.Encoder\\).Encode` is not checked"
	go enc.Encode(1)    // want "Error return value of `\\(\\*encoding/json.Encoder\\).Encode` is not checked"
}

type embeddedSafe struct {
	I int
}

type embeddedUnexportedAny struct {
	V any
}

// JSONMarshalEmbeddedUnexported contains test cases for the exported fields
// promoted from embedded structs of unexported types (see
// https://github.com/breml/errchkjson/issues/93).
func JSONMarshalEmbeddedUnexported() {
	var err error

	_, _ = json.Marshal(struct{ embeddedUnexportedFloat }{})  // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, _ = json.Marshal(struct{ *embeddedUnexportedFloat }{}) // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, _ = json.Marshal(embeddedWithName{})                   // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `float64` found"
	_, _ = json.Marshal(struct{ embeddedUnexportedAny }{})    // want "Error return value of `encoding/json.Marshal` is not checked: unsafe type `any` found"

	// An embedded struct with safe fields only is safe.
	_, err = json.Marshal(struct{ embeddedSafe }{}) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err

	// An embedded struct, which is omitted, is not checked.
	_, err = json.Marshal(embeddedOmitted{}) // want "Error return value of `encoding/json.Marshal` is checked but passed argument is safe"
	_ = err
}

type embeddedWithName struct {
	embeddedUnexportedFloat `json:"in"`
}

type embeddedOmitted struct {
	embeddedUnexportedFloat `json:"-"`
}

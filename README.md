# errchkjson

[![Test Status](https://github.com/breml/errchkjson/actions/workflows/ci.yml/badge.svg)](https://github.com/breml/errchkjson/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/breml/errchkjson)](https://goreportcard.com/report/github.com/breml/errchkjson) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Checks types passed to the json encoding functions. Reports unsupported types and reports occurrences where the check for the returned error can be omitted.

Consider this [http.Handler](https://pkg.go.dev/net/http#Handler):

```Go
func JSONHelloWorld(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Message string
		Code    int
	}{
		Message: "Hello World",
		Code:    200,
	}

	body, err := json.Marshal(response)
	if err != nil {
		panic(err) // unreachable, because json encoding of a struct with just a string and an int will never return an error.
	}

	w.Write(body)
}
```

Because the `panic` is not possible to happen, one might refactor the code like this:

```Go
func JSONHelloWorld(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Message string
		Code    int
	}{
		Message: "Hello World",
		Code:    200,
	}

	body, _ := json.Marshal(response)

	w.Write(body)
}
```

This is ok, as long as the struct is not altered in such a way, that could potentially lead
to `json.Marshal` returning an error.

`errchkjson` allows you to lint your code such that the above error returned from `json.Marshal`
can be omitted while still staying safe, because as soon as an unsafe type is added to the
response type, the linter will warn you.

## Installation

Download `errchkjson` from the [releases](https://github.com/breml/errchkjson/releases) or get the latest version from source with:

```shell
go install github.com/breml/errchkjson/cmd/errchkjson@latest
```

## Usage

### Shell

Check everything:

```shell
errchkjson ./...
```

`errchkjson` also recognizes the following command-line options:

The `-omit-safe` flag disables checking for safe returns of errors from json.Marshal

## Types

### Safe

The following types are safe to use with [json encoding functions](https://pkg.go.dev/encoding/json), that is, the encoding to JSON can not fail:

Safe basic types:

* `bool`
* `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `uintptr`
* `string`
* Pointer type of the above listed basic types

Composed types (struct, map, slice, array) are safe, if the type of the value is
safe. For structs, only exported fields are relevant. For maps, the key needs to be either an integer type or a string.

### Unsafe

The following types are unsafe to use with [json encoding functions](https://pkg.go.dev/encoding/json), that is, the encoding to JSON can fail (return an error):

Unsafe basic types:

* `float32`, `float64`
* `interface{}`
* Pointer type of the above listed basic types

Any composed types (struct, map, slice, array) containing an unsafe basic type.

If a type implements the `json.Marshaler` or `encoding.TextMarshaler` interface (e.g. `json.Number`).

### Forbidden

Forbidden basic types:

* `complex64`, `complex128`
* `chan`
* `func`
* `unsafe.Pointer`

Any composed types (struct, map, slice, array) containing a forbidden basic type. Any map
using a key with a forbidden type (`bool`, `float32`, `float64`, `struct`).

### Go 1.27 and later

As of Go 1.27, [`encoding/json`](https://pkg.go.dev/encoding/json) is implemented on top of
[`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2), because the experiment `GOEXPERIMENT=jsonv2`
is enabled by default (it can still be disabled with `GOEXPERIMENT=nojsonv2`). The behavior of
`encoding/json` therefore depends on the Go version (and `GOEXPERIMENT`) used to build a program. `errchkjson` can not know this
and therefore considers a type unsafe, if encoding it can fail with any supported Go version.
In addition to the above rules, the following is considered unsafe:

* Types implementing the `MarshalJSONTo(*jsontext.Encoder) error` method (`json.MarshalerTo` of `encoding/json/v2`)
  or the `encoding.TextAppender` interface.
* Struct fields with the `format` option in the `json` struct tag (e.g. `json:"when,format:RFC3339"`).
  A `format` option without a valid value (e.g. `format:` or `format:123`) and a `format` option on an
  embedded struct without JSON name are ignored by `encoding/json` and are therefore not reported.

## encoding/json/v2

`errchkjson` also checks the functions `Marshal`, `MarshalWrite` and `MarshalEncode` of
[`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2) (available as of Go 1.27 or with
`GOEXPERIMENT=jsonv2` for Go 1.26). The rules for safe, unsafe and forbidden
types are the same as above, with the following differences:

* `string` (including named string types and string map keys) is unsafe, because `encoding/json/v2`
  returns an error for strings containing invalid UTF-8.
* `time.Duration` is unsafe, because it has no default representation in `encoding/json/v2`.
* Types, which implement one of the marshaler interfaces with a pointer receiver (e.g. `func (*T) MarshalJSON() ([]byte, error)`),
  are unsafe, even if they are passed by value, because `encoding/json/v2` always calls these methods.
* Float map keys (e.g. `map[float64]string`) and pointer map keys (e.g. `map[*int]string`) are unsafe
  instead of forbidden.
* Map keys of byte array types (e.g. `map[[16]byte]string`) are safe instead of forbidden, because they
  are encoded as base64 encoded strings.
* Struct fields with the `string` option in the `json` struct tag are unsafe, if the type of the field is
  not numeric.
* Structs with two or more direct fields using the same JSON object name are unsafe. Conflicts with
  fields promoted from embedded structs are not an error, because `encoding/json/v2` resolves them
  using the Go embedding rules (the less nested field wins, ambiguous fields are ignored).
* Structs with fields, but without any exported field and without any `json` struct tag, are unsafe.
* Struct fields with a JSON object name containing a backslash or quote characters (e.g. `json:"'name'"`)
  or invalid UTF-8 are unsafe.
* Unexported struct fields with a `json` struct tag other than `json:"-"` are unsafe.
* Embedded fields of non-struct types (e.g. `type MyInt int` or an interface) are unsafe, if they are
  not explicitly given a JSON name (e.g. `json:"name"`) or if their type is unexported.
* Struct fields with malformed or misspelled options in the `json` struct tag are unsafe, e.g. a trailing
  comma (`json:"name,"`), `omitEmpty`, duplicate options, `case` with a value other than `ignore` or `strict`
  or `format`. Unknown options are ignored by `encoding/json/v2` and are therefore not reported, unless
  they appear more than once.
* Embedded fields (embedded structs without a JSON name or fields with the `embed` option) are unsafe, if
  they have a JSON name or options other than `embed`, if their type is not a struct, a map with a
  string key or `jsontext.Value`, or if their type (or the map key type) implements one of the marshaler
  or unmarshaler interfaces (e.g. `func (*T) UnmarshalJSON([]byte) error`).
* Fields with the options `inline` or `unknown` (the predecessors of `embed` in Go 1.26) are checked
  according to the same rules, because they are embedded by `encoding/json/v2` of Go 1.26.
* Embedded structs of unexported types with a JSON name (e.g. ``inner `json:"in"` ``) are unsafe, if their
  type implements one of the marshaler or unmarshaler interfaces, or the `IsZero() bool` method in
  combination with the `omitzero` option, because these methods can not be called.
* If options are passed to the function (e.g. `json.WithMarshalers`), the encoding of any type can
  be changed and the call is therefore always considered unsafe.

Because `MarshalWrite` and `MarshalEncode` may return errors unrelated to the encoded type (e.g. from the
`io.Writer`), their returned error is never reported as safe to omit, similar to `(*encoding/json.Encoder).Encode`.

## Accepted edge case

For `encoding/json.MarshalIndent`, there is a (pathological) edge case, where this
function could [return an error](https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/encoding/json/scanner.go;drc=refs%2Ftags%2Fgo1.18;l=181) for an otherwise safe argument, if the argument has
a nesting depth larger than [`10000`](https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/encoding/json/scanner.go;drc=refs%2Ftags%2Fgo1.18;l=144) (as of Go 1.18).

## Bugs found during development

During the development of `errcheckjson`, the following issues in package `encoding/json` of the Go standard library have been found and PR have been merged:

* [Issue #34154: encoding/json: string option (struct tag) on string field with SetEscapeHTML(false) escapes anyway](https://github.com/golang/go/issues/34154)
* [PR #34127: encoding/json: fix and optimize marshal for quoted string](https://github.com/golang/go/pull/34127)
* [Issue #34268: encoding/json: wrong encoding for json.Number field with string option (struct tag)](https://github.com/golang/go/issues/34268)
* [PR #34269: encoding/json: make Number with the ,string option marshal with quotes](https://github.com/golang/go/pull/34269)
* [PR #34272: encoding/json: validate strings when decoding into Number](https://github.com/golang/go/pull/34272)

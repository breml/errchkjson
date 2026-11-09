// Package errchkjson defines an Analyzer that finds places, where it is
// safe to omit checking the error returned from json.Marshal.
package errchkjson

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

type errchkjson struct {
	omitSafe         bool // -omit-safe flag
	reportNoExported bool // -report-no-exported flag
}

// NewAnalyzer returns a new errchkjson analyzer.
func NewAnalyzer() *analysis.Analyzer {
	errchkjson := &errchkjson{}

	a := &analysis.Analyzer{
		Name: "errchkjson",
		Doc:  "Checks types passed to the json encoding functions. Reports unsupported types and reports occurrences where the check for the returned error can be omitted.",
		Run:  errchkjson.run,
	}

	a.Flags.Init("errchkjson", flag.ExitOnError)
	a.Flags.BoolVar(&errchkjson.omitSafe, "omit-safe", false, "if omit-safe is true, checking of safe returns is omitted")
	a.Flags.BoolVar(&errchkjson.reportNoExported, "report-no-exported", false, "if report-no-exported is true, encoding a struct without exported fields is reported as issue")
	a.Flags.Var(versionFlag{}, "V", "print version and exit")

	return a
}

func (e *errchkjson) run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			return e.inspect(pass, n)
		})
	}

	return nil, nil
}

func (e *errchkjson) inspect(pass *analysis.Pass, n ast.Node) bool {
	switch n := n.(type) {
	case *ast.ReturnStmt:
		// if the error is returned, it is the caller's responsibility to check
		// the return value.
		return false

	case *ast.CallExpr:
		fn, _ := typeutil.Callee(pass.TypesInfo, n).(*types.Func)
		if fn == nil {
			return true
		}

		if jf, ok := jsonFuncs[fn.FullName()]; ok {
			e.handleJSONMarshal(pass, n, fn.FullName(), jf, blankIdentifier)
		} else {
			e.inspectArgs(pass, n.Args)
		}
		return false

	case *ast.AssignStmt:
		return e.inspectAssignment(pass, n.Lhs, n.Rhs, false)

	case *ast.ValueSpec:
		lhs := make([]ast.Expr, 0, len(n.Names))
		for _, name := range n.Names {
			lhs = append(lhs, name)
		}
		return e.inspectAssignment(pass, lhs, n.Values, true)
	}

	return true
}

// inspectAssignment handles the assignment of the values rhs to lhs, either
// by an assignment statement or by a var declaration (valueSpec is true). It
// returns false, if the children of the assignment must not be inspected.
func (e *errchkjson) inspectAssignment(pass *analysis.Pass, lhs []ast.Expr, rhs []ast.Expr, valueSpec bool) bool {
	if len(rhs) == 1 {
		ce, fnName, jf, ok := jsonFuncCall(pass, rhs[0])
		if !ok || len(lhs) <= jf.errIdx {
			return true
		}
		// For backwards compatibility, var declarations are not handled for
		// encoding/json.Marshal and encoding/json.MarshalIndent, because this
		// would add new findings (error checked but argument is safe).
		if valueSpec && !jf.v2 && !jf.alwaysOmitSafe {
			return true
		}
		e.handleJSONMarshal(pass, ce, fnName, jf, evaluateMarshalErrorTarget(lhs[jf.errIdx]))
		return false
	}

	if len(lhs) != len(rhs) {
		return true
	}

	// Multi-value assignment (e.g. `n, err := 0, json.MarshalWrite(w, v)`),
	// only functions returning a single value (the error) are possible.
	for i, expr := range rhs {
		ce, fnName, jf, ok := jsonFuncCall(pass, expr)
		if ok && jf.errIdx == 0 {
			e.handleJSONMarshal(pass, ce, fnName, jf, evaluateMarshalErrorTarget(lhs[i]))
			continue
		}
		ast.Inspect(expr, func(n ast.Node) bool {
			return e.inspect(pass, n)
		})
	}
	return false
}

// jsonFuncCall reports, if expr is a call of one of the JSON encoding
// functions checked by errchkjson.
func jsonFuncCall(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, string, jsonFunc, bool) {
	ce, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, "", jsonFunc{}, false
	}
	fn, _ := typeutil.Callee(pass.TypesInfo, ce).(*types.Func)
	if fn == nil {
		return nil, "", jsonFunc{}, false
	}
	jf, ok := jsonFuncs[fn.FullName()]
	return ce, fn.FullName(), jf, ok
}

func evaluateMarshalErrorTarget(n ast.Expr) marshalErrorTarget {
	if errIdent, ok := n.(*ast.Ident); ok {
		if errIdent.Name == "_" {
			return blankIdentifier
		}
	}
	return variableAssignment
}

type marshalErrorTarget int

const (
	blankIdentifier    = iota // the returned error from the JSON marshal function is assigned to the blank identifier "_".
	variableAssignment        // the returned error from the JSON marshal function is assigned to a variable.
	functionArgument          // the returned error from the JSON marshal function is passed to an other function as argument.
)

// jsonFunc describes a JSON encoding function, which is checked by errchkjson.
type jsonFunc struct {
	argIdx         int  // index of the argument, which is encoded to JSON.
	errIdx         int  // index of the error in the returned values.
	alwaysOmitSafe bool // the function may return errors not related to the encoded type (e.g. from an io.Writer).
	v2             bool // the function is part of encoding/json/v2.
	encoderOpts    bool // the encoding is affected by the options of the passed jsontext.Encoder.
}

var jsonFuncs = map[string]jsonFunc{
	"encoding/json.Marshal":           {argIdx: 0, errIdx: 1},
	"encoding/json.MarshalIndent":     {argIdx: 0, errIdx: 1},
	"(*encoding/json.Encoder).Encode": {argIdx: 0, errIdx: 0, alwaysOmitSafe: true},
	"encoding/json/v2.Marshal":        {argIdx: 0, errIdx: 1, v2: true},
	"encoding/json/v2.MarshalWrite":   {argIdx: 1, errIdx: 0, alwaysOmitSafe: true, v2: true},
	"encoding/json/v2.MarshalEncode":  {argIdx: 1, errIdx: 0, alwaysOmitSafe: true, v2: true, encoderOpts: true},
}

func (e *errchkjson) handleJSONMarshal(pass *analysis.Pass, ce *ast.CallExpr, fnName string, jf jsonFunc, errorTarget marshalErrorTarget) {
	argTypes := callArgTypes(pass, ce, jf.v2)
	if len(argTypes) <= jf.argIdx {
		return
	}

	omitSafe := e.omitSafe || jf.alwaysOmitSafe

	var err error
	if jf.v2 && len(argTypes) > jf.argIdx+1 {
		// Options (e.g. json.WithMarshalers) can change the encoding of any type,
		// therefore the safety of the encoding can not be determined.
		err = fmt.Errorf("options passed, safety of the encoding can not be determined")
	} else {
		t := argTypes[jf.argIdx]
		if t == nil {
			// Not sure, if this is at all possible
			if errorTarget == blankIdentifier {
				pass.Reportf(ce.Pos(), "Type of argument to `%s` could not be evaluated and error return value is not checked", fnName)
			}
			return
		}

		if _, ok := t.(*types.Pointer); ok {
			t = t.(*types.Pointer).Elem()
		}

		err = e.jsonSafe(t, 0, map[types.Type]bool{}, jf.v2)
		if jf.encoderOpts {
			// The options of the encoder (e.g. json.WithMarshalers passed to
			// jsontext.NewEncoder) can change the encoding of any type,
			// therefore a type is never considered unsupported.
			err = demoteUnsupported(err)
		}
	}

	if err != nil {
		if _, ok := err.(unsupported); ok {
			pass.Reportf(ce.Pos(), "`%s` for %v", fnName, err)
			return
		}
		if _, ok := err.(noexported); ok {
			pass.Reportf(ce.Pos(), "Error argument passed to `%s` does not contain any exported field", fnName)
		}
		// Only care about unsafe types if they are assigned to the blank identifier.
		if errorTarget == blankIdentifier {
			pass.Reportf(ce.Pos(), "Error return value of `%s` is not checked: %v", fnName, err)
		}
	}
	if err == nil && errorTarget == variableAssignment && !omitSafe {
		pass.Reportf(ce.Pos(), "Error return value of `%s` is checked but passed argument is safe", fnName)
	}
	// Report an error, if err for json.Marshal is not checked and safe types are omitted
	if err == nil && errorTarget == blankIdentifier && omitSafe {
		pass.Reportf(ce.Pos(), "Error return value of `%s` is not checked", fnName)
	}
}

const (
	allowedBasicTypes       = types.IsBoolean | types.IsInteger | types.IsString
	allowedMapKeyBasicTypes = types.IsInteger | types.IsString
	unsupportedBasicTypes   = types.IsComplex
)

// callArgTypes returns the types of the arguments of the call ce. If
// expandTuple is true and the only argument is a call returning multiple
// values (e.g. json.MarshalWrite(f())), the types of the returned values are
// returned instead.
//
// For encoding/json, multi-value arguments are not expanded for backwards
// compatibility, the tuple type is checked as is (and considered unsafe).
func callArgTypes(pass *analysis.Pass, ce *ast.CallExpr, expandTuple bool) []types.Type {
	if expandTuple && len(ce.Args) == 1 {
		if tuple, ok := pass.TypesInfo.TypeOf(ce.Args[0]).(*types.Tuple); ok {
			argTypes := make([]types.Type, 0, tuple.Len())
			for v := range tuple.Variables() {
				argTypes = append(argTypes, v.Type())
			}
			return argTypes
		}
	}

	argTypes := make([]types.Type, 0, len(ce.Args))
	for _, arg := range ce.Args {
		argTypes = append(argTypes, pass.TypesInfo.TypeOf(arg))
	}
	return argTypes
}

// jsonSafe checks, if encoding a value of type t to JSON can fail. If v2 is
// true, the rules of encoding/json/v2 are applied.
//
// seenTypes contains the named types and structs, which have already been
// checked (false) or are currently being checked (true).
func (e *errchkjson) jsonSafe(t types.Type, level int, seenTypes map[types.Type]bool, v2 bool) error {
	if inProgress, ok := seenTypes[t]; ok {
		// A type, which is currently being checked, is recursive (e.g.
		// `type Node struct{ Next *Node }` or `type Tree map[string]Tree`).
		// encoding/json/v2 fails for cyclic values and values exceeding the
		// maximum nesting depth. For encoding/json, recursive types are
		// considered safe for backwards compatibility (see
		// https://github.com/breml/errchkjson/issues/94).
		if inProgress && v2 {
			return fmt.Errorf("unsafe recursive type `%s` found", t.String())
		}
		return nil
	}
	// Named types may be recursive without a struct being involved (e.g.
	// `type Tree map[string]Tree`), mark them as seen to prevent an infinite
	// recursion.
	_, isNamed := types.Unalias(t).(*types.Named)
	_, isStruct := t.Underlying().(*types.Struct)
	if isNamed || isStruct {
		seenTypes[t] = true
		defer func() { seenTypes[t] = false }()
	}

	if implementsMarshaler(t, v2) {
		return fmt.Errorf("unsafe type `%s` found", t.String())
	}

	// In encoding/json/v2, time.Duration has no default representation.
	if v2 && isTimeDuration(t) {
		return fmt.Errorf("unsafe type `%s` found", t.String())
	}

	switch ut := t.Underlying().(type) {
	case *types.Basic:
		if ut.Info()&allowedBasicTypes > 0 { // bool, int-family, string
			if ut.Info()&types.IsString > 0 {
				if t.String() == "encoding/json.Number" {
					return fmt.Errorf("unsafe type `%s` found", t.String())
				}
				// encoding/json/v2 rejects strings containing invalid UTF-8.
				if v2 {
					return fmt.Errorf("unsafe type `%s` found", t.String())
				}
			}
			return nil
		}
		if ut.Info()&unsupportedBasicTypes > 0 { // complex64, complex128
			return newUnsupportedError(fmt.Errorf("unsupported type `%s` found", ut.String()))
		}
		switch ut.Kind() {
		case types.UntypedNil:
			return nil
		case types.UnsafePointer:
			return newUnsupportedError(fmt.Errorf("unsupported type `%s` found", ut.String()))
		default:
			// E.g. float32, float64
			return fmt.Errorf("unsafe type `%s` found", ut.String())
		}

	case *types.Array:
		if v2 && ut.Len() == 0 {
			// encoding/json/v2 encodes arrays of length 0 without
			// considering the element type.
			return nil
		}
		err := e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if err != nil {
			return err
		}
		return nil

	case *types.Slice:
		err := e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if v2 {
			// encoding/json/v2 encodes nil and empty slices without
			// considering the element type.
			err = demoteUnsupported(err)
		}
		if err != nil {
			return err
		}
		return nil

	case *types.Struct:
		s := structState{names: map[string]struct{}{}}
		var firstErr error
		for i := 0; i < ut.NumFields(); i++ {
			err := e.jsonSafeField(t, ut, i, level, seenTypes, v2, &s)
			if err == nil {
				continue
			}
			if _, ok := err.(unsupported); ok || !v2 {
				return err
			}
			// For encoding/json/v2, many types are unsafe (e.g. string),
			// therefore the remaining fields are checked as well, because an
			// unsupported field takes precedence over an unsafe field.
			if firstErr == nil {
				firstErr = err
			}
		}
		if firstErr != nil {
			return firstErr
		}
		if e.reportNoExported && level == 0 && s.exported == 0 {
			return newNoexportedError(fmt.Errorf("struct does not export any field"))
		}
		// encoding/json/v2 rejects structs, which have fields, but neither a
		// field considered for encoding nor any json struct tag.
		if v2 && ut.NumFields() > 0 && !s.hasAnyJSONField && !s.hasAnyJSONTag {
			return fmt.Errorf("unsafe struct `%s` without exported fields found", t.String())
		}
		return nil

	case *types.Pointer:
		err := e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if v2 {
			// encoding/json/v2 encodes a nil pointer as null.
			err = demoteUnsupported(err)
		}
		if err != nil {
			return err
		}
		return nil

	case *types.Map:
		err := jsonSafeMapKey(ut.Key(), v2)
		if v2 {
			// encoding/json/v2 encodes nil and empty maps without
			// considering the key type.
			err = demoteUnsupported(err)
		}
		if err != nil {
			return err
		}
		err = e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if v2 {
			// encoding/json/v2 encodes nil and empty maps without
			// considering the element type.
			err = demoteUnsupported(err)
		}
		if err != nil {
			return err
		}
		return nil

	case *types.Chan, *types.Signature:
		// Types that are not supported for encoding to json:
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` found", ut.String()))

	default:
		// Types that are not supported for encoding to json or are not completely safe, like: interfaces
		return fmt.Errorf("unsafe type `%s` found", t.String())
	}
}

// structState holds the state collected by jsonSafeField while checking the
// fields of a struct.
type structState struct {
	exported        int                 // number of exported fields.
	hasAnyJSONTag   bool                // any field has a json struct tag (including `json:"-"`).
	hasAnyJSONField bool                // any field is considered for encoding.
	names           map[string]struct{} // JSON object names of the direct fields (encoding/json/v2).
}

// jsonSafeField checks the i-th field of the struct ut (the underlying type of
// t), if encoding it to JSON can fail.
func (e *errchkjson) jsonSafeField(t types.Type, ut *types.Struct, i int, level int, seenTypes map[types.Type]bool, v2 bool, s *structState) error {
	field := ut.Field(i)
	tag, hasTag := reflect.StructTag(ut.Tag(i)).Lookup("json")
	s.hasAnyJSONTag = s.hasAnyJSONTag || hasTag
	if tag == "-" {
		// Fields omitted in json can be ignored
		return nil
	}
	name, options := parseJSONTag(tag)
	isExported := field.Exported()
	if !isExported && v2 && field.Embedded() && isStruct(field.Type()) {
		// encoding/json/v2 encodes the exported fields of embedded
		// structs of unexported types. Such fields are therefore
		// checked like exported fields, but do not count as exported
		// for the report-no-exported flag.
		isExported = true
	}
	if !isExported {
		if field.Embedded() {
			// encoding/json/v2 rejects embedded fields of unexported
			// non-struct types (with or without a JSON name). Embedded
			// structs of unexported types are handled above.
			if v2 {
				return fmt.Errorf("unsafe embedded field `%s` of unexported non-struct type found", field.Name())
			}
			// Embedded fields of unexported types may still forward
			// exported fields.
			s.hasAnyJSONField = true
			// Since Go 1.27, encoding/json rejects the `format` option on
			// embedded structs of unexported types with a JSON name, unless
			// the methods of the type prevent the field from being encoded.
			option, hasFormat := formatOptionWithValue(tag)
			if hasFormat && name != "" && isStruct(field.Type()) && !unexportedMethodsV2(field.Type(), slices.Contains(options, "omitzero")) {
				return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
			}
		} else if v2 && hasTag {
			// encoding/json/v2 rejects json struct tags (other than
			// `json:"-"`) on unexported fields.
			return fmt.Errorf("unsafe struct tag on unexported field `%s` found", field.Name())
		}
		// Otherwise, unexported fields are ignored.
		return nil
	}
	s.hasAnyJSONField = true

	tagErr := checkJSONTag(t, field, tag, name, options, v2, s)

	err := e.jsonSafe(field.Type(), level+1, seenTypes, v2)
	if v2 && slices.Contains(options, "omitzero") {
		// encoding/json/v2 omits the zero value without encoding it.
		err = demoteUnsupported(err)
	}
	if _, ok := err.(unsupported); ok && v2 {
		// An unsupported type takes precedence over an unsafe struct tag.
		return err
	}
	if tagErr != nil {
		return tagErr
	}
	if err != nil {
		return err
	}
	if field.Exported() {
		s.exported++
	}
	return nil
}

// checkJSONTag checks the json struct tag of the field, which is considered
// for encoding.
func checkJSONTag(t types.Type, field *types.Var, tag string, name string, options []string, v2 bool, s *structState) error {
	// encoding/json/v2 rejects JSON object names containing a backslash
	// or quote characters (including single-quoted names).
	if v2 && strings.ContainsAny(name, "\\'\"`") {
		return fmt.Errorf("unsafe JSON object name `%s` on field `%s` found", name, field.Name())
	}
	if v2 && !utf8.ValidString(name) {
		return fmt.Errorf("unsafe JSON object name with invalid UTF-8 on field `%s` found", field.Name())
	}
	// encoding/json/v2 of Go 1.26 (GOEXPERIMENT=jsonv2) rejects the JSON
	// object name `-` (e.g. `json:"-,omitempty"`).
	if v2 && name == "-" {
		return fmt.Errorf("unsafe JSON object name `-` on field `%s` found", field.Name())
	}
	// encoding/json/v2 rejects embedded fields of non-struct types
	// (e.g. named basic types, slices or interfaces) without a JSON name.
	if v2 && field.Embedded() && name == "" && !isStruct(field.Type()) {
		return fmt.Errorf("unsafe embedded field `%s` of non-struct type without JSON name found", field.Name())
	}
	// Embedded struct without JSON name.
	implicitEmbed := field.Embedded() && name == "" && isStruct(field.Type())
	// Since Go 1.27, the `format` option is rejected by encoding/json and
	// encoding/json/v2 for all types, because support for it was removed
	// (see https://go.dev/issue/79071). encoding/json/v2 rejects any `format`
	// option, encoding/json only a `format` option with a valid value on
	// fields other than embedded structs without JSON name.
	if option, ok := formatOptionWithValue(tag); !v2 && !implicitEmbed && ok {
		return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
	}
	if !v2 {
		return nil
	}
	for _, option := range options {
		if strings.HasPrefix(option, "format:") {
			return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
		}
		// encoding/json/v2 rejects the `string` option on non-numeric types.
		if option == "string" && !isNumeric(field.Type()) {
			return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
		}
	}
	seenOptions, option, err := checkJSONTagOptionsV2(options)
	if err != nil {
		return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found: %v", option, field.Name(), err)
	}
	// Fields embedded by encoding/json/v2: embedded struct fields without
	// a JSON name and fields with the `embed` option. Fields with the
	// `inline` or `unknown` options are embedded by encoding/json/v2 of
	// Go 1.26 (GOEXPERIMENT=jsonv2), but are regular fields as of Go 1.27.
	switch {
	case implicitEmbed || seenOptions["embed"] || seenOptions["inline"] || seenOptions["unknown"]:
		err := checkJSONEmbedV2(field.Type(), name, seenOptions)
		if err != nil {
			return fmt.Errorf("unsafe embedded field `%s` found: %v", field.Name(), err)
		}
	case !field.Exported():
		// Embedded struct of an unexported type with a JSON name,
		// encoded as regular field. Methods of the unexported type
		// can not be called.
		if unexportedMethodsV2(field.Type(), seenOptions["omitzero"]) {
			return fmt.Errorf("unsafe field `%s` of unexported type with methods for JSON encoding found", field.Name())
		}
	}
	// encoding/json/v2 rejects structs, where two direct fields use the
	// same JSON object name. Conflicts with fields promoted from embedded
	// structs (without an explicit name) are not an error, they are
	// resolved by the Go embedding rules. Fields with the `embed` option
	// are regular fields for encoding/json/v2 of Go 1.26 (the option is
	// unknown) and are therefore checked as well.
	if !implicitEmbed {
		if name == "" {
			name = field.Name()
		}
		if _, ok := s.names[name]; ok {
			return fmt.Errorf("unsafe duplicate JSON object name `%s` in struct `%s` found", name, t.String())
		}
		s.names[name] = struct{}{}
	}
	return nil
}

// unexportedMethodsV2 reports, if encoding/json/v2 rejects an embedded struct
// field of the unexported type t (or *t) with a JSON name, because methods of
// the type can not be called.
func unexportedMethodsV2(t types.Type, omitzero bool) bool {
	if p, ok := t.Underlying().(*types.Pointer); ok {
		t = p.Elem()
	}
	return implementsJSONMethods(t) || (omitzero && implementsIsZero(t))
}

// demoteUnsupported turns an unsupported error into an unsafe error. It is
// used, where encoding/json/v2 only fails for some values of an unsupported
// type (e.g. a non-empty slice of channels).
func demoteUnsupported(err error) error {
	if _, ok := err.(unsupported); ok {
		return errors.New(err.Error())
	}
	return err
}

func jsonSafeMapKey(t types.Type, v2 bool) error {
	if implementsMarshaler(t, v2) || (v2 && isTimeDuration(t)) {
		return fmt.Errorf("unsafe type `%s` as map key found", t.String())
	}
	switch ut := t.Underlying().(type) {
	case *types.Basic:
		if ut.Info()&types.IsString > 0 && (t.String() == "encoding/json.Number" || v2) {
			return fmt.Errorf("unsafe type `%s` as map key found", t.String())
		}
		if ut.Info()&allowedMapKeyBasicTypes > 0 { // int-family, string
			return nil
		}
		// encoding/json/v2 supports float map keys, but NaN and Inf values fail.
		if v2 && ut.Info()&types.IsFloat > 0 {
			return fmt.Errorf("unsafe type `%s` as map key found", t.String())
		}
		// E.g. bool, float32, float64, complex64, complex128
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` as map key found", t.String()))
	case *types.Interface:
		return fmt.Errorf("unsafe type `%s` as map key found", t.String())
	case *types.Array:
		// encoding/json/v2 encodes arrays of bytes as base64 encoded string.
		if v2 && isByte(ut.Elem()) {
			return nil
		}
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` as map key found", t.String()))
	case *types.Pointer:
		if !v2 {
			return newUnsupportedError(fmt.Errorf("unsupported type `%s` as map key found", t.String()))
		}
		// encoding/json/v2 encodes pointer map keys like the pointed to
		// value, but fails for nil pointers.
		return fmt.Errorf("unsafe type `%s` as map key found", t.String())
	default:
		// E.g. struct composed solely of basic types, that are comparable
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` as map key found", t.String()))
	}
}

// jsonTagOptionsV2 are the json struct tag options known by encoding/json/v2.
// The options `inline` and `unknown` are only known by encoding/json/v2 of
// Go 1.26 (GOEXPERIMENT=jsonv2), they have been replaced by `embed` in Go 1.27.
var jsonTagOptionsV2 = []string{"case", "embed", "inline", "unknown", "omitzero", "omitempty", "string", "format"}

// checkJSONTagOptionsV2 checks the json struct tag options (as returned by
// parseJSONTag) according to the rules of encoding/json/v2. It returns the
// set of options. If encoding/json/v2 rejects an option, the raw option
// and an error are returned. Unknown options are ignored by encoding/json/v2,
// unless they appear more than once.
func checkJSONTagOptionsV2(options []string) (map[string]bool, string, error) {
	seen := map[string]bool{}
	for _, option := range options {
		// Options are Go identifiers, optionally followed by `:value` for
		// the `case` and `format` options.
		ident := ""
		if r, _ := utf8.DecodeRuneInString(option); r == '_' || unicode.IsLetter(r) {
			ident = option[:len(option)-len(strings.TrimLeftFunc(option, isLetterOrDigit))]
		}
		rest := option[len(ident):]

		switch ident {
		case "":
			// E.g. trailing comma, empty or quoted option.
			return nil, option, fmt.Errorf("malformed option")
		case "case":
			if rest != ":ignore" && rest != ":strict" {
				return nil, option, fmt.Errorf("invalid value, only `case:ignore` and `case:strict` are supported")
			}
		case "format":
			// Since Go 1.27, the `format` option is rejected for all types.
			return nil, option, fmt.Errorf("`format` is not supported")
		default:
			if rest != "" {
				return nil, option, fmt.Errorf("malformed option")
			}
			// encoding/json/v2 rejects options resembling a known option
			// (e.g. `omitEmpty` or `omit_empty`).
			normalized := strings.ReplaceAll(strings.ToLower(ident), "_", "")
			if normalized != ident && slices.Contains(jsonTagOptionsV2, normalized) {
				return nil, option, fmt.Errorf("invalid spelling of `%s`", normalized)
			}
		}

		if seen[ident] {
			return nil, option, fmt.Errorf("duplicate option")
		}
		seen[ident] = true
	}
	return seen, "", nil
}

// checkJSONEmbedV2 checks a field, which is embedded by encoding/json/v2
// (embedded struct without JSON name or field with the `embed` option, or the
// `inline` or `unknown` options for Go 1.26), according to the rules of
// encoding/json/v2.
func checkJSONEmbedV2(t types.Type, name string, options map[string]bool) error {
	if name != "" {
		return fmt.Errorf("JSON object name `%s` is not allowed", name)
	}
	if options["inline"] && options["unknown"] {
		return fmt.Errorf("options `inline` and `unknown` can not be combined")
	}
	for _, option := range jsonTagOptionsV2 {
		switch option {
		case "embed", "inline", "unknown":
			continue
		}
		if options[option] {
			return fmt.Errorf("option `%s` is not allowed, only `embed` is supported", option)
		}
	}

	// Pointers to the embedded type are allowed.
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	if isJSONTextValue(t) {
		return nil
	}
	// The fields to embed can not be determined, if the type customizes its
	// encoding or decoding.
	if implementsJSONMethods(t) {
		return fmt.Errorf("type `%s` must not implement marshal or unmarshal methods", t.String())
	}
	switch ut := t.Underlying().(type) {
	case *types.Struct:
		if options["unknown"] {
			return fmt.Errorf("type `%s` must be a map with string key or jsontext.Value for option `unknown`", t.String())
		}
		return nil
	case *types.Map:
		if b, ok := ut.Key().Underlying().(*types.Basic); ok && b.Info()&types.IsString > 0 {
			if implementsJSONMethods(ut.Key()) {
				return fmt.Errorf("map key type `%s` must not implement marshal or unmarshal methods", ut.Key().String())
			}
			return nil
		}
	}
	return fmt.Errorf("type `%s` must be a struct, a map with string key or jsontext.Value", t.String())
}

// isJSONTextValue reports, if t is encoding/json/jsontext.Value.
func isJSONTextValue(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "encoding/json/jsontext" && obj.Name() == "Value"
}

// formatOptionWithValue reports, if the json struct tag contains a `format`
// option with a value, which is accepted by the parser of the json struct tag
// (a Go identifier or a non-empty single-quoted string), and returns the
// option. Since Go 1.27, encoding/json rejects such options. Without a valid
// value, the option is ignored by encoding/json.
func formatOptionWithValue(tag string) (string, bool) {
	_, rest, found := strings.Cut(tag, ",")
	for found {
		if value, ok := strings.CutPrefix(rest, "format:"); ok {
			v, n, ok := consumeTagOptionValue(value)
			if ok && v != "" {
				return "format:" + value[:n], true
			}
			rest = value[n:]
		}
		_, rest, found = strings.Cut(rest, ",")
	}
	return "", false
}

// consumeTagOptionValue parses the value of a json struct tag option (a Go
// identifier or a single-quoted string) at the start of in, like the parser of
// encoding/json/v2 does. It returns the value and the number of consumed
// bytes. If the value is malformed, ok is false and the bytes up to the next
// comma are consumed.
func consumeTagOptionValue(in string) (value string, n int, ok bool) {
	i := strings.IndexByte(in, ',')
	if i < 0 {
		i = len(in)
	}

	switch r, _ := utf8.DecodeRuneInString(in); {
	case r == '_' || unicode.IsLetter(r):
		n := len(in) - len(strings.TrimLeftFunc(in, isLetterOrDigit))
		return in[:n], n, true
	case r == '\'':
		// Convert the single-quoted string to a double-quoted string and
		// rely on strconv.Unquote to handle the rest.
		inEscape := false
		b := []byte{'"'}
		n := len("'")
		for len(in) > n {
			r, rn := utf8.DecodeRuneInString(in[n:])
			switch {
			case inEscape:
				if r == '\'' {
					b = b[:len(b)-1] // remove escape character: `\'` => `'`
				}
				inEscape = false
			case r == '\\':
				inEscape = true
			case r == '"':
				b = append(b, '\\') // insert escape character: `"` => `\"`
			case r == '\'':
				b = append(b, '"')
				n += len("'")
				out, err := strconv.Unquote(string(b))
				if err != nil {
					return "", i, false
				}
				return out, n, true
			}
			b = append(b, in[n:][:rn]...)
			n += rn
		}
		// Single-quoted string not terminated.
		return "", i, false
	default:
		return "", i, false
	}
}

// isByte reports, if t is the predeclared type byte (uint8). Named byte types
// are not considered.
func isByte(t types.Type) bool {
	b, ok := types.Unalias(t).(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

// isLetterOrDigit reports, if r is allowed in a json struct tag option.
func isLetterOrDigit(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// parseJSONTag splits a json struct tag into the name and the options.
func parseJSONTag(tag string) (string, []string) {
	name, rest, found := strings.Cut(tag, ",")
	if !found {
		return name, nil
	}
	return name, strings.Split(rest, ",")
}

// isNumeric reports, if t (after dereferencing pointers) is an integer or
// float type. A named pointer type referencing itself (e.g. `type P *P`) is
// considered numeric, because no value of such a type ends in a non-numeric
// value (only nil or a cycle).
func isNumeric(t types.Type) bool {
	seen := map[types.Type]bool{}
	for {
		p, ok := t.Underlying().(*types.Pointer)
		if !ok {
			break
		}
		if seen[t] {
			return true
		}
		seen[t] = true
		t = p.Elem()
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&(types.IsInteger|types.IsFloat) > 0
}

// isStruct reports, if t is a struct or a pointer to a struct.
func isStruct(t types.Type) bool {
	if p, ok := t.Underlying().(*types.Pointer); ok {
		t = p.Elem()
	}
	_, ok := t.Underlying().(*types.Struct)
	return ok
}

// isTimeDuration reports, if t is time.Duration.
func isTimeDuration(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "Duration"
}

// implementsMarshaler reports, if t implements one of the interfaces, which
// allow a type to customize its JSON encoding (and therefore may return an error).
//
// If v2 is true, the method set of *t is considered as well, because
// encoding/json/v2 always calls marshal methods with a pointer receiver, even
// if the value is not addressable.
func implementsMarshaler(t types.Type, v2 bool) bool {
	if implementsMarshalerMethodSet(t) {
		return true
	}
	if !v2 {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return false
	}
	return implementsMarshalerMethodSet(types.NewPointer(t))
}

// implementsMarshalerMethodSet reports, if the method set of t contains one
// of the methods, which allow a type to customize its JSON encoding.
func implementsMarshalerMethodSet(t types.Type) bool {
	return types.Implements(t, textMarshalerInterface()) ||
		types.Implements(t, jsonMarshalerInterface()) ||
		types.Implements(t, textAppenderInterface()) ||
		implementsJSONMarshalerTo(t)
}

// implementsJSONMethods reports, if t or *t implements one of the interfaces,
// which allow a type to customize its JSON encoding or decoding.
func implementsJSONMethods(t types.Type) bool {
	return implementsMarshalerMethodSet(t) || implementsUnmarshalerMethodSet(t) ||
		implementsMarshalerMethodSet(types.NewPointer(t)) || implementsUnmarshalerMethodSet(types.NewPointer(t))
}

// implementsUnmarshalerMethodSet reports, if the method set of t contains one
// of the methods, which allow a type to customize its JSON decoding.
func implementsUnmarshalerMethodSet(t types.Type) bool {
	return types.Implements(t, unmarshalerInterface("UnmarshalText")) ||
		types.Implements(t, unmarshalerInterface("UnmarshalJSON")) ||
		implementsJSONTextMethod(t, "UnmarshalJSONFrom", "Decoder")
}

// implementsIsZero reports, if t or *t implements the method IsZero() bool,
// which is used by encoding/json/v2 for the `omitzero` option.
func implementsIsZero(t types.Type) bool {
	isZeroer := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, nil, "IsZero", types.NewSignatureType(
			nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])),
			false)),
	}, nil)
	isZeroer.Complete()
	return types.Implements(t, isZeroer) || types.Implements(types.NewPointer(t), isZeroer)
}

func (e *errchkjson) inspectArgs(pass *analysis.Pass, args []ast.Expr) {
	for _, a := range args {
		ast.Inspect(a, func(n ast.Node) bool {
			if n == nil {
				return true
			}

			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return false
			}

			fn, _ := typeutil.Callee(pass.TypesInfo, ce).(*types.Func)
			if fn == nil {
				return true
			}

			if jf, ok := jsonFuncs[fn.FullName()]; ok {
				e.handleJSONMarshal(pass, ce, fn.FullName(), jf, functionArgument)
			} else {
				e.inspectArgs(pass, ce.Args)
			}
			return false
		})
	}
}

// Construct *types.Interface for interface encoding.TextMarshaler
//
//	type TextMarshaler interface {
//	    MarshalText() (text []byte, err error)
//	}
func textMarshalerInterface() *types.Interface {
	textMarshalerInterface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, nil, "MarshalText", types.NewSignatureType(
			nil, nil, nil, nil, types.NewTuple(
				types.NewVar(token.NoPos, nil, "text",
					types.NewSlice(
						types.Universe.Lookup("byte").Type())),
				types.NewVar(token.NoPos, nil, "err", types.Universe.Lookup("error").Type())),
			false)),
	}, nil)
	textMarshalerInterface.Complete()

	return textMarshalerInterface
}

// Construct *types.Interface for interface json.Marshaler
//
//	type Marshaler interface {
//	    MarshalJSON() ([]byte, error)
//	}
func jsonMarshalerInterface() *types.Interface {
	textMarshalerInterface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, nil, "MarshalJSON", types.NewSignatureType(
			nil, nil, nil, nil, types.NewTuple(
				types.NewVar(token.NoPos, nil, "",
					types.NewSlice(
						types.Universe.Lookup("byte").Type())),
				types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
			false)),
	}, nil)
	textMarshalerInterface.Complete()

	return textMarshalerInterface
}

// Construct *types.Interface for interface encoding.TextAppender
//
//	type TextAppender interface {
//	    AppendText(b []byte) ([]byte, error)
//	}
func textAppenderInterface() *types.Interface {
	byteSlice := types.NewSlice(types.Universe.Lookup("byte").Type())
	textAppenderInterface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, nil, "AppendText", types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "b", byteSlice)),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", byteSlice),
				types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
			false)),
	}, nil)
	textAppenderInterface.Complete()

	return textAppenderInterface
}

// Construct *types.Interface for the interfaces encoding.TextUnmarshaler and
// json.Unmarshaler
//
//	type TextUnmarshaler interface {
//	    UnmarshalText(text []byte) error
//	}
//
//	type Unmarshaler interface {
//	    UnmarshalJSON([]byte) error
//	}
func unmarshalerInterface(method string) *types.Interface {
	unmarshalerInterface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, nil, method, types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.NewSlice(types.Universe.Lookup("byte").Type()))),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())),
			false)),
	}, nil)
	unmarshalerInterface.Complete()

	return unmarshalerInterface
}

// implementsJSONMarshalerTo reports, if t implements the interface
// encoding/json/v2.MarshalerTo. The interface can not be constructed with
// go/types, because this requires the type object of *jsontext.Encoder of the
// analyzed program. Therefore, the method is matched by name, signature and
// the package path of its parameter type.
//
//	type MarshalerTo interface {
//	    MarshalJSONTo(*jsontext.Encoder) error
//	}
func implementsJSONMarshalerTo(t types.Type) bool {
	return implementsJSONTextMethod(t, "MarshalJSONTo", "Encoder")
}

// implementsJSONTextMethod reports, if the method set of t contains the method
// with the signature `method(*jsontext.<typeName>) error` (e.g.
// `MarshalJSONTo(*jsontext.Encoder) error` or
// `UnmarshalJSONFrom(*jsontext.Decoder) error`).
func implementsJSONTextMethod(t types.Type, method string, typeName string) bool {
	obj, _, _ := types.LookupFieldOrMethod(t, false, nil, method)
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig := fn.Signature()
	if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return false
	}
	if !types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup("error").Type()) {
		return false
	}
	ptr, ok := sig.Params().At(0).Type().(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	if !ok {
		return false
	}
	obj2 := named.Obj()
	return obj2.Pkg() != nil && obj2.Pkg().Path() == "encoding/json/jsontext" && obj2.Name() == typeName
}

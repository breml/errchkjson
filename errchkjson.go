// Package errchkjson defines an Analyzer that finds places, where it is
// safe to omit checking the error returned from json.Marshal.
package errchkjson

import (
	"cmp"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
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

		isPtr := false
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
			isPtr = true
		}

		err = e.jsonSafe(t, 0, newSafeState(jf.v2))
		if jf.v2 && err != nil && safeConstantArgV2(pass, ce, jf, t) {
			err = nil
		}
		if isPtr && !nonNilPointerArg(pass, ce, jf) {
			// A nil pointer is encoded as null.
			err = demoteUnsupported(err)
		}
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

// safeConstantArgV2 reports, if the argument to be encoded of the call ce is a
// constant of type t, which is always encoded by encoding/json/v2 without
// error, although other values of type t may fail: a string constant with
// valid UTF-8 or a float constant (constants are always finite).
//
// For backwards compatibility, this is not applied to encoding/json, where
// float constants would otherwise be reported as safe.
func safeConstantArgV2(pass *analysis.Pass, ce *ast.CallExpr, jf jsonFunc, t types.Type) bool {
	if len(ce.Args) <= jf.argIdx {
		// Multi-value argument (e.g. json.MarshalWrite(f())).
		return false
	}
	tv, ok := pass.TypesInfo.Types[ce.Args[jf.argIdx]]
	if !ok || tv.Value == nil {
		return false
	}
	// The encoding of types with methods (or of encoding/json.Number) does
	// not only depend on the value.
	if implementsMarshaler(t, true) || t.String() == "encoding/json.Number" {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch {
	case b.Info()&types.IsString != 0:
		return tv.Value.Kind() == constant.String && utf8.ValidString(constant.StringVal(tv.Value))
	case b.Info()&types.IsFloat != 0:
		return true
	}
	return false
}

// nonNilPointerArg reports, if the argument to be encoded of the call ce is a
// pointer, which is known to be non-nil (e.g. `&v` or `new(T)`).
func nonNilPointerArg(pass *analysis.Pass, ce *ast.CallExpr, jf jsonFunc) bool {
	if len(ce.Args) <= jf.argIdx {
		// Multi-value argument (e.g. json.MarshalWrite(f())).
		return false
	}
	switch arg := ast.Unparen(ce.Args[jf.argIdx]).(type) {
	case *ast.UnaryExpr:
		return arg.Op == token.AND
	case *ast.CallExpr:
		ident, ok := ast.Unparen(arg.Fun).(*ast.Ident)
		if !ok {
			return false
		}
		b, ok := pass.TypesInfo.Uses[ident].(*types.Builtin)
		return ok && b.Name() == "new"
	}
	return false
}

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

// safeState holds the state of checking the type of a single argument to a
// JSON encoding function.
type safeState struct {
	v2 bool // apply the rules of encoding/json/v2.

	// stack contains the named types and structs, which are currently being
	// checked, with their position in the stack.
	stack map[types.Type]int
	// results contains the results of the named types and structs, which
	// have been checked already.
	results map[types.Type]error
	// low is the lowest position in the stack of a type, which has been
	// reached again while it was being checked (recursion).
	low int
}

func newSafeState(v2 bool) *safeState {
	return &safeState{
		v2:      v2,
		stack:   map[types.Type]int{},
		results: map[types.Type]error{},
		low:     math.MaxInt,
	}
}

// jsonSafe checks, if encoding a value of type t to JSON can fail. If
// st.v2 is true, the rules of encoding/json/v2 are applied.
//
// The returned error is nil (safe), an unsupported error (encoding fails for
// all values of type t) or any other error (unsafe, encoding fails for some
// values of type t).
func (e *errchkjson) jsonSafe(t types.Type, level int, st *safeState) (err error) {
	if pos, ok := st.stack[t]; ok {
		// A type, which is currently being checked, is recursive (e.g.
		// `type Node struct{ Next *Node }` or `type Tree map[string]Tree`).
		// encoding/json/v2 fails for cyclic values and values exceeding the
		// maximum nesting depth. For encoding/json, recursive types are
		// considered safe for backwards compatibility (see
		// https://github.com/breml/errchkjson/issues/94).
		st.low = min(st.low, pos)
		if st.v2 {
			return fmt.Errorf("unsafe recursive type `%s` found", t.String())
		}
		return nil
	}
	if res, ok := st.results[t]; ok {
		return res
	}
	// Named types may be recursive without a struct being involved (e.g.
	// `type Tree map[string]Tree`), they are therefore tracked as well.
	_, isNamed := types.Unalias(t).(*types.Named)
	_, isStruct := t.Underlying().(*types.Struct)
	if isNamed || isStruct {
		pos := len(st.stack)
		st.stack[t] = pos
		outerLow := st.low
		st.low = math.MaxInt
		defer func() {
			delete(st.stack, t)
			if st.low >= pos {
				// The result does not depend on a type further up in the
				// stack and can therefore be reused.
				st.results[t] = err
				st.low = outerLow
				return
			}
			st.low = min(outerLow, st.low)
		}()
	}

	if implementsMarshaler(t, st.v2) {
		return fmt.Errorf("unsafe type `%s` found", t.String())
	}

	// In encoding/json/v2, time.Duration has no default representation and
	// encoding it fails for all values.
	if st.v2 && isTimeDuration(t) {
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` found", t.String()))
	}

	switch ut := t.Underlying().(type) {
	case *types.Basic:
		if ut.Info()&allowedBasicTypes > 0 { // bool, int-family, string
			if ut.Info()&types.IsString > 0 {
				if t.String() == "encoding/json.Number" {
					return fmt.Errorf("unsafe type `%s` found", t.String())
				}
				// encoding/json/v2 rejects strings containing invalid UTF-8.
				if st.v2 {
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
		if ut.Len() == 0 {
			// Arrays of length 0 are encoded without considering the
			// element type.
			return nil
		}
		return e.jsonSafe(ut.Elem(), level+1, st)

	case *types.Slice:
		// nil and empty slices are encoded without considering the element
		// type.
		return demoteUnsupported(e.jsonSafe(ut.Elem(), level+1, st))

	case *types.Struct:
		if st.v2 {
			return e.jsonSafeStructV2(t, ut, level, st)
		}
		return e.jsonSafeStruct(ut, level, st)

	case *types.Pointer:
		// A nil pointer is encoded as null.
		return demoteUnsupported(e.jsonSafe(ut.Elem(), level+1, st))

	case *types.Map:
		// nil and empty maps are encoded without considering the key type
		// (as of Go 1.27 for encoding/json) and the element type.
		err := demoteUnsupported(jsonSafeMapKey(ut.Key(), st.v2))
		if err != nil {
			return err
		}
		return demoteUnsupported(e.jsonSafe(ut.Elem(), level+1, st))

	case *types.Chan, *types.Signature:
		// Types that are not supported for encoding to json:
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` found", ut.String()))

	default:
		// Types that are not supported for encoding to json or are not completely safe, like: interfaces
		return fmt.Errorf("unsafe type `%s` found", t.String())
	}
}

// jsonSafeStruct checks the struct ut according to the rules of encoding/json.
func (e *errchkjson) jsonSafeStruct(ut *types.Struct, level int, st *safeState) error {
	exported := 0
	var firstErr error
	for i := range ut.NumFields() {
		err := e.jsonSafeField(ut, i, level, st)
		if err == nil {
			if ut.Field(i).Exported() && !ignoredField(ut, i) {
				exported++
			}
			continue
		}
		if isUnsupported(err) {
			return err
		}
		if firstErr != nil {
			continue
		}
		// For backwards compatibility, the check stops at the first unsafe
		// field. If the field is unsafe only because nil and empty values
		// of an unsupported type are encoded without error (e.g. a nil
		// slice of channels), the remaining fields are checked for
		// unsupported types, because such a field has been considered as
		// unsupported before.
		if !isDemoted(err) {
			return err
		}
		firstErr = err
	}
	if firstErr != nil {
		return firstErr
	}
	if e.reportNoExported && level == 0 && exported == 0 {
		return newNoexportedError(fmt.Errorf("struct does not export any field"))
	}
	return nil
}

// ignoredField reports, if the i-th field of the struct ut is omitted in
// JSON (`json:"-"`).
func ignoredField(ut *types.Struct, i int) bool {
	return reflect.StructTag(ut.Tag(i)).Get("json") == "-"
}

// jsonSafeField checks the i-th field of the struct ut according to the rules
// of encoding/json.
func (e *errchkjson) jsonSafeField(ut *types.Struct, i int, level int, st *safeState) error {
	field := ut.Field(i)
	tag := reflect.StructTag(ut.Tag(i)).Get("json")
	if tag == "-" {
		// Fields omitted in json can be ignored
		return nil
	}
	name, options := parseJSONTag(tag)
	if !field.Exported() {
		// Since Go 1.27, encoding/json rejects the `format` option on
		// embedded structs of unexported types with a JSON name, unless the
		// methods of the type prevent the field from being encoded.
		// Otherwise, unexported fields are ignored.
		option, hasFormat := formatOptionWithValue(tag)
		if field.Embedded() && hasFormat && name != "" && isStruct(field.Type()) && !unexportedMethodsV2(field.Type(), slices.Contains(options, "omitzero")) {
			return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
		}
		return nil
	}

	// Since Go 1.27, the `format` option is rejected by encoding/json for all
	// types, because support for it was removed (see
	// https://go.dev/issue/79071). A `format` option without a valid value
	// and a `format` option on an embedded struct without JSON name are
	// ignored.
	var tagErr error
	implicitEmbed := field.Embedded() && name == "" && isStruct(field.Type())
	if option, ok := formatOptionWithValue(tag); !implicitEmbed && ok {
		tagErr = fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
	}

	err := e.jsonSafe(field.Type(), level+1, st)
	if slices.Contains(options, "omitzero") {
		// As of Go 1.24, the zero value is omitted without encoding it.
		err = demoteUnsupported(err)
	}
	if isUnsupported(err) {
		// An unsupported type takes precedence over an unsafe struct tag.
		return err
	}
	if tagErr != nil {
		return tagErr
	}
	return err
}

// fieldV2 is a field of a struct, which is encoded as JSON object member by
// encoding/json/v2. This includes the fields promoted from embedded structs.
type fieldV2 struct {
	field   *types.Var
	path    []int // index path of the field, starting at the checked struct.
	name    string
	hasName bool // the JSON object name is given in the json struct tag.
	// weak fields are encoded differently by the supported versions of
	// encoding/json/v2 (e.g. fields with the `embed` option). They are
	// always checked and do not take part in the resolution of name
	// conflicts between fields.
	weak    bool
	viaPtr  bool // the field is promoted from an embedded pointer.
	options map[string]bool
}

// embeddedV2 is a struct, whose fields are considered by encoding/json/v2.
type embeddedV2 struct {
	t             types.Type
	ut            *types.Struct
	path          []int // index path of the embedded field.
	viaPtr        bool  // the struct is embedded through a pointer.
	visitChildren bool  // the embedded structs of the struct are considered.
}

// pathError is an error found for the field with the given index path.
type pathError struct {
	path []int
	err  error
}

// jsonSafeStructV2 checks the struct ut (the underlying type of t) according
// to the rules of encoding/json/v2.
//
// Like encoding/json/v2, the fields of embedded structs are collected with a
// breadth-first search and conflicts between fields with the same JSON object
// name are resolved with the Go embedding rules. Only the type of the
// resulting fields is checked. An embedded struct, which has been visited
// already, is not visited again, therefore embedding a struct in itself (e.g.
// `type T struct{ *T }`) is not recursive.
func (e *errchkjson) jsonSafeStructV2(t types.Type, ut *types.Struct, level int, st *safeState) error {
	var errs []pathError
	var fields []fieldV2
	var root structState
	exported := 0

	queue := []embeddedV2{{t: t, ut: ut, visitChildren: true}}
	seen := map[types.Type]bool{t: true}
	for qi := 0; qi < len(queue); qi++ {
		qe := queue[qi]
		s := structState{names: map[string]nameKind{}}
		for i := range qe.ut.NumFields() {
			path := append(slices.Clip(qe.path), i)
			f, embedded, err := checkFieldV2(qe.ut, i, qe.t, &s)
			if err != nil {
				errs = append(errs, pathError{path: path, err: err})
			}
			if qi == 0 && qe.ut.Field(i).Exported() && (f != nil || embedded != nil) {
				exported++
			}
			if embedded != nil {
				tf, isPtr := embedded, false
				if p, ok := tf.Underlying().(*types.Pointer); ok {
					tf, isPtr = p.Elem(), true
				}
				if qe.visitChildren {
					queue = append(queue, embeddedV2{
						t:             tf,
						ut:            tf.Underlying().(*types.Struct),
						path:          path,
						viaPtr:        qe.viaPtr || isPtr,
						visitChildren: !seen[tf],
					})
				}
				seen[tf] = true
				continue
			}
			if f == nil {
				continue
			}
			f.path = path
			f.viaPtr = qe.viaPtr
			fields = append(fields, *f)
		}
		if qi == 0 {
			// The checked struct itself is handled below, because the
			// report-no-exported flag takes precedence.
			root = s
			continue
		}
		if qe.ut.NumFields() > 0 && !s.hasAnyJSONField && !s.hasAnyJSONTag {
			errs = append(errs, pathError{
				path: append(slices.Clip(qe.path), math.MaxInt),
				err:  newUnsupportedError(fmt.Errorf("unsupported struct `%s` without exported fields found", qe.t.String())),
			})
		}
	}

	for _, f := range dominantFieldsV2(fields) {
		err := e.jsonSafe(f.field.Type(), level+1, st)
		if f.options["omitzero"] || f.viaPtr {
			// The zero value of a field with the `omitzero` option is
			// omitted and fields promoted from a nil embedded pointer are
			// skipped without encoding them.
			err = demoteUnsupported(err)
		}
		if f.options["format"] && isTimeDuration(f.field.Type()) {
			// encoding/json/v2 of Go 1.26 supports time.Duration with
			// the `format` option.
			err = demoteUnsupported(err)
		}
		if err != nil {
			errs = append(errs, pathError{path: f.path, err: err})
		}
	}

	// Report the first error in the order of the fields, but an unsupported
	// type takes precedence over an unsafe type or struct tag.
	slices.SortStableFunc(errs, func(a, b pathError) int {
		return slices.Compare(a.path, b.path)
	})
	sortedErrs := make([]error, 0, len(errs))
	for _, pe := range errs {
		sortedErrs = append(sortedErrs, pe.err)
	}
	if err := combineErrors(sortedErrs...); err != nil {
		return err
	}
	if e.reportNoExported && level == 0 && exported == 0 {
		return newNoexportedError(fmt.Errorf("struct does not export any field"))
	}
	// encoding/json/v2 rejects structs, which have fields, but neither a
	// field considered for encoding nor any json struct tag.
	if ut.NumFields() > 0 && !root.hasAnyJSONField && !root.hasAnyJSONTag {
		return newUnsupportedError(fmt.Errorf("unsupported struct `%s` without exported fields found", t.String()))
	}
	return nil
}

// dominantFieldsV2 returns the fields, which are encoded by encoding/json/v2.
// If multiple fields have the same JSON object name, only the field, which
// exists alone at the shallowest depth or is uniquely tagged with a JSON
// name, is encoded. Otherwise, none of these fields is encoded.
func dominantFieldsV2(fields []fieldV2) []fieldV2 {
	byName := map[string][]fieldV2{}
	var names []string
	var result []fieldV2
	for _, f := range fields {
		if f.weak {
			result = append(result, f)
			continue
		}
		if _, ok := byName[f.name]; !ok {
			names = append(names, f.name)
		}
		byName[f.name] = append(byName[f.name], f)
	}
	for _, name := range names {
		fs := byName[name]
		slices.SortStableFunc(fs, func(x, y fieldV2) int {
			return cmp.Or(
				cmp.Compare(len(x.path), len(y.path)),
				boolCompare(!x.hasName, !y.hasName))
		})
		if len(fs) == 1 || len(fs[0].path) != len(fs[1].path) || fs[0].hasName != fs[1].hasName {
			result = append(result, fs[0])
		}
	}
	return result
}

func boolCompare(x, y bool) int {
	switch {
	case x == y:
		return 0
	case !x:
		return -1
	default:
		return +1
	}
}

// nameKind records, which kind of fields use a JSON object name in a struct.
type nameKind uint8

const (
	regularName nameKind = 1 << iota // used by a field encoded the same way by all versions of encoding/json/v2.
	weakName                         // used by a field with the `embed`, `inline` or `unknown` option.
)

// structState holds the state collected while checking the fields of a
// struct according to the rules of encoding/json/v2.
type structState struct {
	hasAnyJSONTag   bool                // any field has a json struct tag (including `json:"-"`).
	hasAnyJSONField bool                // any field is considered for encoding.
	names           map[string]nameKind // JSON object names of the direct fields.
}

// checkFieldV2 checks the definition of the i-th field of the struct ut (the
// underlying type of t) according to the rules of encoding/json/v2.
//
// If the field is a struct (or a pointer to a struct), whose fields are
// promoted (embedded struct without JSON name), its type is returned as
// embedded. Otherwise, if the field is encoded as JSON object member, it is
// returned as f. The returned error is related to the definition of the field,
// not to its type.
func checkFieldV2(ut *types.Struct, i int, t types.Type, s *structState) (f *fieldV2, embedded types.Type, err error) {
	field := ut.Field(i)
	tag, hasTag := reflect.StructTag(ut.Tag(i)).Lookup("json")
	s.hasAnyJSONTag = s.hasAnyJSONTag || hasTag
	if tag == "-" {
		// Fields omitted in json can be ignored
		return nil, nil, nil
	}
	if !field.Exported() && !field.Embedded() {
		if hasTag {
			// encoding/json/v2 rejects json struct tags (other than
			// `json:"-"`) on unexported fields.
			return nil, nil, newUnsupportedError(fmt.Errorf("unsupported struct tag on unexported field `%s` found", field.Name()))
		}
		// Otherwise, unexported fields are ignored.
		return nil, nil, nil
	}
	s.hasAnyJSONField = true
	if !field.Exported() && !isStruct(field.Type()) {
		// encoding/json/v2 rejects embedded fields of unexported non-struct
		// types (with or without a JSON name). The exported fields of
		// embedded structs of unexported types are encoded.
		return nil, nil, newUnsupportedError(fmt.Errorf("unsupported embedded field `%s` of unexported non-struct type found", field.Name()))
	}

	name, options := parseJSONTag(tag)
	seenOptions, err := checkJSONTagV2(t, field, name, options, s)
	if field.Embedded() && name == "" && isStruct(field.Type()) {
		return nil, field.Type(), err
	}
	f = &fieldV2{
		field:   field,
		name:    name,
		hasName: name != "",
		// Single-quoted names are only supported by encoding/json/v2 of
		// Go 1.26.
		weak:    seenOptions["embed"] || seenOptions["inline"] || seenOptions["unknown"] || strings.HasPrefix(name, "'"),
		options: seenOptions,
	}
	if f.name == "" {
		f.name = field.Name()
	}
	return f, nil, err
}

// checkJSONTagV2 checks the json struct tag of the field, which is
// considered for encoding, according to the rules of encoding/json/v2. It
// returns the set of options.
//
// Errors for struct tags, which are rejected by all supported versions of
// encoding/json/v2, are unsupported errors (the encoding of the struct always
// fails). Struct tags, which are only rejected by some versions, are unsafe.
func checkJSONTagV2(t types.Type, field *types.Var, name string, options []string, s *structState) (map[string]bool, error) {
	var errs []error
	switch {
	case strings.HasPrefix(name, "'"):
		// Single-quoted names are only supported by encoding/json/v2 of
		// Go 1.26.
		errs = append(errs, newVersionError(go127, fmt.Errorf("unsafe JSON object name `%s` on field `%s` found", name, field.Name())))
	case strings.ContainsAny(name, "\\'\"`"):
		// encoding/json/v2 rejects JSON object names containing a backslash
		// or quote characters.
		errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported JSON object name `%s` on field `%s` found", name, field.Name())))
	}
	if !utf8.ValidString(name) {
		errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported JSON object name with invalid UTF-8 on field `%s` found", field.Name())))
	}
	// encoding/json/v2 of Go 1.26 (GOEXPERIMENT=jsonv2) rejects the JSON
	// object name `-` (e.g. `json:"-,omitempty"`).
	if name == "-" {
		errs = append(errs, newVersionError(go126, fmt.Errorf("unsafe JSON object name `-` on field `%s` found", field.Name())))
	}
	// encoding/json/v2 rejects embedded fields of non-struct types
	// (e.g. named basic types, slices or interfaces) without a JSON name.
	if field.Embedded() && name == "" && !isStruct(field.Type()) {
		errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported embedded field `%s` of non-struct type without JSON name found", field.Name())))
	}
	// Embedded struct without JSON name.
	implicitEmbed := field.Embedded() && name == "" && isStruct(field.Type())
	for _, option := range options {
		// Since Go 1.27, the `format` option is rejected for all types,
		// because support for it was removed (see
		// https://go.dev/issue/79071).
		if strings.HasPrefix(option, "format:") {
			errs = append(errs, newVersionError(go127, fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())))
		}
		// encoding/json/v2 of Go 1.27 rejects the `string` option on
		// non-numeric types.
		if option == "string" && !isNumeric(field.Type()) {
			errs = append(errs, newVersionError(go127, fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())))
		}
	}
	seenOptions, err := checkJSONTagOptionsV2(field.Name(), options)
	if strings.HasPrefix(name, "'") {
		// encoding/json/v2 of Go 1.26 parses single-quoted names, which may
		// contain commas (e.g. `json:"'a,b'"`), the options are therefore
		// parsed differently by the versions of encoding/json/v2.
		err = demoteUnsupported(err)
	}
	errs = append(errs, err)
	// Fields embedded by encoding/json/v2: embedded struct fields without
	// a JSON name and fields with the `embed` option. Fields with the
	// `inline` or `unknown` options are embedded by encoding/json/v2 of
	// Go 1.26 (GOEXPERIMENT=jsonv2), but are regular fields as of Go 1.27.
	switch {
	case implicitEmbed || seenOptions["embed"] || seenOptions["inline"] || seenOptions["unknown"]:
		if err := checkJSONEmbedV2(field.Type(), name, seenOptions); err != nil {
			err = fmt.Errorf("embedded field `%s` found: %v", field.Name(), err)
			// Embedded structs without JSON name are embedded by all
			// versions of encoding/json/v2, unless the version specific
			// options `inline` or `unknown` are given. Otherwise, the
			// field is only embedded by the version, which knows the
			// option (`embed` as of Go 1.27, `inline` and `unknown` for
			// Go 1.26).
			inlineOrUnknown := seenOptions["inline"] || seenOptions["unknown"]
			switch {
			case implicitEmbed && !inlineOrUnknown:
				errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported %w", err)))
			case !implicitEmbed && seenOptions["embed"] && !inlineOrUnknown:
				errs = append(errs, newVersionError(go127, fmt.Errorf("unsafe %w", err)))
			case !implicitEmbed && !seenOptions["embed"]:
				errs = append(errs, newVersionError(go126, fmt.Errorf("unsafe %w", err)))
			default:
				errs = append(errs, fmt.Errorf("unsafe %w", err))
			}
		}
	case !field.Exported():
		// Embedded struct of an unexported type with a JSON name,
		// encoded as regular field. Methods of the unexported type
		// can not be called.
		if unexportedMethodsV2(field.Type(), seenOptions["omitzero"]) {
			errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported field `%s` of unexported type with methods for JSON encoding found", field.Name())))
		}
	}
	// encoding/json/v2 rejects structs, where two direct fields use the
	// same JSON object name. Conflicts with fields promoted from embedded
	// structs (without an explicit name) are not an error, they are
	// resolved by the Go embedding rules. Fields with the `embed` option
	// are regular fields for encoding/json/v2 of Go 1.26 (the option is
	// unknown) and fields with the `inline` or `unknown` options are
	// regular fields as of Go 1.27, conflicts with these fields are
	// therefore only rejected by one of the versions.
	if !implicitEmbed {
		if name == "" {
			name = field.Name()
		}
		kind := regularName
		if seenOptions["embed"] || seenOptions["inline"] || seenOptions["unknown"] || strings.HasPrefix(name, "'") {
			kind = weakName
		}
		switch prev := s.names[name]; {
		case kind == regularName && prev&regularName != 0:
			errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported duplicate JSON object name `%s` in struct `%s` found", name, t.String())))
		case prev != 0:
			errs = append(errs, fmt.Errorf("unsafe duplicate JSON object name `%s` in struct `%s` found", name, t.String()))
		}
		s.names[name] |= kind
	}
	return seenOptions, combineErrors(errs...)
}

// combineErrors returns the first unsupported error of errs. Otherwise, if
// the struct definition errors of errs together are rejected by all versions
// of encoding/json/v2, an unsupported error is returned. Otherwise, the first
// non-nil error is returned.
func combineErrors(errs ...error) error {
	var first error
	var versions goVersions
	for _, err := range errs {
		if err == nil {
			continue
		}
		if isUnsupported(err) {
			return err
		}
		if ve, ok := err.(versionError); ok {
			versions |= ve.versions
		}
		if first == nil {
			first = err
		}
	}
	switch {
	case first == nil:
		return nil
	case versions == allVersions:
		return newUnsupportedError(fmt.Errorf("unsupported struct definition (rejected by all versions of encoding/json/v2) found: %v", first))
	case versions != 0:
		return newVersionError(versions, first)
	}
	return first
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

// isUnsupported reports, if err is an unsupported error.
func isUnsupported(err error) bool {
	_, ok := err.(unsupported)
	return ok
}

// demotedError is an unsupported error, which has been turned into an unsafe
// error.
type demotedError struct {
	err error
}

func (d demotedError) Error() string {
	return d.err.Error()
}

// isDemoted reports, if err is an unsupported error, which has been turned
// into an unsafe error.
func isDemoted(err error) bool {
	_, ok := err.(demotedError)
	return ok
}

// demoteUnsupported turns an unsupported error into an unsafe error. It is
// used, where encoding only fails for some values of an unsupported type
// (e.g. a non-empty slice of channels). The versions of a version specific
// struct definition error are removed as well.
func demoteUnsupported(err error) error {
	if isUnsupported(err) {
		return demotedError{err: err}
	}
	if ve, ok := err.(versionError); ok {
		return ve.err
	}
	return err
}

// goVersions is a set of versions of encoding/json/v2.
type goVersions uint8

const (
	go126       goVersions = 1 << iota // encoding/json/v2 of Go 1.26 (GOEXPERIMENT=jsonv2).
	go127                              // encoding/json/v2 as of Go 1.27.
	allVersions = go126 | go127
)

// versionError is an unsafe error for a struct definition, which is rejected
// by the given versions of encoding/json/v2 (for all values).
type versionError struct {
	err      error
	versions goVersions
}

func (v versionError) Error() string {
	return v.err.Error()
}

// newVersionError returns an error for a struct definition, which is rejected
// by the given versions of encoding/json/v2.
func newVersionError(versions goVersions, err error) error {
	if ve, ok := err.(versionError); ok {
		err = ve.err
	}
	return versionError{err: err, versions: versions}
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
// parseJSONTag) of the field fieldName according to the rules of
// encoding/json/v2. It returns the set of options. Unknown options are ignored
// by encoding/json/v2, unless they appear more than once.
//
// If an option is rejected by all supported versions of encoding/json/v2, an
// unsupported error is returned, if it is only rejected by some versions, a
// version specific (unsafe) error is returned.
func checkJSONTagOptionsV2(fieldName string, options []string) (map[string]bool, error) {
	seen := map[string]bool{}
	var errs []error
	report := func(option string, versions goVersions, reason string) {
		if versions == allVersions {
			errs = append(errs, newUnsupportedError(fmt.Errorf("unsupported struct tag option `%s` on field `%s` found: %s", option, fieldName, reason)))
			return
		}
		errs = append(errs, newVersionError(versions, fmt.Errorf("unsafe struct tag option `%s` on field `%s` found: %s", option, fieldName, reason)))
	}
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
			// E.g. trailing comma, empty or quoted option. encoding/json/v2
			// of Go 1.26 accepts single-quoted options, unless they only
			// consist of letters and digits (e.g. `'omitempty'`).
			versions := allVersions
			if strings.HasPrefix(option, "'") && !isQuotedIdent(option) {
				versions = go127
			}
			report(option, versions, "malformed option")
			continue
		case "case":
			if rest != ":ignore" && rest != ":strict" {
				report(option, allVersions, "invalid value, only `case:ignore` and `case:strict` are supported")
			}
		case "format":
			// The `format` option with a value is reported by the caller.
			// A missing value is rejected by all versions.
			if !strings.HasPrefix(rest, ":") {
				report(option, allVersions, "missing value for `format`")
			}
		default:
			if rest != "" {
				report(option, allVersions, "malformed option")
			}
			// encoding/json/v2 rejects options resembling a known option
			// (e.g. `omitEmpty` or `omit_empty`). The options `embed`,
			// `inline` and `unknown` are only known by some versions.
			normalized := strings.ReplaceAll(strings.ToLower(ident), "_", "")
			if normalized != ident && slices.Contains(jsonTagOptionsV2, normalized) {
				versions := allVersions
				switch normalized {
				case "embed":
					versions = go127
				case "inline", "unknown":
					versions = go126
				}
				report(option, versions, fmt.Sprintf("invalid spelling of `%s`", normalized))
			}
		}

		if seen[ident] {
			report(option, allVersions, "duplicate option")
		}
		seen[ident] = true
	}
	return seen, combineErrors(errs...)
}

// isQuotedIdent reports, if option is a single-quoted string, which only
// consists of letters and digits (including the empty string).
func isQuotedIdent(option string) bool {
	inner, ok := strings.CutPrefix(option, "'")
	if !ok {
		return false
	}
	inner, ok = strings.CutSuffix(inner, "'")
	return ok && strings.TrimFunc(inner, isLetterOrDigit) == ""
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

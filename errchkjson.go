// Package errchkjson defines an Analyzer that finds places, where it is
// safe to omit checking the error returned from json.Marshal.
package errchkjson

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

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
			if n == nil {
				return true
			}

			// if the error is returned, it is the caller's responsibility to check
			// the return value.
			if _, ok := n.(*ast.ReturnStmt); ok {
				return false
			}

			ce, ok := n.(*ast.CallExpr)
			if ok {
				fn, _ := typeutil.Callee(pass.TypesInfo, ce).(*types.Func)
				if fn == nil {
					return true
				}

				if jf, ok := jsonFuncs[fn.FullName()]; ok {
					e.handleJSONMarshal(pass, ce, fn.FullName(), jf, blankIdentifier)
				} else {
					e.inspectArgs(pass, ce.Args)
				}
				return false
			}

			as, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			ce, ok = as.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}

			fn, _ := typeutil.Callee(pass.TypesInfo, ce).(*types.Func)
			if fn == nil {
				return true
			}

			jf, ok := jsonFuncs[fn.FullName()]
			if !ok || len(as.Lhs) <= jf.errIdx {
				return true
			}
			e.handleJSONMarshal(pass, ce, fn.FullName(), jf, evaluateMarshalErrorTarget(as.Lhs[jf.errIdx]))
			return false
		})
	}

	return nil, nil
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
}

var jsonFuncs = map[string]jsonFunc{
	"encoding/json.Marshal":           {argIdx: 0, errIdx: 1},
	"encoding/json.MarshalIndent":     {argIdx: 0, errIdx: 1},
	"(*encoding/json.Encoder).Encode": {argIdx: 0, errIdx: 0, alwaysOmitSafe: true},
	"encoding/json/v2.Marshal":        {argIdx: 0, errIdx: 1, v2: true},
	"encoding/json/v2.MarshalWrite":   {argIdx: 1, errIdx: 0, alwaysOmitSafe: true, v2: true},
	"encoding/json/v2.MarshalEncode":  {argIdx: 1, errIdx: 0, alwaysOmitSafe: true, v2: true},
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

		err = e.jsonSafe(t, 0, map[types.Type]struct{}{}, jf.v2)
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
func (e *errchkjson) jsonSafe(t types.Type, level int, seenTypes map[types.Type]struct{}, v2 bool) error {
	if _, ok := seenTypes[t]; ok {
		return nil
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
		err := e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if err != nil {
			return err
		}
		return nil

	case *types.Slice:
		err := e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if err != nil {
			return err
		}
		return nil

	case *types.Struct:
		seenTypes[t] = struct{}{}
		exported := 0
		hasAnyJSONTag := false   // any field has a json struct tag (including `json:"-"`).
		hasAnyJSONField := false // any field is considered for encoding.
		names := map[string]struct{}{}
		for i := 0; i < ut.NumFields(); i++ {
			field := ut.Field(i)
			tag, hasTag := reflect.StructTag(ut.Tag(i)).Lookup("json")
			hasAnyJSONTag = hasAnyJSONTag || hasTag
			if tag == "-" {
				// Fields omitted in json can be ignored
				continue
			}
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
					hasAnyJSONField = true
				} else if v2 && hasTag {
					// encoding/json/v2 rejects json struct tags (other than
					// `json:"-"`) on unexported fields.
					return fmt.Errorf("unsafe struct tag on unexported field `%s` found", field.Name())
				}
				// Otherwise, unexported fields are ignored.
				continue
			}
			hasAnyJSONField = true
			name, options := parseJSONTag(tag)
			// encoding/json/v2 rejects JSON object names containing a backslash
			// or quote characters (including single-quoted names).
			if v2 && strings.ContainsAny(name, "\\'\"`") {
				return fmt.Errorf("unsafe JSON object name `%s` on field `%s` found", name, field.Name())
			}
			// encoding/json/v2 rejects embedded fields of non-struct types
			// (e.g. named basic types, slices or interfaces) without a JSON name.
			if v2 && field.Embedded() && name == "" && !isStruct(field.Type()) {
				return fmt.Errorf("unsafe embedded field `%s` of non-struct type without JSON name found", field.Name())
			}
			for _, option := range options {
				// Since Go 1.27, the `format` option is rejected by encoding/json
				// and encoding/json/v2 for all types, because support for it was
				// removed (see https://go.dev/issue/79071).
				if strings.HasPrefix(option, "format:") {
					return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
				}
				// encoding/json/v2 rejects the `string` option on non-numeric types.
				if v2 && option == "string" && !isNumeric(field.Type()) {
					return fmt.Errorf("unsafe struct tag option `%s` on field `%s` found", option, field.Name())
				}
			}
			// encoding/json/v2 rejects structs, where two direct fields use the
			// same JSON object name. Conflicts with fields promoted from embedded
			// structs (without an explicit name) are not an error, they are
			// resolved by the Go embedding rules.
			if v2 && (!field.Embedded() || name != "") {
				if name == "" {
					name = field.Name()
				}
				if _, ok := names[name]; ok {
					return fmt.Errorf("unsafe duplicate JSON object name `%s` in struct `%s` found", name, t.String())
				}
				names[name] = struct{}{}
			}
			err := e.jsonSafe(field.Type(), level+1, seenTypes, v2)
			if err != nil {
				return err
			}
			if field.Exported() {
				exported++
			}
		}
		if e.reportNoExported && level == 0 && exported == 0 {
			return newNoexportedError(fmt.Errorf("struct does not export any field"))
		}
		// encoding/json/v2 rejects structs, which have fields, but neither a
		// field considered for encoding nor any json struct tag.
		if v2 && ut.NumFields() > 0 && !hasAnyJSONField && !hasAnyJSONTag {
			return fmt.Errorf("unsafe struct `%s` without exported fields found", t.String())
		}
		return nil

	case *types.Pointer:
		err := e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
		if err != nil {
			return err
		}
		return nil

	case *types.Map:
		err := jsonSafeMapKey(ut.Key(), v2)
		if err != nil {
			return err
		}
		err = e.jsonSafe(ut.Elem(), level+1, seenTypes, v2)
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
	default:
		// E.g. struct composed solely of basic types, that are comparable
		return newUnsupportedError(fmt.Errorf("unsupported type `%s` as map key found", t.String()))
	}
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
// float type.
func isNumeric(t types.Type) bool {
	for {
		p, ok := t.Underlying().(*types.Pointer)
		if !ok {
			break
		}
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
	obj, _, _ := types.LookupFieldOrMethod(t, false, nil, "MarshalJSONTo")
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
	return obj2.Pkg() != nil && obj2.Pkg().Path() == "encoding/json/jsontext" && obj2.Name() == "Encoder"
}

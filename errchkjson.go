// Package errchkjson defines an Analyzer that finds places, where it is
// save to omit checking the error returned from json.Marshal.
package errchkjson

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

type errchkjson struct {
	omitSafe bool // -omit-safe flag
}

// NewAnalyzer returns a new errchkjson analyzer.
func NewAnalyzer() *analysis.Analyzer {
	errchkjson := &errchkjson{}

	a := &analysis.Analyzer{
		Name: "errchkjson",
		Doc:  "Checks types passed to the json encoding functions. Reports unsupported types and reports occations, where the check for the returned error can be omitted.",
		Run:  errchkjson.run,
	}

	a.Flags.Init("errchkjson", flag.ExitOnError)
	a.Flags.BoolVar(&errchkjson.omitSafe, "omit-safe", false, "if omit-safe is true, checking of safe returns is omitted")

	return a
}

func (e *errchkjson) run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return true
			}

			ce, ok := n.(*ast.CallExpr)
			if ok && e.omitSafe {
				fn, _ := typeutil.Callee(pass.TypesInfo, ce).(*types.Func)
				if fn == nil {
					return true
				}

				switch fn.FullName() {
				case "encoding/json.Marshal", "encoding/json.MarshalIndent", "(*encoding/json.Encoder).Encode":
					pass.Reportf(n.Pos(), "Error return value of `%s` is not checked", fn.FullName())
					return false
				}
				return true
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

			switch fn.FullName() {
			case "encoding/json.Marshal", "encoding/json.MarshalIndent":
				e.handleJSONMarshal(pass, n, fn.FullName(), 1)
			case "(*encoding/json.Encoder).Encode":
				e.handleJSONMarshal(pass, n, fn.FullName(), 0)
			default:
				return true
			}
			return false
		})
	}

	return nil, nil
}

func (e *errchkjson) handleJSONMarshal(pass *analysis.Pass, n ast.Node, fnName string, errPos int) {
	as := n.(*ast.AssignStmt)
	ce := as.Rhs[0].(*ast.CallExpr)

	var blankIdentifier bool
	if errIdent, ok := as.Lhs[errPos].(*ast.Ident); ok {
		if errIdent.Name == "_" {
			blankIdentifier = true
		}
	}

	t := pass.TypesInfo.TypeOf(ce.Args[0])
	if t == nil {
		// Not sure, if this is at all possible
		if blankIdentifier {
			pass.Reportf(n.Pos(), "Type of argument to `%s` could not be evaluated and error return value is not checked", fnName)
		}
		return
	}

	if _, ok := t.(*types.Pointer); ok {
		t = t.(*types.Pointer).Elem()
	}

	err := jsonSafe(t)
	if err != nil {
		if _, ok := err.(unsupported); ok {
			pass.Reportf(n.Pos(), "`%s` for %v", fnName, err)
			return
		}
		if blankIdentifier {
			pass.Reportf(n.Pos(), "Error return value of `%s` is not checked: %v", fnName, err)
		}
	}
	if err == nil && !blankIdentifier && !e.omitSafe {
		pass.Reportf(n.Pos(), "Error return value of `%s` is checked but passed argument is safe", fnName)
	}
	// Report an error, if err for json.Marshal is not checked and save types are omitted
	if err == nil && blankIdentifier && e.omitSafe {
		pass.Reportf(n.Pos(), "Error return value of `%s` is not checked", fnName)
	}
}

const (
	allowedBasicTypes       = types.IsBoolean | types.IsInteger | types.IsString
	allowedMapKeyBasicTypes = types.IsInteger | types.IsString
	unsupportedBasicTypes   = types.IsComplex
)

func jsonSafe(t types.Type) error {
	if types.Implements(t, textMarshalerInterface()) {
		return fmt.Errorf("unsafe type `%s` found", t.String())
	}

	switch ut := t.Underlying().(type) {
	case *types.Basic:
		if ut.Info()&allowedBasicTypes > 0 { // bool, int-family, string
			if ut.Info()&types.IsString > 0 && t.String() == "encoding/json.Number" {
				return fmt.Errorf("unsafe type `%s` found", t.String())
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
		err := jsonSafe(ut.Elem())
		if err != nil {
			return err
		}
		return nil

	case *types.Slice:
		err := jsonSafe(ut.Elem())
		if err != nil {
			return err
		}
		return nil

	case *types.Struct:
		for i := 0; i < ut.NumFields(); i++ {
			if !ut.Field(i).Exported() {
				// Unexported fields can be ignored
				continue
			}
			if tag, ok := reflect.StructTag(ut.Tag(i)).Lookup("json"); ok {
				if tag == "-" {
					// Fields omitted in json can be ignored
					continue
				}
			}
			err := jsonSafe(ut.Field(i).Type())
			if err != nil {
				return err
			}
		}
		return nil

	case *types.Pointer:
		err := jsonSafe(ut.Elem())
		if err != nil {
			return err
		}
		return nil

	case *types.Map:
		err := jsonSafeMapKey(ut.Key())
		if err != nil {
			return err
		}
		err = jsonSafe(ut.Elem())
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

func jsonSafeMapKey(t types.Type) error {
	if types.Implements(t, textMarshalerInterface()) {
		return fmt.Errorf("unsafe type `%s` as map key found", t.String())
	}
	switch ut := t.Underlying().(type) {
	case *types.Basic:
		if ut.Info()&types.IsString > 0 && t.String() == "encoding/json.Number" {
			return fmt.Errorf("unsafe type `%s` as map key found", t.String())
		}
		if ut.Info()&allowedMapKeyBasicTypes > 0 { // bool, int-family, string
			return nil
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

// Construct *types.Interface for interface TextMarshaler
//     type TextMarshaler interface {
//         MarshalText() (text []byte, err error)
//     }
//
func textMarshalerInterface() *types.Interface {
	textMarshalerInterface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, nil, "MarshalText", types.NewSignature(
			nil, nil, types.NewTuple(
				types.NewVar(token.NoPos, nil, "text",
					types.NewSlice(
						types.Universe.Lookup("byte").Type())),
				types.NewVar(token.NoPos, nil, "err", types.Universe.Lookup("error").Type())),
			false)),
	}, nil)
	textMarshalerInterface.Complete()

	return textMarshalerInterface
}

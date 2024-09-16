package convert

import (
	"fmt"
	"go/token"
	"go/types"
	"strconv"

	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/gentypes/typmap"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type ConvertExpr struct {
	e ast.Expr
}

func NewConvertExpr(e ast.Expr) *ConvertExpr {
	return &ConvertExpr{e: e}
}

func (p *ConvertExpr) ToInt() (int, error) {
	v, ok := p.e.(*ast.BasicLit)
	if ok && v.Kind == ast.IntLit {
		return strconv.Atoi(v.Value)
	}
	return 0, fmt.Errorf("%v can't convert to int", p.e)
}

func (p *ConvertExpr) ToFloat(bitSize int) (float64, error) {
	v, ok := p.e.(*ast.BasicLit)
	if ok && v.Kind == ast.FloatLit {
		return strconv.ParseFloat(v.Value, bitSize)
	}
	return 0, fmt.Errorf("%v can't convert to float", v)
}

func (p *ConvertExpr) ToString() (string, error) {
	v, ok := p.e.(*ast.BasicLit)
	if ok && v.Kind == ast.StringLit {
		return v.Value, nil
	}
	return "", fmt.Errorf("%v can't convert to string", v)
}

func (p *ConvertExpr) ToChar() (int8, error) {
	v, ok := p.e.(*ast.BasicLit)
	if ok && v.Kind == ast.CharLit {
		iV, err := strconv.Atoi(v.Value)
		if err == nil {
			return int8(iV), nil
		}
	}
	return 0, fmt.Errorf("%v can't convert to char", p.e)
}

func (p *ConvertExpr) ToBuiltinType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	builtinType, ok := p.e.(*ast.BuiltinType)
	if ok {
		return m.FindBuiltinType(*builtinType)
	}
	return nil, fmt.Errorf("unsupported type %v", builtinType)
}

// - void* -> c.Pointer
// - Function pointers -> Function types (pointer removed)
// - Other cases -> Pointer to the base type
func (p *ConvertExpr) ToPointerType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	pointerType, ok := p.e.(*ast.PointerType)
	if ok {
		baseType, err := NewConvertExpr(pointerType.X).ToType(m)
		if err != nil {
			return nil, err
		}
		// void * -> c.Pointer
		// todo(zzy):alias visit the origin type unsafe.Pointer,c.Pointer is better
		if m.IsVoidType(baseType) {
			return m.CType("Pointer"), nil
		}
		if baseFuncType, ok := baseType.(*types.Signature); ok {
			return baseFuncType, nil
		}
		return types.NewPointer(baseType), nil
	}
	return nil, fmt.Errorf("%s", "is not a pointer type")
}

func (p *ConvertExpr) ToArrayType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	t, ok := p.e.(*ast.ArrayType)
	if !ok {
		return nil, fmt.Errorf("expr is not a ArrayType")
	}
	if t.Len == nil {
		// in param handle
		eltType, err := NewConvertExpr(t.Elt).ToType(m)
		if err != nil {
			return nil, err
		}
		// array in the parameter,ignore the len,convert as pointer
		return types.NewPointer(eltType), nil
	}
	if t.Len == nil {
		return nil, fmt.Errorf("%s", "unsupport field with array without length")
	}
	elemType, err := NewConvertExpr(t.Elt).ToType(m)
	if err != nil {
		return nil, err
	}
	len, err := NewConvertExpr(t.Len).ToInt()
	if err != nil {
		return nil, fmt.Errorf("%s", "can't determine the array length")
	}
	return types.NewArray(elemType, int64(len)), nil
}

func (p *ConvertExpr) ToType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	switch t := p.e.(type) {
	case *ast.BuiltinType:
		return p.ToBuiltinType(m)
	case *ast.PointerType:
		return p.ToPointerType(m)
	case *ast.ArrayType:
		return p.ToArrayType(m)
	default:
		return nil, fmt.Errorf("unexpected type %T", t)
	}
}

func (p *ConvertExpr) ToTuple(typesPacakge *types.Package, m *typmap.BuiltinTypeMap) (*types.Tuple, error) {
	typ, err := p.ToType(m)
	if err == nil && !m.IsVoidType(typ) {
		// in c havent multiple return
		return types.NewTuple(types.NewVar(token.NoPos, typesPacakge, "", typ)), nil
	}
	return types.NewTuple(), nil
}

func (p *ConvertExpr) ToSignature(typesPacakge *types.Package, m *typmap.BuiltinTypeMap) (*types.Signature, error) {
	funcType, ok := p.e.(*ast.FuncType)
	if ok {
		fieldList := NewConvertFieldList(funcType.Params)
		params, _ := fieldList.ToTuple(typesPacakge, m)
		ret := NewConvertExpr(funcType.Ret)
		results, _ := ret.ToTuple(typesPacakge, m)
		return types.NewSignatureType(nil, nil, nil, params, results, false), nil
	}
	return nil, fmt.Errorf("%s", "expr not a *ast.FuncType")
}

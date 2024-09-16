package convert

import (
	"fmt"
	"go/token"
	"go/types"
	"strconv"

	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/gentypes/typmap"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type Expr struct {
	ast.Expr
}

func NewExpr(e ast.Expr) *Expr {
	return &Expr{Expr: e}
}

func (p *Expr) ToInt() (int, error) {
	v, ok := p.Expr.(*ast.BasicLit)
	if ok && v.Kind == ast.IntLit {
		return strconv.Atoi(v.Value)
	}
	return 0, fmt.Errorf("%v can't convert to int", p.Expr)
}

func (p *Expr) ToFloat(bitSize int) (float64, error) {
	v, ok := p.Expr.(*ast.BasicLit)
	if ok && v.Kind == ast.FloatLit {
		return strconv.ParseFloat(v.Value, bitSize)
	}
	return 0, fmt.Errorf("%v can't convert to float", v)
}

func (p *Expr) ToString() (string, error) {
	v, ok := p.Expr.(*ast.BasicLit)
	if ok && v.Kind == ast.StringLit {
		return v.Value, nil
	}
	return "", fmt.Errorf("%v can't convert to string", v)
}

func (p *Expr) ToChar() (int8, error) {
	v, ok := p.Expr.(*ast.BasicLit)
	if ok && v.Kind == ast.CharLit {
		iV, err := strconv.Atoi(v.Value)
		if err == nil {
			return int8(iV), nil
		}
	}
	return 0, fmt.Errorf("%v can't convert to char", p.Expr)
}

func (p *Expr) ToBuiltinType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	builtinType, ok := p.Expr.(*ast.BuiltinType)
	if ok {
		return m.FindBuiltinType(*builtinType)
	}
	return nil, fmt.Errorf("unsupported type %v", builtinType)
}

// - void* -> c.Pointer
// - Function pointers -> Function types (pointer removed)
// - Other cases -> Pointer to the base type
func (p *Expr) ToPointerType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	pointerType, ok := p.Expr.(*ast.PointerType)
	if ok {
		baseType, err := NewExpr(pointerType.X).ToType(m)
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

func (p *Expr) ToType(m *typmap.BuiltinTypeMap) (types.Type, error) {
	switch t := p.Expr.(type) {
	case *ast.BuiltinType:
		return p.ToBuiltinType(m)
	case *ast.PointerType:
		return p.ToPointerType(m)
	case *ast.ArrayType:
		if t.Len == nil {
			eltType, err := NewExpr(t.Elt).ToType(m)
			if err != nil {
				return nil, err
			}
			// array in the parameter,ignore the len,convert as pointer
			return types.NewPointer(eltType), nil
		}
		if t.Len == nil {
			return nil, fmt.Errorf("%s", "unsupport field with array without length")
		}
		elemType, err := NewExpr(t.Elt).ToType(m)
		if err != nil {
			return nil, err
		}
		len, err := NewExpr(t.Len).ToInt()
		if err != nil {
			return nil, fmt.Errorf("%s", "can't determine the array length")
		}
		return types.NewArray(elemType, int64(len)), nil
		/*
			case *ast.FuncType:
				return p.toSignature(t)*/
	default:
		return nil, fmt.Errorf("%s", "unexpected type")
	}
}

func (p *Expr) ToTuple(typesPacakge *types.Package, m *typmap.BuiltinTypeMap) (*types.Tuple, error) {
	typ, err := p.ToType(m)
	if err == nil && !m.IsVoidType(typ) {
		// in c havent multiple return
		return types.NewTuple(types.NewVar(token.NoPos, typesPacakge, "", typ)), nil
	}
	return types.NewTuple(), nil
}

func (p *Expr) ToSignature(typesPacakge *types.Package, m *typmap.BuiltinTypeMap) (*types.Signature, error) {
	funcType, ok := p.Expr.(*ast.FuncType)
	if ok {
		fieldList := NewFieldList(funcType.Params)
		params, _ := fieldList.ToTuple(typesPacakge, m)
		ret := NewExpr(funcType.Ret)
		results, _ := ret.ToTuple(typesPacakge, m)
		return types.NewSignatureType(nil, nil, nil, params, results, false), nil
	}
	return nil, fmt.Errorf("%s", "expr not a *ast.FuncType")
}

package convert

import (
	"fmt"
	"strconv"

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

package convert

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/gentypes/typmap"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type ConvertField struct {
	f *ast.Field
}

func NewConvertField(field *ast.Field) *ConvertField {
	return &ConvertField{f: field}
}

func (p *ConvertField) ToVar(typesPackage *types.Package, m *typmap.BuiltinTypeMap) (*types.Var, error) {
	if p == nil || len(p.f.Names) <= 0 {
		return nil, fmt.Errorf("%s", "invalid input ToVar")
	}
	typ, err := NewConvertExpr(p.f.Type).ToType(m)
	if err != nil {
		return nil, err
	}
	return types.NewVar(token.NoPos, typesPackage, p.f.Names[0].Name, typ), nil
}

type ConvertFieldList struct {
	list *ast.FieldList
}

func NewConvertFieldList(fieldList *ast.FieldList) *ConvertFieldList {
	return &ConvertFieldList{list: fieldList}
}

func (p *ConvertFieldList) ToVars(typesPackage *types.Package, m *typmap.BuiltinTypeMap) ([]*types.Var, error) {
	var vars []*types.Var
	if p.list == nil || p.list.List == nil {
		return vars, nil
	}
	for _, field := range p.list.List {
		fieldVar, err := NewConvertField(field).ToVar(typesPackage, m)
		if err != nil {
			//todo handle field _Type=Variadic case
			continue
		}
		vars = append(vars, fieldVar)
	}
	return vars, nil
}

func (p *ConvertFieldList) ToTuple(typesPackage *types.Package, m *typmap.BuiltinTypeMap) (*types.Tuple, error) {
	if p.list == nil {
		return types.NewTuple(), nil
	}
	vars, err := p.ToVars(typesPackage, m)
	if err != nil {
		return types.NewTuple(), err
	}
	return types.NewTuple(vars...), nil
}

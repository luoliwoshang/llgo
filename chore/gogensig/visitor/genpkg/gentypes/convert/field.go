package convert

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/gentypes/typmap"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type Field struct {
	*ast.Field
}

func NewField(field *ast.Field) *Field {
	return &Field{Field: field}
}

func (p *Field) ToVar(typesPackage *types.Package, m *typmap.BuiltinTypeMap) (*types.Var, error) {
	if p == nil || len(p.Names) <= 0 {
		return nil, fmt.Errorf("%s", "invalid input ToVar")
	}
	typ, err := NewExpr(p.Type).ToType(m)
	if err != nil {
		return nil, err
	}
	return types.NewVar(token.NoPos, typesPackage, p.Names[0].Name, typ), nil
}

type FieldList struct {
	*ast.FieldList
}

func NewFieldList(fieldList *ast.FieldList) *FieldList {
	return &FieldList{FieldList: fieldList}
}

func (p *FieldList) ToVars(typesPackage *types.Package, m *typmap.BuiltinTypeMap) ([]*types.Var, error) {
	var vars []*types.Var
	if p.FieldList == nil || p.FieldList.List == nil {
		return vars, nil
	}
	for _, field := range p.List {
		fieldVar, err := NewField(field).ToVar(typesPackage, m)
		if err != nil {
			//todo handle field _Type=Variadic case
			continue
		}
		vars = append(vars, fieldVar)
	}
	return vars, nil
}

func (p *FieldList) ToTuple(typesPackage *types.Package, m *typmap.BuiltinTypeMap) (*types.Tuple, error) {
	if p.FieldList == nil {
		return types.NewTuple(), nil
	}
	vars, err := p.ToVars(typesPackage, m)
	if err != nil {
		return types.NewTuple(), err
	}
	return types.NewTuple(vars...), nil
}

package pkg

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/goplus/llgo/chore/llcppg/ast"

	"github.com/goplus/gogen"
)

type Package struct {
	p *gogen.Package
}

func NewPackage(pkgPath, name string, conf *gogen.Config) *Package {
	pkg := &Package{}
	pkg.p = gogen.NewPackage(pkgPath, name, conf)
	return pkg
}

func (p *Package) NewFuncDecl(funcDecl *ast.FuncDecl) error {
	sig, err := toSigniture(funcDecl.Type)
	if err != nil {
		return err
	}
	p.p.NewFuncDecl(token.NoPos, funcDecl.Name.Name, sig)
	return nil
}

func (p *Package) Write() error {
	return fmt.Errorf("%s", "todo Write package")
}

func toSigniture(funcType *ast.FuncType) (*types.Signature, error) {
	params := fieldListToParams(funcType.Params)
	return types.NewSignatureType(nil, nil, nil, params, nil, false), nil
}

func fieldListToParams(params *ast.FieldList) *types.Tuple {
	var vars []*types.Var
	if params != nil {
		for _, field := range params.List {
			vars = append(vars, fieldToVar(field))
		}
	}
	return types.NewTuple(vars...)
}

func fieldToVar(field *ast.Field) *types.Var {
	return types.NewVar(token.NoPos, nil, field.Names[0].Name, toType(field.Type))
}

func toType(expr ast.Expr) types.Type {
	switch t := expr.(type) {
	case *ast.BuiltinType:
		return toBuiltinType(t)
	}
	return nil
}

func toBuiltinType(typ *ast.BuiltinType) types.Type {
	return nil
}

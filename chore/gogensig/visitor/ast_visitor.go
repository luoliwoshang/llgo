package visitor

import (
	"fmt"

	"github.com/goplus/llgo/chore/llcppg/ast"
)

type DocVisitor interface {
	Visit(_Type string, node ast.Node)
	VisitFuncDecl(funcDecl *ast.FuncDecl)
	VisitDone(docPath string)
	VisitClass(className *ast.Ident, fields *ast.FieldList, typeDecl *ast.TypeDecl)
	VisitMethod(className *ast.Ident, method *ast.FuncDecl, typeDecl *ast.TypeDecl)
}

type BaseDocVisitor struct {
	DocVisitor
}

func NewBaseDocVisitor(Visitor DocVisitor) *BaseDocVisitor {
	return &BaseDocVisitor{DocVisitor: Visitor}
}

func (p *BaseDocVisitor) visitNode(decl ast.Node) {
	switch v := decl.(type) {
	case *ast.FuncDecl:
		p.visitFuncDecl(v)
	case *ast.TypeDecl:
		p.visitTypeDecl(v)
	default:
		panic(fmt.Errorf("todo visit %v", v))
	}
}

func (p *BaseDocVisitor) Visit(_Type string, node ast.Node) {
	switch v := node.(type) {
	case *ast.File:
		for _, decl := range v.Decls {
			p.visitNode(decl)
		}
	default:
		p.visitNode(v)
	}
}

func (p *BaseDocVisitor) visitFuncDecl(funcDecl *ast.FuncDecl) {
	p.VisitFuncDecl(funcDecl)
}

func (p *BaseDocVisitor) visitTypeDecl(typeDecl *ast.TypeDecl) {
	if typeDecl.Type.Tag == ast.Class {
		//todo new struct and convert fields
		p.visitClass(typeDecl.Name, typeDecl.Type.Fields, typeDecl)
		for _, method := range typeDecl.Type.Methods {
			p.visitMethod(typeDecl.Name, method, typeDecl)
		}
	}
}

func (p *BaseDocVisitor) visitClass(className *ast.Ident, fields *ast.FieldList, typeDecl *ast.TypeDecl) {
	p.VisitClass(className, fields, typeDecl)
}

func (p *BaseDocVisitor) visitMethod(className *ast.Ident, method *ast.FuncDecl, typeDecl *ast.TypeDecl) {
	p.VisitMethod(className, method, typeDecl)
}

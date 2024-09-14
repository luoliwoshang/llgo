package visitor

import (
	"fmt"

	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg"
	"github.com/goplus/llgo/chore/gogensig/visitor/symb"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type AstConvert struct {
	*BaseDocVisitor
	pkg *genpkg.Package
}

func NewAstConvert(pkgName string, symbFile string) *AstConvert {
	p := new(AstConvert)
	p.BaseDocVisitor = NewBaseDocVisitor(p)
	pkg := genpkg.NewPackage(".", pkgName, nil)
	p.pkg = pkg
	p.setupSymbleTableFile(symbFile)
	return p
}

func (p *AstConvert) setupSymbleTableFile(fileName string) error {
	symbTable, err := symb.NewSymbolTable(fileName)
	if err != nil {
		return err
	}
	p.pkg.SetSymbolTable(symbTable)
	return nil
}

func (p *AstConvert) VisitFuncDecl(funcDecl *ast.FuncDecl) {
	p.pkg.NewFuncDecl(funcDecl)
}

func (p *AstConvert) VisitClass(className *ast.Ident, fields *ast.FieldList, typeDecl *ast.TypeDecl) {
	fmt.Printf("visit class %s\n", className.Name)
}

func (p *AstConvert) VisitMethod(className *ast.Ident, method *ast.FuncDecl, typeDecl *ast.TypeDecl) {
	fmt.Printf("visit method %s of %s\n", method.Name.Name, className.Name)
}

func (p *AstConvert) VisitDone(docPath string) {
	p.pkg.Write(docPath)
}

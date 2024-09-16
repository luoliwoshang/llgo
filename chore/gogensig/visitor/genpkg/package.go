package genpkg

import (
	"bytes"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/gentypes/convert"
	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg/gentypes/typmap"
	"github.com/goplus/llgo/chore/gogensig/visitor/symb"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type Package struct {
	name      string
	p         *gogen.Package
	cvt       *convert.TypeConv
	typeBlock *gogen.TypeDefs // type decls block.
}

func NewPackage(pkgPath, name string, conf *gogen.Config) *Package {
	p := &Package{
		p: gogen.NewPackage(pkgPath, name, conf),
	}
	clib := p.p.Import("github.com/goplus/llgo/c")
	typeMap := typmap.NewBuiltinTypeMap(clib)
	p.cvt = convert.NewConv(p.p.Types, typeMap)
	p.name = name
	return p
}

func (p *Package) SetSymbolTable(symbolTable *symb.SymbolTable) {
	p.cvt.SetSymbolTable(symbolTable)
}

func (p *Package) getTypeBlock() *gogen.TypeDefs {
	if p.typeBlock == nil {
		p.typeBlock = p.p.NewTypeDefs()
	}
	return p.typeBlock
}

func (p *Package) NewFuncDecl(funcDecl *ast.FuncDecl) error {
	// todo(zzy) accept the name of llcppg.symb.json
	sig := p.cvt.ToSignature(funcDecl.Type)
	goFuncName := toGoFuncName(funcDecl.Name.Name)
	decl := p.p.NewFuncDecl(token.NoPos, goFuncName, sig)
	decl.SetComments(p.p, NewFuncDocComments(funcDecl.Name.Name, goFuncName))
	return nil
}

func (p *Package) NewTypeDecl(typeDecl *ast.TypeDecl) error {
	decl := p.getTypeBlock().NewType(typeDecl.Name.Name)
	structType := p.cvt.RecordTypeToStruct(typeDecl.Type)
	decl.InitType(p.p, structType)
	return nil
}

func (p *Package) NewTypedefDecl(typedefDecl *ast.TypedefDecl) error {
	decl := p.getTypeBlock().NewType(typedefDecl.Name.Name)
	typ := p.ToType(typedefDecl.Type)
	decl.InitType(p.p, typ)
	return nil
}

// Convert ast.Expr to types.Type
func (p *Package) ToType(expr ast.Expr) types.Type {
	return p.cvt.ToType(expr)
}

func (p *Package) NewEnumTypeDecl(enumTypeDecl *ast.EnumTypeDecl) {
	if len(enumTypeDecl.Type.Items) > 0 {
		for _, item := range enumTypeDecl.Type.Items {
			name := toTitle(enumTypeDecl.Name.Name) + "_" + item.Name.Name
			val, err := convert.Expr(item.Value).ToInt()
			if err != nil {
				continue
			}
			p.p.CB().NewConstStart(types.Typ[types.Int], name).Val(val).EndInit(1)
		}
	}
}

func (p *Package) Write(docPath string) error {
	_, fileName := filepath.Split(docPath)
	dir, err := p.makePackageDir("")
	if err != nil {
		return err
	}
	ext := filepath.Ext(fileName)
	if len(ext) > 0 {
		fileName = strings.TrimSuffix(fileName, ext)
	}
	if len(fileName) <= 0 {
		fileName = "temp"
	}
	fileName = fileName + ".go"
	p.p.WriteFile(filepath.Join(dir, fileName))
	return nil
}

func (p *Package) WriteToBuffer(buf *bytes.Buffer) error {
	return p.p.WriteTo(buf)
}

func (p *Package) makePackageDir(dir string) (string, error) {
	if len(dir) <= 0 {
		dir = "."
	}
	curDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(curDir, p.name)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return "", err
	}
	return path, nil
}

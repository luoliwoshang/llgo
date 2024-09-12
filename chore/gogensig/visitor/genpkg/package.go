package genpkg

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/llgo/chore/llcppg/ast"

	"github.com/goplus/gogen"
)

type Package struct {
	name string
	p    *gogen.Package
}

func NewPackage(pkgPath, name string, conf *gogen.Config) *Package {
	pkg := &Package{name: name}
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

func (p *Package) Write(curName string) error {
	fileDir, fileName := filepath.Split(curName)
	dir, err := p.makePackageDir(fileDir)
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

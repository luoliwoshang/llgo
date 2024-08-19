package main

import (
	"fmt"
	"unsafe"

	"github.com/goplus/llgo/c"
	"github.com/goplus/llgo/c/clang"
	"github.com/goplus/llgo/chore/_xtool/llcppsigfetch/parse"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

func main() {
	TestBuiltinType()
}
func TestBuiltinType() {
	tests := []struct {
		name     string
		typ      clang.Type
		expected ast.BuiltinType
	}{
		{"DefaultComplex", mockComplexType(0), ast.BuiltinType{Kind: ast.Complex}},
	}
	converter := &parse.Converter{}
	converter.Convert()
	for _, bt := range tests {
		if bt.typ.Kind == clang.TypeComplex {

			c.Printf(c.Str("TestBuiltinType data %p %p\n"), bt.typ.Data[0], bt.typ.Data[1]) // 与 mockComplexType data %p %p 一致
			c.Printf(c.Str("TestBuiltinType kind %d\n"), bt.typ.Kind)
			c.Printf(c.Str("TestBuiltinType element kind %d\n"), bt.typ.ElementType().Kind) // 空指针

		}
		res := converter.ProcessBuiltinType(bt.typ)
		if res.Kind != bt.expected.Kind {
			fmt.Printf("%s Kind mismatch:got %d want %d, \n", bt.name, res.Kind, bt.expected.Kind)
		}
		if res.Flags != bt.expected.Flags {
			fmt.Printf("%s Flags mismatch:got %d,want %d\n", bt.name, res.Flags, bt.expected.Flags)
		}
		fmt.Printf("%s:flags:%d kind:%d\n", bt.name, res.Flags, res.Kind)
	}
}

type complex struct {
	typ clang.Type
}

func visit(cursor, parent clang.Cursor, clientData unsafe.Pointer) clang.ChildVisitResult {
	typ := (*complex)(clientData)
	if cursor.Kind == clang.CursorVarDecl && cursor.Type().Kind == clang.TypeComplex {
		typ.typ = cursor.Type()
	}
	return clang.ChildVisit_Continue
}

// mock complex type, this type cannot be directly created in Go
func mockComplexType(flag ast.TypeFlag) clang.Type {
	var typeStr string
	if flag&(ast.Long|ast.Double) == (ast.Long | ast.Double) {
		typeStr = "long double"
	} else if flag&ast.Double != 0 {
		typeStr = "double"
	} else {
		typeStr = "float"
	}

	code := fmt.Sprintf("#include <complex.h>\n%s complex z;", typeStr)
	// println(code)
	// must include complex.h & specify the language is c
	// code := "#include <complex.h>\ndouble complex z;"
	index, unit, err := parse.CreateTranslationUnit(&parse.Config{
		File: code,
		Temp: true,
		Args: []string{"-x", "c", "-std=c99"},
	})
	if err != nil {
		panic(err)
	}

	defer unit.Dispose()
	defer index.Dispose()

	cursor := unit.Cursor()
	complex := &complex{}
	clang.VisitChildren(cursor, visit, unsafe.Pointer(complex))

	c.Printf(c.Str("mockComplexType kind %d\n"), complex.typ.Kind)                            // 这里可以正常输出
	c.Printf(c.Str("mockComplexType data %p %p\n"), complex.typ.Data[0], complex.typ.Data[1]) //
	c.Printf(c.Str("mockComplexType element kind %d\n"), complex.typ.ElementType().Kind)      // 这里可以正常输出

	return complex.typ
}

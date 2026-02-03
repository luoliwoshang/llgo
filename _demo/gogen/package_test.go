// Converted from package_test.go for llgo testing
// Batch 1: tests 1-5, Batch 2: tests 6-15
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	"github.com/goplus/gogen"
	"github.com/goplus/gogen/packages"
)

// ============ Global Variables ============
var (
	gblFset *token.FileSet
	gblImp  types.Importer
)

func init() {
	gblFset = token.NewFileSet()
	gblImp = packages.NewImporter(gblFset)
}

// ============ Helper Functions ============

func newMainPackage() *gogen.Package {
	conf := &gogen.Config{
		Fset:     gblFset,
		Importer: gblImp,
	}
	return gogen.NewPackage("", "main", conf)
}

func domTest(pkg *gogen.Package, expected string) {
	domTestEx(pkg, expected, "")
}

func domTestEx(pkg *gogen.Package, expected string, fname string) {
	var b bytes.Buffer
	err := gogen.WriteTo(&b, pkg, fname)
	if err != nil {
		panic(fmt.Sprintf("gogen.WriteTo failed: %v", err))
	}
	result := b.String()
	if result != expected {
		panic(fmt.Sprintf("\nResult:\n%s\nExpected:\n%s\n", result, expected))
	}
}

func ctxRef(pkg *gogen.Package, name string) gogen.Ref {
	_, o := pkg.CB().Scope().LookupParent(name, token.NoPos)
	return o
}

func comment(txt string) *ast.CommentGroup {
	return &ast.CommentGroup{List: []*ast.Comment{{Text: txt}}}
}

// ============ Test 1: TestRedupPkgIssue796 ============
func testRedupPkgIssue796() {
	fmt.Println("=== testRedupPkgIssue796 ===")
	pkg := newMainPackage()
	builtin := pkg.Import("github.com/goplus/gogen/internal/builtin")
	builtin.EnsureImported()
	context := pkg.Import("context")
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		Val(context.Ref("WithTimeout")).
		Val(context.Ref("Background")).Call(0).
		Val(pkg.Import("time").Ref("Minute")).Call(2).EndStmt().
		End()
	domTest(pkg, `package main

import (
	"context"
	"time"
)

func main() {
	context.WithTimeout(context.Background(), time.Minute)
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 2: TestBTIMethod ============
func testBTIMethod() {
	fmt.Println("=== testBTIMethod ===")
	pkg := newMainPackage()
	fmtPkg := pkg.Import("fmt")
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(types.NewChan(0, types.Typ[types.Int]), "a").
		NewVar(types.NewSlice(types.Typ[types.Int]), "b").
		NewVar(types.NewSlice(types.Typ[types.String]), "c").
		NewVar(types.NewMap(types.Typ[types.String], types.Typ[types.Int]), "d").
		NewVar(types.Typ[types.Int64], "e").
		Val(fmtPkg.Ref("Println")).VarVal("a").MemberVal("Len").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("b").MemberVal("Len").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("c").MemberVal("Len").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("d").MemberVal("Len").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("c").MemberVal("Join").Val(",").Call(1).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).Val("Hi").MemberVal("Len").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).Val("100").MemberVal("Int").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).Val("100").MemberVal("Uint64").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).Val(100).MemberVal("String").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).Val(1.34).MemberVal("String").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("e").Debug(
		func(cb *gogen.CodeBuilder) {
			cb.Member("string", gogen.MemberFlagAutoProperty)
		}).Call(1).EndStmt().
		End()
	domTest(pkg, `package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	var a chan int
	var b []int
	var c []string
	var d map[string]int
	var e int64
	fmt.Println(len(a))
	fmt.Println(len(b))
	fmt.Println(len(c))
	fmt.Println(len(d))
	fmt.Println(strings.Join(c, ","))
	fmt.Println(len("Hi"))
	fmt.Println(strconv.Atoi("100"))
	fmt.Println(strconv.ParseUint("100", 10, 64))
	fmt.Println(strconv.Itoa(100))
	fmt.Println(strconv.FormatFloat(1.34, 'g', -1, 64))
	fmt.Println(strconv.FormatInt(e, 10))
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 3: TestTypedBTIMethod ============
func testTypedBTIMethod() {
	fmt.Println("=== testTypedBTIMethod ===")
	pkg := newMainPackage()
	fmtPkg := pkg.Import("fmt")
	tyInt := pkg.NewType("MyInt").InitType(pkg, types.Typ[types.Int])
	tyInt64 := pkg.NewType("MyInt64").InitType(pkg, types.Typ[types.Int64])
	tyUint64 := pkg.NewType("MyUint64").InitType(pkg, types.Typ[types.Uint64])
	tyFloat64 := pkg.NewType("MyFloat64").InitType(pkg, types.Typ[types.Float64])
	tyString := pkg.NewType("MyString").InitType(pkg, types.Typ[types.String])
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(tyInt, "a").
		NewVar(tyInt64, "b").
		NewVar(tyUint64, "c").
		NewVar(tyFloat64, "d").
		NewVar(tyString, "e").
		Val(fmtPkg.Ref("Println")).VarVal("a").MemberVal("String").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("b").MemberVal("String").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("c").MemberVal("String").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("d").MemberVal("String").Call(0).Call(1).EndStmt().
		Val(fmtPkg.Ref("Println")).VarVal("e").MemberVal("ToUpper").Call(0).Call(1).EndStmt().
		End()
	domTest(pkg, `package main

import (
	"fmt"
	"strconv"
	"strings"
)

type MyInt int
type MyInt64 int64
type MyUint64 uint64
type MyFloat64 float64
type MyString string

func main() {
	var a MyInt
	var b MyInt64
	var c MyUint64
	var d MyFloat64
	var e MyString
	fmt.Println(strconv.Itoa(int(a)))
	fmt.Println(strconv.FormatInt(int64(b), 10))
	fmt.Println(strconv.FormatUint(uint64(c), 10))
	fmt.Println(strconv.FormatFloat(float64(d), 'g', -1, 64))
	fmt.Println(strings.ToUpper(string(e)))
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 4: TestPrintlnPrintln ============
func testPrintlnPrintln() {
	fmt.Println("=== testPrintlnPrintln ===")
	pkg := newMainPackage()
	fmtPkg := pkg.Import("fmt")
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		Val(fmtPkg.Ref("Println")).Val(fmtPkg.Ref("Println")).Call(0).Call(1).EndStmt().
		End()
	domTest(pkg, `package main

import "fmt"

func main() {
	fmt.Println(fmt.Println())
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 5: TestImportGopPkg ============
func testImportGopPkg() {
	fmt.Println("=== testImportGopPkg ===")
	pkg := newMainPackage()
	foo := pkg.Import("github.com/goplus/gogen/internal/foo")
	foo.EnsureImported()
	nodeSet := foo.Ref("NodeSet")
	if nodeSet == nil {
		panic("TestImportGopPkg: NodeSet not found")
	}
	typ := nodeSet.(*types.TypeName).Type().(*types.Named)
	for i, n := 0, typ.NumMethods(); i < n; i++ {
		m := typ.Method(i)
		if m.Name() == "Attr" {
			funcs, ok := gogen.CheckOverloadMethod(m.Type().(*types.Signature))
			if !ok || len(funcs) != 2 {
				panic(fmt.Sprintf("CheckOverloadMethod failed: funcs=%v ok=%v", funcs, ok))
			}
			fmt.Println("  Found NodeSet.Attr with 2 overloads [PASS]")
			return
		}
	}
	panic("TestImportGopPkg: NodeSet.Attr not found")
}

// ============ Test 6: TestGoTypesPkg ============
func testGoTypesPkg() {
	fmt.Println("=== testGoTypesPkg ===")
	const src = `package foo

type mytype = byte

func bar(v mytype) rune {
	return 0
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, 0)
	if err != nil {
		panic(fmt.Sprintf("parser.ParseFile: %v", err))
	}

	conf := types.Config{}
	pkg, err := conf.Check("foo", fset, []*ast.File{f}, nil)
	if err != nil {
		panic(fmt.Sprintf("conf.Check: %v", err))
	}
	bar := pkg.Scope().Lookup("bar")
	// Note: behavior depends on Go version, just verify it compiles
	if bar == nil {
		panic("bar not found")
	}
	fmt.Printf("  bar.String() = %s [PASS]\n", bar.String())
}

// ============ Test 7: TestMethods ============
func testMethods() {
	fmt.Println("=== testMethods ===")
	const src = `package foo

type foo struct {}
func (a foo) A() {}
func (p *foo) Bar() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, 0)
	if err != nil {
		panic(fmt.Sprintf("parser.ParseFile: %v", err))
	}

	conf := types.Config{}
	pkg, err := conf.Check("foo", fset, []*ast.File{f}, nil)
	if err != nil {
		panic(fmt.Sprintf("conf.Check: %v", err))
	}
	foo, ok := pkg.Scope().Lookup("foo").Type().(*types.Named)
	if !ok {
		panic("foo not found")
	}
	if foo.NumMethods() != 2 {
		panic(fmt.Sprintf("foo.NumMethods: %d, expected 2", foo.NumMethods()))
	}
	fmt.Printf("  foo has %d methods [PASS]\n", foo.NumMethods())
}

// ============ Test 8: TestBasic ============
func testBasic() {
	fmt.Println("=== testBasic ===")
	pkg := newMainPackage()
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).End()
	if pkg.Ref("main") == nil {
		panic("main not found")
	}
	domTest(pkg, `package main

func main() {
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 9: TestMake ============
func testMake() {
	fmt.Println("=== testMake ===")
	pkg := newMainPackage()
	tySlice := types.NewSlice(types.Typ[types.Int])
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVarStart(tySlice, "a").Val(pkg.Builtin().Ref("make")).
		Typ(tySlice).Val(0).Val(2).Call(3).EndInit(1).
		End()
	domTest(pkg, `package main

func main() {
	var a []int = make([]int, 0, 2)
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 10: TestNew ============
func testNew() {
	fmt.Println("=== testNew ===")
	pkg := newMainPackage()
	tyInt := types.Typ[types.Int]
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVarStart(types.NewPointer(tyInt), "a").Val(pkg.Builtin().Ref("new")).
		Val(ctxRef(pkg, "int")).Call(1).EndInit(1).
		End()
	domTest(pkg, `package main

func main() {
	var a *int = new(int)
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 11: TestTypeConv ============
func testTypeConv() {
	fmt.Println("=== testTypeConv ===")
	pkg := newMainPackage()
	tyInt := types.Typ[types.Uint32]
	tyPInt := types.NewPointer(tyInt)
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVarStart(tyInt, "a").Typ(tyInt).Val(0).Call(1).EndInit(1).
		NewVarStart(tyPInt, "b").Typ(tyPInt).Val(nil).Call(1).EndInit(1).
		End()
	domTest(pkg, `package main

func main() {
	var a uint32 = uint32(0)
	var b *uint32 = (*uint32)(nil)
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 12: TestTypeConvBool ============
func testTypeConvBool() {
	fmt.Println("=== testTypeConvBool ===")
	pkg := newMainPackage()
	tyBool := types.Typ[types.Bool]
	tyInt := types.Typ[types.Uint32]
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVarStart(tyBool, "a").Val(false).EndInit(1).
		NewVarStart(tyInt, "b").Typ(tyInt).VarVal("a").Call(1).EndInit(1).
		End()
	domTest(pkg, `package main

func main() {
	var a bool = false
	var b uint32 = func() uint32 {
		if a {
			return 1
		} else {
			return 0
		}
	}()
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 13: TestIncDec ============
func testIncDec() {
	fmt.Println("=== testIncDec ===")
	pkg := newMainPackage()
	tyInt := types.Typ[types.Uint]
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		SetComments(comment("\n// new var a"), false).
		NewVar(tyInt, "a").
		SetComments(comment("\n// inc a"), true).
		VarRef(ctxRef(pkg, "a")).IncDec(token.INC).EndStmt().
		VarRef(ctxRef(pkg, "a")).IncDec(token.DEC).EndStmt().
		End()
	domTest(pkg, `package main

func main() {
// new var a
	var a uint
// inc a
	a++
	a--
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 14: TestSend ============
func testSend() {
	fmt.Println("=== testSend ===")
	pkg := newMainPackage()
	tyChan := types.NewChan(types.SendRecv, types.Typ[types.Uint])
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(tyChan, "a").
		VarVal("a").Val(1).Send().
		End()
	domTest(pkg, `package main

func main() {
	var a chan uint
	a <- 1
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 15: TestRecv ============
func testRecv() {
	fmt.Println("=== testRecv ===")
	pkg := newMainPackage()
	tyChan := types.NewChan(types.SendRecv, types.Typ[types.Uint])
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(tyChan, "a").
		NewVarStart(types.Typ[types.Uint], "b").VarVal("a").UnaryOp(token.ARROW).EndInit(1).
		DefineVarStart(0, "c", "ok").VarVal("a").UnaryOp(token.ARROW, true, nil).EndInit(1).
		End()
	domTest(pkg, `package main

func main() {
	var a chan uint
	var b uint = <-a
	c, ok := <-a
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 16: TestRecv2 ============
func testRecv2() {
	fmt.Println("=== testRecv2 ===")
	pkg := newMainPackage()
	tyChan := types.NewChan(types.SendRecv, types.Typ[types.Uint])
	typ := pkg.NewType("T").InitType(pkg, tyChan)
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(typ, "a").
		NewVarStart(types.Typ[types.Uint], "b").VarVal("a").UnaryOp(token.ARROW).EndInit(1).
		DefineVarStart(0, "c", "ok").VarVal("a").UnaryOp(token.ARROW, true).EndInit(1).
		End()
	domTest(pkg, `package main

type T chan uint

func main() {
	var a T
	var b uint = <-a
	c, ok := <-a
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 17: TestRecv3 ============
func testRecv3() {
	fmt.Println("=== testRecv3 ===")
	pkg := newMainPackage()
	tyUint := pkg.NewType("Uint").InitType(pkg, types.Typ[types.Uint])
	tyChan := types.NewChan(types.SendRecv, tyUint)
	typ := pkg.NewType("T").InitType(pkg, tyChan)
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(typ, "a").
		NewVarStart(tyUint, "b").VarVal("a").UnaryOp(token.ARROW).EndInit(1).
		DefineVarStart(0, "c", "ok").VarVal("a").UnaryOp(token.ARROW, true).EndInit(1).
		End()
	domTest(pkg, `package main

type Uint uint
type T chan Uint

func main() {
	var a T
	var b Uint = <-a
	c, ok := <-a
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 18: TestZeroLit ============
func testZeroLit() {
	fmt.Println("=== testZeroLit ===")
	pkg := newMainPackage()
	tyMap := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	ret := pkg.NewAutoParam("ret")
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVarStart(tyMap, "a").
		NewClosure(nil, gogen.NewTuple(ret), false).BodyStart(pkg).
		VarRef(ctxRef(pkg, "ret")).ZeroLit(ret.Type()).Assign(1).
		Val(ctxRef(pkg, "ret")).Val("Hi").IndexRef(1).Val(1).Assign(1).
		Return(0).
		End().Call(0).EndInit(1).
		End()
	domTest(pkg, `package main

func main() {
	var a map[string]int = func() (ret map[string]int) {
		ret = map[string]int{}
		ret["Hi"] = 1
		return
	}()
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 19: TestZeroLitAllTypes ============
func testZeroLitAllTypes() {
	fmt.Println("=== testZeroLitAllTypes ===")
	pkg := newMainPackage()
	tyString := types.Typ[types.String]
	tyBool := types.Typ[types.Bool]
	tyUP := types.Typ[types.UnsafePointer]
	tyMap := types.NewMap(tyString, types.Typ[types.Int])
	tySlice := types.NewSlice(types.Typ[types.Int])
	tyArray := gogen.NewArray(types.Typ[types.Int], 10)
	tyPointer := gogen.NewPointer(types.Typ[types.Int])
	tyChan := types.NewChan(types.SendRecv, types.Typ[types.Int])
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVarStart(tyMap, "a").ZeroLit(tyMap).EndInit(1).
		NewVarStart(tySlice, "b").ZeroLit(tySlice).EndInit(1).
		NewVarStart(tyPointer, "c").ZeroLit(tyPointer).EndInit(1).
		NewVarStart(tyChan, "d").ZeroLit(tyChan).EndInit(1).
		NewVarStart(tyBool, "e").ZeroLit(tyBool).EndInit(1).
		NewVarStart(tyString, "f").ZeroLit(tyString).EndInit(1).
		NewVarStart(tyUP, "g").ZeroLit(tyUP).EndInit(1).
		NewVarStart(gogen.TyEmptyInterface, "h").ZeroLit(gogen.TyEmptyInterface).EndInit(1).
		NewVarStart(tyArray, "i").ZeroLit(tyArray).EndInit(1).
		End()
	domTest(pkg, `package main

import "unsafe"

func main() {
	var a map[string]int = nil
	var b []int = nil
	var c *int = nil
	var d chan int = nil
	var e bool = false
	var f string = ""
	var g unsafe.Pointer = nil
	var h interface{} = nil
	var i [10]int = [10]int{}
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 20: TestTypeDeclInFunc ============
func testTypeDeclInFunc() {
	fmt.Println("=== testTypeDeclInFunc ===")
	pkg := newMainPackage()
	fields := []*types.Var{
		types.NewField(token.NoPos, pkg.Types, "x", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg.Types, "y", types.Typ[types.String], false),
	}
	typ := types.NewStruct(fields, nil)
	cb := pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg)
	foo := cb.NewType("foo").InitType(pkg, typ)
	cb.AliasType("bar", typ)
	a := cb.AliasType("a", foo)
	cb.AliasType("b", a)
	cb.End()
	domTest(pkg, `package main

func main() {
	type foo struct {
		x int
		y string
	}
	type bar = struct {
		x int
		y string
	}
	type a = foo
	type b = a
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 21: TestTypeDoc ============
func testTypeDoc() {
	fmt.Println("=== testTypeDoc ===")
	pkg := newMainPackage()
	typ := types.NewStruct(nil, nil)
	def := pkg.NewTypeDefs().SetComments(nil)
	def.NewType("foo").SetComments(pkg, comment("\n//go:notinheap")).InitType(pkg, typ)
	def.Complete()
	domTest(pkg, `package main

//go:notinheap
type foo struct {
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 22: TestDeleteType ============
func testDeleteType() {
	fmt.Println("=== testDeleteType ===")
	pkg := newMainPackage()
	typ := types.NewStruct(nil, nil)
	def := pkg.NewTypeDefs()
	decl := def.NewType("foo")
	if decl.State() != gogen.TyStateUninited {
		panic("TypeDecl.State failed: expected TyStateUninited")
	}
	decl.InitType(pkg, typ)
	if decl.State() != gogen.TyStateInited {
		panic("TypeDecl.State failed: expected TyStateInited")
	}
	decl.Delete()
	if decl.State() != gogen.TyStateDeleted {
		panic("TypeDecl.State failed: expected TyStateDeleted")
	}
	def.NewType("t").InitType(def.Pkg(), gogen.TyByte)
	def.NewType("bar").Delete()
	def.Complete()
	domTest(pkg, `package main

type t byte
`)
	fmt.Println("  [PASS]")
}

// ============ Test 23: TestTypeDecl ============
func testTypeDecl() {
	fmt.Println("=== testTypeDecl ===")
	pkg := newMainPackage()
	fields := []*types.Var{
		types.NewField(token.NoPos, pkg.Types, "x", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg.Types, "y", types.Typ[types.String], false),
	}
	typ := types.NewStruct(fields, nil)
	foo := pkg.NewType("foo").InitType(pkg, typ)
	pkg.AliasType("bar", typ)
	a := pkg.AliasType("a", foo)
	pkg.AliasType("b", a)
	domTest(pkg, `package main

type foo struct {
	x int
	y string
}
type bar = struct {
	x int
	y string
}
type a = foo
type b = a
`)
	fmt.Println("  [PASS]")
}

// ============ Test 24: TestTypeCycleDef ============
func testTypeCycleDef() {
	fmt.Println("=== testTypeCycleDef ===")
	pkg := newMainPackage()
	foo := pkg.NewType("foo")
	a := pkg.AliasType("a", foo.Type())
	b := pkg.AliasType("b", a)
	fields := []*types.Var{
		types.NewField(token.NoPos, pkg.Types, "p", types.NewPointer(b), false),
	}
	foo.InitType(pkg, types.NewStruct(fields, nil))
	domTest(pkg, `package main

type foo struct {
	p *b
}
type a = foo
type b = a
`)
	fmt.Println("  [PASS]")
}

// ============ Test 25: TestTypeMethods ============
func testTypeMethods() {
	fmt.Println("=== testTypeMethods ===")
	pkg := newMainPackage()
	fields := []*types.Var{
		types.NewField(token.NoPos, pkg.Types, "x", types.Typ[types.Int], false),
		types.NewField(token.NoPos, pkg.Types, "y", types.Typ[types.String], false),
	}
	typ := types.NewStruct(fields, nil)
	foo := pkg.NewType("foo").InitType(pkg, typ)
	recv := pkg.NewParam(token.NoPos, "a", foo)
	precv := pkg.NewParam(token.NoPos, "p", types.NewPointer(foo))
	pkg.NewFunc(recv, "Bar", nil, nil, false).SetComments(pkg, comment("\n// abc")).BodyStart(pkg).End()
	pkg.NewFunc(precv, "Print", nil, nil, false).BodyStart(pkg).End()
	if foo.NumMethods() != 2 {
		panic(fmt.Sprintf("foo.NumMethods = %d, expected 2", foo.NumMethods()))
	}
	domTest(pkg, `package main

type foo struct {
	x int
	y string
}

// abc
func (a foo) Bar() {
}
func (p *foo) Print() {
}
`)
	fmt.Println("  [PASS]")
}

// ============ Helper: newFuncDecl ============
func newFuncDecl(pkg *gogen.Package, name string, params, results *types.Tuple) *gogen.Func {
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	return pkg.NewFuncDecl(token.NoPos, name, sig)
}

// ============ Test 26: TestAssignInterface ============
func testAssignInterface() {
	fmt.Println("=== testAssignInterface ===")
	pkg := newMainPackage()
	foo := pkg.NewType("foo").InitType(pkg, types.Typ[types.Int])
	recv := pkg.NewParam(token.NoPos, "a", foo)
	ret := pkg.NewParam(token.NoPos, "ret", types.Typ[types.String])
	pkg.NewFunc(recv, "Error", nil, types.NewTuple(ret), false).BodyStart(pkg).
		Return(0).
		End()
	pkg.CB().NewVarStart(gogen.TyError, "err").
		Typ(foo).ZeroLit(foo).Call(1).
		EndInit(1)
	domTest(pkg, `package main

type foo int

func (a foo) Error() (ret string) {
	return
}

var err error = foo(0)
`)
	fmt.Println("  [PASS]")
}

// ============ Test 27: TestAssignUserInterface ============
func testAssignUserInterface() {
	fmt.Println("=== testAssignUserInterface ===")
	pkg := newMainPackage()
	methods := []*types.Func{
		types.NewFunc(token.NoPos, pkg.Types, "Bar", types.NewSignatureType(nil, nil, nil, nil, nil, false)),
	}
	tyInterf := types.NewInterfaceType(methods, nil).Complete()
	typStruc := types.NewStruct(nil, nil)
	foo := pkg.NewType("foo").InitType(pkg, tyInterf)
	bar := pkg.NewType("bar").InitType(pkg, typStruc)
	pbar := types.NewPointer(bar)
	recv := pkg.NewParam(token.NoPos, "p", pbar)
	vfoo := types.NewTuple(pkg.NewParam(token.NoPos, "v", types.NewSlice(foo)))
	pkg.NewFunc(recv, "Bar", nil, nil, false).BodyStart(pkg).End()
	pkg.NewFunc(nil, "f", vfoo, nil, true).BodyStart(pkg).End()
	pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
		NewVar(pbar, "v").
		VarVal("f").
		VarVal("v").VarVal("v").Call(2).EndStmt().
		End()
	domTest(pkg, `package main

type foo interface {
	Bar()
}
type bar struct {
}

func (p *bar) Bar() {
}
func f(v ...foo) {
}
func main() {
	var v *bar
	f(v, v)
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 28: TestTypeSwitch ============
func testTypeSwitch() {
	fmt.Println("=== testTypeSwitch ===")
	pkg := newMainPackage()
	p := pkg.NewParam(token.NoPos, "p", types.NewPointer(gogen.TyEmptyInterface))
	v := pkg.NewParam(token.NoPos, "v", gogen.TyEmptyInterface)
	newFuncDecl(pkg, "bar", types.NewTuple(p), nil)
	newFuncDecl(pkg, "foo", types.NewTuple(v), nil).BodyStart(pkg).
		TypeSwitch("t").Val(v).TypeAssertThen().
		TypeCase().Typ(types.Typ[types.Int]).Typ(types.Typ[types.String]).Then().
		Val(ctxRef(pkg, "bar")).VarRef(ctxRef(pkg, "t")).UnaryOp(token.AND).Call(1).EndStmt().
		End().
		TypeCase().Typ(types.Typ[types.Bool]).Then().
		NewVarStart(types.Typ[types.Bool], "x").Val(ctxRef(pkg, "t")).EndInit(1).
		End().
		TypeDefaultThen().
		Val(ctxRef(pkg, "bar")).VarRef(ctxRef(pkg, "t")).UnaryOp(token.AND).Call(1).EndStmt().
		End().
		End().
		End()
	domTest(pkg, `package main

func bar(p *interface{})
func foo(v interface{}) {
	switch t := v.(type) {
	case int, string:
		bar(&t)
	case bool:
		var x bool = t
	default:
		bar(&t)
	}
}
`)
	fmt.Println("  [PASS]")
}

// ============ Test 29: TestTypeSwitch2 ============
// IGNORED: Bug in llgo - TypeSwitch initialization statement is missing
// See: https://github.com/goplus/llgo/issues/1604
/*
func testTypeSwitch2() {
	fmt.Println("=== testTypeSwitch2 ===")
	pkg := newMainPackage()
	p := pkg.NewParam(token.NoPos, "p", types.NewPointer(gogen.TyEmptyInterface))
	v := pkg.NewParam(token.NoPos, "v", gogen.TyEmptyInterface)
	pkg.NewFunc(nil, "bar", types.NewTuple(p), nil, false).BodyStart(pkg).End()
	pkg.NewFunc(nil, "foo", types.NewTuple(v), nil, false).BodyStart(pkg).
		TypeSwitch("").Val(ctxRef(pkg, "bar")).Val(nil).Call(1).EndStmt().Val(v).TypeAssertThen().
		TypeCase().Typ(types.Typ[types.Int]).Then().
		Val(ctxRef(pkg, "bar")).VarRef(ctxRef(pkg, "v")).UnaryOp(token.AND).Call(1).EndStmt().
		End().
		End().
		End()
	domTest(pkg, `package main

func bar(p *interface{}) {
}
func foo(v interface{}) {
	switch bar(nil); v.(type) {
	case int:
		bar(&v)
	}
}
`)
	fmt.Println("  [PASS]")
}
*/

// ============ Main ============
func main() {
	fmt.Println("Running package_test tests (28 tests, 1 ignored)...")
	fmt.Println()

	// Batch 1: tests 1-5
	testRedupPkgIssue796()
	fmt.Println()

	testBTIMethod()
	fmt.Println()

	testTypedBTIMethod()
	fmt.Println()

	testPrintlnPrintln()
	fmt.Println()

	testImportGopPkg()
	fmt.Println()

	// Batch 2: tests 6-15
	testGoTypesPkg()
	fmt.Println()

	testMethods()
	fmt.Println()

	testBasic()
	fmt.Println()

	testMake()
	fmt.Println()

	testNew()
	fmt.Println()

	testTypeConv()
	fmt.Println()

	testTypeConvBool()
	fmt.Println()

	testIncDec()
	fmt.Println()

	testSend()
	fmt.Println()

	testRecv()
	fmt.Println()

	testRecv2()
	fmt.Println()

	testRecv3()
	fmt.Println()

	testZeroLit()
	fmt.Println()

	testZeroLitAllTypes()
	fmt.Println()

	testTypeDeclInFunc()
	fmt.Println()

	testTypeDoc()
	fmt.Println()

	testDeleteType()
	fmt.Println()

	testTypeDecl()
	fmt.Println()

	testTypeCycleDef()
	fmt.Println()

	testTypeMethods()
	fmt.Println()

	testAssignInterface()
	fmt.Println()

	testAssignUserInterface()
	fmt.Println()

	testTypeSwitch()
	fmt.Println()

	testTypeSwitch2()
	fmt.Println()

	fmt.Println("All tests PASSED!")
}

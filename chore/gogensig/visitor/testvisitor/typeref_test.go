package testvisitor_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/llgo/chore/gogensig/unmarshal"
	"github.com/goplus/llgo/chore/gogensig/util"
	"github.com/goplus/llgo/chore/gogensig/visitor"
	"github.com/goplus/llgo/chore/gogensig/visitor/genpkg"
	"github.com/goplus/llgo/chore/gogensig/visitor/symb"
	"github.com/goplus/llgo/chore/gogensig/visitor/testvisitor/cmptest"
)

func TestStructDeclRef(t *testing.T) {
	symbolfile := []symb.SymbolEntry{
		{
			MangleName: "ExecuteFoo",
			CppName:    "ExecuteFoo",
			GoName:     "CustomExecuteFoo",
		},
	}
	filePath, err := createAndWriteTempSymbFile(symbolfile)
	if err != nil {
		t.Fatal(err)
	}
	astConvert := visitor.NewAstConvert("typeref", filePath)
	var buf bytes.Buffer
	astConvert.SetVisitDone(func(pkg *genpkg.Package, docPath string) {
		err := pkg.WriteToBuffer(&buf)
		if err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
	})
	docVisitors := []visitor.DocVisitor{astConvert}
	p := unmarshal.NewDocFileSetUnmarshaller(docVisitors)
	orginCode :=
		`
struct Foo { int a; double b; bool c; };
int ExecuteFoo(int a,Foo b);
`
	bytes, err := util.Llcppsigfetch(orginCode, true, false)
	if err != nil {
		t.Fatal(err)
	}
	p.UnmarshalBytes(bytes)

	expectedString := `
package typeref

import "github.com/goplus/llgo/c"

type Foo struct {
	a c.Int
	b float64
	c c.Int
}

//go:linkname CustomExecuteFoo C.ExecuteFoo
func CustomExecuteFoo(a c.Int, b Foo) c.Int
	`
	result := buf.String()
	isEqual, diff := cmptest.EqualStringIgnoreSpace(result, expectedString)
	if !isEqual {
		t.Errorf("%s", diff)
	}
}

// struct Foo { int a; double b; bool c; }

func TestCustomStruct(t *testing.T) {
	symbolfile := []symb.SymbolEntry{
		{
			MangleName: "lua_close",
			CppName:    "lua_close",
			GoName:     "Close",
		},
		{
			MangleName: "lua_newthread",
			CppName:    "lua_newthread",
			GoName:     "Newthread",
		},
		{
			MangleName: "lua_closethread",
			CppName:    "lua_closethread",
			GoName:     "Closethread",
		},
		{
			MangleName: "lua_resetthread",
			CppName:    "lua_resetthread",
			GoName:     "Resetthread",
		},
	}
	filePath, err := createAndWriteTempSymbFile(symbolfile)
	if err != nil {
		t.Fatal(err)
	}
	astConvert := visitor.NewAstConvert("typeref", filePath)
	var buf bytes.Buffer
	astConvert.SetVisitDone(func(pkg *genpkg.Package, docPath string) {
		err := pkg.WriteToBuffer(&buf)
		if err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
	})
	docVisitors := []visitor.DocVisitor{astConvert}
	p := unmarshal.NewDocFileSetUnmarshaller(docVisitors)
	orginCode :=
		`
typedef struct lua_State lua_State;
LUA_API void(lua_close)(lua_State *L);
LUA_API lua_State *(lua_newthread)(lua_State *L);
LUA_API int(lua_closethread)(lua_State *L, lua_State *from);
LUA_API int(lua_resetthread)(lua_State *L); 
`
	bytes, err := util.Llcppsigfetch(orginCode, true, false)
	if err != nil {
		t.Fatal(err)
	}
	p.UnmarshalBytes(bytes)
	expectedString := `
package typeref

import "github.com/goplus/llgo/c"

type lua_State struct {
	Unused [8]uint8
}

//go:linkname Close C.lua_close
func Close(L *lua_State) c.Int

//go:linkname Closethread C.lua_closethread
func Closethread(L *lua_State, from *lua_State) c.Int

//go:linkname Resetthread C.lua_resetthread
func Resetthread(L *lua_State) c.Int
	`
	result := buf.String()
	isEqual, diff := cmptest.EqualStringIgnoreSpace(result, expectedString)
	if !isEqual {
		t.Errorf("%s", diff)
	}
}

func createAndWriteTempSymbFile(entries []symb.SymbolEntry) (string, error) {
	tempDir := os.TempDir()

	filePath := filepath.Join(tempDir, "llcppg.symb.json")

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(entries)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

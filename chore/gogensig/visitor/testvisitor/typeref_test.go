package testvisitor_test

import (
	"testing"

	"github.com/goplus/llgo/chore/gogensig/unmarshal"
	"github.com/goplus/llgo/chore/gogensig/util"
	"github.com/goplus/llgo/chore/gogensig/visitor"
)

func TestStructDeclRef(t *testing.T) {
	astConvert := visitor.NewAstConvert("typeref", "")
	docVisitors := []visitor.DocVisitor{astConvert}
	p := unmarshal.NewDocFileSetUnmarshaller(docVisitors)
	orginCode :=
		`
struct Foo { int a; double b; bool c; };
int ExecuteFoo(int a,Foo b);
`
	bytes, err := util.Llcppsigfetch(orginCode, true, true)
	if err != nil {
		t.Fatal(err)
	}
	p.UnmarshalBytes(bytes)
	// todo(zzy):compare test

	// package typeref

	// import "github.com/goplus/llgo/c"

	// type Foo struct {
	// 	a c.Int
	// 	b float64
	// 	c bool
	// }

	// //go:linkname Executefoo C.ExecuteFoo
	// func Executefoo(a c.Int, b Foo) c.Int
}

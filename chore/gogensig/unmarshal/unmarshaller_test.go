package unmarshal_test

// TODO to improve test case
import (
	"testing"

	"github.com/goplus/llgo/chore/gogensig/file"
	"github.com/goplus/llgo/chore/gogensig/unmarshal"
	"github.com/goplus/llgo/chore/gogensig/visitor"
)

func TestUnmarshalFiles(t *testing.T) {
	filesBytes, err := file.ReadFile("./jsons/files.json")
	if err != nil {
		t.Error(err)
	}
	docVisitors := []visitor.DocVisitor{visitor.NewAstConvert("files")}
	p := unmarshal.NewDocFileSetUnmarshaller(docVisitors)
	p.Unmarshal(filesBytes)
}

func TestUnmarshalAnyNode(t *testing.T) {
	nodeBytes, err := file.ReadFile("./jsons/anynode.json")
	if err != nil {
		t.Error(err)
	}
	docVisitors := []visitor.DocVisitor{visitor.NewAstConvert("anynode")}
	p := unmarshal.NewDocFileUnmarshaller(docVisitors)
	rawDocFile := unmarshal.NewRawDocFile("./jsons/anynode.json", nodeBytes)
	p.Unmarshal(rawDocFile)
}

func TestFunc1(t *testing.T) {
	bytes, err := file.ReadFile("./jsons/func1.json")
	if err != nil {
		t.Fatal(err)
	}
	docVisitors := []visitor.DocVisitor{visitor.NewAstConvert("func1")}
	p := unmarshal.NewDocFileSetUnmarshaller(docVisitors)
	err = p.Unmarshal(bytes)
	if err != nil {
		t.Fatal(err)
	}
}

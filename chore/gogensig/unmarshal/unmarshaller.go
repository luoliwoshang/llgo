package unmarshal

import (
	"encoding/json"
	"fmt"

	"github.com/goplus/llgo/chore/gogensig/util"
	"github.com/goplus/llgo/chore/gogensig/visitor"
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type PathDoc struct {
	Path string          `json:"path"`
	Doc  json.RawMessage `json:"doc"`
}

type RawMessageMap map[string]json.RawMessage

type PathDocArray []PathDoc

func NewPathDoc(path string, doc json.RawMessage) *PathDoc {
	return &PathDoc{Path: path, Doc: doc}
}

type DocFileUnmarshaller struct {
	VisitorList []visitor.DocVisitor
}

func NewDocFileUnmarshaller(visitorList []visitor.DocVisitor) *DocFileUnmarshaller {
	return &DocFileUnmarshaller{VisitorList: visitorList}
}

func (p *DocFileUnmarshaller) visit(_Type string, node ast.Node, docPath string) bool {
	for _, v := range p.VisitorList {
		v.Visit(_Type, node)
		v.VisitDone(docPath)
	}
	return true
}

func (p *DocFileUnmarshaller) UnmarshalBytes(raw []byte, docPath string) error {
	var temp struct {
		Type string `json:"_Type"`
	}
	if err := json.Unmarshal(raw, &temp); err != nil {
		return fmt.Errorf("error unmarshalling node type: %v", err)
	}
	unmarshaler, ok := nodeUnmarshalers[temp.Type]
	if !ok {
		return fmt.Errorf("unknown node type: %s", temp.Type)
	}
	node, err := unmarshaler(raw)
	if err != nil {
		return err
	}
	p.visit(temp.Type, node, docPath)
	return nil
}

func (p *DocFileUnmarshaller) UnmarshalPathDocFile(rawDocFile *PathDoc) error {
	var temp struct {
		Type string `json:"_Type"`
	}
	if err := json.Unmarshal(rawDocFile.Doc, &temp); err != nil {
		return fmt.Errorf("error unmarshalling node type: %v", err)
	}
	unmarshaler, ok := nodeUnmarshalers[temp.Type]
	if !ok {
		return fmt.Errorf("unknown node type: %s", temp.Type)
	}
	node, err := unmarshaler(rawDocFile.Doc)
	if err != nil {
		return err
	}
	p.visit(temp.Type, node, rawDocFile.Path)
	return nil
}

type DocFileSetUnmarshaller struct {
	docVisitorList []visitor.DocVisitor
}

func NewDocFileSetUnmarshaller(docVisitorList []visitor.DocVisitor) *DocFileSetUnmarshaller {
	return &DocFileSetUnmarshaller{docVisitorList: docVisitorList}
}

func (p *DocFileSetUnmarshaller) UnmarshalBytes(raw []byte) error {
	var filesWrapper PathDocArray
	if err := json.Unmarshal(raw, &filesWrapper); err != nil {
		return fmt.Errorf("error unmarshalling FilesWithPath: %w", err)
	}
	for _, fileData := range filesWrapper {
		docVisitor := NewDocFileUnmarshaller(p.docVisitorList)
		docVisitor.UnmarshalBytes(fileData.Doc, fileData.Path)
	}
	return nil
}

func (p *DocFileSetUnmarshaller) UnmarshalFile(jsonFilePath string) error {
	raw, err := util.ReadFile(jsonFilePath)
	if err != nil {
		return err
	}
	return p.UnmarshalBytes(raw)
}

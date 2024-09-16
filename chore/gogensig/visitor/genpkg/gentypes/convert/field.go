package convert

import (
	"github.com/goplus/llgo/chore/llcppg/ast"
)

type ConvertField struct {
	f *ast.Field
}

func Field(field *ast.Field) *ConvertField {
	return &ConvertField{f: field}
}

type ConvertFieldList struct {
	list *ast.FieldList
}

func FieldList(fieldList *ast.FieldList) *ConvertFieldList {
	return &ConvertFieldList{list: fieldList}
}

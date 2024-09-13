package symb

import (
	"encoding/json"
	"fmt"

	"github.com/goplus/llgo/chore/gogensig/file"
)

type MangleNameType string

type CppNameType string

type GoNameType string

type symbolntry struct {
	MangleName MangleNameType `json:"mangle"`
	CppName    CppNameType    `json:"c++"`
	GoName     GoNameType     `json:"go"`
}

type SymbolTable struct {
	t map[MangleNameType]symbolntry
}

func NewSymbolTable(filePath string) (*SymbolTable, error) {
	bytes, err := file.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var symbs []symbolntry
	err = json.Unmarshal(bytes, &symbs)
	if err != nil {
		return nil, err
	}
	var symbolTable SymbolTable
	symbolTable.t = make(map[MangleNameType]symbolntry)
	for _, symb := range symbs {
		symbolTable.t[symb.MangleName] = symb
	}
	return &symbolTable, nil
}

func (t *SymbolTable) LookupSymbol(name MangleNameType) (*symbolntry, error) {
	symbol, ok := t.t[name]
	if ok {
		return &symbol, nil
	}
	return nil, fmt.Errorf("symbol not found")
}

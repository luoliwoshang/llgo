package main

import (
	"errors"
)

// 模拟外部库
type cjson struct{}

func (c *cjson) Array() *cjson { return &cjson{} }
func (c *cjson) Delete()       {}

var cjsonLib = &cjson{}

type SymbolInfo struct {
	Mangle string
	CPP    string
	Go     string
}

func main() {
	err := processSymbols()
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func processSymbols() error {
	symbols := []SymbolInfo{{Mangle: "sym1", CPP: "Class::func1", Go: ""}}
	err := genSymbolTableFile(symbols)
	return err // 这里不使用 check，让错误传播到上层
}

func genSymbolTableFile(symbolInfos []SymbolInfo) error {
	existingSymbols := make(map[string]SymbolInfo)

	err := errors.New("failed to read symbol table file")
	// 正常运行
	// if err != nil {
	// 	return err
	// }
	// 错误运行（取消注释以模拟错误）
	check(err)
	for i := range symbolInfos {
		if existingSymbol, exists := existingSymbols[symbolInfos[i].Mangle]; exists {
			symbolInfos[i].Go = existingSymbol.Go
		}
	}

	root := cjsonLib.Array()
	defer root.Delete()

	for _, symbol := range symbolInfos {
		_ = symbol
	}

	return nil
}

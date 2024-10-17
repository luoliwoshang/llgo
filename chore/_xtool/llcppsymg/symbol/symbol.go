package symbol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/goplus/llgo/c"
	"github.com/goplus/llgo/c/cjson"
	"github.com/goplus/llgo/chore/_xtool/llcppsymg/config"
	"github.com/goplus/llgo/chore/_xtool/llcppsymg/parse"
	"github.com/goplus/llgo/chore/llcppg/types"
	"github.com/goplus/llgo/xtool/nm"
)

type LibConfig struct {
	Paths []string
	Names []string
}

func ParseLibConfig(lib string) *LibConfig {
	parts := strings.Fields(lib)
	config := &LibConfig{}

	for _, part := range parts {
		if strings.HasPrefix(part, "-L") {
			config.Paths = append(config.Paths, part[2:])
		} else if strings.HasPrefix(part, "-l") {
			config.Names = append(config.Names, part[2:])
		}
	}

	return config
}

func GenDylibPaths(config *LibConfig, defaultPaths []string) ([]string, error) {
	var foundPaths []string
	var notFound []string
	affix := ".dylib"
	if runtime.GOOS == "linux" {
		affix = ".so"
	}
	searchPaths := append(config.Paths, defaultPaths...)
	for _, name := range config.Names {
		var foundPath string
		for _, path := range searchPaths {
			dylibPath := filepath.Join(path, "lib"+name+affix)
			if _, err := os.Stat(dylibPath); err == nil {
				foundPath = dylibPath
			}
		}
		if foundPath != "" {
			foundPaths = append(foundPaths, foundPath)
		} else {
			notFound = append(notFound, name)
		}
	}
	if len(notFound) > 0 {
		fmt.Printf("Warning: Some libraries were not found: %s\n", strings.Join(notFound, ", "))
	}
	if len(foundPaths) == 0 {
		return nil, fmt.Errorf("failed to find any libraries")
	}
	return foundPaths, nil
}

// ParseDylibSymbols parses symbols from dynamic libraries specified in the lib string.
// It handles multiple libraries (e.g., -L/opt/homebrew/lib -llua -lm) and returns
// symbols if at least one library is successfully parsed. Errors from inaccessible
// libraries (like standard libs) are logged as warnings.
//
// Returns symbols and nil error if any symbols are found, or nil and error if none found.
func ParseDylibSymbols(lib string) ([]*nm.Symbol, error) {
	fmt.Printf("parse dylib symbols from config lib:%s\n", lib)

	conf := ParseLibConfig(lib)
	defaultPaths := getSysLibPaths()
	dylibPaths, err := GenDylibPaths(conf, defaultPaths)
	if err != nil {
		fmt.Printf("Warning: failed to generate some dylib paths: %v\n", err)
	}

	var symbols []*nm.Symbol
	var parseErrors []string

	for _, dylibPath := range dylibPaths {
		if _, err := os.Stat(dylibPath); err != nil {
			fmt.Printf("Warning: Failed to access dylib %s: %v\n", dylibPath, err)
			continue
		}

		files, err := nm.New("").List(dylibPath)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("Failed to list symbols in dylib %s: %v", dylibPath, err))
			continue
		}

		for _, file := range files {
			symbols = append(symbols, file.Symbols...)
		}
	}

	if len(symbols) > 0 {
		if len(parseErrors) > 0 {
			fmt.Printf("Warning: Some libraries could not be parsed: %v\n", parseErrors)
		}
		return symbols, nil
	}

	return nil, fmt.Errorf("no symbols found in any dylib. Errors: %v", parseErrors)
}

func getSysLibPaths() []string {
	var paths []string
	if runtime.GOOS == "linux" {
		paths = []string{
			"/usr/lib",
			"/usr/local/lib",
		}
		paths = append(paths, getPath("/etc/ld.so.conf")...)
		confd := "/etc/ld.so.conf.d"
		if dir, err := os.Stat(confd); err == nil && dir.IsDir() {
			_ = dir
			// todo(zzy) : wait llgo os.ReadDir support
			// files, err := os.ReadDir(confd)
			// if err == nil {
			// 	for _, file := range files {
			// 		filepath := filepath.Join(confd, file.Name())
			// 		paths = append(paths, getPath(filepath)...)
			// 	}
			// }
		}
	}
	return paths
}

func getPath(file string) []string {
	var paths []string
	content, err := os.ReadFile(file)
	if err != nil {
		return paths
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if file, err := os.Stat(line); err == nil && file.IsDir() {
				paths = append(paths, line)
			}
		}
	}
	return paths
}

// finds the intersection of symbols from the dynamic library's symbol table and the symbols parsed from header files.
// It returns a list of symbols that can be externally linked.
func GetCommonSymbols(dylibSymbols []*nm.Symbol, headerSymbols map[string]*parse.SymbolInfo) []*types.SymbolInfo {
	var commonSymbols []*types.SymbolInfo
	for _, dylibSym := range dylibSymbols {
		symName := strings.TrimLeft(dylibSym.Name, "_")
		if symInfo, ok := headerSymbols[symName]; ok {
			symbolInfo := &types.SymbolInfo{
				Mangle: symName,
				CPP:    symInfo.ProtoName,
				Go:     symInfo.GoName,
			}
			commonSymbols = append(commonSymbols, symbolInfo)
		}
	}
	return commonSymbols
}

func ReadExistingSymbolTable(fileName string) (map[string]types.SymbolInfo, bool) {
	if _, err := os.Stat(fileName); err != nil {
		return nil, false
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, false
	}

	parsedJSON := cjson.ParseBytes(data)
	if parsedJSON == nil {
		return nil, false
	}

	existingSymbols := make(map[string]types.SymbolInfo)
	arraySize := parsedJSON.GetArraySize()

	for i := 0; i < int(arraySize); i++ {
		item := parsedJSON.GetArrayItem(c.Int(i))
		symbol := types.SymbolInfo{
			Mangle: config.GetStringItem(item, "mangle", ""),
			CPP:    config.GetStringItem(item, "c++", ""),
			Go:     config.GetStringItem(item, "go", ""),
		}
		existingSymbols[symbol.Mangle] = symbol
	}

	return existingSymbols, true
}

func GenSymbolTableData(commonSymbols []*types.SymbolInfo, existingSymbols map[string]types.SymbolInfo) ([]byte, error) {
	// todo(zzy): len(existingSymbols) !=0
	// https://github.com/goplus/llgo/issues/808 will cause unexpected panic
	// https://github.com/goplus/llgo/pull/793 this pr can fix it
	for i := range commonSymbols {
		if existingSymbol, exists := existingSymbols[commonSymbols[i].Mangle]; exists {
			commonSymbols[i].Go = existingSymbol.Go
		}
	}

	root := cjson.Array()
	defer root.Delete()

	for _, symbol := range commonSymbols {
		item := cjson.Object()
		item.SetItem(c.Str("mangle"), cjson.String(c.AllocaCStr(symbol.Mangle)))
		item.SetItem(c.Str("c++"), cjson.String(c.AllocaCStr(symbol.CPP)))
		item.SetItem(c.Str("go"), cjson.String(c.AllocaCStr(symbol.Go)))
		root.AddItem(item)
	}

	cStr := root.Print()
	if cStr == nil {
		return nil, errors.New("symbol table is empty")
	}
	defer c.Free(unsafe.Pointer(cStr))
	result := []byte(c.GoString(cStr))
	return result, nil
}

func GenerateAndUpdateSymbolTable(symbols []*nm.Symbol, headerInfos map[string]*parse.SymbolInfo, symbFile string) ([]byte, error) {
	commonSymbols := GetCommonSymbols(symbols, headerInfos)

	existSymbols, _ := ReadExistingSymbolTable(symbFile)

	symbolData, err := GenSymbolTableData(commonSymbols, existSymbols)
	if err != nil {
		return nil, err
	}

	return symbolData, nil
}

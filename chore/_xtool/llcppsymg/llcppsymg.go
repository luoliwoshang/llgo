/*
 * Copyright (c) 2024 The GoPlus Authors (goplus.org). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/goplus/llgo/chore/_xtool/llcppsymg/common"
	"github.com/goplus/llgo/chore/llcppg/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	cfgFile := "llcppg.cfg"
	if len(os.Args) > 1 {
		cfgFile = os.Args[1]
	}

	var data []byte
	var err error
	if cfgFile == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(cfgFile)
	}
	check(err)

	var config types.Config
	err = json.Unmarshal(data, &config)
	check(err)

	symbols, err := parseDylibSymbols(config.Libs)
	check(err)

	files, err := parseHeaderFile(config)
	check(err)

	symbolInfo := getCommonSymbols(symbols, files)

	jsonData, err := json.MarshalIndent(symbolInfo, "", "  ")
	check(err)

	// 写入文件
	fileName := "llcppg.symb.json"
	err = os.WriteFile(fileName, jsonData, 0644) // 使用 0644 权限
	check(err)

}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func parseDylibSymbols(lib string) ([]common.CPPSymbol, error) {
	dylibPath, _ := generateDylibPath(lib)
	nmCmd := exec.Command("nm", "-gU", dylibPath)
	nmOutput, err := nmCmd.Output()
	if err != nil {
		return nil, errors.New("failed to execute nm command")
	}

	symbols := parseNmOutput(nmOutput)

	for i, sym := range symbols {
		decodedName, err := decodeSymbolName(sym.Name)
		if err != nil {
			return nil, err
		}
		symbols[i].Name = decodedName
	}

	return symbols, nil
}

func generateDylibPath(lib string) (string, error) {
	// 执行pkg-config命令
	output := expandEnv(lib)
	// 解析输出
	libPath := ""
	libName := ""
	for _, part := range strings.Fields(string(output)) {
		if strings.HasPrefix(part, "-L") {
			libPath = part[2:] // 去掉-L前缀
		} else if strings.HasPrefix(part, "-l") {
			libName = part[2:] // 去掉-l前缀
		}
	}

	if libPath == "" || libName == "" {
		return "", fmt.Errorf("failed to parse pkg-config output: %s", output)
	}

	// 构造dylib路径
	dylibPath := filepath.Join(libPath, "lib"+libName+".dylib")
	return dylibPath, nil
}

func parseNmOutput(output []byte) []common.CPPSymbol {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var symbols []common.CPPSymbol

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		symbolName := fields[2]
		// Check if the symbol name starts with an underscore and remove it if present
		if strings.HasPrefix(symbolName, "_") {
			symbolName = symbolName[1:]
		}
		symbols = append(symbols, common.CPPSymbol{
			Symbol: symbolName,
			Type:   fields[1],
			Name:   fields[2],
		})
	}

	return symbols
}

func decodeSymbolName(symbolName string) (string, error) {
	cppfiltCmd := exec.Command("c++filt", symbolName)
	cppfiltOutput, err := cppfiltCmd.Output()
	if err != nil {
		return "", errors.New("failed to execute c++filt command")
	}

	decodedName := strings.TrimSpace(string(cppfiltOutput))
	// 将特定的模板类型转换为 std::string
	decodedName = strings.ReplaceAll(decodedName, "std::__1::basic_string<char, std::__1::char_traits<char>, std::__1::allocator<char> > const", "std::string")
	return decodedName, nil
}

// parseHeaderFile
func parseHeaderFile(config types.Config) ([]common.ASTInformation, error) {
	files := generateHeaderFilePath(config.CFlags, config.Include)
	headerFileCmd := exec.Command("llcppinfofetch", files...)
	headerFileOutput, err := headerFileCmd.Output()
	if err != nil {
		return nil, errors.New("failed to execute header file command")
	}
	t := make([]common.ASTInformation, 0)
	err = json.Unmarshal(headerFileOutput, &t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func generateHeaderFilePath(cflags string, files []string) []string {
	// 执行pkg-config命令
	prefixPath := expandEnv(cflags)
	if strings.HasPrefix(prefixPath, "-I") {
		prefixPath = prefixPath[2:]
	}
	// 去掉首尾空白字符（包括换行符）
	prefixPath = strings.TrimSpace(prefixPath)
	var includePaths []string
	for _, file := range files {
		includePaths = append(includePaths, filepath.Join(prefixPath, "/"+file))
	}
	return includePaths
}

func getCommonSymbols(dylibSymbols []common.CPPSymbol, astInfoList []common.ASTInformation) []common.SymbolInfo {
	var commonSymbols []common.SymbolInfo
	functionNameMap := make(map[string]int)

	for _, astInfo := range astInfoList {
		for _, dylibSym := range dylibSymbols {
			if dylibSym.Symbol == astInfo.Symbol {
				cppName := generateCPPName(astInfo)
				functionNameMap[cppName]++
				symbolInfo := common.SymbolInfo{
					Mangle: dylibSym.Symbol,
					CPP:    cppName,
					Go:     generateMangle(astInfo, functionNameMap[cppName]),
				}
				commonSymbols = append(commonSymbols, symbolInfo)
				break
			}
		}
	}

	return commonSymbols
}

func generateCPPName(astInfo common.ASTInformation) string {
	cppName := astInfo.Name
	if astInfo.Class != "" {
		cppName = astInfo.Class + "::" + astInfo.Name
	}
	return cppName
}

func generateMangle(astInfo common.ASTInformation, count int) string {
	res := ""
	if astInfo.Class != "" {
		if astInfo.Class == astInfo.Name {
			res = "(*" + astInfo.Class + ")." + "Init"
			if count > 1 {
				res += "__" + strconv.Itoa(count-1)
			}
		} else if astInfo.Name == "~"+astInfo.Class {
			res = "(*" + astInfo.Class + ")." + "Dispose"
			if count > 1 {
				res += "__" + strconv.Itoa(count-1)
			}
		} else {
			res = "(*" + astInfo.Class + ")." + astInfo.Name + "__" + string(rune(count))
		}
	} else {
		res = astInfo.Name
		if count > 0 {
			res += "__" + strconv.Itoa(count-1)
		}
	}
	return res
}

var (
	reSubcmd = regexp.MustCompile(`\$\([^)]+\)`)
	reFlag   = regexp.MustCompile(`[^ \t\n]+`)
)

func expandEnv(s string) string {
	return expandEnvWithCmd(s)
}

func expandEnvWithCmd(s string) string {
	expanded := reSubcmd.ReplaceAllStringFunc(s, func(m string) string {
		subcmd := strings.TrimSpace(s[2 : len(s)-1])

		args := parseSubcmd(subcmd)

		cmd := args[0]

		if cmd != "pkg-config" && cmd != "llvm-config" {
			fmt.Fprintf(os.Stderr, "expand cmd only support pkg-config and llvm-config: '%s'\n", subcmd)
			return ""
		}

		var out []byte
		var err error
		out, err = exec.Command(cmd, args[1:]...).Output()

		if err != nil {
			// TODO(kindy): log in verbose mode
			return ""
		}

		return string(out)
	})
	return os.Expand(expanded, os.Getenv)
}

func parseSubcmd(s string) []string {
	return reFlag.FindAllString(s, -1)
}

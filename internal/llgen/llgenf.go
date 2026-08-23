/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package llgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/internal/goflags"
	"github.com/xgo-dev/llgo/internal/targets"
)

// GeneratedIR is an LLVM IR snapshot together with its effective target.
type GeneratedIR struct {
	Text   string
	GOOS   string
	GOARCH string
	Target string
}

func GenFrom(fileOrPkg string) string {
	return Generate(fileOrPkg).Text
}

// Generate returns the historical pre-target-ABI module together with its
// effective target. GenFrom is the text-only compatibility wrapper.
func Generate(fileOrPkg string) GeneratedIR {
	return GenerateWithConf(fileOrPkg, nil)
}

// GenerateWithConf is Generate with caller-provided target and frontend
// settings. It uses generation mode and does not invoke ModuleHook.
func GenerateWithConf(fileOrPkg string, input *build.Config) GeneratedIR {
	return generateWithConf(fileOrPkg, input, cabi.ModeNone)
}

// GeneratePostABI returns the module after aggregate and target ABI lowering
// and before LLVM optimization. GenFrom intentionally retains its historical
// pre-target-ABI behavior.
func GeneratePostABI(fileOrPkg string) GeneratedIR {
	return GeneratePostABIWithConf(fileOrPkg, nil)
}

// GeneratePostABIWithConf is GeneratePostABI with caller-provided target and
// frontend settings. It uses generation mode and does not invoke ModuleHook.
func GeneratePostABIWithConf(fileOrPkg string, input *build.Config) GeneratedIR {
	return generateWithConf(fileOrPkg, input, cabi.ModeAllFunc)
}

func generateWithConf(fileOrPkg string, input *build.Config, abiMode cabi.Mode) GeneratedIR {
	conf := &build.Config{}
	if input != nil {
		*conf = *input
	}
	conf.Mode = build.ModeGen
	conf.AbiMode = abiMode
	conf.GenLL = true
	conf.ModuleHook = nil
	// Cache hits can skip the module production needed by snapshot generation.
	conf.ForceRebuild = true
	pkg, err := genFromConf(fileOrPkg, conf)
	check(err)
	return consumeGeneratedIR(pkg)
}

func consumeGeneratedIR(pkg build.Package) GeneratedIR {
	target := pkg.LPkg.Prog.Target()
	out := GeneratedIR{
		Text:   pkg.LPkg.String(),
		GOOS:   target.GOOS,
		GOARCH: target.GOARCH,
		Target: target.Target,
	}
	pkg.LPkg.Prog.Dispose()
	return out
}

func genFrom(pkgPath string, abiMode build.AbiMode) (build.Package, error) {
	conf := &build.Config{
		Mode:    build.ModeGen,
		AbiMode: abiMode,
		GenLL:   true,
	}
	return genFromConf(pkgPath, conf)
}

func genFromConf(pkgPath string, conf *build.Config) (build.Package, error) {
	if err := applyFlagsFile(conf, filepath.Join(pkgPath, "flags.txt")); err != nil {
		return nil, err
	}
	pkgs, err := build.Do([]string{pkgPath}, conf)
	if err != nil {
		return nil, err
	}
	return pkgs[0], nil
}

func DoFile(fileOrPkg, outFile string) {
	ret := GenFrom(fileOrPkg)
	err := os.WriteFile(outFile, []byte(ret), 0644)
	check(err)
}

func readFlags(flagsFile string) ([]string, error) {
	data, err := os.ReadFile(flagsFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	flags, err := goflags.ParseFlagFile(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", flagsFile, err)
	}
	return flags, nil
}

func applyFlagsFile(conf *build.Config, flagsFile string) error {
	flags, err := readFlags(flagsFile)
	if err != nil {
		return err
	}
	next := *conf
	goFlags := make([]string, 0, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		switch {
		case strings.HasPrefix(flag, "GOOS="):
			next.Goos = strings.TrimPrefix(flag, "GOOS=")
		case strings.HasPrefix(flag, "GOARCH="):
			next.Goarch = strings.TrimPrefix(flag, "GOARCH=")
		case flag == "-target" || flag == "--target":
			if i+1 == len(flags) {
				return fmt.Errorf("apply %s: %s requires a value", flagsFile, flag)
			}
			i++
			next.Target = flags[i]
		case strings.HasPrefix(flag, "-target="):
			next.Target = strings.TrimPrefix(flag, "-target=")
		case strings.HasPrefix(flag, "--target="):
			next.Target = strings.TrimPrefix(flag, "--target=")
		default:
			goFlags = append(goFlags, flag)
		}
	}
	if next.Target != "" {
		target, err := targets.NewDefaultResolver().Resolve(next.Target)
		if err != nil {
			return fmt.Errorf("apply %s: %w", flagsFile, err)
		}
		next.Goos = target.GOOS
		next.Goarch = target.GOARCH
	}
	if err := goflags.ApplyBuildFlags(&next, goFlags); err != nil {
		return fmt.Errorf("apply %s: %w", flagsFile, err)
	}
	*conf = next
	return nil
}

func SmartDoFile(pkgPath string) {
	SmartDoFileEx(pkgPath, 0)
}

func SmartDoFileEx(pkgPath string, abiMode build.AbiMode) {
	pkg, err := genFrom(pkgPath, abiMode)
	check(err)

	const autgenFile = "llgo_autogen.ll"
	dir, _ := filepath.Split(pkg.GoFiles[0])
	outFile := dir + autgenFile

	b, err := os.ReadFile(outFile)
	if err == nil && len(b) == 1 && b[0] == ';' {
		return // skip to gen
	}

	if err = os.WriteFile(outFile, []byte(pkg.LPkg.String()), 0644); err != nil {
		panic(err)
	}
}

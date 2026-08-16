/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package gotest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const floatIntConversionProbe = `package main

import (
	"fmt"
	"math"
)

//go:noinline
func id[T any](x T) T { return x }

func emit[T ~float32 | ~float64](x T) {
	fmt.Printf("%T %g i=%d i8=%d i16=%d i32=%d i64=%d u=%d u8=%d u16=%d u32=%d u64=%d up=%d\n",
		x, x, int(x), int8(x), int16(x), int32(x), int64(x),
		uint(x), uint8(x), uint16(x), uint32(x), uint64(x), uintptr(x))
}

func main() {
	one := id(1.0)
	for _, x := range []float32{
		id(float32(-math.MaxFloat32)), id(float32(-one / 0)), id(float32(-1.5)),
		id(float32(math.NaN())), id(float32(1.5)), id(float32(1 << 31)),
		id(float32(1 << 32)), id(float32(math.MaxFloat32)), id(float32(one / 0)),
	} {
		emit(x)
	}
	for _, x := range []float64{
		id(-math.MaxFloat64), id(-one / 0), id(-1.5), id(math.NaN()), id(1.5),
		id(float64(1 << 31)), id(float64(1 << 32)), id(float64(1 << 63)),
		id(math.MaxFloat64), id(one / 0),
	} {
		emit(x)
	}
}
`

func TestFloatToIntegerConversionSemantics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(floatIntConversionProbe), 0644); err != nil {
		t.Fatal(err)
	}
	run := func(dir, name string, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		cmd.Env = os.Environ()
		out, err := cmd.Output()
		if err != nil {
			var stderr []byte
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr = exitErr.Stderr
			}
			t.Fatalf("%s %v failed: %v\n%s", name, args, err, stderr)
		}
		return out
	}
	want := run(filepath.Dir(file), "go", "run", file)
	got := run(dir, acceptanceLLGoBinary(t), "run", file)
	if !bytes.Equal(got, want) {
		t.Fatalf("float-to-integer conversions differ from gc\ngc:\n%s\nllgo:\n%s", want, got)
	}
}

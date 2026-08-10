package dcepass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	qtest "github.com/qiniu/x/test"
	"github.com/xgo-dev/llvm"
)

const (
	taskTypeName    = "_llgo_main.Task"
	ptrTaskTypeName = "*_llgo_main.Task"
)

func TestEmitStrongTypeOverrides(t *testing.T) {
	tests := []struct {
		name      string
		liveSlots map[string][]int
	}{
		{
			name: "method_slots",
			liveSlots: map[string][]int{
				taskTypeName:    {1}, // Run
				ptrTaskTypeName: {1}, // Run
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			dir := filepath.Join("testdata", tt.name)
			src := parseModule(t, &ctx, filepath.Join(dir, "in.ll"))
			defer src.Dispose()
			dst := ctx.NewModule("dst")
			defer dst.Dispose()

			EmitStrongTypeOverrides(dst, []llvm.Module{src}, tt.liveSlots, true)
			want, err := os.ReadFile(filepath.Join(dir, "expect.ll"))
			if err != nil {
				t.Fatal(err)
			}
			qtest.Diff(t, filepath.Join(dir, "expect.ll.new"), []byte(dst.String()), want)
		})
	}
}

func TestRewriteTypeMethodTablesPreservesLinkage(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := parseModule(t, &ctx, filepath.Join("testdata", "method_slots", "in.ll"))
	defer mod.Dispose()

	if got := RewriteTypeMethodTables(mod, map[string][]int{
		taskTypeName:    {1},
		ptrTaskTypeName: {1},
	}, false); got != 2 {
		t.Fatalf("RewriteTypeMethodTables rewrote %d globals, want 2", got)
	}
	out := mod.String()
	if !strings.Contains(out, `@_llgo_main.Task = weak_odr constant`) {
		t.Fatalf("rewrite changed the source type linkage:\n%s", out)
	}
	if strings.Contains(out, `@_llgo_main.Task = constant`) {
		t.Fatalf("rewrite introduced a strong duplicate:\n%s", out)
	}
	if !strings.Contains(out, `ptr @"github.com/goplus/llgo/runtime/internal/runtime.unreachableMethod"`) {
		t.Fatalf("rewrite did not replace the dead method slot:\n%s", out)
	}
}

func TestMethodArray(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	initWithLast := func(last llvm.Value) llvm.Value {
		return llvm.ConstStruct([]llvm.Value{llvm.ConstNull(ctx.Int8Type()), last}, false)
	}
	intValue := llvm.ConstInt(ctx.Int8Type(), 1, false)
	methodTy := ctx.StructCreateNamed(abiMethodTypeName)
	methodTy.StructSetBody([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
	method := llvm.ConstNamedStruct(methodTy, []llvm.Value{intValue, intValue, intValue, intValue})
	methods := llvm.ConstArray(methodTy, []llvm.Value{method, method})

	methodsVal, elemTy, ok := methodArray(initWithLast(methods))
	if !ok {
		t.Fatal("methodArray failed to recognize an ABI method array")
	}
	if methodsVal.OperandsCount() != 2 {
		t.Fatalf("methodArray returned %d methods, want 2", methodsVal.OperandsCount())
	}
	if elemTy.StructElementTypesCount() != 4 {
		t.Fatalf("methodArray returned %d fields, want 4", elemTy.StructElementTypesCount())
	}

	arrayOfInts := llvm.ConstArray(ctx.Int8Type(), []llvm.Value{intValue})
	wrongFieldsTy := ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
	wrongFields := llvm.ConstNamedStruct(wrongFieldsTy, []llvm.Value{intValue, intValue, intValue})
	wrongNameTy := ctx.StructCreateNamed("external/" + abiMethodTypeName)
	wrongNameTy.StructSetBody([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
	wrongName := llvm.ConstNamedStruct(wrongNameTy, []llvm.Value{intValue, intValue, intValue, intValue})

	tests := []struct {
		name string
		init llvm.Value
	}{
		{name: "nil", init: llvm.Value{}},
		{name: "no operands", init: llvm.ConstNull(ctx.Int32Type())},
		{name: "last operand is not array", init: initWithLast(intValue)},
		{name: "array element is not struct", init: initWithLast(arrayOfInts)},
		{name: "struct has wrong field count", init: initWithLast(llvm.ConstArray(wrongFieldsTy, []llvm.Value{wrongFields}))},
		{name: "struct name only contains ABI name", init: initWithLast(llvm.ConstArray(wrongNameTy, []llvm.Value{wrongName}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, ok := methodArray(tt.init); ok {
				t.Fatalf("methodArray recognized invalid initializer: %s", tt.name)
			}
		})
	}
}

func parseModule(t *testing.T, ctx *llvm.Context, path string) llvm.Module {
	t.Helper()
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	return mod
}

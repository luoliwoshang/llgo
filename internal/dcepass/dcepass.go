// Package dcepass rewrites ABI metadata so link-time dead code elimination can
// drop method bodies that are no longer referenced by live method slots.
package dcepass

import (
	"fmt"
	"os"

	"github.com/xgo-dev/llvm"
)

const (
	abiMethodTypeName     = "github.com/goplus/llgo/runtime/abi.Method"
	unreachableMethodName = "github.com/goplus/llgo/runtime/internal/runtime.unreachableMethod"
)

// EmitStrongTypeOverrides emits method-pruned strong ABI type symbols into dst.
// srcMods contains the original package modules. For each constant ABI type
// global with a method array, it creates a same-name strong global in dst and
// clears IFn/TFn for method slots not listed in liveSlots[typeName].
// When verbose is true, dropped method slots are reported to os.Stderr.
func EmitStrongTypeOverrides(dst llvm.Module, srcMods []llvm.Module, liveSlots map[string][]int, verbose bool) {
	emitted := make(map[string]bool)
	emitter := newOverrideEmitter(dst)
	for _, src := range srcMods {
		for g := src.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
			name := g.Name()
			if emitted[name] || !g.IsGlobalConstant() {
				continue
			}
			methodsVal, elemTy, ok := methodArray(g.Initializer())
			if !ok {
				continue
			}
			emitter.emitTypeOverride(g, methodsVal, elemTy, liveSlotSet(liveSlots[name]), verbose)
			emitted[name] = true
		}
	}
}

// RewriteTypeMethodTables rewrites ABI method table initializers in mod in
// place. It preserves each type descriptor's existing linkage and COMDAT, so
// ThinLTO sees one ordinary definition per package instead of an entry-module
// strong override competing with weak_odr definitions.
//
// A missing type entry in liveSlots means that no method slot is demanded.
// The method name and type are retained while IFn/TFn point at the runtime
// unreachable stub, preserving ABI table shape and method matching metadata.
func RewriteTypeMethodTables(mod llvm.Module, liveSlots map[string][]int, verbose bool) int {
	if mod.IsNil() {
		return 0
	}
	rewriter := &moduleRewriter{mod: mod}
	rewritten := 0
	for g := mod.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
		if g.IsDeclaration() || !g.IsGlobalConstant() {
			continue
		}
		methodsVal, elemTy, ok := methodArray(g.Initializer())
		if !ok {
			continue
		}
		if rewriter.rewriteGlobal(g, methodsVal, elemTy, liveSlotSet(liveSlots[g.Name()]), verbose) {
			rewritten++
		}
	}
	return rewritten
}

type moduleRewriter struct {
	mod         llvm.Module
	unreachable llvm.Value
}

func (r *moduleRewriter) unreachableMethod() llvm.Value {
	if r.unreachable.IsNil() {
		r.unreachable = r.mod.NamedFunction(unreachableMethodName)
	}
	if r.unreachable.IsNil() {
		r.unreachable = llvm.AddFunction(r.mod, unreachableMethodName,
			llvm.FunctionType(r.mod.Context().VoidType(), nil, false))
	}
	return r.unreachable
}

func (r *moduleRewriter) rewriteGlobal(g, methodsVal llvm.Value, elemTy llvm.Type, keepIdx map[int]bool, verbose bool) bool {
	init := g.Initializer()
	fields := make([]llvm.Value, init.OperandsCount())
	for i := range fields {
		fields[i] = init.Operand(i)
	}
	methods := make([]llvm.Value, methodsVal.OperandsCount())
	dropped := false
	for i := range methods {
		orig := methodsVal.Operand(i)
		if keepIdx[i] {
			methods[i] = orig
			continue
		}
		dropped = true
		if verbose {
			fmt.Fprintf(os.Stderr, "[dce] drop method %s[%d] ifn=%s tfn=%s\n", g.Name(), i, orig.Operand(2).Name(), orig.Operand(3).Name())
		}
		unreachable := r.unreachableMethod()
		methods[i] = llvm.ConstNamedStruct(elemTy, []llvm.Value{
			orig.Operand(0),
			orig.Operand(1),
			unreachable,
			unreachable,
		})
	}
	if !dropped {
		return false
	}
	fields[len(fields)-1] = llvm.ConstArray(elemTy, methods)
	g.SetInitializer(constStructOfType(init.Type(), fields))
	return true
}

type overrideEmitter struct {
	dst    llvm.Module
	values map[llvm.Value]llvm.Value
}

func newOverrideEmitter(dst llvm.Module) *overrideEmitter {
	return &overrideEmitter{dst: dst, values: make(map[llvm.Value]llvm.Value)}
}

func (e *overrideEmitter) emitTypeOverride(srcType, methodsVal llvm.Value, elemTy llvm.Type, keepIdx map[int]bool, verbose bool) {
	init := srcType.Initializer()
	dstType := e.ensureOverrideGlobal(srcType)
	e.values[srcType] = dstType

	fieldCount := init.OperandsCount()
	fields := make([]llvm.Value, fieldCount)
	for i := 0; i < fieldCount-1; i++ {
		fields[i] = e.cloneConst(init.Operand(i))
	}

	unreachableMethod := e.unreachableMethod()
	methods := make([]llvm.Value, methodsVal.OperandsCount())
	for i := range methods {
		orig := methodsVal.Operand(i)
		if keepIdx[i] {
			methods[i] = e.cloneConst(orig)
			continue
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "[dce] drop method %s[%d] ifn=%s tfn=%s\n", srcType.Name(), i, orig.Operand(2).Name(), orig.Operand(3).Name())
		}
		name := e.cloneConst(orig.Operand(0))
		mtype := e.cloneConst(orig.Operand(1))
		methods[i] = llvm.ConstNamedStruct(elemTy, []llvm.Value{
			name,
			mtype,
			unreachableMethod,
			unreachableMethod,
		})
	}
	fields[fieldCount-1] = llvm.ConstArray(elemTy, methods)

	dstType.SetInitializer(constStructOfType(init.Type(), fields))
	dstType.SetGlobalConstant(true)
	dstType.SetLinkage(llvm.ExternalLinkage)
	copyGlobalAttrs(dstType, srcType)
}

func (e *overrideEmitter) unreachableMethod() llvm.Value {
	fn := e.dst.NamedFunction(unreachableMethodName)
	if fn.IsNil() {
		fn = llvm.AddFunction(e.dst, unreachableMethodName,
			llvm.FunctionType(e.dst.Context().VoidType(), nil, false))
	}
	return fn
}

func (e *overrideEmitter) ensureOverrideGlobal(src llvm.Value) llvm.Value {
	name := src.Name()
	dst := e.dst.NamedGlobal(name)
	if dst.IsNil() {
		dst = llvm.AddGlobal(e.dst, src.GlobalValueType(), name)
	}
	e.values[src] = dst
	return dst
}

func (e *overrideEmitter) cloneConst(v llvm.Value) llvm.Value {
	if mapped, ok := e.values[v]; ok {
		return mapped
	}
	if gv := v.IsAGlobalValue(); !gv.IsNil() {
		return e.cloneGlobalValue(gv)
	}
	if !v.IsAConstantStruct().IsNil() {
		clone := constStructOfType(v.Type(), e.cloneOperands(v))
		e.values[v] = clone
		return clone
	}
	return v
}

func (e *overrideEmitter) cloneOperands(v llvm.Value) []llvm.Value {
	ops := make([]llvm.Value, v.OperandsCount())
	for i := range ops {
		ops[i] = e.cloneConst(v.Operand(i))
	}
	return ops
}

func (e *overrideEmitter) cloneGlobalValue(v llvm.Value) llvm.Value {
	// Rebind a source-module function reference to a declaration in dst. The
	// function body remains in its package object and resolves by name at link
	// time; the override initializer only needs a destination-owned reference
	// with the same function type.
	if fn := v.IsAFunction(); !fn.IsNil() {
		dstFn := e.dst.NamedFunction(fn.Name())
		if dstFn.IsNil() {
			dstFn = llvm.AddFunction(e.dst, fn.Name(), fn.GlobalValueType())
		}
		e.values[v] = dstFn
		return dstFn
	}
	if gv := v.IsAGlobalVariable(); !gv.IsNil() {
		return e.cloneGlobalVariable(gv)
	}
	panic("dcepass: unsupported global value")
}

func (e *overrideEmitter) cloneGlobalVariable(src llvm.Value) llvm.Value {
	if mapped, ok := e.values[src]; ok {
		return mapped
	}
	name := src.Name()
	if name != "" && !isLocalLinkage(src.Linkage()) {
		dst := e.dst.NamedGlobal(name)
		if dst.IsNil() {
			dst = llvm.AddGlobal(e.dst, src.GlobalValueType(), name)
			dst.SetLinkage(llvm.ExternalLinkage)
		}
		e.values[src] = dst
		return dst
	}

	dst := llvm.AddGlobal(e.dst, src.GlobalValueType(), "")
	e.values[src] = dst
	copyGlobalAttrs(dst, src)
	dst.SetLinkage(src.Linkage())
	dst.SetGlobalConstant(src.IsGlobalConstant())
	if init := src.Initializer(); !init.IsNil() {
		dst.SetInitializer(e.cloneConst(init))
	}
	return dst
}

func methodArray(init llvm.Value) (llvm.Value, llvm.Type, bool) {
	if init.IsNil() || init.OperandsCount() == 0 {
		return llvm.Value{}, llvm.Type{}, false
	}
	methodsVal := init.Operand(init.OperandsCount() - 1)
	if methodsVal.Type().TypeKind() != llvm.ArrayTypeKind {
		return llvm.Value{}, llvm.Type{}, false
	}
	elemTy := methodsVal.Type().ElementType()
	if elemTy.TypeKind() != llvm.StructTypeKind || elemTy.StructElementTypesCount() != 4 {
		return llvm.Value{}, llvm.Type{}, false
	}
	if elemTy.StructName() != abiMethodTypeName {
		return llvm.Value{}, llvm.Type{}, false
	}
	return methodsVal, elemTy, true
}

func liveSlotSet(slots []int) map[int]bool {
	out := make(map[int]bool, len(slots))
	for _, slot := range slots {
		out[slot] = true
	}
	return out
}

func copyGlobalAttrs(dst, src llvm.Value) {
	dst.SetVisibility(src.Visibility())
	dst.SetThreadLocal(src.IsThreadLocal())
	if align := src.Alignment(); align > 0 {
		dst.SetAlignment(align)
	}
}

func isLocalLinkage(linkage llvm.Linkage) bool {
	return linkage == llvm.PrivateLinkage || linkage == llvm.InternalLinkage
}

func constStructOfType(typ llvm.Type, fields []llvm.Value) llvm.Value {
	if typ.StructName() != "" {
		return llvm.ConstNamedStruct(typ, fields)
	}
	return llvm.ConstStruct(fields, typ.IsStructPacked())
}

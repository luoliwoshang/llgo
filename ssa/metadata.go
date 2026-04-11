package ssa

import (
	"go/token"
	"strings"

	"github.com/goplus/llvm"
)

const (
	// llgoUseIfaceMetadata is a module-level named metadata table whose rows are
	// {owner name, concrete type name}. Each row means that if the named owner
	// function is reachable, the named concrete type should enter the UsedInIface
	// semantic state during deadcode analysis. Metadata stores only symbol names
	// so emission does not force extra LLVM globals into the module.
	llgoUseIfaceMetadata = "llgo.useiface"
	// llgoUseIfaceMethodMetadata is a module-level named metadata table whose
	// rows are {owner name, interface type name, normalized method name, mtyp
	// name}. Each row means that if the named owner function is reachable, the
	// referenced interface method demand should participate in later method
	// matching.
	llgoUseIfaceMethodMetadata = "llgo.useifacemethod"
	// llgoInterfaceInfoMetadata is a module-level named metadata table whose rows
	// are {interface type name, [normalized method name, mtyp name]...}. Each
	// row describes one interface type's full method set in declaration order.
	llgoInterfaceInfoMetadata = "llgo.interfaceinfo"
	// llgoMethodInfoMetadata is a module-level named metadata table whose rows
	// are {concrete type name, [index, normalized method name, mtyp name, ifn
	// name, tfn name]...}. Each row describes one concrete type's full method
	// table in canonical abi.Method order.
	llgoMethodInfoMetadata = "llgo.methodinfo"
	// llgoUseNamedMethodMetadata is a module-level named metadata table whose
	// rows are {owner name, normalized method name}. Each row means that if the
	// named owner function is reachable, deadcode analysis should treat the named
	// method as requested by MethodByName-like semantics.
	llgoUseNamedMethodMetadata = "llgo.usenamedmethod"
	// llgoReflectMethodMetadata is a module-level named metadata table whose rows
	// are {owner name}. Each row means that if the named owner function is
	// reachable, deadcode analysis should conservatively keep reachable exported
	// methods for reflection-driven method lookup.
	llgoReflectMethodMetadata = "llgo.reflectmethod"
)

type semMetaEmitter struct {
	seen map[string]struct{}
}

func newSemMetaEmitter() *semMetaEmitter {
	return &semMetaEmitter{
		seen: make(map[string]struct{}),
	}
}

func (e *semMetaEmitter) add(mod llvm.Module, table, key string, fields ...llvm.Metadata) {
	fullKey := table + ":" + key
	if _, ok := e.seen[fullKey]; ok {
		return
	}
	e.seen[fullKey] = struct{}{}
	mod.AddNamedMetadataOperand(table, mod.Context().MDNode(fields))
}

func metadataKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func metadataString(ctx llvm.Context, s string) llvm.Metadata {
	return ctx.MDString(s)
}

func metadataInt32(ctx llvm.Context, i int) llvm.Metadata {
	return llvm.ConstInt(ctx.Int32Type(), uint64(i), false).ConstantAsMetadata()
}

func (p Package) emitUseIface(owner, target string) {
	ctx := p.mod.Context()
	p.semMetaEmitter.add(
		p.mod,
		llgoUseIfaceMetadata,
		metadataKey(owner, target),
		metadataString(ctx, owner),
		metadataString(ctx, target),
	)
}

func (p Package) emitUseIfaceMethod(owner, target, name, mtyp string) {
	ctx := p.mod.Context()
	p.semMetaEmitter.add(
		p.mod,
		llgoUseIfaceMethodMetadata,
		metadataKey(owner, target, name, mtyp),
		metadataString(ctx, owner),
		metadataString(ctx, target),
		metadataString(ctx, name),
		metadataString(ctx, mtyp),
	)
}

type interfaceInfoMethod struct {
	Name  string
	MType string
}

func (p Package) emitInterfaceInfo(target string, methods []interfaceInfoMethod) {
	if target == "" || len(methods) == 0 {
		return
	}
	ctx := p.mod.Context()
	fields := make([]llvm.Metadata, 0, 1+len(methods)*2)
	fields = append(fields, metadataString(ctx, target))
	for _, method := range methods {
		fields = append(
			fields,
			metadataString(ctx, method.Name),
			metadataString(ctx, method.MType),
		)
	}
	p.semMetaEmitter.add(
		p.mod,
		llgoInterfaceInfoMetadata,
		target,
		fields...,
	)
}

type methodInfoSlot struct {
	Index int
	Name  string
	MType string
	IFn   string
	TFn   string
}

func (p Package) emitMethodInfo(typeSym string, slots []methodInfoSlot) {
	if typeSym == "" {
		return
	}
	ctx := p.mod.Context()
	fields := make([]llvm.Metadata, 0, 1+len(slots)*5)
	fields = append(fields, metadataString(ctx, typeSym))
	for _, slot := range slots {
		fields = append(
			fields,
			metadataInt32(ctx, slot.Index),
			metadataString(ctx, slot.Name),
			metadataString(ctx, slot.MType),
			metadataString(ctx, slot.IFn),
			metadataString(ctx, slot.TFn),
		)
	}
	p.semMetaEmitter.add(
		p.mod,
		llgoMethodInfoMetadata,
		typeSym,
		fields...,
	)
}

func (p Package) emitUseNamedMethod(owner, name string) {
	if !token.IsExported(name) && p.Path() != "" {
		name = p.Path() + "." + name
	}
	ctx := p.mod.Context()
	p.semMetaEmitter.add(
		p.mod,
		llgoUseNamedMethodMetadata,
		metadataKey(owner, name),
		metadataString(ctx, owner),
		metadataString(ctx, name),
	)
}

func (p Package) emitReflectMethod(owner string) {
	ctx := p.mod.Context()
	p.semMetaEmitter.add(
		p.mod,
		llgoReflectMethodMetadata,
		owner,
		metadataString(ctx, owner),
	)
}

// EmitUseNamedMethod records a MethodByName-like exact method-name demand for
// later whole-program deadcode analysis.
func (p Package) EmitUseNamedMethod(owner, name string) {
	p.emitUseNamedMethod(owner, name)
}

// EmitReflectMethod records a conservative reflection marker for later
// whole-program deadcode analysis.
func (p Package) EmitReflectMethod(owner string) {
	p.emitReflectMethod(owner)
}

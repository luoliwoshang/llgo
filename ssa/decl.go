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

package ssa

import (
	"go/types"
	"log"
	"strconv"

	"github.com/goplus/llvm"
)

// -----------------------------------------------------------------------------

const (
	NameValist = "__llgo_va_list"
)

func VArg() *types.Var {
	return types.NewParam(0, nil, NameValist, types.NewSlice(tyAny))
}

// -----------------------------------------------------------------------------

type aNamedConst struct {
}

// A NamedConst is a Member of a Package representing a package-level
// named constant.
//
// Pos() returns the position of the declaring ast.ValueSpec.Names[*]
// identifier.
//
// NB: a NamedConst is not a Value; it contains a constant Value, which
// it augments with the name and position of its 'const' declaration.
type NamedConst = *aNamedConst

/*
// NewConst creates a new named constant.
func (p Package) NewConst(name string, val constant.Value) NamedConst {
	return &aNamedConst{}
}
*/

// -----------------------------------------------------------------------------

type aGlobal struct {
	Expr
	//array bool
}

// A Global is a named Value holding the address of a package-level
// variable.
type Global = *aGlobal

// NewVar 创建一个新的全局变量
func (p Package) NewVar(name string, typ types.Type, bg Background) Global {
	// bg: InGo, InC
	if v, ok := p.vars[name]; ok {
		// 如果已经存在于当前的package，则直接返回
		return v
	}

	t := p.Prog.Type(typ, bg)  // 获得aType，其中包含了LLVM的类型和原始的类型
	return p.doNewVar(name, t) // 根据该类型创建一个新的global变量（LLVM）
}

// NewVarEx creates a new global variable.
func (p Package) NewVarEx(name string, t Type) Global {
	if v, ok := p.vars[name]; ok {
		return v
	}
	return p.doNewVar(name, t)
}

// 创建一个全局变量，并且存放于package的vars中
func (p Package) doNewVar(name string, t Type) Global {
	typ := p.Prog.Elem(t).ll
	gbl := llvm.AddGlobal(p.mod, typ, name)
	alignment := p.Prog.td.ABITypeAlignment(typ)
	gbl.SetAlignment(alignment)
	ret := &aGlobal{Expr{gbl, t}}
	p.vars[name] = ret
	return ret
}

// VarOf returns a global variable by name.
func (p Package) VarOf(name string) Global {
	return p.vars[name]
}

// Init initializes the global variable with the given value.
func (g Global) Init(v Expr) {
	g.impl.SetInitializer(v.impl)
}

func (g Global) InitNil() {
	g.impl.SetInitializer(llvm.ConstNull(g.impl.GlobalValueType()))
}

// -----------------------------------------------------------------------------

// Function represents the parameters, results, and code of a function
// or method.
//
// If Blocks is nil, this indicates an external function for which no
// Go source code is available.  In this case, FreeVars, Locals, and
// Params are nil too.  Clients performing whole-program analysis must
// handle external functions specially.
//
// Blocks contains the function's control-flow graph (CFG).
// Blocks[0] is the function entry point; block order is not otherwise
// semantically significant, though it may affect the readability of
// the disassembly.
// To iterate over the blocks in dominance order, use DomPreorder().
//
// Recover is an optional second entry point to which control resumes
// after a recovered panic.  The Recover block may contain only a return
// statement, preceded by a load of the function's named return
// parameters, if any.
//
// A nested function (Parent()!=nil) that refers to one or more
// lexically enclosing local variables ("free variables") has FreeVars.
// Such functions cannot be called directly but require a
// value created by MakeClosure which, via its Bindings, supplies
// values for these parameters.
//
// If the function is a method (Signature.Recv() != nil) then the first
// element of Params is the receiver parameter.
//
// A Go package may declare many functions called "init".
// For each one, Object().Name() returns "init" but Name() returns
// "init#1", etc, in declaration order.
//
// Pos() returns the declaring ast.FuncLit.Type.Func or the position
// of the ast.FuncDecl.Name, if the function was explicit in the
// source.  Synthetic wrappers, for which Synthetic != "", may share
// the same position as the function they wrap.
// Syntax.Pos() always returns the position of the declaring "func" token.
//
// Type() returns the function's Signature.
//
// A generic function is a function or method that has uninstantiated type
// parameters (TypeParams() != nil). Consider a hypothetical generic
// method, (*Map[K,V]).Get. It may be instantiated with all ground
// (non-parameterized) types as (*Map[string,int]).Get or with
// parameterized types as (*Map[string,U]).Get, where U is a type parameter.
// In both instantiations, Origin() refers to the instantiated generic
// method, (*Map[K,V]).Get, TypeParams() refers to the parameters [K,V] of
// the generic method. TypeArgs() refers to [string,U] or [string,int],
// respectively, and is nil in the generic method.
type aFunction struct {
	Expr
	Pkg  Package
	Prog Program

	blks []BasicBlock // 基本块数组
	blks []BasicBlock // 基本块数组

	defer_ *aDefer
	recov  BasicBlock

	params   []Type
	freeVars Expr
	base     int // base = 1 if hasFreeVars; base = 0 otherwise
	hasVArg  bool
}

// Function represents a function or method.
type Function = *aFunction

// 创建一个新的Func
// 创建一个新的Func
func (p Package) NewFunc(name string, sig *types.Signature, bg Background) Function {
	return p.NewFuncEx(name, sig, bg, false)
}

// 创建一个新的Func
// 创建一个新的Func
func (p Package) NewFuncEx(name string, sig *types.Signature, bg Background, hasFreeVars bool) Function {
	// 如果存在于当前的package，则直接返回
	// 如果存在于当前的package，则直接返回
	if v, ok := p.fns[name]; ok {
		return v
	}
	t := p.Prog.FuncDecl(sig, bg)
	if debugInstr {
		log.Println("NewFunc", name, t.raw.Type, "hasFreeVars:", hasFreeVars)
	}
	fn := llvm.AddFunction(p.mod, name, t.ll)
	ret := newFunction(fn, t, p, p.Prog, hasFreeVars)
	p.fns[name] = ret
	return ret
}

// 通过名称获得函数，p.fns[name]
func (p Package) FuncOf(name string) Function {
	return p.fns[name]
}

func newFunction(fn llvm.Value, t Type, pkg Package, prog Program, hasFreeVars bool) Function {
	params, hasVArg := newParams(t, prog)
	base := 0
	if hasFreeVars {
		base = 1
	}
	return &aFunction{
		Expr:    Expr{fn, t},
		Pkg:     pkg,
		Prog:    prog,
		params:  params,
		base:    base,
		hasVArg: hasVArg,
	}
}

func newParams(fn Type, prog Program) (params []Type, hasVArg bool) {
	sig := fn.raw.Type.(*types.Signature)
	in := sig.Params()
	if n := in.Len(); n > 0 {
		if hasVArg = sig.Variadic(); hasVArg {
			n--
		}
		params = make([]Type, n)
		for i := 0; i < n; i++ {
			params[i] = prog.rawType(in.At(i).Type())
		}
	}
	return
}

// Name returns the function's name.
func (p Function) Name() string {
	return p.impl.Name()
}

// Params returns the function's ith parameter.
func (p Function) Param(i int) Expr {
	i += p.base // skip if hasFreeVars
	return Expr{p.impl.Param(i), p.params[i]}
}

func (p Function) closureCtx(b Builder) Expr {
	if p.freeVars.IsNil() {
		if p.base == 0 {
			panic("ssa: function has no free variables")
		}
		ptr := Expr{p.impl.Param(0), p.params[0]}
		p.freeVars = b.Load(ptr)
	}
	return p.freeVars
}

// FreeVar returns the function's ith free variable.
func (p Function) FreeVar(b Builder, i int) Expr {
	ctx := p.closureCtx(b)
	return b.getField(ctx, i)
}

// NewBuilder 创建了一个新的函数Builder，可以向函数体中插入指令
// SetInsertPointAtEnd 是 LLVM C++ API 中的一个方法,它用于设置 Builder 的当前插入点(insertion point)。
// 插入点决定了当使用 Builder 添加新的指令时,这些指令将被插入到哪里。
func (p Function) NewBuilder() Builder {
	prog := p.Prog
	b := prog.ctx.NewBuilder()
	// TODO(xsw): Finalize may cause panic, so comment it.
	// b.Finalize()
	return &aBuilder{b, nil, p, p.Pkg, prog}
}

// HasBody reports whether the function has a body.
func (p Function) HasBody() bool {
	return len(p.blks) > 0
}

// MakeBody 创建函数的nblk个基本块，以及创建一个新的Builder指向第一个基本块 #0
// MakeBody 创建函数的nblk个基本块，以及创建一个新的Builder指向第一个基本块 #0
func (p Function) MakeBody(nblk int) Builder {
	p.MakeBlocks(nblk)                     // 为函数创建nblk个基本块
	b := p.NewBuilder()                    // 创建一个Builder
	b.blk = p.blks[0]                      //将 Builder 的当前基本块设置为刚刚创建的第一个基本块。
	b.impl.SetInsertPointAtEnd(b.blk.last) //插入点决定了当使用 Builder 添加新的指令时,这些指令将被插入到哪里。
	return b
}

// MakeBlocks 为函数新创建nblk个基本块，返回新创建的那些基本块
// MakeBlocks 为函数新创建nblk个基本块，返回新创建的那些基本块
func (p Function) MakeBlocks(nblk int) []BasicBlock {
	n := len(p.blks) // 当前函数的block个数
	n := len(p.blks) // 当前函数的block个数
	if n == 0 {
		// 创建函数的nblk个block的空间
		// 创建函数的nblk个block的空间
		p.blks = make([]BasicBlock, 0, nblk)
	}
	for i := 0; i < nblk; i++ {
		// 创建函数的nblk个block，如果
		// 创建函数的nblk个block，如果
		p.addBlock(n + i)
	}
	return p.blks[n:] //返回新创建的block从n开始
	return p.blks[n:] //返回新创建的block从n开始
}

// 为函数创建一个新的基本块
// 为函数创建一个新的基本块
func (p Function) addBlock(idx int) BasicBlock {
	label := "_llgo_" + strconv.Itoa(idx)    //为基本块创建label
	blk := llvm.AddBasicBlock(p.impl, label) //创建基本块
	ret := &aBasicBlock{blk, blk, p, idx}    //创建基本块对象
	label := "_llgo_" + strconv.Itoa(idx)    //为基本块创建label
	blk := llvm.AddBasicBlock(p.impl, label) //创建基本块
	ret := &aBasicBlock{blk, blk, p, idx}    //创建基本块对象
	p.blks = append(p.blks, ret)
	return ret
}

// MakeBlock creates a new basic block for the function.
func (p Function) MakeBlock() BasicBlock {
	return p.addBlock(len(p.blks))
}

// Block returns the ith basic block of the function.
func (p Function) Block(idx int) BasicBlock {
	return p.blks[idx]
}

// SetRecover sets the recover block for the function.
func (p Function) SetRecover(blk BasicBlock) {
	p.recov = blk
}

// -----------------------------------------------------------------------------

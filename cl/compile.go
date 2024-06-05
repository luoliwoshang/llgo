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

package cl

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"os"
	"sort"

	"github.com/goplus/llgo/cl/blocks"
	"github.com/goplus/llgo/internal/typepatch"
	"golang.org/x/tools/go/ssa"

	llssa "github.com/goplus/llgo/ssa"
)

// -----------------------------------------------------------------------------

type dbgFlags = int

const (
	DbgFlagInstruction dbgFlags = 1 << iota
	DbgFlagGoSSA

	DbgFlagAll = DbgFlagInstruction | DbgFlagGoSSA
)

var (
	debugInstr bool
	debugGoSSA bool
)

// SetDebug sets debug flags.
func SetDebug(dbgFlags dbgFlags) {
	debugInstr = (dbgFlags & DbgFlagInstruction) != 0
	debugGoSSA = (dbgFlags & DbgFlagGoSSA) != 0
}

// -----------------------------------------------------------------------------

type instrOrValue interface {
	ssa.Instruction
	ssa.Value
}

// 在PkgNoInit前的种类的包，都是需要执行Init的PkgNormal，PkgLLGo，PkgPyModule
const (
	PkgNormal     = iota // 正常的包，非LLGO包（不包含LLGoPackage常量）
	PkgLLGo              // 一个基本的LLGo包
	PkgPyModule          //TODO: py.<module>
	PkgNoInit            //TODO: noinit: a package that don't need to be initialized
	PkgDeclOnly          //TODO:(了解这里的应用) decl: a package that only have declarations
	PkgLinkIR            //TODO: link llvm ir (.ll)
	PkgLinkExtern        //TODO: link external object (.a/.so/.dll/.dylib/etc.)
	// PkgLinkBitCode // link bitcode (.bc)
)

type pkgInfo struct {
	kind int
}

type none struct{}

type context struct {
	prog   llssa.Program
	pkg    llssa.Package
	fn     llssa.Function
	fset   *token.FileSet
	goProg *ssa.Program
	goTyps *types.Package //TODO: 这个goTyps是何时导入的
	goPkg  *ssa.Package
	pyMod  string
	link   map[string]string // pkgPath.nameInPkg => linkname
	skips  map[string]none
	loaded map[*types.Package]*pkgInfo // loaded packages
	bvals  map[ssa.Value]llssa.Expr    // block values go ssa -> llvm ir
	vargs  map[*ssa.Alloc][]llssa.Expr // varargs

	patches  Patches
	blkInfos []blocks.Info

	inits []func() // 存储初始化函数
	phis  []func()

	state   pkgState
	inCFunc bool
	skipall bool
}

type pkgState byte

const (
	pkgNormal pkgState = iota
	pkgHasPatch
	pkgInPatch

	pkgFNoOldInit = 0x80 // flag if no initFnNameOld
)

func (p *context) inMain(instr ssa.Instruction) bool {
	return p.fn.Name() == "main"
}

func (p *context) compileType(pkg llssa.Package, t *ssa.Type) {
	tn := t.Object().(*types.TypeName)
	if tn.IsAlias() { // don't need to compile alias type
		return
	}
	tnName := tn.Name()
	typ := tn.Type()
	name := llssa.FullName(tn.Pkg(), tnName)
	if ignoreName(name) {
		return
	}
	if debugInstr {
		log.Println("==> NewType", name, typ)
	}
	p.compileMethods(pkg, typ)
	p.compileMethods(pkg, types.NewPointer(typ))
}

func (p *context) compileMethods(pkg llssa.Package, typ types.Type) {
	prog := p.goProg
	mthds := prog.MethodSets.MethodSet(typ)
	for i, n := 0, mthds.Len(); i < n; i++ {
		mthd := mthds.At(i)
		if ssaMthd := prog.MethodValue(mthd); ssaMthd != nil {
			p.compileFuncDecl(pkg, ssaMthd)
		}
	}
}

// 编译全局变量
func (p *context) compileGlobal(pkg llssa.Package, gbl *ssa.Global) {
	typ := globalType(gbl)
	name, vtype, define := p.varName(gbl.Pkg.Pkg, gbl)
	if vtype == pyVar || ignoreName(name) || checkCgo(gbl.Name()) {
		return
	}
	if debugInstr {
		log.Println("==> NewVar", name, typ)
	}
	g := pkg.NewVar(name, typ, llssa.Background(vtype))
	if define {
		g.InitNil()
	}
}

func makeClosureCtx(pkg *types.Package, vars []*ssa.FreeVar) *types.Var {
	n := len(vars)
	flds := make([]*types.Var, n)
	for i, v := range vars {
		flds[i] = types.NewField(token.NoPos, pkg, v.Name(), v.Type(), false)
	}
	t := types.NewPointer(types.NewStruct(flds, nil))
	return types.NewParam(token.NoPos, pkg, "__llgo_ctx", t)
}

var (
	argvTy = types.NewPointer(types.NewPointer(types.Typ[types.Int8]))
)

// 将ssa.Function编译为llssa.Function，在p.Inits中注册 编译该函数基本块的函数
func (p *context) compileFuncDecl(pkg llssa.Package, f *ssa.Function) (llssa.Function, llssa.PyObjRef, int) {
	pkgTypes, name, ftype := p.funcName(f, true) // ftype:inGo inC
	if ftype != goFunc {
		/*
			if ftype == pyFunc {
				// TODO(xsw): pyMod == ""
				fnName := pysymPrefix + p.pyMod + "." + name
				return nil, pkg.NewPyFunc(fnName, f.Signature, call), pyFunc
			}
		*/
		return nil, nil, ignoredFunc
	}
	sig := f.Signature
	state := p.state
	isInit := (f.Name() == "init" && sig.Recv() == nil)
	if isInit && state == pkgHasPatch {
		name = initFnNameOfHasPatch(name)
	}

	fn := pkg.FuncOf(name)
	if fn != nil && fn.HasBody() {
		return fn, nil, goFunc
	}

	var hasCtx = len(f.FreeVars) > 0
	if hasCtx {
		if debugInstr {
			log.Println("==> NewClosure", name, "type:", sig)
		}
		ctx := makeClosureCtx(pkgTypes, f.FreeVars)
		sig = llssa.FuncAddCtx(ctx, sig)
	} else {
		if debugInstr {
			log.Println("==> NewFunc", name, "type:", sig.Recv(), sig, "ftype:", ftype)
		}
	}
	if fn == nil {
		if name == "main" { // 对main包的main函数进行处理，添加对c的命令行参数接受
			argc := types.NewParam(token.NoPos, pkgTypes, "", types.Typ[types.Int32]) // c:argc：命令行参数
			argv := types.NewParam(token.NoPos, pkgTypes, "", argvTy)                 // c:argv 命令行参数数组，第一项是程序名（还未确定）
			params := types.NewTuple(argc, argv)
			ret := types.NewParam(token.NoPos, pkgTypes, "", p.prog.CInt().RawType()) // c语言中main函数的返回值
			results := types.NewTuple(ret)
			sig = types.NewSignatureType(nil, nil, nil, params, results, false) // 生成函数签名
		}
		fn = pkg.NewFuncEx(name, sig, llssa.Background(ftype), hasCtx)
	}
	// 对于存在函数体的函数，进行编译
	if nblk := len(f.Blocks); nblk > 0 {
		fn.MakeBlocks(nblk)   // to set fn.HasBody() = true
		if f.Recover != nil { // set recover block
			fn.SetRecover(fn.Block(f.Recover.Index))
		}
		p.inits = append(p.inits, func() {
			p.fn = fn
			p.state = state // restore pkgState when compiling funcBody
			defer func() {
				p.fn = nil
			}()
			p.phis = nil // TODO: 不清楚这是什么
			if debugGoSSA {
				f.WriteTo(os.Stderr)
			}
			if debugInstr {
				log.Println("==> FuncBody", name)
			}
			b := fn.NewBuilder()                     // 创建一个函数的构建器
			p.bvals = make(map[ssa.Value]llssa.Expr) // TODO: 不清楚这是为什么
			off := make([]int, len(f.Blocks))        // 获得原始函数基本块数量
			for i, block := range f.Blocks {         //将每一个基本块的Phi指令都进行编译
				off[i] = p.compilePhis(b, block) // 获得每个基本块的Phi指令的数量
			}
			p.blkInfos = blocks.Infos(f.Blocks)
			i := 0
			for {
				block := f.Blocks[i] // 根据索引获得原本函数的每一个基本块
				doMainInit := (i == 0 && name == "main")
				doModInit := (i == 1 && isInit)
				p.compileBlock(b, block, off[i], doMainInit, doModInit)
				if i = p.blkInfos[i].Next; i < 0 {
					break
				}
			}
			for _, phi := range p.phis { // TODO: 执行phi指令
				phi()
			}
			b.EndBuild()
		})
		for _, af := range f.AnonFuncs {
			p.compileFuncDecl(pkg, af)
		}
	}
	return fn, nil, goFunc
}

type blockInfo struct {
	kind llssa.DoAction
	next int
}

func blockInfos(blks []*ssa.BasicBlock) []blockInfo {
	n := len(blks)
	infos := make([]blockInfo, n)
	for i := range blks {
		next := i + 1
		if next >= n {
			next = -1
		}
		infos[i] = blockInfo{kind: llssa.DeferInCond, next: next}
	}
	return infos
}

// funcOf returns a function by name and set ftype = goFunc, cFunc, etc.
// or returns nil and set ftype = llgoCstr, llgoAlloca, llgoUnreachable, etc.
func (p *context) funcOf(fn *ssa.Function) (aFn llssa.Function, pyFn llssa.PyObjRef, ftype int) {
	pkgTypes, name, ftype := p.funcName(fn, false)
	switch ftype {
	case pyFunc:
		if kind, mod := pkgKindByScope(pkgTypes.Scope()); kind == PkgPyModule {
			pkg := p.pkg
			fnName := pysymPrefix + mod + "." + name
			if pyFn = pkg.PyObjOf(fnName); pyFn == nil {
				pyFn = pkg.PyNewFunc(fnName, fn.Signature, true)
			}
			return
		}
		ftype = ignoredFunc
	case llgoInstr:
		switch name {
		case "cstr":
			ftype = llgoCstr
		case "advance":
			ftype = llgoAdvance
		case "index":
			ftype = llgoIndex
		case "alloca":
			ftype = llgoAlloca
		case "allocaCStr":
			ftype = llgoAllocaCStr
		case "stringData":
			ftype = llgoStringData
		case "pyList":
			ftype = llgoPyList
		case "unreachable":
			ftype = llgoUnreachable
		default:
			panic("unknown llgo instruction: " + name)
		}
	default:
		pkg := p.pkg
		if aFn = pkg.FuncOf(name); aFn == nil {
			if len(fn.FreeVars) > 0 {
				return nil, nil, ignoredFunc
			}
			sig := fn.Signature
			aFn = pkg.NewFuncEx(name, sig, llssa.Background(ftype), false)
		}
	}
	return
}

// 编译函数的某个基本块，对块中的每一个指令进行编译
func (p *context) compileBlock(b llssa.Builder, block *ssa.BasicBlock, n int, doMainInit, doModInit bool) llssa.BasicBlock {
	var last int
	var pyModInit bool
	var prog = p.prog
	var pkg = p.pkg
	var fn = p.fn
	var instrs = block.Instrs[n:]
	var ret = fn.Block(block.Index)
	b.SetBlock(ret)
	if doModInit {
		if pyModInit = p.pyMod != ""; pyModInit {
			last = len(instrs) - 1
			instrs = instrs[:last]
		} else {
			// TODO(xsw): confirm pyMod don't need to call AfterInit
			p.inits = append(p.inits, func() {
				pkg.AfterInit(b, ret)
			})
		}
	} else if doMainInit {
		argc := pkg.NewVar("__llgo_argc", types.NewPointer(types.Typ[types.Int32]), llssa.InC)
		argv := pkg.NewVar("__llgo_argv", types.NewPointer(argvTy), llssa.InC)
		argc.InitNil()
		argv.InitNil()
		b.Store(argc.Expr, fn.Param(0))
		b.Store(argv.Expr, fn.Param(1))
		callRuntimeInit(b, pkg)              // 调用运行时的初始化函数
		b.Call(pkg.FuncOf("main.init").Expr) // 创建函数调用指令，调用main.init函数
	}
	for i, instr := range instrs {
		if i == 1 && doModInit && p.state == pkgInPatch {
			initFnNameOld := initFnNameOfHasPatch(p.fn.Name())
			fnOld := pkg.NewFunc(initFnNameOld, llssa.NoArgsNoRet, llssa.InC)
			b.Call(fnOld.Expr)
		}
		p.compileInstr(b, instr)
	}
	if pyModInit {
		jump := block.Instrs[n+last].(*ssa.Jump)
		jumpTo := p.jumpTo(jump)
		modPath := p.pyMod
		modName := pysymPrefix + modPath
		modPtr := pkg.PyNewModVar(modName, true).Expr
		mod := b.Load(modPtr)
		cond := b.BinOp(token.NEQ, mod, prog.Nil(mod.Type))
		newBlk := fn.MakeBlock()
		b.If(cond, jumpTo, newBlk)
		b.SetBlockEx(newBlk, llssa.AtEnd, false)
		b.Store(modPtr, b.PyImportMod(modPath))
		b.Jump(jumpTo)
	}
	return ret
}

const (
	RuntimeInit = llssa.PkgRuntime + ".init"
)

// 当前位置插入运行时初始化函数
func callRuntimeInit(b llssa.Builder, pkg llssa.Package) {
	// 在IR中声明这个函数 declare void @"github.com/goplus/llgo/internal/runtime.init"()
	fn := pkg.NewFunc(RuntimeInit, llssa.NoArgsNoRet, llssa.InC) // don't need to convert runtime.init
	// 当前位置调用该函数 call void @"github.com/goplus/llgo/internal/runtime.init"()
	b.Call(fn.Expr)
}

func isAny(t types.Type) bool {
	if t, ok := t.(*types.Interface); ok {
		return t.Empty()
	}
	return false
}

func intVal(v ssa.Value) int64 {
	if c, ok := v.(*ssa.Const); ok {
		if iv, exact := constant.Int64Val(c.Value); exact {
			return iv
		}
	}
	panic("intVal: ssa.Value is not a const int")
}

func (p *context) isVArgs(v ssa.Value) (ret []llssa.Expr, ok bool) {
	switch v := v.(type) {
	case *ssa.Alloc:
		ret, ok = p.vargs[v] // varargs: this is a varargs index
	}
	return
}

func (p *context) checkVArgs(v *ssa.Alloc, t *types.Pointer) bool {
	if v.Comment == "varargs" { // this maybe a varargs allocation
		if arr, ok := t.Elem().(*types.Array); ok {
			if isAny(arr.Elem()) && isAllocVargs(p, v) {
				p.vargs[v] = make([]llssa.Expr, arr.Len())
				return true
			}
		}
	}
	return false
}

func isAllocVargs(ctx *context, v *ssa.Alloc) bool {
	refs := *v.Referrers()
	n := len(refs)
	lastref := refs[n-1]
	if i, ok := lastref.(*ssa.Slice); ok {
		if refs = *i.Referrers(); len(refs) == 1 {
			var call *ssa.CallCommon
			switch ref := refs[0].(type) {
			case *ssa.Call:
				call = &ref.Call
			case *ssa.Defer:
				call = &ref.Call
			case *ssa.Go:
				call = &ref.Call
			default:
				return false
			}
			return ctx.funcKind(call.Value) == fnHasVArg
		}
	}
	return false
}

func isPhi(i ssa.Instruction) bool {
	_, ok := i.(*ssa.Phi)
	return ok
}

func (p *context) compilePhis(b llssa.Builder, block *ssa.BasicBlock) int {
	fn := p.fn
	ret := fn.Block(block.Index)
	b.SetBlockEx(ret, llssa.AtEnd, false)
	if ninstr := len(block.Instrs); ninstr > 0 {
		if isPhi(block.Instrs[0]) {
			n := 1
			for n < ninstr && isPhi(block.Instrs[n]) {
				n++
			}
			rets := make([]llssa.Expr, n) // TODO(xsw): check to remove this
			for i := 0; i < n; i++ {
				iv := block.Instrs[i].(*ssa.Phi)
				rets[i] = p.compilePhi(b, iv)
			}
			for i := 0; i < n; i++ {
				iv := block.Instrs[i].(*ssa.Phi)
				p.bvals[iv] = rets[i]
			}
			return n
		}
	}
	return 0
}

func (p *context) compilePhi(b llssa.Builder, v *ssa.Phi) (ret llssa.Expr) {
	phi := b.Phi(p.prog.Type(v.Type(), llssa.InGo))
	ret = phi.Expr
	p.phis = append(p.phis, func() {
		preds := v.Block().Preds
		bblks := make([]llssa.BasicBlock, len(preds))
		for i, pred := range preds {
			bblks[i] = p.fn.Block(pred.Index)
		}
		edges := v.Edges
		phi.AddIncoming(b, bblks, func(i int, blk llssa.BasicBlock) llssa.Expr {
			b.SetBlockEx(blk, llssa.BeforeLast, false)
			return p.compileValue(b, edges[i])
		})
	})
	return
}

func (p *context) compileInstrOrValue(b llssa.Builder, iv instrOrValue, asValue bool) (ret llssa.Expr) {
	if asValue {
		if v, ok := p.bvals[iv]; ok {
			return v
		}
		log.Panicln("unreachable:", iv)
	}
	switch v := iv.(type) {
	case *ssa.Call:
		ret = p.call(b, llssa.Call, &v.Call)
	case *ssa.BinOp:
		x := p.compileValue(b, v.X)
		y := p.compileValue(b, v.Y)
		ret = b.BinOp(v.Op, x, y)
	case *ssa.UnOp: // 构建单目运算符
		x := p.compileValue(b, v.X)
		ret = b.UnOp(v.Op, x)
	case *ssa.ChangeType:
		t := v.Type()
		x := p.compileValue(b, v.X)
		ret = b.ChangeType(p.prog.Type(t, llssa.InGo), x)
	case *ssa.Convert:
		t := v.Type()
		x := p.compileValue(b, v.X)
		ret = b.Convert(p.prog.Type(t, llssa.InGo), x)
	case *ssa.FieldAddr:
		x := p.compileValue(b, v.X)
		ret = b.FieldAddr(x, v.Field)
	case *ssa.Alloc:
		t := v.Type().(*types.Pointer)
		if p.checkVArgs(v, t) { // varargs: this maybe a varargs allocation
			return
		}
		elem := p.prog.Type(t.Elem(), llssa.InGo)
		ret = b.Alloc(elem, v.Heap)
	case *ssa.IndexAddr:
		vx := v.X                       //获得变量
		if _, ok := p.isVArgs(vx); ok { // TODO:(了解这里是什么) varargs: this is a varargs index
			return
		}
		x := p.compileValue(b, vx)        // 获得对应的全局变量
		idx := p.compileValue(b, v.Index) // 获得常量表达式
		ret = b.IndexAddr(x, idx)         // 获得地址访问指针（不构建）
	case *ssa.Index:
		x := p.compileValue(b, v.X)
		idx := p.compileValue(b, v.Index)
		ret = b.Index(x, idx, func(e llssa.Expr) (ret llssa.Expr, zero bool) {
			if e == x {
				switch n := v.X.(type) {
				case *ssa.Const:
					zero = true
					return
				case *ssa.UnOp:
					return p.compileValue(b, n.X), false
				}
			}
			panic(fmt.Errorf("todo: addr of %v", e))
		})
	case *ssa.Lookup:
		x := p.compileValue(b, v.X)
		idx := p.compileValue(b, v.Index)
		ret = b.Lookup(x, idx, v.CommaOk)
	case *ssa.Slice:
		vx := v.X
		if _, ok := p.isVArgs(vx); ok { // varargs: this is a varargs slice
			return
		}
		var low, high, max llssa.Expr
		x := p.compileValue(b, vx)
		if v.Low != nil {
			low = p.compileValue(b, v.Low)
		}
		if v.High != nil {
			high = p.compileValue(b, v.High)
		}
		if v.Max != nil {
			max = p.compileValue(b, v.Max)
		}
		ret = b.Slice(x, low, high, max)
	case *ssa.MakeInterface:
		if refs := *v.Referrers(); len(refs) == 1 {
			switch ref := refs[0].(type) {
			case *ssa.Store:
				if va, ok := ref.Addr.(*ssa.IndexAddr); ok {
					if _, ok = p.isVArgs(va.X); ok { // varargs: this is a varargs store
						return
					}
				}
			case *ssa.Call:
				if fn, ok := ref.Call.Value.(*ssa.Function); ok {
					if _, _, ftype := p.funcOf(fn); ftype == llgoFuncAddr { // llgo.funcAddr
						return
					}
				}
			}
		}
		t := p.prog.Type(v.Type(), llssa.InGo)
		x := p.compileValue(b, v.X)
		ret = b.MakeInterface(t, x)
	case *ssa.MakeSlice:
		var nCap llssa.Expr
		t := p.prog.Type(v.Type(), llssa.InGo)
		nLen := p.compileValue(b, v.Len)
		if v.Cap != nil {
			nCap = p.compileValue(b, v.Cap)
		}
		ret = b.MakeSlice(t, nLen, nCap)
	case *ssa.MakeMap:
		var nReserve llssa.Expr
		t := p.prog.Type(v.Type(), llssa.InGo)
		if v.Reserve != nil {
			nReserve = p.compileValue(b, v.Reserve)
		}
		ret = b.MakeMap(t, nReserve)
	case *ssa.MakeClosure:
		fn := p.compileValue(b, v.Fn)
		bindings := p.compileValues(b, v.Bindings, 0)
		ret = b.MakeClosure(fn, bindings)
	case *ssa.TypeAssert:
		x := p.compileValue(b, v.X)
		t := p.prog.Type(v.AssertedType, llssa.InGo)
		ret = b.TypeAssert(x, t, v.CommaOk)
	case *ssa.Extract:
		x := p.compileValue(b, v.Tuple)
		ret = b.Extract(x, v.Index)
	case *ssa.Range:
		x := p.compileValue(b, v.X)
		ret = b.Range(x)
	case *ssa.Next:
		iter := p.compileValue(b, v.Iter)
		ret = b.Next(iter, v.IsString)
	case *ssa.ChangeInterface:
		t := v.Type()
		x := p.compileValue(b, v.X)
		ret = b.ChangeInterface(p.prog.Type(t, llssa.InGo), x)
	case *ssa.Field:
		x := p.compileValue(b, v.X)
		ret = b.Field(x, v.Field)
	default:
		panic(fmt.Sprintf("compileInstrAndValue: unknown instr - %T\n", iv))
	}
	p.bvals[iv] = ret
	return ret
}

func (p *context) jumpTo(v *ssa.Jump) llssa.BasicBlock {
	fn := p.fn
	succs := v.Block().Succs
	return fn.Block(succs[0].Index)
}

// 编译函数中的某个基本块中的指定指令
func (p *context) compileInstr(b llssa.Builder, instr ssa.Instruction) {
	if iv, ok := instr.(instrOrValue); ok { //TODO: BinOp符合这个条件
		p.compileInstrOrValue(b, iv, false)
		return
	}
	switch v := instr.(type) {
	case *ssa.Store: //存储指令
		va := v.Addr // 获得对应的IndexAddr表达式，对应某个指针
		if va, ok := va.(*ssa.IndexAddr); ok {
			if args, ok := p.isVArgs(va.X); ok { //TODO: 考虑这个情况 varargs: this is a varargs store
				idx := intVal(va.Index)
				val := v.Val
				if vi, ok := val.(*ssa.MakeInterface); ok {
					val = vi.X
				}
				args[idx] = p.compileValue(b, val)
				return
			}
		}
		ptr := p.compileValue(b, va)    //获取IndexAddr获得对应的指针
		val := p.compileValue(b, v.Val) //获取常量表达式
		b.Store(ptr, val)               //构建存储指令
	case *ssa.Jump:
		jmpb := p.jumpTo(v)
		b.Jump(jmpb)
	case *ssa.Return:
		var results []llssa.Expr
		if n := len(v.Results); n > 0 {
			results = make([]llssa.Expr, n)
			for i, r := range v.Results {
				results[i] = p.compileValue(b, r) // 如果返回内容的某个参数为函数形参的某一个，那么该项就会是形参的表达
			}
		}
		if p.inMain(instr) {
			results = make([]llssa.Expr, 1)
			results[0] = p.prog.IntVal(0, p.prog.CInt())
		}
		b.Return(results...)
	case *ssa.If:
		fn := p.fn                        //获得当前正在处理的LLVM func
		cond := p.compileValue(b, v.Cond) //获得条件表达式的结果的类型
		succs := v.Block().Succs          //获得这个指令对应的基本块的if true 和 else的基本块
		thenb := fn.Block(succs[0].Index) //获得if true的基本块
		elseb := fn.Block(succs[1].Index) // 获得else的基本块
		b.If(cond, thenb, elseb)          // 为该基本块创建对应的IF指令，此时仅仅构建了对应的块的IF跳转指令，对应块中还未生成对应的指令
	case *ssa.MapUpdate:
		m := p.compileValue(b, v.Map)
		key := p.compileValue(b, v.Key)
		val := p.compileValue(b, v.Value)
		b.MapUpdate(m, key, val)
	case *ssa.Defer:
		p.call(b, p.blkInfos[v.Block().Index].Kind, &v.Call)
	case *ssa.Go:
		p.call(b, llssa.Go, &v.Call)
	case *ssa.RunDefers:
		b.RunDefers()
	case *ssa.Panic:
		arg := p.compileValue(b, v.X)
		b.Panic(arg)
	default:
		panic(fmt.Sprintf("compileInstr: unknown instr - %T\n", instr))
	}
}

func (p *context) compileFunction(v *ssa.Function) (goFn llssa.Function, pyFn llssa.PyObjRef, kind int) {
	// TODO(xsw) v.Pkg == nil: means auto generated function?
	if v.Pkg == p.goPkg || v.Pkg == nil {
		// function in this package
		goFn, pyFn, kind = p.compileFuncDecl(p.pkg, v)
		if kind != ignoredFunc {
			return
		}
	}
	return p.funcOf(v)
}

func (p *context) compileValue(b llssa.Builder, v ssa.Value) llssa.Expr {
	if iv, ok := v.(instrOrValue); ok {
		return p.compileInstrOrValue(b, iv, true)
	}
	switch v := v.(type) {
	case *ssa.Parameter: //函数的返回值和形参都为该类型
		fn := v.Parent() // 获得参数对应的Func
		for idx, param := range fn.Params {
			if param == v {
				return p.fn.Param(idx) // 如果某个返回值正好与函数的形参对应，那么返回该形参（保证引用）
			}
		}
	case *ssa.Function:
		aFn, pyFn, _ := p.compileFunction(v)
		if aFn != nil {
			return aFn.Expr
		}
		return pyFn.Expr
	case *ssa.Global: // 从LLVM包中获得该全局变量的引用
		return p.varOf(b, v)
	case *ssa.Const: // 获得该常量对应的类型的表达式
		t := types.Default(v.Type())
		bg := llssa.InGo
		if p.inCFunc {
			bg = llssa.InC
		}
		return b.Const(v.Value, p.prog.Type(t, bg))
	case *ssa.FreeVar:
		fn := v.Parent()
		for idx, freeVar := range fn.FreeVars {
			if freeVar == v {
				return p.fn.FreeVar(b, idx)
			}
		}
	}
	panic(fmt.Sprintf("compileValue: unknown value - %T\n", v))
}

func (p *context) compileVArg(ret []llssa.Expr, b llssa.Builder, v ssa.Value) []llssa.Expr {
	_ = b
	switch v := v.(type) {
	case *ssa.Slice: // varargs: this is a varargs slice
		if args, ok := p.isVArgs(v.X); ok {
			return append(ret, args...)
		}
	case *ssa.Const:
		if v.Value == nil {
			return ret
		}
	}
	panic(fmt.Sprintf("compileVArg: unknown value - %T\n", v))
}

func (p *context) compileValues(b llssa.Builder, vals []ssa.Value, hasVArg int) []llssa.Expr {
	n := len(vals) - hasVArg
	ret := make([]llssa.Expr, n)
	for i := 0; i < n; i++ {
		ret[i] = p.compileValue(b, vals[i])
	}
	if hasVArg > 0 {
		ret = p.compileVArg(ret, b, vals[n])
	}
	return ret
}

// -----------------------------------------------------------------------------

// Patches is patches of some packages.
type Patches = map[string]*ssa.Package

// NewPackage compiles a Go package to LLVM IR package.
func NewPackage(prog llssa.Program, pkg *ssa.Package, files []*ast.File) (ret llssa.Package, err error) {
	type namedMember struct {
		name string
		val  ssa.Member
	}
	// pkg.Members 存放了所有的包级别的变量和函数
	members := make([]*namedMember, 0, len(pkg.Members))
	for name, v := range pkg.Members {
		members = append(members, &namedMember{name, v})
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].name < members[j].name
	})

	pkgProg := pkg.Prog // 正在分析的go程序
	pkgTypes := pkg.Pkg // 包中的package类型信息
	pkgName, pkgPath := pkgTypes.Name(), llssa.PathOf(pkgTypes)
	alt, hasPatch := patches[pkgPath]
	if hasPatch {
		pkgTypes = typepatch.Pkg(pkgTypes, alt.Pkg)
		pkg.Pkg = pkgTypes
		alt.Pkg = pkgTypes
	}
	if pkgPath == llssa.PkgRuntime {
		prog.SetRuntime(pkgTypes)
	}
	ret = prog.NewPackage(pkgName, pkgPath) //初始化

	ctx := &context{
		prog:    prog,
		pkg:     ret,
		fset:    pkgProg.Fset,
		goProg:  pkgProg,
		goTyps:  pkgTypes,
		goPkg:   pkg,
		patches: patches,
		link:    make(map[string]string),
		skips:   make(map[string]none),
		vargs:   make(map[*ssa.Alloc][]llssa.Expr),
		loaded: map[*types.Package]*pkgInfo{
			types.Unsafe: {kind: PkgDeclOnly}, // TODO(xsw): PkgNoInit or PkgDeclOnly?
		},
	}
	ctx.initPyModule()
	ctx.initFiles(pkgPath, files)

	if hasPatch {
		skips := ctx.skips
		ctx.skips = nil
		ctx.state = pkgInPatch
		if _, ok := skips["init"]; ok || ctx.skipall {
			ctx.state |= pkgFNoOldInit
		}
		processPkg(ctx, ret, alt)
		ctx.state = pkgHasPatch
		ctx.skips = skips
	}
	if !ctx.skipall {
		processPkg(ctx, ret, pkg)
	}
	for len(ctx.inits) > 0 {
		inits := ctx.inits
		ctx.inits = nil
		for _, ini := range inits {
			ini()
		}
	}
	return
}

func initFnNameOfHasPatch(name string) string {
	return name + "$hasPatch"
}

func processPkg(ctx *context, ret llssa.Package, pkg *ssa.Package) {
	type namedMember struct {
		name string
		val  ssa.Member
	}

	members := make([]*namedMember, 0, len(pkg.Members))
	skips := ctx.skips
	for name, v := range pkg.Members {
		if _, ok := skips[name]; !ok {
			members = append(members, &namedMember{name, v})
		}
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].name < members[j].name
	})

	for _, m := range members {
		member := m.val
		switch member := member.(type) {
		case *ssa.Function:
			// TypeParams 是泛型函数定义中声明的类型参数，TypeArgs是调用泛型函数时传递的类型参数
			// 暂时不处理这种情况
			if member.TypeParams() != nil || member.TypeArgs() != nil {
				// TODO(xsw): don't compile generic functions
				// Do not try to build generic (non-instantiated) functions.
				continue
			}
			ctx.compileFuncDecl(ret, member)
		case *ssa.Type:
			ctx.compileType(ret, member)
		case *ssa.Global:
			ctx.compileGlobal(ret, member)
		}
	}
}

func globalType(gbl *ssa.Global) types.Type {
	t := gbl.Type()
	if t, ok := t.(*types.Named); ok {
		o := t.Obj()
		if pkg := o.Pkg(); typepatch.IsPatched(pkg) {
			return gbl.Pkg.Pkg.Scope().Lookup(o.Name()).Type()
		}
	}
	return t
}

// -----------------------------------------------------------------------------

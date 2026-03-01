package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	llssa "github.com/goplus/llgo/ssa"
	gllvm "github.com/goplus/llvm"
)

const (
	invokeThunkPrefix    = "__llgo_invoke."
	ifacePtrDataFuncName = "github.com/goplus/llgo/runtime/internal/runtime.IfacePtrData"
)

const llvmFunctionAttributeIndex = -1

var invokeThunkNameRE = regexp.MustCompile(`^__llgo_invoke\.(.+)\$m([0-9]+)\.[^.]+$`)

type invokeIfaceMethod struct {
	key string
}

type invokeTypeMethods struct {
	TypeSymbol string
	Methods    map[string]string // method key -> IFn symbol
}

type invokeThunkTarget struct {
	TypeSymbol string
	IFnSymbol  string
}

type invokeThunkPlan struct {
	ThunkName   string
	MethodIndex int
	Targets     []invokeThunkTarget
}

func buildInvokeLoweringPatchObject(ctx *context, linkedPkgIDs map[string]bool, verbose bool) (string, error) {
	bitcodeFiles := collectLinkedBitcodeFiles(ctx, linkedPkgIDs)
	if len(bitcodeFiles) == 0 {
		return "", nil
	}
	sort.Strings(bitcodeFiles)

	llctx := gllvm.NewContext()
	defer llctx.Dispose()

	mergedMod, err := parseAndLinkBitcodeModules(llctx, bitcodeFiles)
	if err != nil {
		return "", err
	}
	defer mergedMod.Dispose()
	if IsMethodLateBindingEnabled() {
		if err := runInvokePreLoweringPasses(ctx, mergedMod); err != nil {
			return "", err
		}
	}

	plans := collectInvokeThunkPlans(mergedMod)
	if len(plans) == 0 {
		return "", nil
	}

	patchMod, patched := emitInvokeThunkPatchModule(mergedMod, plans)
	if patchMod.C == nil {
		return "", nil
	}
	defer patchMod.Dispose()
	if patched == 0 {
		return "", nil
	}

	if verbose || ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "invoke-lowering: linked %d bc files, generated %d invoke thunks\n", len(bitcodeFiles), patched)
	}
	objFile, err := compileLLVMIRToObject(ctx, "invoke-lowering", patchMod.String())
	if err != nil {
		return "", err
	}
	return objFile, nil
}

func collectLinkedBitcodeFiles(ctx *context, linkedPkgIDs map[string]bool) []string {
	files := make([]string, 0, len(linkedPkgIDs))
	for pkgID := range linkedPkgIDs {
		aPkg := ctx.pkgByID[pkgID]
		if aPkg == nil || aPkg.BitcodeFile == "" {
			continue
		}
		files = append(files, aPkg.BitcodeFile)
	}
	return files
}

func parseAndLinkBitcodeModules(llctx gllvm.Context, files []string) (gllvm.Module, error) {
	var merged gllvm.Module
	hasMerged := false
	for _, file := range files {
		mod, err := llctx.ParseBitcodeFile(file)
		if err != nil {
			// Some modules may currently fail bitcode round-trip; skip them so the
			// lowering pass remains best-effort and does not block normal linking.
			fmt.Fprintf(os.Stderr, "warning: invoke-lowering skip invalid bitcode %s: %v\n", file, err)
			continue
		}
		if !hasMerged {
			merged = mod
			hasMerged = true
			continue
		}
		if err := gllvm.LinkModules(merged, mod); err != nil {
			// Fall back to the existing weak thunk path if global linking fails.
			merged.Dispose()
			fmt.Fprintf(os.Stderr, "warning: invoke-lowering disable pass, link bitcode %s failed: %v\n", file, err)
			return gllvm.Module{}, nil
		}
	}
	if !hasMerged {
		return gllvm.Module{}, nil
	}
	return merged, nil
}

func runInvokePreLoweringPasses(ctx *context, mod gllvm.Module) error {
	pbo := gllvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	if err := mod.RunPasses("globaldce", ctx.prog.TargetMachine(), pbo); err != nil {
		return fmt.Errorf("run invoke pre-lowering passes failed: %w", err)
	}
	return nil
}

func collectInvokeThunkPlans(mod gllvm.Module) []invokeThunkPlan {
	types := collectConcreteTypeMethods(mod)
	if len(types) == 0 {
		return nil
	}

	ifaceCache := make(map[string][]invokeIfaceMethod)
	var plans []invokeThunkPlan

	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		thunkName := fn.Name()
		if !strings.HasPrefix(thunkName, invokeThunkPrefix) {
			continue
		}
		ifaceSym, methodIdx, ok := parseInvokeThunkName(thunkName)
		if !ok {
			continue
		}

		ifaceMethods, ok := ifaceCache[ifaceSym]
		if !ok {
			ifaceMethods = parseInterfaceMethods(mod, ifaceSym)
			ifaceCache[ifaceSym] = ifaceMethods
		}
		if len(ifaceMethods) == 0 || methodIdx >= len(ifaceMethods) {
			continue
		}

		targetKey := ifaceMethods[methodIdx].key
		targets := make([]invokeThunkTarget, 0, 16)
		for _, typ := range types {
			if !typeImplementsInterface(typ.Methods, ifaceMethods) {
				continue
			}
			if ifnSym := typ.Methods[targetKey]; ifnSym != "" {
				targets = append(targets, invokeThunkTarget{
					TypeSymbol: typ.TypeSymbol,
					IFnSymbol:  ifnSym,
				})
			}
		}
		targets = dedupInvokeThunkTargets(targets)
		if len(targets) == 0 {
			continue
		}
		plans = append(plans, invokeThunkPlan{
			ThunkName:   thunkName,
			MethodIndex: methodIdx,
			Targets:     targets,
		})
	}
	return plans
}

func dedupInvokeThunkTargets(targets []invokeThunkTarget) []invokeThunkTarget {
	if len(targets) < 2 {
		return targets
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].TypeSymbol == targets[j].TypeSymbol {
			return targets[i].IFnSymbol < targets[j].IFnSymbol
		}
		return targets[i].TypeSymbol < targets[j].TypeSymbol
	})
	out := targets[:1]
	for i := 1; i < len(targets); i++ {
		prev := out[len(out)-1]
		cur := targets[i]
		if prev.TypeSymbol == cur.TypeSymbol && prev.IFnSymbol == cur.IFnSymbol {
			continue
		}
		out = append(out, cur)
	}
	return out
}

func parseInvokeThunkName(thunkName string) (ifaceSym string, methodIdx int, ok bool) {
	m := invokeThunkNameRE.FindStringSubmatch(thunkName)
	if len(m) != 3 {
		return "", 0, false
	}
	idx, err := strconv.Atoi(m[2])
	if err != nil || idx < 0 {
		return "", 0, false
	}
	return m[1], idx, true
}

func parseInterfaceMethods(mod gllvm.Module, ifaceSym string) []invokeIfaceMethod {
	ifaceGlobal := mod.NamedGlobal(ifaceSym)
	if isNilValue(ifaceGlobal) {
		return nil
	}
	init := ifaceGlobal.Initializer()
	if isNilValue(init) || init.OperandsCount() < 3 {
		return nil
	}
	methodSlice := init.Operand(2)
	if isNilValue(methodSlice) || methodSlice.OperandsCount() < 3 {
		return nil
	}
	methodArrayGlobal := resolveGlobalSymbol(methodSlice.Operand(0))
	if isNilValue(methodArrayGlobal) {
		return nil
	}
	methodArray := methodArrayGlobal.Initializer()
	if isNilValue(methodArray) {
		return nil
	}

	n := constIntValue(methodSlice.Operand(1))
	if n <= 0 {
		return nil
	}
	if n > methodArray.OperandsCount() {
		n = methodArray.OperandsCount()
	}
	out := make([]invokeIfaceMethod, 0, n)
	for i := 0; i < n; i++ {
		im := methodArray.Operand(i)
		if isNilValue(im) || im.OperandsCount() < 2 {
			continue
		}
		name, ok := decodeRuntimeStringLiteral(im.Operand(0))
		if !ok {
			continue
		}
		typeSym := valueSymbol(im.Operand(1))
		if typeSym == "" {
			continue
		}
		out = append(out, invokeIfaceMethod{key: methodKey(name, typeSym)})
	}
	return out
}

func collectConcreteTypeMethods(mod gllvm.Module) []invokeTypeMethods {
	if out := collectConcreteTypeMethodsFromAttrs(mod); len(out) != 0 {
		return out
	}
	return collectConcreteTypeMethodsFromAbiMethodTable(mod)
}

func collectConcreteTypeMethodsFromAttrs(mod gllvm.Module) []invokeTypeMethods {
	types := make(map[string]map[string]string)
	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		name := fn.Name()
		if name == "" {
			continue
		}
		attr := fn.GetStringAttributeAtIndex(llvmFunctionAttributeIndex, llssa.MethodBindingAttrIFN)
		if attr.IsNil() {
			continue
		}
		for _, entry := range llssa.DecodeMethodBindingAttrValue(attr.GetStringValue()) {
			m, ok := types[entry.TypeSymbol]
			if !ok {
				m = make(map[string]string)
				types[entry.TypeSymbol] = m
			}
			m[methodKey(entry.MethodName, entry.MethodTypeSymbol)] = name
		}
	}
	if len(types) == 0 {
		return nil
	}
	typeNames := make([]string, 0, len(types))
	for typeSym := range types {
		typeNames = append(typeNames, typeSym)
	}
	sort.Strings(typeNames)
	out := make([]invokeTypeMethods, 0, len(typeNames))
	for _, typeSym := range typeNames {
		out = append(out, invokeTypeMethods{
			TypeSymbol: typeSym,
			Methods:    types[typeSym],
		})
	}
	return out
}

func collectConcreteTypeMethodsFromAbiMethodTable(mod gllvm.Module) []invokeTypeMethods {
	var out []invokeTypeMethods
	for g := mod.FirstGlobal(); !isNilValue(g); g = gllvm.NextGlobal(g) {
		name := g.Name()
		if name == "" {
			continue
		}
		init := g.Initializer()
		if isNilValue(init) || init.OperandsCount() < 3 {
			continue
		}
		uncommon := init.Operand(1)
		if isNilValue(uncommon) || uncommon.OperandsCount() < 2 {
			continue
		}
		methodArray := init.Operand(2)
		if isNilValue(methodArray) {
			continue
		}

		mcount := constIntValue(uncommon.Operand(1))
		if mcount <= 0 {
			continue
		}
		n := methodArray.OperandsCount()
		if n <= 0 {
			continue
		}
		if mcount < n {
			n = mcount
		}

		methods := make(map[string]string, n)
		for i := 0; i < n; i++ {
			m := methodArray.Operand(i)
			if isNilValue(m) || m.OperandsCount() < 4 {
				continue
			}
			methodName, ok := decodeRuntimeStringLiteral(m.Operand(0))
			if !ok {
				continue
			}
			typeSym := valueSymbol(m.Operand(1))
			if typeSym == "" {
				continue
			}
			ifnSym := valueSymbol(m.Operand(2))
			if ifnSym == "" {
				continue
			}
			methods[methodKey(methodName, typeSym)] = ifnSym
		}
		if len(methods) == 0 {
			continue
		}
		out = append(out, invokeTypeMethods{
			TypeSymbol: name,
			Methods:    methods,
		})
	}
	return out
}

func typeImplementsInterface(typeMethods map[string]string, ifaceMethods []invokeIfaceMethod) bool {
	for _, im := range ifaceMethods {
		if _, ok := typeMethods[im.key]; !ok {
			return false
		}
	}
	return true
}

func methodKey(name, typeSym string) string {
	return name + "\x00" + typeSym
}

type invokePatchEmitter struct {
	srcMod gllvm.Module
	mod    gllvm.Module
	ctx    gllvm.Context

	rtIfacePtrData gllvm.Value
}

func emitInvokeThunkPatchModule(srcMod gllvm.Module, plans []invokeThunkPlan) (gllvm.Module, int) {
	ctx := srcMod.Context()
	patchMod := ctx.NewModule("llgo.invoke.lowering")
	patchMod.SetDataLayout(srcMod.DataLayout())
	patchMod.SetTarget(srcMod.Target())

	emitter := &invokePatchEmitter{
		srcMod: srcMod,
		mod:    patchMod,
		ctx:    ctx,
	}

	patched := 0
	for _, plan := range plans {
		if emitter.emitThunk(plan) {
			patched++
		}
	}
	return patchMod, patched
}

func (e *invokePatchEmitter) emitThunk(plan invokeThunkPlan) bool {
	srcThunk := e.srcMod.NamedFunction(plan.ThunkName)
	if isNilValue(srcThunk) {
		return false
	}
	thunkTy := srcThunk.GlobalValueType()
	if thunkTy.TypeKind() != gllvm.FunctionTypeKind {
		return false
	}

	thunk := gllvm.AddFunction(e.mod, plan.ThunkName, thunkTy)
	thunk.SetLinkage(gllvm.ExternalLinkage)
	thunk.SetFunctionCallConv(srcThunk.FunctionCallConv())

	b := e.ctx.NewBuilder()
	defer b.Dispose()

	entry := e.ctx.AddBasicBlock(thunk, "entry")
	fallback := e.ctx.AddBasicBlock(thunk, "fallback")

	params := thunk.Params()
	if len(params) == 0 {
		return false
	}

	b.SetInsertPointAtEnd(entry)
	ifaceParam := params[0]
	receiver := b.CreateCall(e.ensureIfacePtrDataDecl(ifaceParam.Type()).GlobalValueType(), e.rtIfacePtrData, []gllvm.Value{ifaceParam}, "receiver")

	ptrTy := receiver.Type()
	itab := b.CreateExtractValue(ifaceParam, 0, "itab")
	i64Ty := e.ctx.Int64Type()
	actualTypePtr := b.CreateGEP(ptrTy, itab, []gllvm.Value{gllvm.ConstInt(i64Ty, 1, false)}, "actualType.ptr")
	actualType := b.CreateLoad(ptrTy, actualTypePtr, "actualType")

	retTy := thunkTy.ReturnType()
	callParamTypes := make([]gllvm.Type, 0, len(params))
	callParamTypes = append(callParamTypes, receiver.Type())
	for i := 1; i < len(params); i++ {
		callParamTypes = append(callParamTypes, params[i].Type())
	}
	callTy := gllvm.FunctionType(retTy, callParamTypes, thunkTy.IsFunctionVarArg())
	callArgs := make([]gllvm.Value, 0, len(params))
	callArgs = append(callArgs, receiver)
	for i := 1; i < len(params); i++ {
		callArgs = append(callArgs, params[i])
	}

	dispatch := entry
	for i, target := range plan.Targets {
		b.SetInsertPointAtEnd(dispatch)
		match := e.ctx.AddBasicBlock(thunk, fmt.Sprintf("type.%d", i))
		miss := fallback
		if i < len(plan.Targets)-1 {
			miss = e.ctx.AddBasicBlock(thunk, fmt.Sprintf("type.next.%d", i))
		}

		typeGlobal := e.ensureTypeGlobalDecl(target.TypeSymbol)
		if isNilValue(typeGlobal) {
			continue
		}
		cmp := b.CreateICmp(gllvm.IntEQ, actualType, typeGlobal, "")
		b.CreateCondBr(cmp, match, miss)

		b.SetInsertPointAtEnd(match)
		callee := e.ensureFunctionDecl(target.IFnSymbol, callTy)
		ret := b.CreateCall(callTy, callee, callArgs, "")
		createReturn(b, retTy, ret)

		dispatch = miss
	}

	if len(plan.Targets) == 0 {
		b.SetInsertPointAtEnd(entry)
		b.CreateBr(fallback)
	}

	b.SetInsertPointAtEnd(fallback)
	slotIndex := uint64(plan.MethodIndex + 3)
	fnPtrPtr := b.CreateGEP(ptrTy, itab, []gllvm.Value{gllvm.ConstInt(i64Ty, slotIndex, false)}, "fn.ptr")
	fnPtr := b.CreateLoad(ptrTy, fnPtrPtr, "fn")
	ret := b.CreateCall(callTy, fnPtr, callArgs, "")
	createReturn(b, retTy, ret)
	return true
}

func (e *invokePatchEmitter) ensureIfacePtrDataDecl(ifaceTy gllvm.Type) gllvm.Value {
	if !isNilValue(e.rtIfacePtrData) {
		return e.rtIfacePtrData
	}
	ptrTy := gllvm.PointerType(e.ctx.Int8Type(), 0)
	fnTy := gllvm.FunctionType(ptrTy, []gllvm.Type{ifaceTy}, false)
	e.rtIfacePtrData = e.ensureFunctionDecl(ifacePtrDataFuncName, fnTy)
	return e.rtIfacePtrData
}

func (e *invokePatchEmitter) ensureFunctionDecl(name string, fnTy gllvm.Type) gllvm.Value {
	fn := e.mod.NamedFunction(name)
	if !isNilValue(fn) {
		return fn
	}
	fn = gllvm.AddFunction(e.mod, name, fnTy)
	fn.SetLinkage(gllvm.ExternalLinkage)
	return fn
}

func (e *invokePatchEmitter) ensureTypeGlobalDecl(name string) gllvm.Value {
	g := e.mod.NamedGlobal(name)
	if !isNilValue(g) {
		return g
	}
	src := e.srcMod.NamedGlobal(name)
	if isNilValue(src) {
		src = e.srcMod.NamedFunction(name)
		if !isNilValue(src) {
			// Not a global variable symbol.
			return gllvm.Value{}
		}
	}

	declTy := e.ctx.Int8Type()
	if !isNilValue(src) {
		declTy = src.GlobalValueType()
	}
	g = gllvm.AddGlobal(e.mod, declTy, name)
	g.SetLinkage(gllvm.ExternalLinkage)
	return g
}

func createReturn(b gllvm.Builder, retTy gllvm.Type, ret gllvm.Value) {
	if retTy.TypeKind() == gllvm.VoidTypeKind {
		b.CreateRetVoid()
		return
	}
	b.CreateRet(ret)
}

func valueSymbol(v gllvm.Value) string {
	base := resolvePointerSymbol(v)
	if isNilValue(base) {
		return ""
	}
	name := base.Name()
	if name == "" {
		return ""
	}
	return name
}

func resolveGlobalSymbol(v gllvm.Value) gllvm.Value {
	base := resolvePointerSymbol(v)
	if isNilValue(base) {
		return gllvm.Value{}
	}
	if !isNilValue(base.IsAGlobalVariable()) {
		return base
	}
	return gllvm.Value{}
}

func resolvePointerSymbol(v gllvm.Value) gllvm.Value {
	cur := v
	for !isNilValue(cur) {
		if cur.IsNull() {
			return gllvm.Value{}
		}
		if !isNilValue(cur.IsAGlobalVariable()) || !isNilValue(cur.IsAFunction()) {
			return cur
		}
		if !isNilValue(cur.IsAConstantExpr()) {
			switch cur.Opcode() {
			case gllvm.BitCast, gllvm.GetElementPtr, gllvm.IntToPtr, gllvm.PtrToInt:
				if cur.OperandsCount() > 0 {
					cur = cur.Operand(0)
					continue
				}
			}
		}
		break
	}
	return gllvm.Value{}
}

func decodeRuntimeStringLiteral(v gllvm.Value) (string, bool) {
	if isNilValue(v) || v.OperandsCount() < 2 {
		return "", false
	}
	n := constIntValue(v.Operand(1))
	if n == 0 {
		return "", true
	}
	strGlobal := resolveGlobalSymbol(v.Operand(0))
	if isNilValue(strGlobal) {
		return "", false
	}
	init := strGlobal.Initializer()
	if isNilValue(init) || !init.IsConstantString() {
		return "", false
	}
	s := init.ConstGetAsString()
	if n < len(s) {
		s = s[:n]
	}
	return s, true
}

func constIntValue(v gllvm.Value) int {
	if isNilValue(v) {
		return 0
	}
	cv := v.IsAConstantInt()
	if isNilValue(cv) {
		return 0
	}
	return int(cv.ZExtValue())
}

func isNilValue(v gllvm.Value) bool {
	return v.C == nil
}

func compileLLVMIRToObject(ctx *context, prefix string, ir string) (string, error) {
	llFile, err := os.CreateTemp("", prefix+"-*.ll")
	if err != nil {
		return "", err
	}
	if _, err := llFile.WriteString(ir); err != nil {
		llFile.Close()
		return "", err
	}
	if err := llFile.Close(); err != nil {
		return "", err
	}

	if ctx.buildConf.CheckLLFiles {
		if msg, err := llcCheck(ctx.env, llFile.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "==> llc %v: %v\n%v\n", prefix, llFile.Name(), msg)
		}
	}

	if ctx.buildConf.GenLL {
		genFile := filepath.Join(".", prefix+".ll")
		if err := copyFileAtomic(llFile.Name(), genFile); err != nil {
			return "", err
		}
	}

	objFile, err := os.CreateTemp("", prefix+"-*.o")
	if err != nil {
		return "", err
	}
	objFile.Close()
	args := []string{"-o", objFile.Name(), "-c", llFile.Name(), "-Wno-override-module"}
	if ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", llFile.Name(), prefix)
		fmt.Fprintln(os.Stderr, "clang", args)
	}
	cmd := ctx.compiler()
	if err := cmd.Compile(args...); err != nil {
		return "", fmt.Errorf("compile invoke lowering IR failed: %w", err)
	}
	return objFile.Name(), nil
}

func invokeLoweringEnabled() bool {
	val := strings.TrimSpace(os.Getenv("LLGO_INVOKE_LOWERING"))
	switch strings.ToLower(val) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

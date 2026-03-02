package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	llssa "github.com/goplus/llgo/ssa"
	gllvm "github.com/goplus/llvm"
)

const (
	invokeThunkPrefix        = "__llgo_invoke."
	typeAssertThunkPrefix    = "__llgo_typeassert."
	ifacePtrDataFuncName     = "github.com/goplus/llgo/runtime/internal/runtime.IfacePtrData"
	rtImplementsFuncName     = "github.com/goplus/llgo/runtime/internal/runtime.Implements"
	rtMatchConcreteFuncName  = "github.com/goplus/llgo/runtime/internal/runtime.MatchConcreteType"
	rtMatchesClosureFuncName = "github.com/goplus/llgo/runtime/internal/runtime.MatchesClosure"

	typeAssertThunkKindIface    = "iface"
	typeAssertThunkKindConcrete = "concrete"
	typeAssertThunkKindClosure  = "closure"
)

const llvmFunctionAttributeIndex = -1

var invokeThunkNameRE = regexp.MustCompile(`^__llgo_invoke\.(.+)\$m([0-9]+)\.[^.]+$`)
var typeAssertThunkNameRE = regexp.MustCompile(`^__llgo_typeassert\.(iface|concrete|closure)\.(.+)$`)

type invokeIfaceMethod struct {
	key  string
	name string
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

type typeAssertThunkPlan struct {
	ThunkName          string
	Kind               string
	AssertedTypeSymbol string
	Targets            []string // only used by iface assert
}

type linkedModuleInput struct {
	BitcodeFile string
	IRFile      string
}

func buildGlobalMergedObject(ctx *context, moduleInputs []linkedModuleInput, verbose bool) (string, error) {
	if len(moduleInputs) == 0 {
		return "", nil
	}
	sort.Slice(moduleInputs, func(i, j int) bool {
		return moduleInputKey(moduleInputs[i]) < moduleInputKey(moduleInputs[j])
	})

	llctx := gllvm.NewContext()
	defer llctx.Dispose()

	llvmDis, _ := llvmDisToolPath(ctx)
	mergedMod, preCollectedTypes, err := parseAndLinkModules(ctx, llctx, moduleInputs, llvmDis, verbose)
	if err != nil {
		return "", err
	}
	if isNilModule(mergedMod) {
		return "", nil
	}
	defer mergedMod.Dispose()

	// Collect invoke/typeassert plans first so we can preserve required IFn/TFn
	// symbols before running whole-module DCE.
	preCollectedTypes = mergeInvokeTypeMethods(preCollectedTypes, collectConcreteTypeMethods(mergedMod))
	preCollectedIfaceMethods := collectInvokeIfaceMethodsForThunks(mergedMod)
	var liveTypeSymbols map[string]bool
	if analysisMod, _, err := parseAndLinkModules(ctx, llctx, moduleInputs, llvmDis, false); err == nil && !isNilModule(analysisMod) {
		if err := runInvokePreLoweringPasses(ctx, analysisMod); err == nil {
			liveTypeSymbols = collectGlobalSymbolSet(analysisMod)
		}
		analysisMod.Dispose()
	}
	invokePlans := collectInvokeThunkPlans(mergedMod, preCollectedTypes, preCollectedIfaceMethods, liveTypeSymbols)
	typeAssertPlans := collectTypeAssertThunkPlans(mergedMod, preCollectedTypes, liveTypeSymbols)
	preservedSymbols := collectLoweringPreserveSymbols(invokePlans, typeAssertPlans)
	liveTypeMethods := filterTypeMethodsBySymbolSet(preCollectedTypes, liveTypeSymbols)
	if len(liveTypeMethods) == 0 {
		liveTypeMethods = filterTypeMethodsByLiveTypeSymbol(mergedMod, preCollectedTypes)
	}
	if len(liveTypeMethods) == 0 {
		liveTypeMethods = preCollectedTypes
	}
	preservedSymbols = append(preservedSymbols, collectIFNSymbolsFromTypes(liveTypeMethods)...)
	preservedSymbols = dedupSymbols(preservedSymbols)
	pinned := pinSymbolsExternal(mergedMod, preservedSymbols)

	if err := runGlobalMergedPasses(ctx, mergedMod); err != nil {
		return "", err
	}
	if verbose || ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "global-bc: merged %d modules, pinned %d symbols, invoke=%d typeassert=%d\n", len(moduleInputs), pinned, len(invokePlans), len(typeAssertPlans))
	}
	return compileLLVMIRToObject(ctx, "global-merged", mergedMod.String())
}

func buildInvokeLoweringPatchObject(ctx *context, linkedPkgIDs map[string]bool, extraInputs []linkedModuleInput, verbose bool) (string, error) {
	moduleInputs := collectLinkedModuleInputs(ctx, linkedPkgIDs, extraInputs)
	if len(moduleInputs) == 0 {
		return "", nil
	}
	debug := strings.TrimSpace(os.Getenv("LLGO_INVOKE_LOWERING_DEBUG")) != ""
	sort.Slice(moduleInputs, func(i, j int) bool {
		return moduleInputKey(moduleInputs[i]) < moduleInputKey(moduleInputs[j])
	})
	if debug {
		for _, in := range moduleInputs {
			fmt.Fprintf(os.Stderr, "invoke-lowering(debug): input bc=%s ir=%s\n", in.BitcodeFile, in.IRFile)
		}
	}

	llctx := gllvm.NewContext()
	defer llctx.Dispose()

	llvmDis, _ := llvmDisToolPath(ctx)
	mergedMod, preCollectedTypes, err := parseAndLinkModules(ctx, llctx, moduleInputs, llvmDis, verbose)
	if err != nil {
		return "", err
	}
	if isNilModule(mergedMod) {
		if verbose || ctx.shouldPrintCommands(false) {
			fmt.Fprintln(os.Stderr, "invoke-lowering: skip, no valid linked module")
		}
		return "", nil
	}
	defer mergedMod.Dispose()

	// Merge retained symbols from merged module into pre-collected metadata.
	// The pre-collected set is harvested before module linking, so it can keep
	// method binding attrs from linkonce/weak_odr functions that may disappear
	// during LinkModules resolution.
	preCollectedTypes = mergeInvokeTypeMethods(preCollectedTypes, collectConcreteTypeMethods(mergedMod))
	preCollectedIfaceMethods := collectInvokeIfaceMethodsForThunks(mergedMod)

	var liveTypeSymbols map[string]bool
	if analysisMod, _, err := parseAndLinkModules(ctx, llctx, moduleInputs, llvmDis, false); err == nil && !isNilModule(analysisMod) {
		if err := runInvokePreLoweringPasses(ctx, analysisMod); err == nil {
			liveTypeSymbols = collectGlobalSymbolSet(analysisMod)
		}
		analysisMod.Dispose()
	}
	if verbose || ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "invoke-lowering: discovered %d invoke thunks, %d typeassert thunks\n", countInvokeThunks(mergedMod), countTypeAssertThunks(mergedMod))
	}

	invokePlans := collectInvokeThunkPlans(mergedMod, preCollectedTypes, preCollectedIfaceMethods, liveTypeSymbols)
	typeAssertPlans := collectTypeAssertThunkPlans(mergedMod, preCollectedTypes, liveTypeSymbols)
	if len(invokePlans) == 0 && len(typeAssertPlans) == 0 {
		return "", nil
	}

	patchMod, patched := emitLoweringPatchModule(mergedMod, invokePlans, typeAssertPlans)
	if patchMod.C == nil {
		return "", nil
	}
	defer patchMod.Dispose()
	if patched == 0 {
		return "", nil
	}

	if verbose || ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "invoke-lowering: linked %d modules, generated %d lowered thunks (invoke+typeassert)\n", len(moduleInputs), patched)
	}
	objFile, err := compileLLVMIRToObject(ctx, "invoke-lowering", patchMod.String())
	if err != nil {
		return "", err
	}
	return objFile, nil
}

func collectLinkedModuleInputs(ctx *context, linkedPkgIDs map[string]bool, extraInputs []linkedModuleInput) []linkedModuleInput {
	files := make([]linkedModuleInput, 0, len(linkedPkgIDs)+len(extraInputs))
	for pkgID := range linkedPkgIDs {
		aPkg := ctx.pkgByID[pkgID]
		if aPkg == nil {
			continue
		}
		if aPkg.BitcodeFile == "" && aPkg.IRFile == "" {
			continue
		}
		files = append(files, linkedModuleInput{
			BitcodeFile: aPkg.BitcodeFile,
			IRFile:      aPkg.IRFile,
		})
	}
	for _, in := range extraInputs {
		if in.BitcodeFile == "" && in.IRFile == "" {
			continue
		}
		files = append(files, in)
	}
	return files
}

func moduleInputKey(in linkedModuleInput) string {
	if in.BitcodeFile != "" {
		return in.BitcodeFile
	}
	return in.IRFile
}

func parseAndLinkModules(ctx *context, llctx gllvm.Context, files []linkedModuleInput, llvmDis string, verbose bool) (gllvm.Module, []invokeTypeMethods, error) {
	var merged gllvm.Module
	hasMerged := false
	var preCollectedTypes []invokeTypeMethods
	for _, file := range files {
		mod, source, err := parseLinkedModule(llctx, llvmDis, file)
		if err != nil {
			if verbose || ctx.shouldPrintCommands(false) {
				fmt.Fprintf(os.Stderr, "warning: invoke-lowering skip module %s: %v\n", moduleInputKey(file), err)
			}
			continue
		}
		preCollectedTypes = mergeInvokeTypeMethods(preCollectedTypes, collectConcreteTypeMethods(mod))
		if !hasMerged {
			merged = mod
			hasMerged = true
			continue
		}
		if err := gllvm.LinkModules(merged, mod); err != nil {
			// Fall back to the existing weak thunk path if global linking fails.
			merged.Dispose()
			fmt.Fprintf(os.Stderr, "warning: invoke-lowering disable pass, link %s module %s failed: %v\n", source, moduleInputKey(file), err)
			return gllvm.Module{}, nil, nil
		}
	}
	if !hasMerged {
		return gllvm.Module{}, nil, nil
	}
	return merged, preCollectedTypes, nil
}

func parseLinkedModule(llctx gllvm.Context, llvmDis string, in linkedModuleInput) (gllvm.Module, string, error) {
	if in.BitcodeFile != "" {
		if llvmDis != "" && !canParseBitcodeFile(llvmDis, in.BitcodeFile) {
			goto parseIR
		}
		mod, err := llctx.ParseBitcodeFile(in.BitcodeFile)
		if err == nil {
			return mod, "bitcode", nil
		}
		if in.IRFile == "" {
			return gllvm.Module{}, "", fmt.Errorf("invalid bitcode: %s (%w)", in.BitcodeFile, err)
		}
	}
parseIR:
	if in.IRFile != "" {
		buf, err := gllvm.NewMemoryBufferFromFile(in.IRFile)
		if err != nil {
			return gllvm.Module{}, "", err
		}
		mod, err := (&llctx).ParseIR(buf)
		if err == nil {
			return mod, "ir", nil
		}
		return gllvm.Module{}, "", err
	}
	return gllvm.Module{}, "", fmt.Errorf("no module input")
}

func llvmDisToolPath(ctx *context) (string, error) {
	if ctx != nil && ctx.env != nil {
		if dir := ctx.env.BinDir(); dir != "" {
			tool := filepath.Join(dir, "llvm-dis")
			if _, err := os.Stat(tool); err == nil {
				return tool, nil
			}
		}
	}
	if tool, err := exec.LookPath("llvm-dis"); err == nil {
		return tool, nil
	}
	return "", fmt.Errorf("llvm-dis not found")
}

func canParseBitcodeFile(llvmDis string, file string) bool {
	if llvmDis == "" {
		return false
	}
	cmd := exec.Command(llvmDis, "-o", os.DevNull, file)
	return cmd.Run() == nil
}

func runInvokePreLoweringPasses(ctx *context, mod gllvm.Module) error {
	pbo := gllvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	if err := mod.RunPasses("internalize,globaldce", ctx.prog.TargetMachine(), pbo); err != nil {
		// Fallback for targets/toolchains that don't accept internalize in this context.
		if err2 := mod.RunPasses("globaldce", ctx.prog.TargetMachine(), pbo); err2 != nil {
			return fmt.Errorf("run invoke pre-lowering passes failed: %w", err2)
		}
	}
	return nil
}

func runGlobalMergedPasses(ctx *context, mod gllvm.Module) error {
	passes := strings.TrimSpace(os.Getenv(llgoGlobalBCPasses))
	if passes == "" {
		passes = "globaldce"
	}

	pbo := gllvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	if err := mod.RunPasses(passes, ctx.prog.TargetMachine(), pbo); err != nil {
		if passes != "globaldce" {
			if err2 := mod.RunPasses("globaldce", ctx.prog.TargetMachine(), pbo); err2 == nil {
				return nil
			} else {
				return fmt.Errorf("run global merged passes %q failed: %v (fallback globaldce failed: %w)", passes, err, err2)
			}
		}
		return fmt.Errorf("run global merged passes failed: %w", err)
	}
	return nil
}

func collectLoweringPreserveSymbols(invokePlans []invokeThunkPlan, typeAssertPlans []typeAssertThunkPlan) []string {
	keep := map[string]bool{
		ifacePtrDataFuncName: true,
	}
	for _, plan := range invokePlans {
		for _, target := range plan.Targets {
			if target.IFnSymbol != "" {
				keep[target.IFnSymbol] = true
			}
			if target.TypeSymbol != "" {
				keep[target.TypeSymbol] = true
			}
		}
	}
	for _, plan := range typeAssertPlans {
		if plan.AssertedTypeSymbol != "" {
			keep[plan.AssertedTypeSymbol] = true
		}
		for _, sym := range plan.Targets {
			if sym != "" {
				keep[sym] = true
			}
		}
		switch plan.Kind {
		case typeAssertThunkKindIface:
			keep[rtImplementsFuncName] = true
		case typeAssertThunkKindConcrete:
			keep[rtMatchConcreteFuncName] = true
		case typeAssertThunkKindClosure:
			keep[rtMatchesClosureFuncName] = true
		}
	}
	if len(keep) == 0 {
		return nil
	}
	out := make([]string, 0, len(keep))
	for sym := range keep {
		if sym == "" {
			continue
		}
		out = append(out, sym)
	}
	sort.Strings(out)
	return out
}

func collectIFNSymbolsFromTypes(types []invokeTypeMethods) []string {
	if len(types) == 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, typ := range types {
		for _, ifn := range typ.Methods {
			if ifn == "" {
				continue
			}
			set[ifn] = true
		}
	}
	out := make([]string, 0, len(set))
	for sym := range set {
		out = append(out, sym)
	}
	sort.Strings(out)
	return out
}

func dedupSymbols(symbols []string) []string {
	if len(symbols) < 2 {
		return symbols
	}
	sort.Strings(symbols)
	out := symbols[:0]
	var prev string
	for i, sym := range symbols {
		if sym == "" {
			continue
		}
		if i != 0 && sym == prev {
			continue
		}
		out = append(out, sym)
		prev = sym
	}
	return out
}

func pinSymbolsExternal(mod gllvm.Module, symbols []string) int {
	if len(symbols) == 0 || isNilModule(mod) {
		return 0
	}
	pinned := 0
	for _, sym := range symbols {
		if sym == "" {
			continue
		}
		if fn := mod.NamedFunction(sym); !isNilValue(fn) {
			fn.SetLinkage(gllvm.ExternalLinkage)
			pinned++
			continue
		}
		if g := mod.NamedGlobal(sym); !isNilValue(g) {
			g.SetLinkage(gllvm.ExternalLinkage)
			pinned++
		}
	}
	return pinned
}

func collectInvokeThunkPlans(mod gllvm.Module, preCollectedTypes []invokeTypeMethods, preCollectedIfaceMethods map[string][]invokeIfaceMethod, liveTypeSymbols map[string]bool) []invokeThunkPlan {
	if isNilModule(mod) {
		return nil
	}
	debug := strings.TrimSpace(os.Getenv("LLGO_INVOKE_LOWERING_DEBUG")) != ""
	allTypes := mergeInvokeTypeMethods(nil, preCollectedTypes)
	allTypes = mergeInvokeTypeMethods(allTypes, collectConcreteTypeMethods(mod))
	if len(allTypes) == 0 {
		return nil
	}
	liveTypes := filterTypeMethodsBySymbolSet(allTypes, liveTypeSymbols)
	if len(liveTypes) == 0 {
		liveTypes = filterTypeMethodsByLiveTypeSymbol(mod, allTypes)
	}
	if len(liveTypes) == 0 {
		// globaldce can remove weak type globals that are still needed by invoke
		// devirtualization planning; fall back to pre-collected metadata.
		liveTypes = allTypes
	}

	ifaceCache := cloneInvokeIfaceMethodsMap(preCollectedIfaceMethods)
	var plans []invokeThunkPlan

	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		thunkName := fn.Name()
		if !strings.HasPrefix(thunkName, invokeThunkPrefix) {
			continue
		}
		ifaceSym, methodIdx, ok := parseInvokeThunkName(thunkName)
		if !ok {
			if debug {
				fmt.Fprintf(os.Stderr, "invoke-lowering(debug): skip thunk parse %s\n", thunkName)
			}
			continue
		}

		ifaceMethods, ok := ifaceCache[ifaceSym]
		if !ok || len(ifaceMethods) == 0 {
			if parsed := parseInterfaceMethods(mod, ifaceSym); len(parsed) != 0 || !ok {
				ifaceMethods = parsed
				ifaceCache[ifaceSym] = ifaceMethods
			}
		}
		if len(ifaceMethods) == 0 || methodIdx >= len(ifaceMethods) {
			if debug {
				fmt.Fprintf(os.Stderr, "invoke-lowering(debug): skip thunk %s iface=%s methods=%d idx=%d\n", thunkName, ifaceSym, len(ifaceMethods), methodIdx)
			}
			continue
		}

		targetMethod := ifaceMethods[methodIdx]
		targets := collectThunkTargets(liveTypes, ifaceMethods, targetMethod)
		if len(targets) == 0 && len(liveTypes) < len(allTypes) {
			// If no target survives the live-type filter, retry with all pre-collected
			// method bindings to avoid missing interface-only instantiations.
			targets = collectThunkTargets(allTypes, ifaceMethods, targetMethod)
			if debug && len(targets) != 0 {
				fmt.Fprintf(os.Stderr, "invoke-lowering(debug): thunk %s recovered %d targets via pre-dce fallback\n", thunkName, len(targets))
			}
		}
		targets = dedupInvokeThunkTargets(targets)
		targets = filterInvokeThunkTargetsByDefinedFunction(mod, targets)
		targets = expandInvokeThunkTargetsWithPointerPairs(mod, targets)
		targets = dedupInvokeThunkTargets(targets)
		if len(targets) == 0 {
			if debug {
				fmt.Fprintf(os.Stderr, "invoke-lowering(debug): skip thunk %s no targets, ifaceMethod=%q\n", thunkName, targetMethod.key)
			}
			continue
		}
		if debug {
			fmt.Fprintf(os.Stderr, "invoke-lowering(debug): thunk %s targets=%d ifaceMethod=%q\n", thunkName, len(targets), targetMethod.key)
		}
		plans = append(plans, invokeThunkPlan{
			ThunkName:   thunkName,
			MethodIndex: methodIdx,
			Targets:     targets,
		})
	}
	return plans
}

func collectThunkTargets(types []invokeTypeMethods, ifaceMethods []invokeIfaceMethod, targetMethod invokeIfaceMethod) []invokeThunkTarget {
	targets := make([]invokeThunkTarget, 0, 16)
	for _, typ := range types {
		if !typeImplementsInterface(typ.Methods, ifaceMethods) {
			continue
		}
		targetKey, ok := resolveMethodKey(typ.Methods, targetMethod)
		if !ok {
			continue
		}
		if ifnSym := typ.Methods[targetKey]; ifnSym != "" {
			targets = append(targets, invokeThunkTarget{
				TypeSymbol: typ.TypeSymbol,
				IFnSymbol:  ifnSym,
			})
		}
	}
	return targets
}

func filterTypeMethodsByLiveTypeSymbol(mod gllvm.Module, types []invokeTypeMethods) []invokeTypeMethods {
	if len(types) == 0 {
		return nil
	}
	out := make([]invokeTypeMethods, 0, len(types))
	for _, typ := range types {
		if typ.TypeSymbol == "" || isNilValue(mod.NamedGlobal(typ.TypeSymbol)) {
			continue
		}
		out = append(out, typ)
	}
	return out
}

func filterTypeMethodsBySymbolSet(types []invokeTypeMethods, symbols map[string]bool) []invokeTypeMethods {
	if len(types) == 0 || len(symbols) == 0 {
		return nil
	}
	out := make([]invokeTypeMethods, 0, len(types))
	for _, typ := range types {
		if typ.TypeSymbol == "" || !symbols[typ.TypeSymbol] {
			continue
		}
		out = append(out, typ)
	}
	return out
}

func collectGlobalSymbolSet(mod gllvm.Module) map[string]bool {
	if isNilModule(mod) {
		return nil
	}
	syms := make(map[string]bool)
	for g := mod.FirstGlobal(); !isNilValue(g); g = gllvm.NextGlobal(g) {
		name := g.Name()
		if name == "" {
			continue
		}
		syms[name] = true
	}
	return syms
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

func filterInvokeThunkTargetsByDefinedFunction(mod gllvm.Module, targets []invokeThunkTarget) []invokeThunkTarget {
	if len(targets) == 0 || isNilModule(mod) {
		return targets
	}
	out := targets[:0]
	for _, target := range targets {
		ifnSym := target.IFnSymbol
		if ifnSym == "" {
			continue
		}
		if fn := mod.NamedFunction(ifnSym); !isNilValue(fn) && !fn.IsDeclaration() {
			out = append(out, target)
			continue
		}
		stubSym := "__llgo_stub." + ifnSym
		if fn := mod.NamedFunction(stubSym); !isNilValue(fn) && !fn.IsDeclaration() {
			target.IFnSymbol = stubSym
			out = append(out, target)
		}
	}
	return out
}

func expandInvokeThunkTargetsWithPointerPairs(mod gllvm.Module, targets []invokeThunkTarget) []invokeThunkTarget {
	if len(targets) == 0 || isNilModule(mod) {
		return targets
	}
	out := make([]invokeThunkTarget, 0, len(targets)*2)
	for _, target := range targets {
		out = append(out, target)
		altTypeSym := pairedTypeSymbol(target.TypeSymbol)
		if altTypeSym == "" {
			continue
		}
		if isNilValue(mod.NamedGlobal(altTypeSym)) {
			continue
		}
		out = append(out, invokeThunkTarget{
			TypeSymbol: altTypeSym,
			IFnSymbol:  target.IFnSymbol,
		})
	}
	return out
}

func pairedTypeSymbol(typeSym string) string {
	if typeSym == "" {
		return ""
	}
	if strings.HasPrefix(typeSym, "*") {
		return typeSym[1:]
	}
	return "*" + typeSym
}

func countInvokeThunks(mod gllvm.Module) (count int) {
	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		name := fn.Name()
		if !strings.HasPrefix(name, invokeThunkPrefix) {
			continue
		}
		count++
	}
	return
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

func countTypeAssertThunks(mod gllvm.Module) (count int) {
	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		name := fn.Name()
		if !strings.HasPrefix(name, typeAssertThunkPrefix) {
			continue
		}
		count++
	}
	return
}

func parseTypeAssertThunkName(thunkName string) (kind, assertedTypeSym string, ok bool) {
	m := typeAssertThunkNameRE.FindStringSubmatch(thunkName)
	if len(m) != 3 || m[2] == "" {
		return "", "", false
	}
	return m[1], m[2], true
}

func collectTypeAssertThunkPlans(mod gllvm.Module, preCollectedTypes []invokeTypeMethods, liveTypeSymbols map[string]bool) []typeAssertThunkPlan {
	if isNilModule(mod) {
		return nil
	}
	allTypes := mergeInvokeTypeMethods(nil, preCollectedTypes)
	allTypes = mergeInvokeTypeMethods(allTypes, collectConcreteTypeMethods(mod))
	liveTypes := filterTypeMethodsBySymbolSet(allTypes, liveTypeSymbols)
	if len(liveTypes) == 0 {
		liveTypes = filterTypeMethodsByLiveTypeSymbol(mod, allTypes)
	}
	if len(liveTypes) == 0 {
		liveTypes = allTypes
	}

	ifaceMethodCache := make(map[string][]invokeIfaceMethod)
	var plans []typeAssertThunkPlan
	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		thunkName := fn.Name()
		if !strings.HasPrefix(thunkName, typeAssertThunkPrefix) {
			continue
		}
		kind, assertedTypeSym, ok := parseTypeAssertThunkName(thunkName)
		if !ok {
			continue
		}
		if kind != typeAssertThunkKindIface {
			// Keep concrete/closure assertions on weak thunk fallback for now.
			// They don't benefit from branch expansion the same way as interface asserts.
			continue
		}
		plan := typeAssertThunkPlan{
			ThunkName:          thunkName,
			Kind:               kind,
			AssertedTypeSymbol: assertedTypeSym,
		}

		ifaceMethods, ok := ifaceMethodCache[assertedTypeSym]
		if !ok {
			ifaceMethods = parseInterfaceMethods(mod, assertedTypeSym)
			ifaceMethodCache[assertedTypeSym] = ifaceMethods
		}
		if len(ifaceMethods) == 0 {
			continue
		}
		targets := collectAssertIfaceTargets(liveTypes, ifaceMethods)
		if len(targets) == 0 && len(liveTypes) < len(allTypes) {
			targets = collectAssertIfaceTargets(allTypes, ifaceMethods)
		}
		targets = dedupTypeSymbols(targets)
		if len(targets) == 0 {
			continue
		}
		plan.Targets = targets

		plans = append(plans, plan)
	}
	return plans
}

func collectAssertIfaceTargets(types []invokeTypeMethods, ifaceMethods []invokeIfaceMethod) []string {
	targets := make([]string, 0, 16)
	for _, typ := range types {
		if typ.TypeSymbol == "" {
			continue
		}
		if typeImplementsInterface(typ.Methods, ifaceMethods) {
			targets = append(targets, typ.TypeSymbol)
		}
	}
	return targets
}

func dedupTypeSymbols(targets []string) []string {
	if len(targets) < 2 {
		return targets
	}
	sort.Strings(targets)
	out := targets[:1]
	for i := 1; i < len(targets); i++ {
		if targets[i] == out[len(out)-1] {
			continue
		}
		out = append(out, targets[i])
	}
	return out
}

func collectInvokeIfaceMethodsForThunks(mod gllvm.Module) map[string][]invokeIfaceMethod {
	if isNilModule(mod) {
		return nil
	}
	out := make(map[string][]invokeIfaceMethod)
	for fn := mod.FirstFunction(); !isNilValue(fn); fn = gllvm.NextFunction(fn) {
		thunkName := fn.Name()
		if !strings.HasPrefix(thunkName, invokeThunkPrefix) {
			continue
		}
		ifaceSym, _, ok := parseInvokeThunkName(thunkName)
		if !ok {
			continue
		}
		if _, exists := out[ifaceSym]; exists {
			continue
		}
		ifaceMethods := parseInterfaceMethods(mod, ifaceSym)
		if len(ifaceMethods) == 0 {
			continue
		}
		out[ifaceSym] = ifaceMethods
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneInvokeIfaceMethodsMap(src map[string][]invokeIfaceMethod) map[string][]invokeIfaceMethod {
	if len(src) == 0 {
		return make(map[string][]invokeIfaceMethod)
	}
	dst := make(map[string][]invokeIfaceMethod, len(src))
	for k, methods := range src {
		if len(methods) == 0 {
			dst[k] = nil
			continue
		}
		buf := make([]invokeIfaceMethod, len(methods))
		copy(buf, methods)
		dst[k] = buf
	}
	return dst
}

func parseInterfaceMethods(mod gllvm.Module, ifaceSym string) []invokeIfaceMethod {
	ifaceGlobal := resolveInterfaceTypeGlobal(mod, ifaceSym)
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
		out = append(out, invokeIfaceMethod{
			key:  methodKey(name, typeSym),
			name: name,
		})
	}
	return out
}

func resolveInterfaceTypeGlobal(mod gllvm.Module, ifaceSym string) gllvm.Value {
	if ifaceSym == "" {
		return gllvm.Value{}
	}
	if g := mod.NamedGlobal(ifaceSym); !isNilValue(g) {
		return g
	}
	suffix, ok := ifaceSymbolSuffix(ifaceSym)
	if !ok {
		return gllvm.Value{}
	}
	// Common canonical form for interface type symbols.
	if g := mod.NamedGlobal("_llgo_" + suffix); !isNilValue(g) {
		return g
	}

	// Fallback: package-qualified thunk names may use "<pkg>.iface$<hash>" while
	// the actual type symbol can be emitted as "_llgo_iface$<hash>" or another
	// package-qualified variant that shares the same iface hash.
	for g := mod.FirstGlobal(); !isNilValue(g); g = gllvm.NextGlobal(g) {
		name := g.Name()
		if name == "" || strings.HasPrefix(name, "*") {
			continue
		}
		if strings.HasSuffix(name, suffix) {
			return g
		}
	}
	return gllvm.Value{}
}

func ifaceSymbolSuffix(ifaceSym string) (suffix string, ok bool) {
	idx := strings.Index(ifaceSym, "iface$")
	if idx < 0 {
		return "", false
	}
	return ifaceSym[idx:], true
}

func collectConcreteTypeMethods(mod gllvm.Module) []invokeTypeMethods {
	out := collectConcreteTypeMethodsFromAbiMethodTable(mod)
	out = mergeInvokeTypeMethods(out, collectConcreteTypeMethodsFromAttrs(mod))
	return out
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

func mergeInvokeTypeMethods(dst []invokeTypeMethods, src []invokeTypeMethods) []invokeTypeMethods {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		out := make([]invokeTypeMethods, 0, len(src))
		for _, typ := range src {
			if typ.TypeSymbol == "" || len(typ.Methods) == 0 {
				continue
			}
			methods := make(map[string]string, len(typ.Methods))
			for k, v := range typ.Methods {
				if v == "" {
					continue
				}
				methods[k] = v
			}
			if len(methods) == 0 {
				continue
			}
			out = append(out, invokeTypeMethods{
				TypeSymbol: typ.TypeSymbol,
				Methods:    methods,
			})
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].TypeSymbol < out[j].TypeSymbol
		})
		return out
	}

	indexByType := make(map[string]int, len(dst))
	for i, typ := range dst {
		if typ.TypeSymbol == "" {
			continue
		}
		if typ.Methods == nil {
			dst[i].Methods = make(map[string]string)
		}
		indexByType[typ.TypeSymbol] = i
	}

	for _, typ := range src {
		if typ.TypeSymbol == "" || len(typ.Methods) == 0 {
			continue
		}
		i, ok := indexByType[typ.TypeSymbol]
		if !ok {
			methods := make(map[string]string, len(typ.Methods))
			for k, v := range typ.Methods {
				if v == "" {
					continue
				}
				methods[k] = v
			}
			if len(methods) == 0 {
				continue
			}
			dst = append(dst, invokeTypeMethods{
				TypeSymbol: typ.TypeSymbol,
				Methods:    methods,
			})
			indexByType[typ.TypeSymbol] = len(dst) - 1
			continue
		}
		methods := dst[i].Methods
		for k, v := range typ.Methods {
			if v == "" {
				continue
			}
			methods[k] = v
		}
	}

	sort.Slice(dst, func(i, j int) bool {
		return dst[i].TypeSymbol < dst[j].TypeSymbol
	})
	return dst
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
		if _, ok := resolveMethodKey(typeMethods, im); !ok {
			return false
		}
	}
	return true
}

func resolveMethodKey(typeMethods map[string]string, ifaceMethod invokeIfaceMethod) (string, bool) {
	if typeMethods == nil {
		return "", false
	}
	if _, ok := typeMethods[ifaceMethod.key]; ok {
		return ifaceMethod.key, true
	}
	if ifaceMethod.name == "" {
		return "", false
	}
	prefix := ifaceMethod.name + "\x00"
	match := ""
	for key := range typeMethods {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if match == "" || key < match {
			match = key
		}
	}
	if match == "" {
		return "", false
	}
	return match, true
}

func methodKey(name, typeSym string) string {
	return name + "\x00" + typeSym
}

type loweringPatchEmitter struct {
	srcMod gllvm.Module
	mod    gllvm.Module
	ctx    gllvm.Context

	rtIfacePtrData   gllvm.Value
	rtImplements     gllvm.Value
	rtMatchConcrete  gllvm.Value
	rtMatchesClosure gllvm.Value
}

func emitLoweringPatchModule(srcMod gllvm.Module, invokePlans []invokeThunkPlan, typeAssertPlans []typeAssertThunkPlan) (gllvm.Module, int) {
	ctx := srcMod.Context()
	patchMod := ctx.NewModule("llgo.lowering")
	patchMod.SetDataLayout(srcMod.DataLayout())
	patchMod.SetTarget(srcMod.Target())

	emitter := &loweringPatchEmitter{
		srcMod: srcMod,
		mod:    patchMod,
		ctx:    ctx,
	}

	patched := 0
	for _, plan := range invokePlans {
		if emitter.emitInvokeThunk(plan) {
			patched++
		}
	}
	for _, plan := range typeAssertPlans {
		if emitter.emitTypeAssertThunk(plan) {
			patched++
		}
	}
	return patchMod, patched
}

func (e *loweringPatchEmitter) emitInvokeThunk(plan invokeThunkPlan) bool {
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

func (e *loweringPatchEmitter) emitTypeAssertThunk(plan typeAssertThunkPlan) bool {
	srcThunk := e.srcMod.NamedFunction(plan.ThunkName)
	if isNilValue(srcThunk) {
		return false
	}
	thunkTy := srcThunk.GlobalValueType()
	if thunkTy.TypeKind() != gllvm.FunctionTypeKind {
		return false
	}
	paramTypes := thunkTy.ParamTypes()
	if len(paramTypes) != 1 {
		return false
	}

	thunk := gllvm.AddFunction(e.mod, plan.ThunkName, thunkTy)
	thunk.SetLinkage(gllvm.ExternalLinkage)
	thunk.SetFunctionCallConv(srcThunk.FunctionCallConv())

	b := e.ctx.NewBuilder()
	defer b.Dispose()

	entry := e.ctx.AddBasicBlock(thunk, "entry")
	fallback := e.ctx.AddBasicBlock(thunk, "fallback")
	b.SetInsertPointAtEnd(entry)

	params := thunk.Params()
	if len(params) != 1 {
		return false
	}
	actualType := params[0]
	assertedType := e.ensureTypeGlobalDecl(plan.AssertedTypeSymbol)

	switch plan.Kind {
	case typeAssertThunkKindIface:
		dispatch := entry
		for i, targetTypeSym := range plan.Targets {
			b.SetInsertPointAtEnd(dispatch)
			match := e.ctx.AddBasicBlock(thunk, fmt.Sprintf("assert.type.%d", i))
			miss := fallback
			if i < len(plan.Targets)-1 {
				miss = e.ctx.AddBasicBlock(thunk, fmt.Sprintf("assert.type.next.%d", i))
			}
			targetType := e.ensureTypeGlobalDecl(targetTypeSym)
			if isNilValue(targetType) {
				continue
			}
			cmp := b.CreateICmp(gllvm.IntEQ, actualType, targetType, "")
			b.CreateCondBr(cmp, match, miss)

			b.SetInsertPointAtEnd(match)
			b.CreateRet(gllvm.ConstInt(thunkTy.ReturnType(), 1, false))
			dispatch = miss
		}
		if len(plan.Targets) == 0 {
			b.SetInsertPointAtEnd(entry)
			b.CreateBr(fallback)
		}
	case typeAssertThunkKindConcrete, typeAssertThunkKindClosure:
		if !isNilValue(assertedType) {
			match := e.ctx.AddBasicBlock(thunk, "match")
			cmp := b.CreateICmp(gllvm.IntEQ, actualType, assertedType, "")
			b.CreateCondBr(cmp, match, fallback)
			b.SetInsertPointAtEnd(match)
			b.CreateRet(gllvm.ConstInt(thunkTy.ReturnType(), 1, false))
		} else {
			b.CreateBr(fallback)
		}
	default:
		return false
	}

	if isNilValue(assertedType) {
		// Keep behavior deterministic in malformed modules.
		b.SetInsertPointAtEnd(fallback)
		b.CreateRet(gllvm.ConstInt(thunkTy.ReturnType(), 0, false))
		return true
	}

	b.SetInsertPointAtEnd(fallback)
	boolTy := thunkTy.ReturnType()
	argTy := paramTypes[0]
	checkTy := gllvm.FunctionType(boolTy, []gllvm.Type{argTy, argTy}, false)
	var callee gllvm.Value
	switch plan.Kind {
	case typeAssertThunkKindIface:
		callee = e.ensureImplementsDecl(argTy, boolTy)
	case typeAssertThunkKindConcrete:
		callee = e.ensureMatchConcreteDecl(argTy, boolTy)
	case typeAssertThunkKindClosure:
		callee = e.ensureMatchesClosureDecl(argTy, boolTy)
	default:
		return false
	}
	ret := b.CreateCall(checkTy, callee, []gllvm.Value{assertedType, actualType}, "")
	b.CreateRet(ret)
	return true
}

func (e *loweringPatchEmitter) ensureIfacePtrDataDecl(ifaceTy gllvm.Type) gllvm.Value {
	if !isNilValue(e.rtIfacePtrData) {
		return e.rtIfacePtrData
	}
	ptrTy := gllvm.PointerType(e.ctx.Int8Type(), 0)
	fnTy := gllvm.FunctionType(ptrTy, []gllvm.Type{ifaceTy}, false)
	e.rtIfacePtrData = e.ensureFunctionDecl(ifacePtrDataFuncName, fnTy)
	return e.rtIfacePtrData
}

func (e *loweringPatchEmitter) ensureImplementsDecl(typePtrTy, boolTy gllvm.Type) gllvm.Value {
	if !isNilValue(e.rtImplements) {
		return e.rtImplements
	}
	fnTy := gllvm.FunctionType(boolTy, []gllvm.Type{typePtrTy, typePtrTy}, false)
	e.rtImplements = e.ensureFunctionDecl(rtImplementsFuncName, fnTy)
	return e.rtImplements
}

func (e *loweringPatchEmitter) ensureMatchConcreteDecl(typePtrTy, boolTy gllvm.Type) gllvm.Value {
	if !isNilValue(e.rtMatchConcrete) {
		return e.rtMatchConcrete
	}
	fnTy := gllvm.FunctionType(boolTy, []gllvm.Type{typePtrTy, typePtrTy}, false)
	e.rtMatchConcrete = e.ensureFunctionDecl(rtMatchConcreteFuncName, fnTy)
	return e.rtMatchConcrete
}

func (e *loweringPatchEmitter) ensureMatchesClosureDecl(typePtrTy, boolTy gllvm.Type) gllvm.Value {
	if !isNilValue(e.rtMatchesClosure) {
		return e.rtMatchesClosure
	}
	fnTy := gllvm.FunctionType(boolTy, []gllvm.Type{typePtrTy, typePtrTy}, false)
	e.rtMatchesClosure = e.ensureFunctionDecl(rtMatchesClosureFuncName, fnTy)
	return e.rtMatchesClosure
}

func (e *loweringPatchEmitter) ensureFunctionDecl(name string, fnTy gllvm.Type) gllvm.Value {
	fn := e.mod.NamedFunction(name)
	if !isNilValue(fn) {
		return fn
	}
	fn = gllvm.AddFunction(e.mod, name, fnTy)
	fn.SetLinkage(gllvm.ExternalLinkage)
	return fn
}

func (e *loweringPatchEmitter) ensureTypeGlobalDecl(name string) gllvm.Value {
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

func isNilModule(m gllvm.Module) bool {
	return m.C == nil
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
	if val == "" {
		return true
	}
	switch strings.ToLower(val) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

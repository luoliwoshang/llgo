package dce

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	llvm "github.com/goplus/llvm"
)

// Result is the phase-1 method liveness output keyed by concrete type symbol.
// Each inner set contains the live abi.Method slot indexes for that type.
type Result map[string]map[int]struct{}

type AnalyzeInputStats struct {
	Iterations       int
	ReachableSymbols int
	UsedInIfaceTypes int
	LiveTypes        int
	LiveMethods      int
	Total            time.Duration
}

type AnalyzeStats struct {
	BuildInput   BuildInputStats
	AnalyzeInput AnalyzeInputStats
	Total        time.Duration
}

// Input is the preprocessed analyzer input built from LLVM modules.
// The goal is to isolate LLVM scanning in BuildInput so the core analysis
// can operate on plain Go data structures.
type Input struct {
	OrdinaryEdges map[string]map[string]struct{}
	TypeChildren  map[string]map[string]struct{}
	MethodRefs    map[string]map[int]map[string]struct{}

	InterfaceInfo  []InterfaceInfoRow
	UseIface       []UseIfaceRow
	UseIfaceMethod []UseIfaceMethodRow
	MethodOff      []MethodOffRow
	UseNamedMethod []UseNamedMethodRow
	ReflectMethod  []ReflectMethodRow
}

type InterfaceInfoRow struct {
	Target string
	Name   string
	MTyp   string
}

type UseIfaceRow struct {
	Owner  string
	Target string
}

type UseIfaceMethodRow struct {
	Owner  string
	Target string
	Name   string
	MTyp   string
}

type MethodOffRow struct {
	TypeName string
	Index    int
	Name     string
	MTyp     string
}

type UseNamedMethodRow struct {
	Owner string
	Name  string
}

type ReflectMethodRow struct {
	Owner string
}

type methodSig struct {
	Name string
	MTyp string
}

type analyzer struct {
	input Input

	reachable   map[string]struct{}
	worklist    []string
	usedInIface map[string]struct{}
	ifaceDemand map[string]map[methodSig]struct{}
	namedDemand map[string]struct{}
	reflectSeen bool
	result      Result

	interfaceInfo map[string]map[methodSig]struct{}
	typeMethods   map[string]map[methodSig]struct{}
}

// Analyze is the package-level entry point used by the build pipeline.
// It first builds a pure-Go Input from the provided modules, then runs the
// phase-1 method reachability analysis on that input.
func Analyze(mods []llvm.Module, roots []string) (Result, error) {
	result, _, err := AnalyzeWithStats(mods, roots)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// AnalyzeWithStats runs the phase-1 analysis and returns detailed timing and
// size counters for verbose build diagnostics.
func AnalyzeWithStats(mods []llvm.Module, roots []string) (Result, AnalyzeStats, error) {
	start := time.Now()
	input, buildStats, err := BuildInputWithStats(mods)
	if err != nil {
		return nil, AnalyzeStats{BuildInput: buildStats, Total: time.Since(start)}, err
	}
	result, analyzeStats := AnalyzeInputWithStats(input, roots)
	return result, AnalyzeStats{
		BuildInput:   buildStats,
		AnalyzeInput: analyzeStats,
		Total:        time.Since(start),
	}, nil
}

// AnalyzeInput runs the phase-1 method reachability analysis on preprocessed
// analyzer input.
func AnalyzeInput(input Input, roots []string) Result {
	result, _ := AnalyzeInputWithStats(input, roots)
	return result
}

// AnalyzeInputWithStats runs the core analyzer and records algorithm-side
// timings and counts.
func AnalyzeInputWithStats(input Input, roots []string) (Result, AnalyzeInputStats) {
	start := time.Now()
	a := analyzer{
		input:       input,
		reachable:   make(map[string]struct{}),
		usedInIface: make(map[string]struct{}),
		ifaceDemand: make(map[string]map[methodSig]struct{}),
		namedDemand: make(map[string]struct{}),
		result:      make(Result),
		interfaceInfo: buildSigSets(input.InterfaceInfo, func(row InterfaceInfoRow) string {
			return row.Target
		}, func(row InterfaceInfoRow) methodSig {
			return methodSig{Name: row.Name, MTyp: row.MTyp}
		}),
		typeMethods: buildSigSets(input.MethodOff, func(row MethodOffRow) string {
			return row.TypeName
		}, func(row MethodOffRow) methodSig {
			return methodSig{Name: row.Name, MTyp: row.MTyp}
		}),
	}
	for _, root := range roots {
		a.markReachable(root)
	}
	iterations := 0
	for {
		iterations++
		a.flood()
		changed := a.activateMetadata()
		changed = a.markMethods() || changed
		if len(a.worklist) == 0 && !changed {
			break
		}
	}
	a.ensurePrunableTypes()
	return a.result, AnalyzeInputStats{
		Iterations:       iterations,
		ReachableSymbols: len(a.reachable),
		UsedInIfaceTypes: len(a.usedInIface),
		LiveTypes:        len(a.result),
		LiveMethods:      countLiveMethods(a.result),
		Total:            time.Since(start),
	}
}

func (a *analyzer) flood() {
	for len(a.worklist) != 0 {
		last := len(a.worklist) - 1
		sym := a.worklist[last]
		a.worklist = a.worklist[:last]
		for dst := range a.input.OrdinaryEdges[sym] {
			a.markReachable(dst)
		}
	}
}

func (a *analyzer) activateMetadata() bool {
	changed := false
	for _, row := range a.input.UseIface {
		if a.isReachable(row.Owner) {
			changed = a.markUsedInIface(row.Target) || changed
		}
	}
	for _, row := range a.input.UseIfaceMethod {
		if a.isReachable(row.Owner) {
			changed = a.addIfaceDemand(row.Target, methodSig{Name: row.Name, MTyp: row.MTyp}) || changed
		}
	}
	for _, row := range a.input.UseNamedMethod {
		if a.isReachable(row.Owner) {
			changed = a.addNamedDemand(row.Name) || changed
		}
	}
	for _, row := range a.input.ReflectMethod {
		if a.isReachable(row.Owner) && !a.reflectSeen {
			a.reflectSeen = true
			changed = true
		}
	}
	return changed
}

func (a *analyzer) markMethods() bool {
	changed := false
	for _, row := range a.input.MethodOff {
		if !a.isUsedInIface(row.TypeName) || !a.shouldKeepMethod(row) {
			continue
		}
		if !a.addLiveMethod(row.TypeName, row.Index) {
			continue
		}
		changed = true
		for sym := range a.input.MethodRefs[row.TypeName][row.Index] {
			if a.markReachable(sym) {
				changed = true
			}
		}
	}
	return changed
}

func (a *analyzer) shouldKeepMethod(row MethodOffRow) bool {
	if a.hasSatisfiedIfaceDemand(row.TypeName, methodSig{Name: row.Name, MTyp: row.MTyp}) {
		return true
	}
	if _, ok := a.namedDemand[row.Name]; ok {
		return true
	}
	return a.reflectSeen && isExportedMethod(row.Name)
}

func (a *analyzer) markReachable(sym string) bool {
	if sym == "" {
		return false
	}
	if _, ok := a.reachable[sym]; ok {
		return false
	}
	a.reachable[sym] = struct{}{}
	a.worklist = append(a.worklist, sym)
	return true
}

func (a *analyzer) markUsedInIface(typeName string) bool {
	if typeName == "" {
		return false
	}
	changed := false
	work := []string{typeName}
	for len(work) != 0 {
		last := len(work) - 1
		sym := work[last]
		work = work[:last]
		if _, ok := a.usedInIface[sym]; ok {
			continue
		}
		a.usedInIface[sym] = struct{}{}
		changed = true
		for child := range a.input.TypeChildren[sym] {
			work = append(work, child)
		}
	}
	return changed
}

func (a *analyzer) addIfaceDemand(target string, sig methodSig) bool {
	if target == "" || sig.Name == "" || sig.MTyp == "" {
		return false
	}
	byTarget := a.ifaceDemand[target]
	if byTarget == nil {
		byTarget = make(map[methodSig]struct{})
		a.ifaceDemand[target] = byTarget
	}
	if _, ok := byTarget[sig]; ok {
		return false
	}
	byTarget[sig] = struct{}{}
	return true
}

func (a *analyzer) addNamedDemand(name string) bool {
	if name == "" {
		return false
	}
	if _, ok := a.namedDemand[name]; ok {
		return false
	}
	a.namedDemand[name] = struct{}{}
	return true
}

func (a *analyzer) addLiveMethod(typeName string, index int) bool {
	byIndex := a.result[typeName]
	if byIndex == nil {
		byIndex = make(map[int]struct{})
		a.result[typeName] = byIndex
	}
	if _, ok := byIndex[index]; ok {
		return false
	}
	byIndex[index] = struct{}{}
	return true
}

func (a *analyzer) isReachable(sym string) bool {
	_, ok := a.reachable[sym]
	return ok
}

func (a *analyzer) hasSatisfiedIfaceDemand(typeName string, sig methodSig) bool {
	for target, demanded := range a.ifaceDemand {
		if _, ok := demanded[sig]; !ok {
			continue
		}
		if a.typeImplementsInterface(typeName, target) {
			return true
		}
	}
	return false
}

func (a *analyzer) typeImplementsInterface(typeName, target string) bool {
	required := a.interfaceInfo[target]
	if len(required) == 0 {
		return false
	}
	have := a.typeMethods[typeName]
	if len(have) < len(required) {
		return false
	}
	for sig := range required {
		if _, ok := have[sig]; !ok {
			return false
		}
	}
	return true
}

func (a *analyzer) isUsedInIface(typeName string) bool {
	_, ok := a.usedInIface[typeName]
	return ok
}

func (a *analyzer) ensurePrunableTypes() {
	for _, row := range a.input.MethodOff {
		if !a.isUsedInIface(row.TypeName) {
			continue
		}
		if _, ok := a.result[row.TypeName]; ok {
			continue
		}
		a.result[row.TypeName] = make(map[int]struct{})
	}
}

func isExportedMethod(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

func buildSigSets[T any](rows []T, group func(T) string, sig func(T) methodSig) map[string]map[methodSig]struct{} {
	out := make(map[string]map[methodSig]struct{})
	for _, row := range rows {
		key := group(row)
		ms := sig(row)
		if key == "" || ms.Name == "" || ms.MTyp == "" {
			continue
		}
		set := out[key]
		if set == nil {
			set = make(map[methodSig]struct{})
			out[key] = set
		}
		set[ms] = struct{}{}
	}
	return out
}

func countLiveMethods(result Result) int {
	total := 0
	for _, byIndex := range result {
		total += len(byIndex)
	}
	return total
}

// FormatResult renders the analyzer output as stable text lines:
//
//	type symbol: [sorted method indexes]
func FormatResult(result Result) string {
	if len(result) == 0 {
		return ""
	}
	typeNames := make([]string, 0, len(result))
	for typeName := range result {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	var b strings.Builder
	for _, typeName := range typeNames {
		indexes := make([]int, 0, len(result[typeName]))
		for index := range result[typeName] {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)

		b.WriteString(typeName)
		b.WriteString(": [")
		for i, index := range indexes {
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strconv.Itoa(index))
		}
		b.WriteString("]\n")
	}
	return b.String()
}

// FormatAnalyzeStats renders timing and size counters in a stable text form
// suitable for verbose build logs.
func FormatAnalyzeStats(stats AnalyzeStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "build_input.total: %s\n", stats.BuildInput.Total)
	fmt.Fprintf(&b, "build_input.modules: %d\n", stats.BuildInput.Modules)
	fmt.Fprintf(&b, "build_input.ordinary_edges: %s\n", stats.BuildInput.OrdinaryEdges)
	fmt.Fprintf(&b, "build_input.type_children: %s\n", stats.BuildInput.TypeChildren)
	fmt.Fprintf(&b, "build_input.method_refs: %s\n", stats.BuildInput.MethodRefs)
	fmt.Fprintf(&b, "build_input.metadata: %s\n", stats.BuildInput.Metadata)
	fmt.Fprintf(&b, "analyze_input.total: %s\n", stats.AnalyzeInput.Total)
	fmt.Fprintf(&b, "analyze_input.iterations: %d\n", stats.AnalyzeInput.Iterations)
	fmt.Fprintf(&b, "analyze_input.reachable_symbols: %d\n", stats.AnalyzeInput.ReachableSymbols)
	fmt.Fprintf(&b, "analyze_input.used_in_iface_types: %d\n", stats.AnalyzeInput.UsedInIfaceTypes)
	fmt.Fprintf(&b, "analyze_input.live_types: %d\n", stats.AnalyzeInput.LiveTypes)
	fmt.Fprintf(&b, "analyze_input.live_methods: %d\n", stats.AnalyzeInput.LiveMethods)
	fmt.Fprintf(&b, "total: %s\n", stats.Total)
	return b.String()
}

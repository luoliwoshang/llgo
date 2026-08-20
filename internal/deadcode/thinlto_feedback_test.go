package deadcode_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/dcepass"
	"github.com/goplus/llgo/internal/deadcode"
	"github.com/goplus/llgo/internal/meta"
	"github.com/xgo-dev/llvm"
)

func TestThinLTOFeedbackShrinksMethodPlan(t *testing.T) {
	summary := feedbackSummary(t)
	first := deadcode.BuildPlan(summary, []string{"main"})
	wantFirst := map[string][]int{"_llgo_feedback.T": {0}}
	if !reflect.DeepEqual(first.LiveSlots, wantFirst) {
		t.Fatalf("first plan LiveSlots = %#v, want %#v", first.LiveSlots, wantFirst)
	}

	for _, tt := range []struct {
		name       string
		demandFile string
		wantDead   bool
	}{
		{name: "constant false drops demand", demandFile: "demand.ll", wantDead: true},
		{name: "constant true keeps demand", demandFile: "demand_live.ll", wantDead: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dead := runThinLTOFeedback(t, tt.demandFile)
			_, isDead := dead["semanticDemand"]
			if isDead != tt.wantDead {
				t.Fatalf("post-ThinLTO feedback = %#v, semanticDemand dead = %v, want %v", dead, isDead, tt.wantDead)
			}
			second := deadcode.BuildPlanWithFeedback(summary, []string{"main"}, deadcode.Feedback{DeadFunctions: dead})
			if tt.wantDead {
				if len(second.LiveSlots) != 0 {
					t.Fatalf("feedback plan LiveSlots = %#v, want empty", second.LiveSlots)
				}
			} else if !reflect.DeepEqual(second.LiveSlots, wantFirst) {
				t.Fatalf("feedback plan LiveSlots = %#v, want %#v", second.LiveSlots, wantFirst)
			}
		})
	}
}

func TestThinLTOFeedbackRewriteRemovesMethodBodies(t *testing.T) {
	opt := requireTool(t, "opt")
	linker := requireTool(t, "ld.lld")
	readelf := requireTool(t, "llvm-readelf")
	tmp := t.TempDir()
	mainObj := filepath.Join(tmp, "main.o")
	demandObj := filepath.Join(tmp, "demand.o")
	firstApp := filepath.Join(tmp, "first")
	secondApp := filepath.Join(tmp, "second")
	runTool(t, opt, "-module-summary", filepath.Join("testdata", "thinlto_feedback", "method_main.ll"), "-o", mainObj)
	runTool(t, opt, "-module-summary", filepath.Join("testdata", "thinlto_feedback", "method_demand.ll"), "-o", demandObj)
	runTool(t, linker, "--export-dynamic", "--entry=main", "--save-temps", "--lto-O2", "-o", firstApp, mainObj, demandObj)

	firstSymbols := commandOutput(t, readelf, "-s", firstApp)
	for _, name := range []string{"T.M", "T.N"} {
		if !strings.Contains(firstSymbols, name) {
			t.Fatalf("first ThinLTO link symbols missing %s:\n%s", name, firstSymbols)
		}
	}

	feedbackCtx := llvm.NewContext()
	feedbackMods := make([]llvm.Module, 0, 2)
	for _, path := range []string{mainObj + ".4.opt.bc", demandObj + ".4.opt.bc"} {
		feedbackMod, err := feedbackCtx.ParseBitcodeFile(path)
		if err != nil {
			feedbackCtx.Dispose()
			t.Fatalf("parse optimized feedback module %s: %v", path, err)
		}
		feedbackMods = append(feedbackMods, feedbackMod)
	}
	if feedbackMods[1].NamedFunction("semanticDemand").IsNil() {
		feedbackCtx.Dispose()
		t.Fatal("expected initial ThinLTO round to retain semanticDemand definition")
	}
	deadFunctions := dcepass.DeadNoInlineFunctionsFromModules(feedbackMods, []string{"main"}, []string{"semanticDemand"})
	for _, feedbackMod := range feedbackMods {
		feedbackMod.Dispose()
	}
	feedbackCtx.Dispose()
	if _, ok := deadFunctions["semanticDemand"]; !ok {
		t.Fatalf("post-ThinLTO feedback = %#v, want semanticDemand dead", deadFunctions)
	}
	secondPlan := deadcode.BuildPlanWithFeedback(feedbackSummary(t), []string{"main"}, deadcode.Feedback{DeadFunctions: deadFunctions})
	if len(secondPlan.LiveSlots) != 0 {
		t.Fatalf("second plan LiveSlots = %#v, want empty", secondPlan.LiveSlots)
	}

	ctx := llvm.NewContext()
	mod, err := ctx.ParseBitcodeFile(demandObj + ".0.preopt.bc")
	if err != nil {
		ctx.Dispose()
		t.Fatalf("parse preopt demand module: %v", err)
	}
	if got := dcepass.RewriteTypeMethodTables(mod, secondPlan.LiveSlots, false); got != 1 {
		mod.Dispose()
		ctx.Dispose()
		t.Fatalf("RewriteTypeMethodTables rewrote %d globals, want 1", got)
	}
	buf := llvm.WriteThinLTOBitcodeToMemoryBuffer(mod)
	mod.Dispose()
	ctx.Dispose()
	rewrittenObj := filepath.Join(tmp, "demand.rewritten.o")
	if err := os.WriteFile(rewrittenObj, buf.Bytes(), 0o644); err != nil {
		buf.Dispose()
		t.Fatalf("write rewritten ThinLTO module: %v", err)
	}
	buf.Dispose()

	runTool(t, linker, "--export-dynamic", "--entry=main", "--lto-O2", "-o", secondApp, mainObj, rewrittenObj)
	secondSymbols := commandOutput(t, readelf, "-s", secondApp)
	for _, name := range []string{"T.M", "T.N"} {
		if strings.Contains(secondSymbols, name) {
			t.Fatalf("second ThinLTO link retained rewritten dead method %s:\n%s", name, secondSymbols)
		}
	}
}

func runThinLTOFeedback(t *testing.T, demandFixture string) map[string]struct{} {
	t.Helper()
	opt := requireTool(t, "opt")
	linker := requireTool(t, "ld.lld")
	tmp := t.TempDir()
	mainObj := filepath.Join(tmp, "main.o")
	demandObj := filepath.Join(tmp, "demand.o")
	runTool(t, opt, "-module-summary", filepath.Join("testdata", "thinlto_feedback", "main.ll"), "-o", mainObj)
	runTool(t, opt, "-module-summary", filepath.Join("testdata", "thinlto_feedback", demandFixture), "-o", demandObj)
	runTool(t, linker, "--entry=main", "--save-temps", "--lto-O2", "-o", filepath.Join(tmp, "app"), mainObj, demandObj)

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mods := make([]llvm.Module, 0, 2)
	for _, path := range []string{mainObj + ".4.opt.bc", demandObj + ".4.opt.bc"} {
		mod, err := ctx.ParseBitcodeFile(path)
		if err != nil {
			t.Fatalf("parse ThinLTO optimized module %s: %v", path, err)
		}
		defer mod.Dispose()
		mods = append(mods, mod)
	}

	// In the false case ThinLTO removes main's call, but its initial combined
	// index still makes the other backend retain the definition. Recomputing
	// roots from post-opt references discovers the new global fixed point.
	if mods[1].NamedFunction("semanticDemand").IsNil() {
		t.Fatal("expected initial ThinLTO round to retain semanticDemand definition")
	}
	return dcepass.DeadNoInlineFunctionsFromModules(mods, []string{"main"}, []string{"semanticDemand"})
}

func feedbackSummary(t *testing.T) *meta.GlobalSummary {
	t.Helper()
	b := meta.NewBuilder()
	main := b.Sym("main")
	demand := b.Sym("semanticDemand")
	typ := b.Sym("_llgo_feedback.T")
	iface := b.Sym("_llgo_feedback.I")
	mtype := b.Sym("_llgo_func$M")
	b.AddOrdinaryEdge(mtype, mtype)
	b.AddIfaceMethod(iface, "M", mtype)
	b.AddMethodSlot(typ, "M", mtype, b.Sym("feedback.(*T).M"), b.Sym("feedback.T.M"))
	b.AddOrdinaryEdge(main, demand)
	b.AddOrdinaryEdge(demand, typ)
	b.AddIfaceUse(demand, typ)
	b.AddIfaceMethodUse(demand, iface, 0)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	summary, err := meta.NewGlobalSummary([]*meta.PackageMeta{pm})
	if err != nil {
		t.Fatal(err)
	}
	return summary
}

func requireTool(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s is required for ThinLTO feedback integration test", name)
	}
	return path
}

func runTool(t *testing.T, tool string, args ...string) {
	t.Helper()
	if out, err := exec.Command(tool, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", tool, args, err, out)
	}
}

func commandOutput(t *testing.T, tool string, args ...string) string {
	t.Helper()
	out, err := exec.Command(tool, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", tool, args, err, out)
	}
	return string(out)
}

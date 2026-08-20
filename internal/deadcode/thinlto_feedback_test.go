package deadcode_test

import (
	"os/exec"
	"path/filepath"
	"reflect"
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

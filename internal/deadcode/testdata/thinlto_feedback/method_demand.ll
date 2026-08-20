target triple = "x86_64-unknown-linux-gnu"

%"github.com/goplus/llgo/runtime/abi.Method" = type { ptr, ptr, ptr, ptr }
%T.type = type { ptr, [2 x %"github.com/goplus/llgo/runtime/abi.Method"] }
@sink = hidden global ptr null
@T = constant %T.type {
  ptr null,
  [2 x %"github.com/goplus/llgo/runtime/abi.Method"] [
    %"github.com/goplus/llgo/runtime/abi.Method" { ptr null, ptr null, ptr @T.M, ptr @T.M },
    %"github.com/goplus/llgo/runtime/abi.Method" { ptr null, ptr null, ptr @T.N, ptr @T.N }
  ]
}

define hidden void @semanticDemand() #0 {
entry:
  ret void
}

define hidden void @T.M() #0 {
entry:
  call void asm sideeffect "", "~{memory}"()
  ret void
}

define hidden void @T.N() #0 {
entry:
  call void asm sideeffect "", "~{memory}"()
  ret void
}

define hidden void @"github.com/goplus/llgo/runtime/internal/runtime.unreachableMethod"() {
entry:
  ret void
}

attributes #0 = { noinline }

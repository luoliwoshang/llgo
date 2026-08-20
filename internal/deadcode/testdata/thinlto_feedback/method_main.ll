target triple = "x86_64-unknown-linux-gnu"

%"github.com/goplus/llgo/runtime/abi.Method" = type { ptr, ptr, ptr, ptr }
%T.type = type { ptr, [2 x %"github.com/goplus/llgo/runtime/abi.Method"] }
@T = external constant %T.type
@llvm.used = appending global [1 x ptr] [ptr @T]
@sink = external global ptr
@flag = constant i1 false

define i32 @main() {
entry:
  call void @keepType()
  %enabled = load i1, ptr @flag
  br i1 %enabled, label %demand, label %done
demand:
  call void @semanticDemand()
  br label %done
done:
  ret i32 0
}

define internal void @keepType() {
entry:
  store ptr @T, ptr @sink
  ret void
}

declare hidden void @semanticDemand()

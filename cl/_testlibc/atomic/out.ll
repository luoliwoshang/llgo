; ModuleID = 'main'
source_filename = "main"

@"main.init$guard" = global i1 false, align 1
@__llgo_argc = global i32 0, align 4
@__llgo_argv = global ptr null, align 8
@0 = private unnamed_addr constant [12 x i8] c"store: %ld\0A\00", align 1
@1 = private unnamed_addr constant [8 x i8] c"v: %ld\0A\00", align 1
@2 = private unnamed_addr constant [8 x i8] c"v: %ld\0A\00", align 1
@3 = private unnamed_addr constant [8 x i8] c"v: %ld\0A\00", align 1
@4 = private unnamed_addr constant [8 x i8] c"v: %ld\0A\00", align 1

define void @main.init() {
_llgo_0:
  %0 = load i1, ptr @"main.init$guard", align 1
  br i1 %0, label %_llgo_2, label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_0
  store i1 true, ptr @"main.init$guard", align 1
  br label %_llgo_2

_llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
  ret void
}

define i32 @main(i32 %0, ptr %1) {
_llgo_0:
  store i32 %0, ptr @__llgo_argc, align 4
  store ptr %1, ptr @__llgo_argv, align 8
  call void @"github.com/goplus/llgo/internal/runtime.init"()
  call void @main.init()
  %2 = call ptr @"github.com/goplus/llgo/internal/runtime.AllocZ"(i64 8)
  store atomic i64 100, ptr %2 seq_cst, align 4
  %3 = load atomic i64, ptr %2 seq_cst, align 4
  %4 = call i32 (ptr, ...) @printf(ptr @0, i64 %3)
  %5 = atomicrmw add ptr %2, i64 1 seq_cst, align 8
  %6 = load i64, ptr %2, align 4
  %7 = call i32 (ptr, ...) @printf(ptr @1, i64 %6)
  %8 = cmpxchg ptr %2, i64 100, i64 102 seq_cst seq_cst, align 8
  %9 = load i64, ptr %2, align 4
  %10 = call i32 (ptr, ...) @printf(ptr @2, i64 %9)
  %11 = cmpxchg ptr %2, i64 101, i64 102 seq_cst seq_cst, align 8
  %12 = load i64, ptr %2, align 4
  %13 = call i32 (ptr, ...) @printf(ptr @3, i64 %12)
  %14 = atomicrmw sub ptr %2, i64 1 seq_cst, align 8
  %15 = load i64, ptr %2, align 4
  %16 = call i32 (ptr, ...) @printf(ptr @4, i64 %15)
  ret i32 0
}

declare void @"github.com/goplus/llgo/internal/runtime.init"()

declare ptr @"github.com/goplus/llgo/internal/runtime.AllocZ"(i64)

declare i32 @printf(ptr, ...)

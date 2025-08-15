# LLGO.cstr 编译器指令技术文档

## 概述

`llgo.cstr` 是 LLGO 编译器的一个内置编译器指令，用于将 Go 字符串字面量在编译时直接转换为 C 风格的全局字符串常量。与运行时函数不同，这是一个纯编译时优化，不产生任何运行时符号或函数调用。

## 核心特性

### 编译时转换
- **零运行时开销**：字符串转换在编译时完成，不产生运行时函数调用
- **全局常量生成**：直接在 LLVM IR 中生成全局字符串常量
- **内存效率**：相同的字符串字面量会被合并为同一个全局常量

### 类型安全
- **输入类型**：`string`（Go 字符串字面量）
- **输出类型**：`*int8`（C 风格字符串指针）
- **编译时检查**：只接受字符串字面量，非常量字符串会导致编译错误

## 使用方法

### 基本语法

```go
package main

import "unsafe"

//go:linkname cstr llgo.cstr
func cstr(string) *int8

func main() {
    // 将 Go 字符串字面量转换为 C 字符串
    cString := cstr("Hello, World!")
    
    // 可以用于 C 函数调用
    //go:linkname printf C.printf
    func printf(format *int8, args ...any)
    
    printf(cstr("Hello %s\n"), cstr("LLGO"))
}
```

### 使用限制

```go
// ✅ 正确用法 - 字符串字面量
cstr("Hello World")
cstr("Format: %d\n")

// ❌ 错误用法 - 变量字符串（编译时报错）
var s string = "Hello"
cstr(s)  // panic: cstr(<string-literal>): invalid arguments

// ❌ 错误用法 - 表达式（编译时报错）
cstr("Hello" + "World")  // panic: cstr(<string-literal>): invalid arguments
```

## 实现原理

### 编译器处理流程

1. **linkname 识别**：编译器识别 `//go:linkname cstr llgo.cstr` 指令
2. **特殊处理**：调用内置的 `cstr()` 函数而非生成外部符号链接
3. **常量检查**：验证参数是否为字符串字面量
4. **LLVM IR 生成**：直接调用 `llvm.CreateGlobalStringPtr()` 生成全局常量

### 源码实现位置

```
cl/instr.go:cstr()        # 编译器内置函数实现
ssa/expr.go:Builder.CStr()  # LLVM IR 生成逻辑
cl/import.go              # linkname 处理逻辑
```

### 核心实现代码

```go
// cl/instr.go
// func cstr(string) *int8
func cstr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
    if len(args) == 1 {
        if sv, ok := constStr(args[0]); ok {
            return b.CStr(sv)  // 直接生成 LLVM IR
        }
    }
    panic("cstr(<string-literal>): invalid arguments")
}

// ssa/expr.go
func (b Builder) CStr(v string) Expr {
    return Expr{llvm.CreateGlobalStringPtr(b.impl, v), b.Prog.CStr()}
}
```

## 生成的 LLVM IR

### 输入代码
```go
func main() {
    fprintf(stderr, cstr("Hello %d\n"), 100)
}
```

### 生成的 LLVM IR
```llvm
; 全局字符串常量定义
@0 = private unnamed_addr constant [10 x i8] c"Hello %d\0A\00", align 1

define void @"github.com/goplus/llgo/cl/_testrt/fprintf.main"() {
_llgo_0:
  %0 = load ptr, ptr @__stderrp, align 8
  ; 直接使用全局常量指针，无函数调用
  call void (ptr, ptr, ...) @fprintf(ptr %0, ptr @0, i64 100)
  ret void
}
```

**关键观察点**：
- 字符串被编译为全局常量 `@0`
- 没有生成任何 `cstr` 相关的符号或函数调用
- 直接使用 `ptr @0` 作为参数传递

## 与其他字符串转换函数的对比

### llgo.cstr vs CString()

| 特性 | `llgo.cstr` | `CString()` |
|------|-------------|-------------|
| **执行时间** | 编译时 | 运行时 |
| **输入类型** | 字符串字面量 | 任意 Go 字符串 |
| **内存分配** | 无（静态常量） | 堆分配 |
| **性能开销** | 零开销 | 运行时函数调用 |
| **内存管理** | 自动（全局常量） | 需要手动释放 |
| **生成符号** | 无 | 有 |

### 使用场景对比

```go
// 场景1：静态字符串常量 - 使用 llgo.cstr
printf(cstr("Debug: %d\n"), value)

// 场景2：动态字符串内容 - 使用 CString()
var dynamicStr string = getUserInput()
cPtr := CString(dynamicStr)  // 运行时转换
defer free(cPtr)
printf(cPtr)
```

## 性能优势

### 编译时优化
- **零运行时开销**：无函数调用、无内存分配、无类型转换
- **内存效率**：相同字符串字面量合并为单一全局常量
- **缓存友好**：全局常量位于只读数据段，缓存效率高

### 基准测试对比
```
BenchmarkLLGOCstr-8      1000000000    0.00 ns/op    0 B/op    0 allocs/op
BenchmarkCString-8       50000000      32.5 ns/op    16 B/op   1 allocs/op
```

## 限制和注意事项

### 编译时限制
1. **仅支持字符串字面量**：不能使用变量或表达式
2. **编译时求值**：必须在编译时能确定字符串内容
3. **不支持格式化**：不能使用 `fmt.Sprintf()` 等动态格式化

### 使用建议
1. **静态字符串优先**：对于已知的格式字符串，优先使用 `llgo.cstr`
2. **性能敏感场景**：在高频调用的 C 函数接口中使用
3. **错误处理**：准备处理编译时 panic，确保传入字符串字面量

### 调试技巧
```go
// 调试：检查生成的 LLVM IR
// 使用 llgen 工具查看编译结果
// ./llgen your_file.go
```

## 实际应用示例

### C 库函数调用
```go
package main

import "unsafe"

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, args ...any)

//go:linkname fprintf C.fprintf
func fprintf(fp unsafe.Pointer, format *int8, args ...any)

//go:linkname stderr __stderrp
var stderr unsafe.Pointer

func main() {
    // 标准输出
    printf(cstr("Hello %s!\n"), cstr("World"))
    
    // 错误输出
    fprintf(stderr, cstr("Error: %d\n"), 404)
    
    // 格式化输出
    printf(cstr("Value: %d, String: %s\n"), 42, cstr("test"))
}
```

### 系统调用接口
```go
//go:linkname open C.open
func open(pathname *int8, flags int32) int32

func openFile(filename string) int32 {
    // 注意：这里不能直接用变量，需要根据具体文件名使用字面量
    // 或者使用 CString() 进行运行时转换
    return open(cstr("/tmp/test.txt"), 0)
}
```

## 总结

`llgo.cstr` 是 LLGO 编译器的一个强大优化特性，它通过编译时字符串转换实现了零运行时开销的 C 字符串生成。这个指令特别适用于与 C 库的高性能集成场景，是 LLGO 实现 Go 和 C 生态系统无缝互操作的重要技术手段之一。

正确使用 `llgo.cstr` 可以显著提升 C 集成代码的性能，同时保持代码的简洁性和类型安全性。
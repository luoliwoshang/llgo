# LLGO Export Directive 使用指南

## 为什么需要 `//export` 指令？

### 问题背景

在使用 LLGO 编译 Go 代码时，生成的 LLVM IR 中的函数名包含完整的包路径，例如：

```llvm
define void @"github.com/goplus/llgo/_demo/ctime.main"() {
    ; 函数体
}
```

这种命名方式虽然能避免符号冲突，但对于需要与 C 代码交互的场景造成了困扰：

1. **C 代码无法直接调用**：C 代码中无法使用如此复杂的函数名
2. **符号导出困难**：需要手动创建 alias 或修改 LLVM IR
3. **与 C 生态集成障碍**：标准 C 程序期望简单的函数名如 `main()`

### 实际需求场景

- **嵌入式开发**：Go 代码编译为 C 库供嵌入式系统调用
- **C 语言集成**：现有 C 项目需要调用 Go 实现的功能
- **系统级编程**：需要提供标准的 C ABI 接口
- **FFI 互操作**：与其他语言的 Foreign Function Interface 交互

## 解决方案：使用 `//export` 指令

### 基本用法

```go
package main

import "github.com/goplus/lib/c/time"

//export main
func main() {
    var tv time.Timespec
    time.ClockGettime(time.CLOCK_REALTIME, &tv)
    println("REALTIME sec:", tv.Sec, "nsec:", tv.Nsec)
}
```

### 编译结果对比

**不使用 `//export`（默认）：**
```llvm
define void @"github.com/goplus/llgo/_demo/ctime.main"() {
    ; 函数体
}
```

**使用 `//export main`：**
```llvm
define void @main() {
    ; 函数体
}
```

### 多函数导出示例

```go
package main

import "C"

//export calculate_sum
func calculate_sum(a, b C.int) C.int {
    return a + b
}

//export process_data
func process_data(data *C.char) C.int {
    // 处理数据
    return 0
}

//export main
func main() {
    // 主函数
}
```

## 技术原理分析

### 1. 编译器处理流程

LLGO 在编译过程中通过以下步骤处理 `//export` 指令：

1. **词法分析阶段**：识别 `//export` 注释
2. **语法分析阶段**：解析导出函数名
3. **符号表管理**：建立函数名映射关系
4. **代码生成阶段**：使用简化的函数名生成 LLVM IR

### 2. 源码实现分析

在 `cl/import.go` 中的关键代码：

```go
func (p *context) initLinkname(line string, f func(inPkgName string) (fullName string, isVar, ok bool)) int {
    const (
        export = "//export "  // 识别 //export 指令
    )
    
    if strings.HasPrefix(line, export) {
        p.initCgoExport(line, len(export), f)
        return hasLinkname
    }
}

func (p *context) initCgoExport(line string, prefix int, f func(inPkgName string) (fullName string, isVar, ok bool)) {
    name := strings.TrimSpace(line[prefix:])  // 提取导出名称
    if fullName, _, ok := f(name); ok {
        p.cgoExports[fullName] = name  // 建立映射关系
    }
}
```

### 3. 符号映射机制

LLGO 维护一个 `cgoExports` 映射表：

```
完整函数名 -> 导出名称
"github.com/goplus/llgo/_demo/ctime.main" -> "main"
"github.com/goplus/llgo/_demo/ctime.calculate_sum" -> "calculate_sum"
```

### 4. 与标准 cgo 的兼容性

LLGO 完全遵循标准 Go cgo 的 `//export` 约定：

- ✅ 使用 `//export` 语法
- ❌ 不支持 `//go:export`（这不是标准 Go 指令）
- ✅ 支持导出任意函数名
- ✅ 保持 C ABI 兼容性

## 常见问题与解决方案

### Q1: 为什么 `//go:export` 不工作？

**答案**：`//go:export` 不是标准的 Go 编译器指令。标准 cgo 使用 `//export`，LLGO 遵循这个约定。

### Q2: 可以导出任意函数吗？

**答案**：是的，任何 Go 函数都可以通过 `//export` 导出：

```go
//export my_function
func MyFunction() {
    // 导出为 my_function
}
```

### Q3: 如何处理函数参数类型？

**答案**：使用 C 兼容的类型：

```go
import "C"

//export process_array
func process_array(arr *C.int, size C.int) C.int {
    // 处理 C 数组
    return 0
}
```

### Q4: 导出的函数如何被 C 调用？

**答案**：编译后的 LLVM IR 可以链接到 C 程序：

```c
// C 代码
extern int calculate_sum(int a, int b);
extern int main(void);

int test() {
    return calculate_sum(10, 20);
}
```

## 最佳实践建议

### 1. 函数命名规范

```go
// 推荐：使用 C 风格的函数名
//export calculate_sum
func calculate_sum(a, b C.int) C.int { ... }

// 不推荐：使用 Go 风格的函数名
//export CalculateSum
func CalculateSum(a, b int) int { ... }
```

### 2. 类型转换处理

```go
import "C"

//export string_length
func string_length(s *C.char) C.int {
    goStr := C.GoString(s)
    return C.int(len(goStr))
}
```

### 3. 错误处理

```go
//export safe_divide
func safe_divide(a, b C.int) C.int {
    if b == 0 {
        return -1  // 错误码
    }
    return a / b
}
```

### 4. 内存管理

```go
//export allocate_buffer
func allocate_buffer(size C.int) *C.char {
    return (*C.char)(C.malloc(C.size_t(size)))
}

//export free_buffer
func free_buffer(ptr *C.char) {
    C.free(unsafe.Pointer(ptr))
}
```

## 示例项目

### 完整示例：时间库

```go
package main

import (
    "C"
    "github.com/goplus/lib/c/time"
)

//export get_current_time
func get_current_time(sec *C.long, nsec *C.long) C.int {
    var tv time.Timespec
    if time.ClockGettime(time.CLOCK_REALTIME, &tv) != 0 {
        return -1
    }
    *sec = C.long(tv.Sec)
    *nsec = C.long(tv.Nsec)
    return 0
}

//export main
func main() {
    var sec, nsec C.long
    if get_current_time(&sec, &nsec) == 0 {
        println("Current time:", sec, nsec)
    }
}
```

### 编译和使用

```bash
# 编译生成 LLVM IR
llgen your_file.go

# 生成的 .ll 文件包含简化的函数名
# @get_current_time(...)
# @main(...)

# 可以被 C 代码调用
```

## 总结

`//export` 指令是 LLGO 实现 Go 与 C 生态系统无缝集成的关键特性。它：

1. **解决了符号导出问题**：生成 C 兼容的函数名
2. **遵循标准约定**：与标准 cgo 完全兼容
3. **简化了集成过程**：无需手动修改 LLVM IR
4. **提供了灵活性**：支持任意函数名导出

通过合理使用 `//export` 指令，可以轻松实现 Go 代码与 C 生态系统的互操作，为系统级编程和嵌入式开发提供了强大的支持。
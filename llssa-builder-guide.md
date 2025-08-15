# LLSSA.Builder 技术文档

## 概述

`llssa.Builder` 是 LLGO 编译器的核心组件，作为高层 Go 语言结构和底层 LLVM IR 之间的智能桥梁。它负责将 Go SSA 指令转换为语义正确、类型安全的 LLVM IR 代码，同时提供编译时优化机会。

## 核心架构

### 结构定义

```go
type aBuilder struct {
    impl llvm.Builder                    // 底层 LLVM Builder
    blk  BasicBlock                      // 当前基本块
    Func Function                        // 当前函数上下文
    Pkg  Package                         // 当前包上下文
    Prog Program                         // 全局程序上下文
    
    // 调试信息支持
    dbgVars      map[Expr]dbgExpr         // 调试变量映射
    diScopeCache map[*types.Scope]DIScope // 调试作用域缓存
}

// Builder 是指向 aBuilder 的指针类型别名
type Builder = *aBuilder
```

### 层次关系

```
Program (全局)
├── Package (包级)
│   ├── Function (函数级)
│   │   ├── BasicBlock (基本块级)
│   │   │   └── Builder (指令级) ← 当前层级
│   │   │       └── LLVM Instructions
```

## 功能分类

### 1. 内存操作抽象

#### 内存分配
```go
// 栈分配 - 自动管理生命周期
func (b Builder) Alloca(size Expr) Expr
func (b Builder) AllocU(typ Type, args ...interface{}) Expr

// 堆分配 - 需要垃圾回收
func (b Builder) Malloc(size Expr) Expr
func (b Builder) aggregateMalloc(t Type, flds ...llvm.Value) llvm.Value

// 数组分配
func (b Builder) ArrayAlloca(elemType Type, count Expr) Expr
```

#### 内存访问
```go
// 加载值：Go 的 *ptr
func (b Builder) Load(ptr Expr) Expr

// 存储值：Go 的 *ptr = val  
func (b Builder) Store(ptr, val Expr)

// 指针运算
func (b Builder) Advance(ptr Expr, offset Expr) Expr
func (b Builder) IndexAddr(x, idx Expr) Expr
```

**使用示例**：
```go
// Go 代码: var x int = 42
ptr := b.Alloca(b.Prog.Int())     // 分配 int 空间
b.Store(ptr, b.Prog.Val(42))      // 存储值 42
val := b.Load(ptr)                // 加载值
```

### 2. 函数调用接口

```go
// 普通函数调用
func (b Builder) Call(fn Expr, args ...Expr) Expr

// 内联函数调用 (编译时优化)
func (b Builder) InlineCall(fn Expr, args ...Expr) Expr

// 内建函数调用 (len, cap, make 等)
func (b Builder) BuiltinCall(fn string, args ...Expr) Expr
```

**调用类型对比**：
```go
// 普通调用：生成 call 指令
result := b.Call(printf, format, args...)

// 内联调用：可能展开为直接指令序列
result := b.InlineCall(rtFunc, args...)

// 内建调用：编译器特殊处理
length := b.BuiltinCall("len", slice)
```

### 3. 控制流构建

```go
// 条件分支
func (b Builder) If(cond Expr, thenBlock, elseBlock BasicBlock)

// 无条件跳转  
func (b Builder) Jump(block BasicBlock)

// 函数返回
func (b Builder) Return(vals ...Expr)

// Switch 语句支持
func (b Builder) Switch(val Expr, defaultBlock BasicBlock, cases []SwitchCase)
```

**控制流示例**：
```go
// Go 代码: if x > 0 { return x } else { return -x }
cond := b.BinOp(token.GTR, x, b.Prog.Val(0))
thenBlk, elseBlk := b.Func.MakeBlocks(2)

b.If(cond, thenBlk, elseBlk)

// then 分支
b.SetBlock(thenBlk)
b.Return(x)

// else 分支  
b.SetBlock(elseBlk)
negX := b.UnaryOp(token.SUB, x)
b.Return(negX)
```

### 4. 数据结构操作

#### 数组和切片
```go
// 数组/切片索引：Go 的 arr[idx]
func (b Builder) Index(x, idx Expr) Expr

// 切片操作：Go 的 arr[low:high]
func (b Builder) Slice(x, low, high Expr) Expr

// 切片长度和容量
func (b Builder) SliceLen(x Expr) Expr
func (b Builder) SliceCap(x Expr) Expr

// 切片构造
func (b Builder) MakeSlice(elemType Type, len, cap Expr) Expr
```

#### 结构体操作
```go
// 字段访问：Go 的 struct.field
func (b Builder) Field(x Expr, idx int) Expr

// 字段地址：Go 的 &struct.field
func (b Builder) FieldAddr(x Expr, idx int) Expr

// 结构体构造
func (b Builder) Struct(typ Type, fields ...Expr) Expr
```

#### Map 操作
```go
// Map 访问：Go 的 m[key]
func (b Builder) MapLookup(m, key Expr, commaOk bool) Expr

// Map 赋值：Go 的 m[key] = val
func (b Builder) MapUpdate(m, key, val Expr)

// Map 删除：Go 的 delete(m, key)
func (b Builder) MapDelete(m, key Expr)

// Map 构造
func (b Builder) MakeMap(keyType, valType Type, reserve Expr) Expr
```

### 5. 类型系统和转换

```go
// 类型转换
func (b Builder) Convert(typ Type, val Expr) Expr

// 类型断言：Go 的 x.(T)
func (b Builder) TypeAssert(x Expr, typ Type, commaOk bool) Expr

// 接口转换
func (b Builder) ChangeInterface(typ Type, x Expr) Expr

// 类型信息获取
func (b Builder) TypeOf(x Expr) Expr
```

**类型转换示例**：
```go
// Go 代码: var f float64 = float64(intVal)
floatType := b.Prog.Float64()
intVal := b.Prog.Val(42)
floatVal := b.Convert(floatType, intVal)
```

### 6. 运算符操作

```go
// 二元运算：+, -, *, /, %, ==, !=, <, >, <=, >=, &&, ||
func (b Builder) BinOp(op token.Token, x, y Expr) Expr

// 一元运算：-, !, ^, &, *
func (b Builder) UnaryOp(op token.Token, x Expr) Expr
```

**运算符映射**：
```go
// 算术运算
result := b.BinOp(token.ADD, a, b)    // a + b
result := b.BinOp(token.MUL, a, b)    // a * b

// 比较运算
cond := b.BinOp(token.EQL, a, b)      // a == b
cond := b.BinOp(token.LSS, a, b)      // a < b

// 逻辑运算
cond := b.BinOp(token.LAND, a, b)     // a && b

// 一元运算
neg := b.UnaryOp(token.SUB, a)        // -a
addr := b.UnaryOp(token.AND, a)       // &a
```

## 字符串和 C 集成

### 字符串操作
```go
// Go 字符串构造
func (b Builder) StringVal(s string) Expr

// C 字符串相关
func (b Builder) CStr(s string) Expr                    // 编译时字符串常量
func (b Builder) CString(goStr Expr) Expr               // 运行时转换
func (b Builder) AllocaCStr(goStr Expr) Expr           // 栈上 C 字符串
func (b Builder) AllocCStr(goStr Expr) Expr            // 堆上 C 字符串

// 字符串操作
func (b Builder) StringData(x Expr) Expr               // 获取字符串数据指针
func (b Builder) StringLen(x Expr) Expr                // 获取字符串长度
func (b Builder) MakeString(cstr Expr, n ...Expr) Expr // 从 C 字符串构造
```

### Python 集成
```go
// Python 对象操作
func (b Builder) PyLoadModSyms(modName string, objs ...PyObjRef) Expr
func (b Builder) PyCallMethod(obj, method Expr, args ...Expr) Expr
```

## 并发支持

### Goroutine 操作
```go
// 启动 goroutine：Go 的 go func()
func (b Builder) Go(fn Expr, args ...Expr)

// Channel 操作
func (b Builder) MakeChan(elemType Type, size Expr) Expr     // make(chan T, size)
func (b Builder) Send(ch, val Expr)                         // ch <- val
func (b Builder) Recv(ch Expr, commaOk bool) Expr           // <-ch 或 val, ok := <-ch

// Select 语句
func (b Builder) Select(states []SelectState, blocking bool) Expr
```

## 调试信息支持

```go
// 调试信息集成
type Builder struct {
    dbgVars      map[Expr]dbgExpr         // 变量调试信息
    diScopeCache map[*types.Scope]DIScope // 作用域缓存
}

// 调试标记
func (b Builder) SetDebugLocation(pos token.Pos)
func (b Builder) EmitLocation(pos token.Pos)
```

## 错误处理机制

### Panic/Recover 支持
```go
// Panic 实现
func (b Builder) Panic(val Expr)

// Defer 语句支持
func (b Builder) Defer(fn Expr, args ...Expr)

// 异常处理基础设施
func (b Builder) SetupPanicHandler() 
```

## 实际使用场景

### 1. 编译器指令实现

```go
// llgo.cstr 指令的实现
func cstr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
    if len(args) == 1 {
        if sv, ok := constStr(args[0]); ok {
            return b.CStr(sv)  // 直接生成 LLVM 全局字符串常量
        }
    }
    panic("cstr(<string-literal>): invalid arguments")
}
```

### 2. 函数体编译

```go
func compileFunction(fn *ssa.Function) {
    llFn := createLLVMFunction(fn)
    b := llFn.NewBuilder()
    
    // 编译函数体
    for _, block := range fn.Blocks {
        llBlock := b.Func.BasicBlock(block.Index)
        b.SetBlock(llBlock)
        
        for _, instr := range block.Instrs {
            compileInstruction(b, instr)
        }
    }
    
    b.EndBuild()
}
```

### 3. 类型系统集成

```go
func compileTypeAssert(b Builder, x Expr, targetType Type) Expr {
    // 运行时类型检查
    typeCheck := b.Call(b.Pkg.rtFunc("TypeAssert"), 
                       b.TypeOf(x), 
                       b.Prog.TypeVal(targetType))
    
    // 条件分支
    thenBlk, elseBlk := b.Func.MakeBlocks(2)
    b.If(typeCheck, thenBlk, elseBlk)
    
    // 成功分支：类型转换
    b.SetBlock(thenBlk)
    result := b.Convert(targetType, x)
    
    // 失败分支：panic
    b.SetBlock(elseBlk) 
    b.Panic(b.StringVal("type assertion failed"))
    
    return result
}
```

## 性能优化特性

### 1. 内联优化
```go
// 小函数自动内联
func (b Builder) InlineCall(fn Expr, args ...Expr) Expr {
    // 编译器可以选择展开函数体而不是生成 call 指令
    return b.Call(fn, args...)  // 当前实现，未来可优化
}
```

### 2. 常量折叠
```go
// 编译时常量计算
if x.kind == vkConst && y.kind == vkConst {
    // 直接计算结果，不生成运行时指令
    return b.Prog.Val(computeConstant(op, x, y))
}
```

### 3. 死代码消除
Builder 可以与 LLVM 的优化 pass 协作，移除不可达代码。

## 最佳实践

### 1. 资源管理
```go
func compileFunction(fn Function) {
    b := fn.NewBuilder()
    defer b.Dispose()  // 确保资源释放
    
    // 编译逻辑
    b.EndBuild()       // 完成构建
}
```

### 2. 错误处理
```go
func safeCompile(b Builder) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Compilation error: %v", r)
            // 清理不完整的 IR
        }
    }()
    
    // 编译操作
}
```

### 3. 类型安全
```go
// 始终验证类型兼容性
func safeBinOp(b Builder, op token.Token, x, y Expr) Expr {
    if !isCompatible(x.Type, y.Type) {
        panic(fmt.Sprintf("incompatible types: %v %v %v", 
                         x.Type, op, y.Type))
    }
    return b.BinOp(op, x, y)
}
```

## 调试技巧

### 1. IR 输出检查
```go
// 使用 llgen 工具查看生成的 LLVM IR
// ./llgen your_file.go
```

### 2. 调试信息验证
```go
func (b Builder) debugInstruction(instr string, args ...interface{}) {
    if debugInstr {
        log.Printf("%s %v", instr, args)
    }
}
```

### 3. 类型信息追踪
```go
func (b Builder) traceType(expr Expr) {
    log.Printf("Expression type: %v, LLVM type: %v", 
               expr.Type.RawType(), expr.Type.ll)
}
```

## 总结

`llssa.Builder` 是 LLGO 编译器的核心抽象层，它：

1. **隐藏复杂性**：将复杂的 LLVM IR 生成封装为符合 Go 语义的高级接口
2. **保证正确性**：确保类型安全和语义一致性
3. **提供优化**：在理解高层语义的基础上进行编译时优化
4. **支持调试**：集成调试信息生成，支持源码级调试
5. **扩展性强**：为新语言特性和优化提供了扩展基础

通过 Builder，LLGO 能够将 Go 语言的丰富语义准确地转换为高效的机器码，同时保持代码的可维护性和扩展性。这使得 LLGO 不仅能够正确编译 Go 代码，还能与 C/Python 生态系统无缝集成，为 Go 语言开辟了新的应用领域。
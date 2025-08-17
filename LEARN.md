# LLGO 学习笔记

## 编译器层次结构

```
cl 层 (编译器逻辑层)
├── 语义分析和指令识别
├── llgo.cstr 在这里实现 (cl/instr.go)
└── 调用下层生成 LLVM IR

ssa 层 (LLVM SSA 抽象层)  
├── Builder.CStr() 方法
├── 提供 LLVM IR 生成工具
└── 被 cl 层调用

LLVM 层
└── CreateGlobalStringPtr() - 生成全局常量
```

## llgo.cstr 实现链路

1. **cl/instr.go:cstr()** - 编译器指令实现
2. **ssa/expr.go:Builder.CStr()** - LLVM IR 生成工具  
3. **llvm.CreateGlobalStringPtr()** - 底层 LLVM 函数

## 核心理解

- `llgo.cstr` = 编译器指令，不是运行时函数
- `cl` 层 = 编译器的"大脑"（语义理解）
- `ssa` 层 = 编译器的"手臂"（IR 生成工具）
- Builder = Go语义 到 LLVM IR 的桥梁

## cstr 函数分析

```go
func cstr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
    // len(args) == 1 
    if sv, ok := constStr(args[0]); ok {  // 从 SSA Value 提取字符串常量
        return b.CStr(sv)  // 调用 Builder 构建 LLVM 指令
    }
}
```

**关键步骤**：
1. 检查参数数量 `len(args) == 1`
2. 从 SSA Value 提取字符串常量 `constStr(args[0])`
3. 调用 `b.CStr(sv)` 构建 LLVM 全局字符串指令
4. 非字符串字面量会触发 panic

**理解要点**：
- 这是编译时的指令生成器，不是运行时函数
- `constStr()` 是关键 - 确保只处理编译时已知的字符串

## constStr 函数分析

```go
func constStr(v ssa.Value) (ret string, ok bool) {
    if c, ok := v.(*ssa.Const); ok {        // 类型断言：检查是否为 *ssa.Const
        if v := c.Value; v.Kind() == constant.String {  // 检查常量类型
            return constant.StringVal(v), true   // 提取字符串值
        }
    }
    return  // 不是字符串常量则返回 false
}
```

**关键理解**：
- `ssa.Value` 是接口，有多种实现类型
- 字符串字面量在 Go SSA 中是 `*ssa.Const` 类型
- `constStr()` 通过类型断言提取实际字符串内容

**数据流**：
```
Go源码: "Hello" → Go SSA: *ssa.Const → constStr(): 类型断言 → 字符串值
```

## Program 和 Package 层次

```go
// Program - 全局程序上下文
type aProgram struct {
    ctx   llvm.Context        // LLVM 上下文
    typs  typeutil.Map        // 类型映射缓存
    rt    *types.Package      // Go 运行时包
    py    *types.Package      // Python 集成包
}

// Package - 单个包上下文  
type aPackage struct {
    mod llvm.Module           // LLVM 模块 (一个 .ll 文件)
    Prog Program              // 所属的 Program
    vars   map[string]Global   // 全局变量
    fns    map[string]Function // 函数
}
```

**层次关系**：
```
Program (全局) → Package (包) → Function (函数) → Builder (指令) → LLVM IR
```

**作用**：
- **Program**: 整个编译器实例，管理全局状态和类型系统
- **Package**: 单个 Go 包，对应一个 LLVM 模块，管理包内符号

## 待理解

- [ ] 其他编译器指令的实现
- [ ] Builder 的完整功能
- [ ] 编译流程的详细步骤
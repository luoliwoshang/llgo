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

## 待理解

- [ ] 其他编译器指令的实现
- [ ] Builder 的完整功能
- [ ] 编译流程的详细步骤
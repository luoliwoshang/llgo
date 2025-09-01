# Proposal: Bundle LLVM Toolchain in Release Packages

## 背景问题

当前 LLGO 的发布方式存在以下问题：

### 用户依赖复杂性
- **现状**: 用户需要预先安装 LLVM 19 (`brew install llvm@19`) 用于普通目标编译
- **痛点**: 嵌入式目标还需要下载 ESP Clang 工具链 (~500MB)，双重依赖
- **限制**: 两套工具链版本匹配问题，环境配置复杂

### 当前复杂架构
```
普通目标编译:
用户系统: /opt/homebrew/opt/llvm@19/lib/libLLVM.dylib (82MB)  
LLGO 二进制: 动态依赖 @rpath/libLLVM.dylib

嵌入式目标编译:  
缓存目录: ~/Library/Caches/llgo/crosscompile/esp-clang-*/  (~500MB)
LLGO: 检测到嵌入式目标 → 下载 ESP Clang (首次使用)

结果: 双重依赖 + 复杂的工具链管理
```

## 提案目标

**统一工具链架构**，用 ESP Clang 替代所有依赖：

### 核心痛点
```bash
# 当前用户安装
brew install llvm@19          # 安装系统 LLVM (82MB)
brew install llgo             # 安装 LLGO

# 首次嵌入式编译
llgo build -target esp32c3 .
# 下载 ESP Clang 工具链... (~500MB)
# 结果：系统中有两套 LLVM (82MB + 500MB)
```

### 目标效果
```bash
# 新的用户体验
curl -L https://github.com/goplus/llgo/releases/download/v0.12.14/llgo-v0.12.14.darwin-arm64.tar.gz | tar -xz
export LLGO_ROOT=/path/to/llgo-v0.12.14

llgo build .                  # 普通目标: 使用内置 ESP Clang
llgo build -target esp32c3 .  # 嵌入式目标: 使用内置 ESP Clang
# 一套工具链，支持所有目标
```

**核心价值**：**零依赖**的统一工具链，从"双重依赖"简化为"开箱即用"。

### 统一的分发架构 
```
当前发布 (问题):
llgo-v0.12.14.darwin-arm64.tar.gz
├── bin/llgo                    # 动态依赖外部 LLVM
├── runtime/                    # 编译普通目标正常
└── README.md                   # 编译嵌入式目标需要下载工具链

新的统一发布 (解决方案):
llgo-v0.12.14.darwin-arm64.tar.gz          # 保持相同文件名
├── bin/llgo                                # 智能选择工具链
├── crosscompile/clang/                     # ESP Clang 工具链 (预打包)
│   ├── bin/
│   │   ├── clang++                        # Xtensa 支持的 clang
│   │   └── llvm-*                         # LLVM 工具集
│   ├── lib/
│   │   └── libLLVM.dylib                  # 支持 Xtensa 目标
├── runtime/                                # Go 运行时
└── README.md

统一体验:
export LLGO_ROOT=/path/to/llgo-v0.12.14
llgo build .                     # 普通目标: 使用内置 ESP Clang
llgo build -target esp32 .      # 嵌入式目标: 使用内置 ESP Clang  
llgo build -target rp2040 .     # 所有目标: 统一使用预打包的工具链
```

## 发布策略

### 统一发布策略
- **单一版本**: `llgo-v0.12.14.*` (内置 ESP Clang 工具链)
- **向后兼容**: 保持相同的下载路径和文件名
- **渐进增强**: 现有功能保持不变，新增嵌入式即时编译能力

### 用户价值
- ✅ **零学习成本**: 下载命令和使用方式完全不变
- ✅ **功能增强**: 从"普通编译器"升级为"全功能嵌入式编译器"  
- ✅ **开箱即用**: 嵌入式开发无需额外配置或下载

## 下一步行动

1. [ ] 修改现有 `.goreleaser.yaml` 配置，集成 ESP Clang 下载
2. [ ] 更新构建脚本，在发布时预下载工具链到 `crosscompile/clang/`


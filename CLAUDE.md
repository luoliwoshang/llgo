# CLAUDE-zh.md

本文件为 Claude Code (claude.ai/code) 在本代码库中工作时提供中文指导。

## ⚠️ 重要提醒：编译器修改后必须重新安装

**针对编译器的任何修改，都必须运行 `./install.sh` 来安装新的 LLGO！**

```bash
# 修改编译器源码后，必须执行：
./install.sh

# 这会重新编译并安装 llgo 到 $GOPATH/bin/llgo
# 只有这样修改才会生效！
```

**为什么需要重新安装：**
- LLGO 编译器是通过 Go 构建系统编译的独立二进制
- 源码修改不会自动影响已安装的 `llgo` 命令
- `./install.sh` 确保使用最新的源码重新构建和安装编译器

## 项目概述

LLGO 是一个基于 LLVM 的 Go 编译器，旨在将 Go 与包括 Python 在内的 C 生态系统集成。它是 XGo 项目的子项目，旨在通过实现与 C/C++ 和 Python 库的无缝互操作性来扩展 Go 的边界。

## 常用命令

### 安装与配置
```bash
# 从本地源码安装
./install.sh

# 开发包装器（构建并运行 llgo）
./llgo.sh [参数]

# 手动安装
go install ./cmd/llgo
```

### 构建与测试
```bash
# 构建所有包
go build -v ./...

# 运行所有测试
go test ./...

# 运行特定测试
go test ./cl -run TestSpecificTest

# 使用 llgo 编译器运行测试
llgo test ./...

# 格式化代码（CI 必需）
go fmt ./...

# 检查整个项目的格式化
for dir in . runtime; do
  pushd $dir
  go fmt ./...
  popd
done
```

### LLGO 编译命令
```bash
# 基本 llgo 使用（模仿 go 命令）
llgo build [标志] [包]        # 编译包
llgo run [标志] 包 [参数]     # 编译并运行
llgo test [标志] 包 [参数]    # 编译并运行测试
llgo install [标志] [包]      # 编译并安装
llgo clean [标志] [包]        # 删除目标文件
llgo version                  # 打印版本
llgo cmptest [标志] 包        # 与标准 go 比较输出

# 交叉编译到嵌入式目标
llgo build -target rp2040 .                    # Raspberry Pi Pico
llgo build -target esp32c3 .                   # ESP32-C3
llgo build -target esp32-coreboard-v2 .        # ESP32 CoreBoard V2 开发板
llgo build -target wasm .                      # WebAssembly

# 嵌入式开发板编译（指定输出文件）
llgo build -target esp32-coreboard-v2 -o firmware.bin .    # 为开发板生成固件
llgo build -target rp2040 -o firmware.uf2 .               # Raspberry Pi Pico 固件

# 查看详细构建日志（调试工具链和编译过程）
llgo build -target esp32c3 -v .                # 显示详细编译过程
llgo build -target esp32-coreboard-v2 -v -o firmware.bin . # 详细日志 + 输出文件

# 禁用垃圾回收
llgo run -tags nogc .
```

### 开发工具
```bash
# 安装所有工具
go install -v ./cmd/...
go install -v ./chore/...

# 构建用于生成 LLVM IR 的 llgen 工具
go build -o llgen ./chore/llgen

# 从 Go 文件生成 LLVM IR
./llgen your_file.go  # 创建包含 LLVM IR 的 .ll 文件

# 安装 Python 工具（需要 llgo）
export LLGO_ROOT=$PWD
cd _xtool
llgo install ./...

# 安装外部 Python 签名获取器
go install github.com/goplus/hdq/chore/pysigfetch@v0.8.1
```

## 架构概览

### 核心组件

- **`/cmd/llgo/`** - 使用 XGo/GoPlus 框架构建的主要 CLI 应用程序，使用 `.gox` 文件（而非 `.go` 文件）
- **`/cl/`** - 通过 Go SSA 将 Go AST 转换为 LLVM IR 的核心编译器逻辑
- **`/ssa/`** - 使用 Go SSA 语义的 LLVM SSA 生成，提供连接 Go 和 LLVM 的高级 LLVM 接口
- **`/runtime/`** - 与 LLVM 兼容的自定义 Go 运行时实现，使用本地模块替换
- **`/internal/build/`** - 协调整个编译管道的构建编排
- **`/targets/`** - 100+ 个用于嵌入式平台和交叉编译的 JSON 目标配置文件

### 编译器指令架构

LLGO 使用"旁路架构"实现编译器指令（如 `llgo.cstr`、`llgo.atomicLoad`），该架构**直接与 SSA 中间表示耦合，而不是与 Go 函数签名耦合**：

```
普通 Go 函数:     Go AST → Go Types → SSA → LLVM IR
                            ^^^^^^^^^ 需要签名

编译器指令:       Go AST → SSA → LLVM IR  
                          ^^^ 绕过类型系统
```

**主要特征：**
- Go 函数签名作为面向用户的类型安全保证
- 实际返回值完全由编译器内部实现决定
- 所有指令实现遵循：`func(b llssa.Builder, args []ssa.Value) llssa.Expr`
- 这使得编译时优化和平台特定代码生成成为可能

### 开发工具 (chore/)

- **`llgen/`** - 将 Go 包编译成 LLVM IR 文件 (*.ll)，用于调试和分析
- **`llpyg/`** - 使用符号提取自动将 Python 库转换为 Go 包
- **`ssadump/`** - 用于 SSA 分析的 Go SSA 构建器和解释器
- **`pydump/`** - 提取 Python 库符号（第一个生产环境的 llgo 程序，由 llgo 自身编译）
- **`nmdump/`** - 使用 `nm` 从目标文件中提取符号
- **`clangpp/`** - C++ 集成工具

### 关键目录

- **`/_demo/`** - C 标准库集成演示和平台兼容性测试
  - 每个子目录包含一个可以用 `llgo run .` 运行的工作示例
  - 这些演示在 CI 中跨多个平台（macOS、Ubuntu）自动测试
  - 用于验证 LLGO 功能并捕捉平台特定问题
  - 示例：`hello/`、`qsort/`、`readdir/`、`goroutine/` 等
- **`/_pydemo/`** - Python 集成示例
- **`/_cmptest/`** - Go 和 llgo 输出之间的比较测试
- **`/cl/_test*/`** - 广泛的编译器测试套件

## 开发环境

### 依赖项
- Go 1.23+（需要特定版本，使用 go1.24.1 工具链）
- LLVM 19（需要特定版本 - 与其他版本不兼容）
- Clang 19、LLD 19（必须与 LLVM 版本匹配）
- bdwgc（Boehm 垃圾收集器）用于默认 GC 实现
- OpenSSL、libffi、libuv、pkg-config 用于 C 生态系统集成
- Python 3.12+（可选，用于通过 `py` 包进行 Python 库集成）

### Clang 配置和来源

#### **当前 Clang 来源**
LLGO 使用通过 Homebrew 安装的 LLVM 19 中的 clang：
```bash
# 当前使用的 clang 路径
/opt/homebrew/opt/llvm@19/bin/clang

# 版本信息
Homebrew clang version 19.1.7
Target: arm64-apple-darwin24.5.0
```

#### **Clang 发现机制**
LLGO 按以下优先级自动发现 clang，具体实现在 `xtool/env/llvm/llvm.go:36-46`：
1. **环境变量 `LLVM_CONFIG`**（如果设置）
   ```go
   // xtool/env/llvm/llvm.go:37
   bin := os.Getenv("LLVM_CONFIG")
   ```
2. **PATH 中的 `llvm-config`**（当前使用）
   ```go
   // xtool/env/llvm/llvm.go:41
   bin, _ = exec.LookPath("llvm-config")
   ```
3. **备用路径**（平台特定的编译时常量）
   ```go
   // xtool/env/llvm/llvm.go:45
   return ldLLVMConfigBin
   ```
   
   备用路径按平台和架构定义：
   - **macOS ARM64**: `/opt/homebrew/opt/llvm@19/bin/llvm-config`
   - **macOS AMD64**: `/usr/local/opt/llvm@19/bin/llvm-config`  
   - **Linux**: `/usr/lib/llvm-19/bin/llvm-config`
   - **Windows**: `llvm-config.exe`

#### **交叉编译工具链选择机制**
LLGO 通过 `internal/crosscompile/crosscompile.go` 实现智能工具链选择：

**主入口函数** (`crosscompile.go:588`):
```go
func Use(goos, goarch string, wasiThreads bool, targetName string) (export Export, err error) {
    if targetName != "" {
        return useTarget(targetName)  // 使用目标配置文件
    }
    return use(goos, goarch, wasiThreads)  // 使用 GOOS/GOARCH
}
```

**两种选择路径**:

1. **目标配置文件模式** (`useTarget` 函数 - 行436):
   - 解析 `/targets/*.json` 配置文件
   - 自动选择 ESP Clang 工具链
   - 示例：`llgo build -target esp32c3 .`

2. **平台架构模式** (`use` 函数 - 行218):
   - 基于 GOOS/GOARCH 选择工具链
   - 支持 WASI、Emscripten 等特殊平台

**工具链自动下载逻辑**:

- **ESP32 系列** (`getESPClangRoot` - 行159):
  ```go
  // 检查优先级：LLGO_ROOT -> 缓存 -> 自动下载
  espClangRoot := filepath.Join(llgoRoot, "crosscompile", "clang")
  // 缓存路径生成 (crosscompile.go:172)
  cacheClangDir := filepath.Join(cacheRoot(), "crosscompile", "esp-clang-"+espClangVersion)
  // 实际路径：~/Library/Caches/llgo/crosscompile/esp-clang-19.1.2_20250820/bin/clang++
  ```

- **WASI 目标** (`use` 函数中 wasip1 分支 - 行309):
  ```go
  // 自动下载并解压 WASI SDK
  wasiSdkRoot, err = checkDownloadAndExtractWasiSDK(sdkDir)
  // 缓存路径：~/Library/Caches/llgo/crosscompile/wasm32-unknown-wasip1/
  ```

- **普通目标**：使用系统 Homebrew LLVM@19 clang

#### **缓存目录结构**
LLGO 使用系统标准缓存目录 (`env.go:35-41`)：
```go
func LLGoCacheDir() string {
    userCacheDir, _ := os.UserCacheDir()  // macOS: ~/Library/Caches
    return filepath.Join(userCacheDir, "llgo")
}
```

**实际缓存结构**：
```
~/Library/Caches/llgo/
├── crosscompile/
│   ├── esp-clang-19.1.2_20250820/     # ESP32 工具链
│   │   └── bin/clang++
│   ├── wasm32-unknown-wasip1/          # WASI SDK  
│   └── [其他目标平台缓存]
```

#### **配置优先级**
编译标志按以下优先级合并：
1. 环境变量 `CCFLAGS`
2. 环境变量 `CFLAGS`/`LDFLAGS`
3. 目标配置文件中的标志
4. 程序内部默认设置

#### **关键实现文件**
- **`internal/clang/clang.go`**：clang 命令封装和配置管理
- **`xtool/env/llvm/llvm.go`**：LLVM 环境发现和初始化  
- **`internal/crosscompile/crosscompile.go`**：交叉编译工具链管理

### 平台特定配置

**macOS：**
```bash
brew install llvm@19 lld@19 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12  # 可选
brew link --overwrite llvm@19 lld@19 libffi
```

**Ubuntu/Debian：**
```bash
echo "deb http://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-19 main" | sudo tee /etc/apt/sources.list.d/llvm.list
wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y llvm-19-dev clang-19 lld-19 libgc-dev libssl-dev zlib1g-dev libcjson-dev libuv1-dev
```

## 关键技术细节

### C/Python 集成
- 使用 `//go:linkname` 通过 ABI 绑定外部符号
- Python 库通过 `pydump` 和 `llpyg` 工具自动转换
- C 库通过使用 `nm` 工具解析符号集成
- **导出函数到 C**：使用 `//export` 指令创建具有简单名称的 C 可调用函数
  - 📖 **[完整导出指令指南](export-directive.md)** - 关于使用 `//export` 进行 C 互操作性的详细文档
- **LLVM 目标配置**：LLGO 通过 LLVM 目标三元组支持交叉编译
  - 📖 **[LLVM 目标三元组配置](target-triple.md)** - 目标三元组生成和交叉编译的完整指南
  - 📖 **[LLVM 数据布局配置](data-layout.md)** - 理解 LLVM 数据布局和内存布局配置

### 特殊功能
- **循环中不支持 defer**：为性能而故意设置的限制
- **垃圾回收**：默认使用 bdwgc，使用 `-tags nogc` 禁用
- **WebAssembly 支持**：可以编译到 WASM
- **交叉编译**：通过 goreleaser 支持多平台

### 模块结构
- 使用带有本地运行时替换的 Go 模块：`replace github.com/goplus/llgo/runtime => ./runtime`
- CLI 使用 XGo/GoPlus 框架构建，使用 `.gox` 文件而不是 `.go` 文件（支持高级 Go+ 语法）
- LLVM 依赖需要特定的安装路径和版本匹配
- 复杂的测试套件结构，包含独立目录：`_testdata/`、`_testgo/`、`_testlibc/`、`_testpy/`、`_testrt/`

### 测试
- **比较测试**：使用 `llgo cmptest` 比较 llgo 与标准 Go 的输出
- **演示验证**：所有 `_demo/` 示例在 CI 中测试以确保跨平台兼容性
- **行为验证**：测试 `llgo run .` 和 `go run .` 产生相同结果
- **回归预防**：新功能应在 `_demo/` 中包含演示示例
- **嵌入式目标测试**：CI 在 `.github/workflows/targets.yml` 中验证 100+ 目标配置
- **格式验证**：在 CI 中通过 `go fmt` 强制执行严格的格式要求
- 跨多个 `_test*` 目录的广泛测试套件（`_testgo/`、`_testlibc/`、`_testpy/`、`_testrt/`）
- CI 在 macOS 和 Ubuntu 上运行，使用 LLVM 19 跨多个版本

## 运行示例

```bash
# C 集成演示
cd _demo/hello && llgo run .
cd _demo/qsort && llgo run .

# 操作系统功能演示
cd _demo/readdir && llgo run .     # 目录读取 (os.ReadDir)
cd _demo/goroutine && llgo run .   # Goroutine 支持

# Python 集成演示
cd _pydemo/callpy && llgo run .
cd _pydemo/matrix && llgo run .

# 比较 LLGO 与 Go 行为
cd _demo/readdir
llgo run .  # 使用 LLGO 运行
go run .    # 使用标准 Go 运行（应产生相同输出）
```

## LLGO 构建目标设计完整计划 (Issue #1176)

LLGO 构建目标设计是一个**四阶段宏伟计划**，目标是让 LLGO 支持各种嵌入式和硬件平台，类似 TinyGo 的能力：

### 📋 第一阶段：基本目标参数支持 ✅ (已完成 - Issue #1194)
- ✅ 添加 `-target` 参数支持 `llgo build/run/test`
- ✅ 实现基于 JSON 的目标配置系统
- ✅ 100+ 个嵌入式平台配置文件（`/targets/` 目录）
- ✅ `crosscompile.UseWithTarget()` 函数实现
- ✅ 弱符号 `_start()` 入口点支持无 libc 环境

### 🔄 第二阶段：多平台 LLVM 支持 (进行中)
- 🔄 支持多平台（X86、ARM、RISC-V 等）
- 🔄 生成可启动代码（bootable code）
- 🔄 集成链接器脚本（linker script）
- 🔄 Flash 编程集成（烧录支持）

### ⏳ 第三阶段：通用机器库 (可与第一阶段并行开发)
- 创建统一的硬件抽象层
- 支持 GPIO、SPI、I2C、UART 等接口
- 保持与 TinyGo 的兼容性
- 类似 Arduino 的 `digitalWrite()`、`digitalRead()` 抽象
- **并行开发优势**：硬件接口设计独立于底层编译实现，可以同时进行

### ⏳ 第四阶段：硬件特定机器库 (低优先级)
- 开发平台特定的库
- 使用构建标签区分特定硬件特性
- 例如：STM32 特有功能、ESP32 WiFi、RP2040 PIO 等

## 嵌入式系统支持实现详情 (Issue #1194 - 第一阶段完成)

LLGO 通过从 TinyGo 导入的三个关键功能实现了全面的嵌入式系统支持：

### 目标配置系统
- **`/targets/`** - 包含 100+ 个嵌入式平台的 JSON 目标定义文件
  - Arduino 系列：`arduino-leonardo.json`、`arduino-nano.json`、`arduino-zero.json`
  - ESP32 系列：`esp32c3.json`、`esp32-coreboard-v2.json`、`esp-c3-32s-kit.json`
  - RP2040/RP2350：`rp2040.json`、`pico.json`、`pico2.json`、`feather-rp2040.json`
  - STM32 系列：`stm32f4disco.json`、`nucleo-f103rb.json`、`bluepill.json`
  - RISC-V：`riscv32.json`、`riscv64.json`、`k210.json`、`hifive1b.json`
  - ARM Cortex-M：`cortex-m0.json`、`cortex-m4.json`、`cortex-m7.json`
  - WebAssembly：`wasm.json`、`wasip1.json`、`wasip2.json`
- **继承机制**：目标可以继承其他目标（如 `rp2040.json` 继承自 `cortex-m0plus`）
- **配置字段**：`llvm-target`、`cpu`、`features`、`build-tags`、`goos`、`goarch`、`cflags`、`ldflags`

### 支持目标的交叉编译
```bash
# 使用 -target 标志进行嵌入式编译
llgo build -target rp2040 .
llgo build -target esp32c3 .
llgo build -target wasm .
llgo run -target cortex-m4 .
```

- **`internal/crosscompile/crosscompile.go:273`** - `UseWithTarget()` 函数实现
- **目标解析**：`internal/targets/` 包负责加载和解析目标配置
- **编译器标志生成**：自动将目标配置转换为 CCFLAGS/LDFLAGS
- **LLVM 集成**：使用 LLVM 目标三元组和 CPU 特定优化

### 无 libc 入口点支持
- **`internal/build/build.go:715-726`** - 弱符号 `_start()` 函数定义
- **用途**：当 libc 不可用时提供入口点（裸机/嵌入式环境）
- **实现**：简单的 `_start()` 调用 `main(0, null)` 提供最小运行时
- **LLVM IR**：生成弱符号定义，避免与系统 libc 冲突

### 嵌入式 CI 测试
- **`.github/workflows/targets.yml`** - 所有嵌入式目标的自动化测试
- **测试策略**：使用最小化的 `_demo/empty/empty.go`（空 main 函数）
- **覆盖范围**：测试 100+ 嵌入式目标，验证编译而不依赖复杂依赖
- **验证方式**：每个目标显示 ✅ 成功或 ❌ 失败及文件类型信息

### 目标使用示例
```bash
# 列出可用目标
ls targets/*.json | sed 's/targets\///g' | sed 's/\.json//g'

# 为特定嵌入式平台构建
cd _demo/hello
llgo build -target rp2040 -o firmware.elf .           # Raspberry Pi Pico
llgo build -target esp32c3 -o firmware.bin .          # ESP32-C3
llgo build -target esp32-coreboard-v2 -o firmware.bin . # ESP32 CoreBoard V2 开发板
llgo build -target arduino-nano -o sketch.hex .       # Arduino Nano
llgo build -target cortex-m4 -o firmware.o .          # 通用 ARM Cortex-M4

# 开发板专用编译（查看详细构建过程）
llgo build -target esp32-coreboard-v2 -v -o firmware.bin .  # 显示工具链调用详情
llgo build -target rp2040 -v -o firmware.uf2 .             # Pico 开发详细日志

# WebAssembly 目标
llgo build -target wasm -o module.wasm .
llgo build -target wasip1 -o program.wasm .

# 调试嵌入式编译过程
llgo build -target esp32-coreboard-v2 -v . 2>&1 | grep clang  # 查看使用的编译器
llgo build -target esp32c3 -v . 2>&1 | head -20               # 查看前20行构建日志
```

## 技术架构核心思想

这个构建目标设计计划的核心架构理念：

1. **目标抽象** - 不局限于传统的 GOOS/GOARCH，而是支持具体的硬件平台定义
2. **配置驱动** - 通过 JSON 配置文件灵活定义每个平台的编译特性和硬件能力
3. **LLVM 深度集成** - 充分利用 LLVM 强大的交叉编译和优化能力
4. **渐进式抽象** - 从底层平台支持逐步发展到高层硬件接口抽象

这个嵌入式支持使得 Go 程序能够在微控制器和嵌入式系统上以最小的运行时开销运行，为 Go 语言开辟了嵌入式和 IoT 开发的新领域。

**当前状态**：第一阶段已完成，LLGO 现在可以编译到 100+ 个嵌入式目标平台！🎉

## LLGO 二进制依赖调查 (2025-09-01)

### 问题背景
- **调查目的**: 确定 LLGO 编译器二进制对 libLLVM 是静态依赖还是动态依赖
- **调查原因**: 了解部署和分发时的依赖要求

### 调查步骤
1. **定位 LLGO 二进制**:
   - 通过 `which llgo` 找到 `/opt/homebrew/bin/llgo`
   - 发现这是一个 shell 脚本包装器，实际二进制在 `/opt/homebrew/Cellar/llgo/0.12.13/libexec/bin/llgo`

2. **分析动态依赖**:
   ```bash
   # 查看动态库依赖
   otool -L /opt/homebrew/Cellar/llgo/0.12.13/libexec/bin/llgo
   ```

3. **检查 RPATH 配置**:
   ```bash
   # 查看运行时库搜索路径
   otool -l llgo_binary | grep -A 2 LC_RPATH
   ```

### 核心发现
**LLGO 编译出的二进制是动态依赖 libLLVM 的，不是静态链接**

#### 关键证据:
- **动态依赖**: `@rpath/libLLVM.dylib` (版本 19.1.2)
- **RPATH 设置**: `/opt/homebrew/Cellar/llgo/0.12.13/libexec/crosscompile/clang/lib`
- **运行时库**: `libLLVM.dylib` 约 82MB，位于 RPATH 指定路径
- **完整依赖列表**:
  ```
  @rpath/libLLVM.dylib
  @rpath/libc++.1.dylib  
  @rpath/libunwind.1.dylib
  /usr/lib/libresolv.9.dylib
  /System/Library/Frameworks/CoreFoundation.framework/...
  /System/Library/Frameworks/Security.framework/...
  /usr/lib/libSystem.B.dylib
  ```

#### 技术架构:
- **包装脚本**: `/opt/homebrew/bin/llgo` → shell 脚本，设置 PATH 并调用实际二进制
- **实际二进制**: `/opt/homebrew/Cellar/llgo/0.12.13/libexec/bin/llgo` → 动态链接的可执行文件
- **LLVM 库路径**: RPATH 指向专门的 clang 库目录，包含完整的 LLVM 工具链

### 影响分析
**动态链接的优缺点**:

✅ **优点**:
- 二进制文件更小（不包含 82MB 的 LLVM 库）
- 多个程序可共享同一份 LLVM 库
- 内存使用更高效
- 更新 LLVM 库时无需重新编译所有依赖程序

⚠️ **缺点**:
- 部署复杂性：目标系统必须有兼容的 LLVM 19 库
- 版本依赖：严格依赖 LLVM 19.1.2，与其他版本不兼容
- 分发挑战：不能作为单一可执行文件分发

### 部署建议
1. **Homebrew 用户**: 通过 `brew install llgo` 自动处理所有依赖
2. **手动部署**: 需要确保目标系统有 LLVM 19 和相关动态库
3. **Docker 部署**: 使用包含完整 LLVM 环境的基础镜像
4. **静态链接需求**: 如需独立二进制，需要修改构建配置使用静态链接

### 相关文件位置
- **依赖分析工具**: `otool -L` (macOS), `ldd` (Linux)
- **RPATH 配置**: 编译时设置，运行时库搜索路径
- **实际库位置**: `/opt/homebrew/Cellar/llgo/.../libexec/crosscompile/clang/lib/`

### GoReleaser 配置验证
**`.goreleaser.yaml` 完全验证了动态链接设计**:

#### 多平台构建配置 (4个目标)
- **macOS Intel**: `/usr/local/opt/llvm@19` + `o64-clang` 交叉编译
- **macOS Apple Silicon**: `/opt/homebrew/opt/llvm@19` + `oa64-clang++`  
- **Linux x86_64**: `/usr/lib/llvm-19` + `x86_64-linux-gnu-gcc`
- **Linux ARM64**: `/usr/lib/llvm-19` + `aarch64-linux-gnu-gcc`

#### 动态链接配置证据
- **`byollvm` 构建标签**: Build with Your Own LLVM，使用系统 LLVM
- **CGO_LDFLAGS**: 所有平台都使用 `-lLLVM-19` 动态链接
- **硬编码 llvm-config 路径**: 每平台指定具体的 `ldLLVMConfigBin`
- **Sysroot 依赖**: 使用预构建的 `.sysroot/` 目录进行交叉编译

#### 发布格式支持
- **tar.gz**: 包含 `bin/llgo` + `runtime/` + 文档
- **Linux 包**: deb/rpm 系统包管理器集成
- **Snap**: Ubuntu Store 分发，使用 `classic` 限制模式
- **校验和**: 自动生成文件完整性验证

#### 技术设计理念
这个配置揭示了 LLGO 团队的**技术权衡决策**：
- ✅ **减小分发尺寸**: 避免在每个平台二进制中打包 82MB LLVM
- ✅ **利用系统资源**: 复用已安装的 LLVM 19 提高内存效率
- ⚠️ **增加部署复杂性**: 需要精确匹配的 LLVM 版本和路径
- 🎯 **专业工具定位**: 面向已有 LLVM 环境的开发者，非通用分发
# CLAUDE-zh.md

本文件为 Claude Code (claude.ai/code) 在本代码库中工作时提供中文指导。

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
llgo build -target rp2040 .   # Raspberry Pi Pico
llgo build -target esp32c3 .  # ESP32-C3
llgo build -target wasm .     # WebAssembly

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
llgo build -target rp2040 -o firmware.elf .      # Raspberry Pi Pico
llgo build -target esp32c3 -o firmware.bin .     # ESP32-C3
llgo build -target arduino-nano -o sketch.hex .  # Arduino Nano
llgo build -target cortex-m4 -o firmware.o .     # 通用 ARM Cortex-M4

# WebAssembly 目标
llgo build -target wasm -o module.wasm .
llgo build -target wasip1 -o program.wasm .
```

## 技术架构核心思想

这个构建目标设计计划的核心架构理念：

1. **目标抽象** - 不局限于传统的 GOOS/GOARCH，而是支持具体的硬件平台定义
2. **配置驱动** - 通过 JSON 配置文件灵活定义每个平台的编译特性和硬件能力
3. **LLVM 深度集成** - 充分利用 LLVM 强大的交叉编译和优化能力
4. **渐进式抽象** - 从底层平台支持逐步发展到高层硬件接口抽象

这个嵌入式支持使得 Go 程序能够在微控制器和嵌入式系统上以最小的运行时开销运行，为 Go 语言开辟了嵌入式和 IoT 开发的新领域。

**当前状态**：第一阶段已完成，LLGO 现在可以编译到 100+ 个嵌入式目标平台！🎉
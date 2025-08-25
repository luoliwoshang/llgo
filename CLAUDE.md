# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LLGO is a Go compiler based on LLVM designed to integrate Go with the C ecosystem including Python. It's a subproject of the XGo project that aims to expand Go's boundaries by enabling seamless interoperability with C/C++ and Python libraries.

## Common Commands

### Installation and Setup
```bash
# Install from local source
./install.sh

# Development wrapper (builds and runs llgo)
./llgo.sh [args]

# Manual installation
go install ./cmd/llgo
```

### Building and Testing
```bash
# Build all packages
go build -v ./...

# Run all tests
go test ./...

# Run specific test
go test ./cl -run TestSpecificTest

# Run tests with llgo compiler
llgo test ./...

# Format code (required for CI)
go fmt ./...

# Check formatting across project
for dir in . runtime; do
  pushd $dir
  go fmt ./...
  popd
done
```

### LLGO Compilation Commands
```bash
# Basic llgo usage (mimics go command)
llgo build [flags] [packages]    # Compile packages
llgo run [flags] package [args]  # Compile and run
llgo test [flags] package [args] # Compile and run tests
llgo install [flags] [packages]  # Compile and install
llgo clean [flags] [packages]    # Remove object files
llgo version                     # Print version
llgo cmptest [flags] package     # Compare output with standard go

# Cross-compilation to embedded targets
llgo build -target rp2040 .      # Raspberry Pi Pico
llgo build -target esp32c3 .     # ESP32-C3
llgo build -target wasm .        # WebAssembly

# Disable garbage collection
llgo run -tags nogc .
```

### Development Tools
```bash
# Install all tools
go install -v ./cmd/...
go install -v ./chore/...

# Build llgen tool for generating LLVM IR
go build -o llgen ./chore/llgen

# Generate LLVM IR from Go files
./llgen your_file.go  # Creates .ll file with LLVM IR

# Install Python tools (requires llgo)
export LLGO_ROOT=$PWD
cd _xtool
llgo install ./...

# Install external Python signature fetcher
go install github.com/goplus/hdq/chore/pysigfetch@v0.8.1
```

## Architecture Overview

### Core Components

- **`/cmd/llgo/`** - Main CLI application built with XGo/GoPlus framework using `.gox` files (not `.go`)
- **`/cl/`** - Core compiler logic that converts Go AST to LLVM IR via Go SSA
- **`/ssa/`** - LLVM SSA generation using Go SSA semantics, provides high-level LLVM interface bridging Go and LLVM
- **`/runtime/`** - Custom Go runtime implementation with local module replacement for LLVM compatibility
- **`/internal/build/`** - Build orchestration that coordinates the entire compilation pipeline
- **`/targets/`** - 100+ JSON target configuration files for embedded platforms and cross-compilation

### Compiler Directives Architecture

LLGO implements compiler directives (like `llgo.cstr`, `llgo.atomicLoad`) using a "bypass architecture" that **directly couples with SSA intermediate representation rather than Go function signatures**:

```
Normal Go Functions:  Go AST → Go Types → SSA → LLVM IR
                               ^^^^^^^^^ requires signature

Compiler Directives:  Go AST → SSA → LLVM IR  
                              ^^^ bypasses type system
```

**Key Characteristics:**
- Go function signatures serve as user-facing type safety guarantees
- Actual return values are determined entirely by compiler internal implementation
- All directive implementations follow: `func(b llssa.Builder, args []ssa.Value) llssa.Expr`
- This enables compile-time optimizations and platform-specific code generation

### Development Tools (chore/)

- **`llgen/`** - Compiles Go packages into LLVM IR files (*.ll) for debugging and analysis
- **`llpyg/`** - Converts Python libraries into Go packages automatically using symbol extraction
- **`ssadump/`** - Go SSA builder and interpreter for SSA analysis
- **`pydump/`** - Extracts Python library symbols (first production llgo program, compiled with llgo itself)
- **`nmdump/`** - Symbol extraction from object files using `nm`
- **`clangpp/`** - C++ integration tooling

### Key Directories

- **`/_demo/`** - C standard library integration demos and platform compatibility tests
  - Each subdirectory contains a working example that can be run with `llgo run .`
  - These demos are automatically tested in CI across multiple platforms (macOS, Ubuntu)
  - Used to verify LLGO functionality and catch platform-specific issues
  - Examples: `hello/`, `qsort/`, `readdir/`, `goroutine/`, etc.
- **`/_pydemo/`** - Python integration examples  
- **`/_cmptest/`** - Comparison tests between Go and llgo output
- **`/cl/_test*/`** - Extensive compiler test suites

## Development Environment

### Dependencies
- Go 1.23+ (specific version required, uses go1.24.1 toolchain)
- LLVM 19 (specific version required - not compatible with other versions)
- Clang 19, LLD 19 (must match LLVM version)
- bdwgc (Boehm garbage collector) for default GC implementation
- OpenSSL, libffi, libuv, pkg-config for C ecosystem integration
- Python 3.12+ (optional, for Python library integration via `py` packages)

### Platform-Specific Setup

**macOS:**
```bash
brew install llvm@19 lld@19 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12  # optional
brew link --overwrite llvm@19 lld@19 libffi
```

**Ubuntu/Debian:**
```bash
echo "deb http://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-19 main" | sudo tee /etc/apt/sources.list.d/llvm.list
wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y llvm-19-dev clang-19 lld-19 libgc-dev libssl-dev zlib1g-dev libcjson-dev libuv1-dev
```

## Key Technical Details

### C/Python Integration
- Uses `//go:linkname` to bind external symbols through ABI
- Python libraries automatically converted via `pydump` and `llpyg` tools
- C libraries integrated by parsing symbols with `nm` tool
- **Export Functions to C**: Use `//export` directive to create C-callable functions with simple names
  - 📖 **[Complete Export Directive Guide](export-directive.md)** - Detailed documentation on using `//export` for C interoperability
- **LLVM Target Configuration**: LLGO supports cross-compilation through LLVM target triples
  - 📖 **[LLVM Target Triple Configuration](target-triple.md)** - Complete guide to target triple generation and cross-compilation
  - 📖 **[LLVM Data Layout Configuration](data-layout.md)** - Understanding LLVM data layout and memory layout configuration

### Special Features
- **No defer in loops**: Intentional limitation for performance
- **Garbage Collection**: Uses bdwgc by default, disable with `-tags nogc`
- **WebAssembly support**: Can compile to WASM
- **Cross-compilation**: Multi-platform via goreleaser

### Module Structure
- Uses Go modules with local runtime replacement: `replace github.com/goplus/llgo/runtime => ./runtime`
- CLI built with XGo/GoPlus framework using `.gox` files instead of `.go` (supports advanced Go+ syntax)
- LLVM dependency requires specific installation paths and version matching
- Complex test suite structure with separate directories: `_testdata/`, `_testgo/`, `_testlibc/`, `_testpy/`, `_testrt/`

### Testing
- **Comparison Testing**: Compare llgo output with standard Go using `llgo cmptest`
- **Demo Validation**: All `_demo/` examples are tested in CI to ensure cross-platform compatibility
- **Behavior Verification**: Test that both `llgo run .` and `go run .` produce identical results
- **Regression Prevention**: New functionality should include demo examples in `_demo/`
- **Embedded Target Testing**: CI validates 100+ target configurations in `.github/workflows/targets.yml`
- **Format Validation**: Strict formatting requirements enforced via `go fmt` in CI
- Extensive test suites across multiple `_test*` directories (`_testgo/`, `_testlibc/`, `_testpy/`, `_testrt/`)
- CI runs on macOS and Ubuntu with LLVM 19 across multiple versions

## Running Examples

```bash
# C integration demos
cd _demo/hello && llgo run .
cd _demo/qsort && llgo run .

# OS functionality demos
cd _demo/readdir && llgo run .     # Directory reading (os.ReadDir)
cd _demo/goroutine && llgo run .   # Goroutine support

# Python integration demos  
cd _pydemo/callpy && llgo run .
cd _pydemo/matrix && llgo run .

# Compare LLGO vs Go behavior
cd _demo/readdir
llgo run .  # Run with LLGO
go run .    # Run with standard Go (should produce identical output)
```

## LLGO Build Target Design 完整计划 (Issue #1176)

LLGO 构建目标设计是一个**四阶段宏伟计划**，目标是让 LLGO 支持各种嵌入式和硬件平台，类似 TinyGo 的能力：

### 📋 第一阶段: Basic Target Parameter Support ✅ (已完成 - Issue #1194)
- ✅ 添加 `-target` 参数支持 `llgo build/run/test`
- ✅ 实现基于 JSON 的目标配置系统
- ✅ 100+ 个嵌入式平台配置文件 (`/targets/` 目录)
- ✅ `crosscompile.UseWithTarget()` 函数实现
- ✅ 弱符号 `_start()` 入口点支持无 libc 环境

### 🔄 第二阶段: Multi-Platform LLVM Support (进行中)
- 🔄 支持多平台 (X86, ARM, RISC-V 等)
- 🔄 生成可启动代码 (bootable code)
- 🔄 集成链接器脚本 (linker script)
- 🔄 Flash 编程集成 (烧录支持)

### ⏳ 第三阶段: Generic Machine Library (可与第一阶段平行开发)
- 创建统一的硬件抽象层
- 支持 GPIO, SPI, I2C, UART 等接口
- 保持与 TinyGo 的兼容性
- 类似 Arduino 的 `digitalWrite()`, `digitalRead()` 抽象
- **平行开发优势**: 硬件接口设计独立于底层编译实现，可以同时进行

### ⏳ 第四阶段: Hardware-Specific Machine Library (低优先级)
- 开发平台特定的库
- 使用构建标签区分特定硬件特性
- 例如：STM32 特有功能、ESP32 WiFi、RP2040 PIO 等

## 嵌入式系统支持实现详情 (Issue #1194 - 第一阶段完成)

LLGO 通过从 TinyGo 导入的三个关键功能实现了全面的嵌入式系统支持：

### 目标配置系统
- **`/targets/`** - 包含 100+ 个嵌入式平台的 JSON 目标定义文件
  - Arduino 系列: `arduino-leonardo.json`, `arduino-nano.json`, `arduino-zero.json`
  - ESP32 系列: `esp32c3.json`, `esp32-coreboard-v2.json`, `esp-c3-32s-kit.json`
  - RP2040/RP2350: `rp2040.json`, `pico.json`, `pico2.json`, `feather-rp2040.json`
  - STM32 系列: `stm32f4disco.json`, `nucleo-f103rb.json`, `bluepill.json`
  - RISC-V: `riscv32.json`, `riscv64.json`, `k210.json`, `hifive1b.json`
  - ARM Cortex-M: `cortex-m0.json`, `cortex-m4.json`, `cortex-m7.json`
  - WebAssembly: `wasm.json`, `wasip1.json`, `wasip2.json`
- **继承机制**: 目标可以继承其他目标 (如 `rp2040.json` 继承自 `cortex-m0plus`)
- **配置字段**: `llvm-target`, `cpu`, `features`, `build-tags`, `goos`, `goarch`, `cflags`, `ldflags`

### 支持目标的交叉编译
```bash
# 使用 -target 标志进行嵌入式编译
llgo build -target rp2040 .
llgo build -target esp32c3 .
llgo build -target wasm .
llgo run -target cortex-m4 .
```

- **`internal/crosscompile/crosscompile.go:273`** - `UseWithTarget()` 函数实现
- **目标解析**: `internal/targets/` 包负责加载和解析目标配置
- **编译器标志生成**: 自动将目标配置转换为 CCFLAGS/LDFLAGS
- **LLVM 集成**: 使用 LLVM 目标三元组和 CPU 特定优化

### 无 libc 入口点支持
- **`internal/build/build.go:715-726`** - 弱符号 `_start()` 函数定义
- **用途**: 当 libc 不可用时提供入口点 (裸机/嵌入式环境)
- **实现**: 简单的 `_start()` 调用 `main(0, null)` 提供最小运行时
- **LLVM IR**: 生成弱符号定义，避免与系统 libc 冲突

### 嵌入式 CI 测试
- **`.github/workflows/targets.yml`** - 所有嵌入式目标的自动化测试
- **测试策略**: 使用最小化的 `_demo/empty/empty.go` (空 main 函数)
- **覆盖范围**: 测试 100+ 嵌入式目标，验证编译而不依赖复杂依赖
- **验证方式**: 每个目标显示 ✅ 成功或 ❌ 失败及文件类型信息

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

**当前状态**: 第一阶段已完成，LLGO 现在可以编译到 100+ 个嵌入式目标平台！🎉
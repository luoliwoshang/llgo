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

- **`/cmd/llgo/`** - Main CLI application built with XGo/GoPlus framework using `.gox` files
- **`/cl/`** - Core compiler logic that converts Go AST to LLVM IR
- **`/ssa/`** - LLVM SSA generation using Go SSA semantics, provides high-level LLVM interface
- **`/runtime/`** - Custom Go runtime implementation with local module replacement
- **`/internal/build/`** - Build orchestration that strings together the compilation process

### Development Tools (chore/)

- **`llgen/`** - Compiles Go packages into LLVM IR files (*.ll)
- **`llpyg/`** - Converts Python libraries into Go packages automatically
- **`ssadump/`** - Go SSA builder and interpreter
- **`pydump/`** - Extracts Python library symbols (first production llgo program)

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
- Go 1.23+
- LLVM 19 (specific version required)
- Clang 19, LLD 19
- bdwgc (Boehm garbage collector)
- OpenSSL, libffi, libuv, pkg-config
- Python 3.12+ (optional, for Python integration)

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
- CLI built with XGo framework using `.gox` files instead of `.go`
- LLVM dependency requires specific installation paths

### Testing
- **Comparison Testing**: Compare llgo output with standard Go using `llgo cmptest`
- **Demo Validation**: All `_demo/` examples are tested in CI to ensure cross-platform compatibility
- **Behavior Verification**: Test that both `llgo run .` and `go run .` produce identical results
- **Regression Prevention**: New functionality should include demo examples in `_demo/`
- Extensive test suites across multiple `_test*` directories
- CI runs on macOS and Ubuntu with coverage reporting

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
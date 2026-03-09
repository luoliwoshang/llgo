# ESP32-C3 + goplus/newlib 调试经验总结

本文记录 llgo 在 ESP32-C3 上联动 `goplus/newlib` 的一次完整调试过程，重点是“为什么会失败、怎么稳定复现、怎么修”。

## 1. 范围和背景

- 目标平台：`esp32c3-basic`（QEMU + semihosting）
- libc 来源：`goplus/newlib`（esp 分支补丁版本）
- 主要关注点：
  - 浮点格式化输出（`printf("%f")`）
  - emulator 回归测试可重复性
  - CI 下的常见卡住/误判

## 2. 最关键结论

1. `newlib nano formatted io` 下，浮点 printf 不是默认拉入。
2. 需要在最终链接参数里显式声明 `_printf_float`。
3. 推荐写法：`--undefined=_printf_float`（单参数）或 `-u`, `_printf_float`（双参数）。
4. `-u=_printf_float` 在当前链路下不可用，会导致浮点输出路径没被正确拉入。

## 3. `_printf_float` 的根因和正确配置

`goplus/newlib` 的 README（`--enable-newlib-nano-formatted-io` 小节）说明：

- nano 版格式化 I/O 将浮点支持做成弱符号路径；
- 如果程序需要 `%f/%e` 等浮点格式化，必须通过 linker 显式请求 `_printf_float`（或 scanf 对应符号）。

因此，配置策略建议是：

- 在 **target 外层 ldflags** 声明（例如 `targets/esp32c3-basic.json`）；
- 不在 `getNewlibESP32ConfigRISCV` 的 libc compile group 里重复注入，避免职责混叠。

当前实践中，以下写法可工作：

```json
"ldflags": [
  "--undefined=_printf_float"
]
```

## 4. 为什么 `-u=_printf_float` 会踩坑

调试中观察到：

- `-u=_printf_float` 形式下，float 回归 case 会退化成“只打印整数部分/空浮点字段”；
- 改为 `--undefined=_printf_float` 或 `-u`, `_printf_float` 后恢复正常。

经验结论：

- 对当前 linker 链路，`-u` 的值应作为独立参数或使用 `--undefined=...`；
- 不要依赖 `-u=...` 这种短参数等号写法。

## 5. QEMU 相关问题

常见失败：

```text
exec: "qemu-system-riscv32": executable file not found in $PATH
```

处理方式：

```bash
export PATH="/Users/heulucklu/project/llgo/.cache/qemu/bin:$PATH"
```

建议在本地调试和 CI 脚本中都显式保证 PATH，避免环境差异导致“假失败”。

## 6. 回归测试设计经验

建议保留并持续使用 `_demo/embed/test_esp32c3_startup.sh` 这种“从构建到仿真到输出比对”的端到端脚本，覆盖：

1. startup 是否走 newlib 初始化路径（`__libc_init_array`）
2. `.init_array` 是否进入 `.rodata` 并进入 bin segment
3. emulator 输出是否符合预期
4. 浮点回归 case 是否稳定

其中第 4 点建议使用固定 case（如 `float-1664`），对最终几行输出做严格比较，避免被 boot log 干扰。

## 7. 运行目录与工具链版本的两个坑

### 7.1 运行目录问题（`LLGO_ROOT`/资源定位）

如果运行目录和预期工程结构不一致，可能出现类似 `missing runtime/internal/lib` 的快速失败。

建议：

- 尽量在仓库内固定目录运行；
- 需要时显式设置 `LLGO_ROOT`，避免靠隐式工作目录推断。

### 7.2 Go 版本矩阵问题

某些 `_testgo` case 在低版本 Go（例如 1.21）下会因 `go.mod requires go >= ...` 快速失败。

建议：

- ESP32-C3 浮点回归优先放在 `_demo/embed/esp32c3` 这种最小依赖目录；
- 避免把回归点绑定到高版本 toolchain 才可运行的测试树。

## 8. 一组可复用的排查命令

```bash
# 1) 运行端到端回归（含 QEMU）
PATH="/Users/heulucklu/project/llgo/.cache/qemu/bin:$PATH" \
  bash _demo/embed/test_esp32c3_startup.sh

# 2) 单独跑 emulator case
llgo run -a -target=esp32c3-basic -emulator ./_demo/embed/esp32c3/float-1664

# 3) 跑交叉编译相关单测
go test ./internal/crosscompile/...
```

## 9. 对后续 ESP32-C3 支持的建议

1. 把 `_printf_float` 语义写入 target 文档，避免后续反复踩坑。
2. 固化一条“浮点输出回归”的 CI 检查路径。
3. 对 emulator 输出比对统一采用“尾行提取 + 精确匹配”。
4. 将环境依赖（QEMU、esptool）检查前置到脚本开头。


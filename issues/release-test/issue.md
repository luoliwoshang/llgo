# Issue: LLGO Release 测试机制改进

## 问题描述

当前 LLGO 的发布流程存在以下问题：

1. **发布后不可用问题** - 发布的版本可能存在运行时问题，但在发布前没有充分验证
2. **缺乏 PR 验证** - 只有在创建 Git tag 时才会触发 goreleaser 构建，无法在 PR 阶段验证构建是否正常
3. **发布风险高** - 构建配置问题只有在真正发布时才会暴露，影响发布流程

## 当前状况分析

### 现有 Workflow 触发条件
```yaml
on:
  push:
    tags: ["*"]           # 只在 tag 推送时触发
```

**当前问题**: 只有创建 Git tag 时才会触发构建，无法在开发阶段验证构建配置

### 期望的 Workflow 触发条件
```yaml
on:
  push:
    branches: ["**"]      # 每次分支推送都触发
    tags: ["*"]           # 包括 tag 推送
  pull_request:
    branches: ["**"]      # 每次 PR 都触发
```

**目的**: 确保每次代码变更都经过完整的构建和测试验证，保证 PR 质量

### 当前 GoReleaser 命令
```bash
goreleaser release --clean --skip nfpm,snapcraft
```

**问题**: 这个命令会直接发布到 GitHub Releases，并不经过测试。

## 期望的构建流程设计

我们希望实现一个更安全的发布流程：

### 阶段一：构建验证
1. **触发构建** - Tag 推送时触发 goreleaser
2. **构建所有平台** - 执行 `goreleaser release --skip=publish,nfpm,snapcraft`
3. **保存构建产物** - 将 `dist/` 目录上传到 GitHub Artifacts

### 阶段二：自动化测试
4. **Matrix 并行测试** - 启动多个测试 job，每个对应一个平台
   - macOS x86_64 测试 job
   - macOS ARM64 测试 job  
   - Linux x86_64 测试 job
   - Linux ARM64 测试 job

5. **下载并测试** - 每个测试 job：
   - 从 artifacts 下载对应平台的构建产物
   - 解压并运行对应的 demo 程序
   - 验证基本功能是否正常

### 阶段三：发布决策
6. **等待所有测试完成** - 只有当所有 4 个平台测试都通过时
7. **条件发布判断** - 检查触发事件类型：
   - **如果是 tag push 事件** → 执行正式发布到 GitHub Releases
   - **如果是普通 push/PR 事件** → 输出测试结果并结束，不进行发布

### 优势
- ✅ **每次变更都验证** - 每个 push/PR 都经过完整构建测试
- ✅ **保证 PR 质量** - 合并前就发现潜在问题
- ✅ **安全发布** - 只有测试通过的 tag 才会发布
- ✅ **早期发现问题** - 在发布前发现平台特定的问题
- ✅ **自动化验证** - 减少手动测试工作量
- ✅ **失败快速回滚** - 测试失败时不会发布损坏的版本
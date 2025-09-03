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

### 当前 GoReleaser 命令
```bash
goreleaser release --clean --skip nfpm,snapcraft
```

**问题**: 这个命令会直接发布到 GitHub Releases，在 PR/push 时执行会造成意外发布！
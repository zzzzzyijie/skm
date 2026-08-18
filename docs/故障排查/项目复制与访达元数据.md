# 项目 Skill 副本被误判为已修改

## 问题现象

在项目中以 `copy` 模式部署 `ios-debugger-agent` 后，Skill Library 的操作可能报错：

```text
refusing to overwrite unmanaged target <project>/.claude/skills/ios-debugger-agent
```

在项目页面移除同一 Skill 时，可能报错：

```text
refusing to remove modified managed target <project>/.claude/skills/ios-debugger-agent; use --force
```

这里的项目为本机注册项目，Skill 同时部署给 Claude Code 和 Codex。

## 根因

`copy` 模式会将 Library 的不可变快照复制到项目 Agent 目录，并在状态中保存源内容
hash。后续计划构建会重新计算目标目录 hash，以区分 skm 管理的副本、用户修改的副本
和未受管理的目录。

macOS Finder 会在浏览目录时写入 `.DS_Store`。该文件不是 Skill 内容的改动，但旧逻辑
将它纳入目录 hash。因此一个内容未变的项目副本会被标记为已修改：

1. Library 启用操作构建了所有项目和用户级部署的全局计划；
2. 该项目副本被识别为冲突，导致无关的 Library 操作也失败；
3. 项目移除为防止删除用户内容而拒绝删除，并要求 `--force`。

## 解决方案

### 复制副本的比较

保留原始 `HashDir` 及已保存的 hash，避免改变既有状态格式。仅在比较 `copy` 目标与
源快照时，如果常规 hash 不一致，再使用忽略 Finder 元数据的比较：

- 比较时忽略任意目录中的 `.DS_Store`；
- 其他新增、删除或编辑的文件仍会被判定为修改；
- 真正修改后的副本依旧不能在未确认的情况下覆盖或删除。

### 按部署范围构建计划

新增按 placement 和项目根目录过滤的计划构建：

- Skill Library 的启用操作只检查用户级部署；
- 项目部署和项目状态只检查当前项目；
- 一个项目中的冲突不再阻断另一个项目或用户级的操作。

### Web UI 的移除确认

普通项目解绑继续直接执行。若后端明确返回“目标已修改，需要 `--force`”，项目页面
显示确认窗口；用户确认后才发送 `force: true` 删除项目副本。

## 验证

自动化测试覆盖：

- `.DS_Store` 不影响复制副本的状态与普通解绑；
- 真实内容修改仍要求强制移除；
- 项目副本冲突不会阻断用户级启用；
- Web API 在强制参数存在时可移除已修改副本。

对受影响项目执行只读 `plan` 检查时，Claude Code 和 Codex 的部署均返回
`unchanged`。

## 预防与边界

`.DS_Store` 只是 Finder 元数据，不再参与复制副本的完整性判断。其他系统生成文件和
任何 Skill 内容变更仍会触发保护机制；只有用户明确确认后，Web UI 才会删除已修改的
项目副本。

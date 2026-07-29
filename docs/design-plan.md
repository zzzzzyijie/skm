# skm 产品与技术方案

> 状态：待评审
> 日期：2026-07-29
> 范围：CLI MVP、Git 同步及后续 macOS App

## 1. 项目定位

`skm` 是面向开发者的 AI Agent Skill 管理与部署工具。它的核心职责不是编辑 Skill，而是统一管理 Skill 的来源、版本、标签、作用域和部署状态，并将同一份 Skill 安全地映射到 Claude Code、Codex 等 Agent 的原生目录。

整体数据流：

```text
本地目录 / Git 仓库
        ↓
 Skill Catalog + Validator
        ↓
作用域解析与冲突处理
        ↓
变更计划 Plan
        ↓
Claude Code / Codex Adapter
        ↓
软链接或受控复制到各 Agent 目录
```

核心业务能力统一由 Go 实现。CLI 是第一个用户界面；后续 macOS App 通过稳定的 JSON 协议调用同一套能力，不重复实现 Skill 解析、同步与部署逻辑。

## 2. 作用域模型

README 中的 `Global / Personal / Project` 需要明确语义，避免 Global 和 Personal 都被理解成“当前用户的全局目录”。建议定义为：

| 作用域 | 含义 | 是否同步 | 优先级 |
| --- | --- | --- | --- |
| Global | 团队或公开的基础 Skill 集合 | Git | 低 |
| Personal | 个人私有 Skill 与个人覆盖 | 可选私有 Git | 中 |
| Project | 当前仓库专用 Skill 与覆盖 | 跟随项目 | 高 |

同名 Skill 的解析顺序：

```text
Project > Personal > Global
```

同一作用域中出现两个同名 Skill 时，不暗中选择，必须报告冲突并要求用户指定 `source/name`。`skm list` 应展示被更高作用域覆盖的 Skill。

长期内部模型应拆分为两个正交维度：

- 来源：`local / git / team / public`
- 生效范围：`user / project`

CLI 仍可展示三层视图，但存储模型不应把来源与生效范围永久绑定为同一个枚举。

## 3. 标签模型

支持使用标签区分开发、测试、文档、运维等不同场景。标签是 skm 的管理元数据，用于分类、搜索、筛选和批量操作，不参与作用域优先级计算，也不会改变 Agent 调用 Skill 的触发逻辑。

标签规则：

- 每个 Skill 至少有一个标签。
- 添加 Skill 时未指定标签，使用配置项 `defaults.tags`；初始默认值为 `general`。
- 用户显式指定标签时，使用显式标签替代默认标签，不自动附加 `general`。
- 一个 Skill 可以有多个标签，例如 `frontend`、`testing`、`code-review`。
- 标签 ID 使用小写字母、数字和连字符，最长 32 个字符；比较时不区分输入大小写并统一保存为小写。
- 标签不用于解决同名 Skill 冲突；同名冲突仍由 `source/name` 和作用域规则处理。

示例：

```bash
# 未指定标签，自动使用 general
skm add ./skills/commit --scope personal

# 显式定义一个或多个场景标签
skm add ./skills/review --tag backend --tag code-review

# 按标签查询或批量部署
skm list --tag code-review
skm link --tag backend --scope project --agent claude
```

重复传入多个 `--tag` 时默认使用 AND 语义，即 Skill 必须同时包含全部标签。后续可增加 `--match-tag any` 支持 OR 查询。

标签存放在 skm 自己的 Catalog 或 Project Manifest 中，不直接写入第三方 `SKILL.md`，避免引入 Agent 不认识的扩展字段和修改上游内容。项目示例：

```yaml
version: 1
skills:
  - id: team/code-review
    tags:
      - backend
      - code-review
```

标签管理命令：

```bash
skm tag list
skm tag add <skill> <tag...>
skm tag remove <skill> <tag...>
skm tag rename <old> <new>
```

移除 Skill 的最后一个标签后，系统自动恢复默认标签，确保不会出现无法分类的 Skill。

## 4. Skill 标准与 Agent Adapter

以开放的 Agent Skills 规范为基础。一个 Skill 是至少包含 `SKILL.md` 的目录；`SKILL.md` 使用 YAML Frontmatter 和 Markdown 正文，`name`、`description` 是基础元数据。Agent 私有字段应原样保留，避免读写后丢失信息。

首批 Adapter：

| Agent | 用户级目录 | 项目级目录 |
| --- | --- | --- |
| Claude Code | `~/.claude/skills/` | `.claude/skills/` |
| Codex | `~/.agents/skills/` | `.agents/skills/` |

每个 Adapter 负责：

- 检测 Agent 是否安装以及目标目录是否可用
- 返回用户级和项目级部署路径
- 声明支持的字段、作用域与部署模式
- 检查目标目录中的冲突和损坏状态
- 把统一部署意图转换成 Agent 对应的文件操作
- 输出兼容性警告

MVP 不把 Skills 自动转换成 Claude Code Rules、Codex Rules 或 `AGENTS.md`。这些机制的加载时机和行为不同，自动转换可能改变原始语义。

软链接是优先实现方式，但不应成为不可替换的产品契约。Adapter 需要支持：

- `symlink`：使用软链接
- `copy`：复制文件并由 skm 追踪内容哈希
- `auto`：根据 Agent 能力和当前环境选择

## 5. 本地目录与状态

第一版使用透明的 YAML/JSON 配置，不引入 SQLite：

```text
~/.skm/
├── config.yaml
├── sources/                 # Git 仓库缓存
├── catalog/
│   └── personal/            # 用户维护的 Skill
├── objects/                 # 按内容哈希保存的不可变版本
├── state/
│   └── links.json           # skm 创建过的部署记录
└── locks/                   # 防止 CLI/App 并发修改
```

项目内：

```text
<repo>/.skm/
├── project.yaml             # 项目期望启用的 Skill
├── lock.yaml                # Git commit 和内容哈希
└── skills/                  # 项目原创 Skill，可提交 Git
```

远程 Skill 不直接链接到可变的 Git checkout。Git 更新流程为：

1. 拉取远程内容到 Source 缓存。
2. 解析并验证 Skill。
3. 生成内容哈希与不可变快照。
4. 计算并展示部署计划。
5. 用户确认后原子切换链接或受控复制。
6. 保留上一版本，以支持回滚。

这样 Git 更新不会在用户确认前改变当前已生效的 Skill。

## 6. 领域模型

核心对象建议如下：

| 对象 | 关键字段 | 职责 |
| --- | --- | --- |
| Skill | ID、name、description、tags、content hash、metadata | 描述一个可部署能力包 |
| Source | ID、type、URL/path、ref、trust | 描述 Skill 来源 |
| Snapshot | Skill ID、source revision、content hash、path | 保存不可变版本 |
| Installation | Skill ID、scope、project、agents、mode | 表达期望部署状态 |
| Deployment | target、snapshot、owner、status | 记录实际文件状态 |
| Plan | operations、warnings、digest | 表示待执行的变更集合 |

Skill 的稳定标识使用 `source/name`。只有在当前上下文中名称唯一时，才允许用户仅输入 `name`。

## 7. CLI 命令设计

基础命令：

```bash
skm init
skm add <path|git-url> --scope personal|project [--tag ...]
skm list [--scope ...] [--agent ...] [--tag ...]
skm show <skill>
skm validate <path|skill>
skm link [skill...] --scope project --agent claude,codex
skm link --tag <tag> --scope project --agent claude,codex
skm unlink [skill...] --scope project
skm status
skm plan
skm apply
skm remove <skill>
skm doctor
skm tag list|add|remove|rename
```

Git 来源管理：

```bash
skm source add <git-url> --name team
skm source list
skm source update [name]
skm source remove <name>
skm sync
```

其中 `sync` 表示 `update + validate + plan + apply` 的组合流程。默认在应用变更前要求确认，自动化环境可使用 `--yes`。

所有会改变文件系统的命令都应支持：

- `--dry-run`
- `--json`
- `--yes`
- `--no-color`
- 稳定且有文档的退出码
- 非交互环境运行
- 默认不覆盖非 skm 管理的文件

`link` 是方便用户理解的命令，但底层仍统一调用 `plan -> apply` 引擎，避免不同命令产生不一致的文件操作逻辑。

标签筛选只决定哪些 Skill 进入本次 Plan，不改变被选中 Skill 的作用域解析、冲突检测和安全策略。

## 8. Plan 与冲突处理

Planner 扫描期望状态和实际文件系统后，将目标分类为：

| 状态 | 含义 | 默认操作 |
| --- | --- | --- |
| create | 目标不存在 | 创建 |
| unchanged | 当前部署正确 | 跳过 |
| replace-managed | 由 skm 创建，但版本落后 | 更新 |
| conflict-unmanaged | 存在用户文件或未知链接 | 停止并提示 |
| broken | 链接目标失效 | 提供修复计划 |
| shadowed | 被更高作用域同名 Skill 覆盖 | 展示但不部署 |

`apply` 应使用临时路径加原子重命名完成切换。Plan 带有输入状态摘要 `digest`；应用前重新校验，避免从预览到执行之间文件已经改变。

移除操作只处理 `state/links.json` 能确认属于 skm 的目标，不根据目录名猜测所有权。

## 9. Git 同步策略

CLI MVP 优先调用系统 `git`，复用用户现有的 SSH Agent、Keychain、代理和 Git 配置，不自行管理账号密码。

Source 支持：

- Git 仓库根目录包含多个 Skill
- 通过子目录选择单个 Skill
- branch、tag 或 commit 引用
- 项目 `lock.yaml` 锁定准确 commit 与内容哈希
- 更新前后展示 Skill 新增、删除和内容变化

第一版不自动处理 Git submodule，不执行远程仓库中的任何 Hook 或脚本。

## 10. 安全边界

- `skm` 管理 Skill 文件，但不执行 Skill 内脚本。
- 导入时阻止 `../` 路径穿越和逃逸 Skill 根目录的软链接。
- 设置单文件大小、文件数量和总目录大小限制。
- 更新后重新校验 Frontmatter、目录结构和内容哈希。
- 远程 Source 标记信任状态，首次启用前展示来源和变更。
- 配置文件写入使用临时文件和原子替换。
- CLI 和 App 使用文件锁，避免同时修改状态。
- 日志不得输出 Git 凭证、Token 或敏感环境变量。

## 11. Go 工程结构

```text
cmd/skm/
internal/
├── domain/       # Skill、Source、Scope、Deployment
├── skill/        # SKILL.md 解析与验证
├── catalog/      # 查询、版本、冲突解析
├── source/       # local/git
├── adapter/      # claude/codex
├── planner/      # plan、apply、rollback
├── config/
├── state/
├── fsx/
└── output/       # text/json
```

技术选择：

- Cobra：命令、参数和 Shell Completion
- YAML 结构化解析：保留未知 Frontmatter 字段
- 系统 `git`：复用已有认证环境
- 普通表格输出：CLI MVP 的默认交互
- Bubble Tea：推迟到确实需要复杂交互选择器时再加入

因为后续已有 SwiftUI App，第一版投入大量时间实现完整 TUI 的收益较低。

## 12. 测试策略

单元测试：

- YAML Frontmatter 解析与未知字段保留
- Skill 命名、路径和目录结构验证
- 三层作用域解析与同名冲突
- 默认标签、显式多标签、标签规范化和组合筛选
- 每个 Adapter 的路径映射
- Plan 分类、摘要和幂等性

集成测试：

- 在临时 HOME 中创建和切换软链接
- 复制模式的内容更新与回滚
- 未管理文件冲突
- 损坏链接修复
- 本地 Git 仓库的 add/update/lock 流程
- 两个进程争用状态锁

端到端测试不得直接操作开发者真实的 `~/.claude`、`~/.agents` 或 `~/.codex`。

## 13. macOS App

第一版采用常规主窗口，不优先做 Menu Bar App：

- 左侧栏：All Skills、Global、Personal、Projects、Sources
- 中间区域：可按名称、来源和标签搜索、排序、筛选的 Skill 列表
- 详情区域：说明、来源、版本、兼容 Agent、部署状态
- Project 视图：各 Agent 的启用状态、冲突和损坏链接
- Sync 视图：展示版本与内容变化，确认后应用
- 操作：添加、同步、启用、停用、回滚、在 Finder 中显示

架构方案：

- SwiftUI 构建主要界面，AppKit 处理文件选择和必要的系统能力。
- App 内置同版本 `skm` 可执行文件。
- 通过版本化 JSON/JSON Lines 协议调用 CLI。
- CLI 负责文件锁、Plan、Apply 和所有状态写入。
- App 只维护展示状态和用户交互，不直接改 Agent 目录。

第一版不引入常驻 daemon、Go-to-Swift 桥接或 XPC。只有后续需要后台定时同步、FSEvents 文件监控或特权操作时，再增加独立服务进程。

由于 App 需要访问用户主目录下多个隐藏目录，首版更适合签名、公证后通过 DMG 和 Homebrew 分发，不优先以严格沙盒的 Mac App Store 为目标。

## 14. JSON 接口要求

为了让 macOS App 和第三方工具稳定调用，CLI JSON 输出应从 MVP 开始设计：

- 顶层包含 `schemaVersion`、`command`、`success` 和 `data/error`。
- 错误包含稳定的机器可读 `code`，不能只返回英文字符串。
- 长操作使用 JSON Lines 输出进度事件。
- Text 输出和 JSON 输出共享领域结果，不分别实现业务逻辑。
- App 启动时检查 CLI 协议版本与二进制版本。

## 15. 实施阶段

### 阶段 0：规范冻结

- 确定作用域语义和覆盖规则
- 确定标签格式、默认值和筛选语义
- 定义配置文件 Schema
- 定义 Skill ID、Source 和 Snapshot
- 定义 Adapter 接口
- 定义 JSON 输出与退出码

### 阶段 1：CLI 本地 MVP

- `init/add/list/show/validate`
- Claude Code、Codex 检测
- 标签添加、编辑、筛选和批量选择
- `plan/apply/link/unlink/status/doctor`
- 软链接和复制模式
- 冲突保护和状态记录

### 阶段 2：Git 与可复现安装

- Source 管理
- Git 缓存与不可变 Snapshot
- `lock.yaml`
- `source update/sync`
- 变更预览和回滚

### 阶段 3：发布级加固

- 并发锁和中断恢复
- Shell Completion
- JSON 协议稳定化
- Homebrew Formula
- 多版本 Agent 兼容测试

### 阶段 4：macOS App

- Skill 浏览和搜索
- Project 与 Agent 管理
- 同步变更预览
- 冲突解决和回滚
- 签名、公证、DMG 发布

### 阶段 5：扩展能力

- 其他 Agent Adapter
- 团队策略与可信 Source
- Skill 签名和安全扫描
- 可选后台同步
- Skill 编辑器与模板创建

以一名熟悉 Go 和 Swift 的开发者估算，CLI 可用 MVP 约需 2 到 3 周；Git 同步和发布级加固再需 1 到 2 周；macOS App 第一版约需 4 到 6 周。实际周期取决于各 Agent 对软链接的兼容验证结果。

## 16. MVP 验收标准

- 能从本地目录导入一个合法 Skill。
- 能验证并展示 Skill 元数据和来源。
- 能把 Skill 部署到 Claude Code 和 Codex 的用户级或项目级目录。
- 未指定标签时使用 `general`，指定标签时不附加默认标签。
- 能通过一个或多个标签查询 Skill 并生成批量部署计划。
- 重复执行 `apply` 不产生额外变更。
- 不覆盖或删除用户已有的未管理文件。
- 能识别同名覆盖、未知冲突和损坏链接。
- 能从 Git Source 更新并锁定准确版本。
- 更新失败时保留原有可用部署。
- `--json` 足以支持后续 macOS App，不需要解析终端文本。

## 17. 非目标

以下内容不进入 CLI MVP：

- 在线 Skill 商店
- Skill 自动生成和 AI 编辑
- Rules、Commands、Subagents、MCP 的统一转换
- 远程代码执行
- 多用户权限系统
- 云端账号和同步服务
- 后台常驻进程
- 完整 Bubble Tea TUI

## 18. 开工前待确认决策

建议采用以下默认决策：

1. `Global` 表示团队/公共基线，`Personal` 表示私有覆盖，`Project` 优先级最高。
2. 首批只支持 Claude Code 和 Codex。
3. `SKILL.md` 保持原始格式，不做 Rules 自动转换。
4. 部署采用 `auto` 模式，优先软链接，必要时受控复制。
5. 项目使用 `.skm/project.yaml + lock.yaml` 实现可复现安装。
6. 标签为一等管理元数据；默认标签是 `general`，显式标签替代默认标签。
7. CLI 先实现普通命令，不实现完整 Bubble Tea TUI。
8. macOS App 调用内置 CLI Core，首版不使用 daemon。
9. App 首发采用 DMG/Homebrew，不以 Mac App Store 为目标。

其中第 1 项作用域定义会直接影响配置格式、覆盖规则和后续 UI，需要在进入实现前优先确认。

## 19. 参考资料

- [Agent Skills Specification](https://agentskills.io/specification)
- [Claude Code Skills](https://code.claude.com/docs/en/slash-commands)
- [Codex Customization and Skills](https://developers.openai.com/codex/concepts/customization#skills)

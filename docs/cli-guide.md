# skm CLI 使用指南

本文对应 skm `0.2.x` 和持久化 schema v2。

## 1. 核心概念

```text
Library     用户拥有的 Skill 集合
Activation  Library Skill 对 Agent 的启用状态
Project     项目 require 依赖或 vendor 副本
```

重要区别：

- `add` 只加入个人 Library，不自动启用。
- `enable/disable` 控制用户级 Agent 链接，不删除 Library 内容。
- `project require` 引用可从 Git 恢复的锁定依赖。
- `project vendor` 复制成项目独立维护版本，个人原版保留。
- 标签属于个人 Library Skill。

## 2. 环境与安装

Git 是可选依赖，仅 Git Source 和项目 Git 依赖恢复需要。Homebrew 和 curl 使用
预编译文件，不需要 Go。

### 2.1 Homebrew

```bash
brew install zzzzzyijie/tap/skm
skm version
```

Homebrew 会把 `skm` 安装到其已配置的可执行目录，安装完成后可以直接使用。

### 2.2 curl

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | sh
skm version
```

安装器自动识别 macOS/Linux 和 amd64/arm64，下载 Release 压缩包并使用
`checksums.txt` 验证 SHA-256。它优先选择当前 `PATH` 中可写的目录，因此通常安装后
可以直接执行。系统没有可写 PATH 目录时，会回退到 `~/.local/bin` 并打印配置提示。

指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | \
  sh -s -- --version v0.2.0
```

指定安装目录：

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | \
  sh -s -- --install-dir "$HOME/.local/bin"
```

出于安全考虑，也可以先下载并检查脚本，再执行：

```bash
curl -fsSLO https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh
less install.sh
sh install.sh
```

### 2.3 从源码构建

开发项目时要求 Go 1.25+：

```bash
go version

go build -trimpath -o ./bin/skm ./cmd/skm
./bin/skm version
```

开发版覆盖测试不应直接使用真实的 `~/.skm` 或 Agent 目录。完整的隔离构建、`--home`、
`--user-home`、`--project` 测试与发布步骤见
[隔离开发与发布流程](development-release-workflow.md)。

### 2.4 首次初始化

```bash
skm init
```

个人 Library 初始化后即可使用 `add` 和 `enable`，不需要初始化任何项目。
`project require`、`project vendor` 和 `project apply` 会在需要时自动创建项目状态。

## 3. Skill 格式

参数必须是包含 `SKILL.md` 的目录：

```text
code-review/
├── SKILL.md
├── scripts/
├── references/
└── assets/
```

最小 `SKILL.md`：

```markdown
---
name: code-review
description: Review code changes for correctness and risk.
---

Follow the repository review checklist.
```

先验证：

```bash
skm validate "$HOME/my-skills/code-review"
```

## 4. 个人 Library

### 4.1 添加本地 Skill

```bash
skm add "$HOME/my-skills/code-review" \
  --tag development \
  --tag review
```

默认 ID：

```text
local/code-review
```

自定义逻辑 Source 名：

```bash
skm add ./code-review --source my-skills
```

对应 ID 为 `my-skills/code-review`。

### 4.2 查看

```bash
skm list
skm list --tag development
skm list --tag development --tag review
skm show local/code-review
```

短名称只有在 Library 中唯一时可用：

```bash
skm show code-review
```

如果 `team/code-review` 和 `local/code-review` 同时存在，必须使用完整 ID。

### 4.3 移除

```bash
skm remove local/code-review
```

如果 Skill 仍处于用户 Activation，先运行：

```bash
skm disable local/code-review
```

`disable` 只处理用户级 Activation，不会移除项目级 Activation。如果被项目
require，针对报错中显示的项目根目录运行：

```bash
skm --project /path/to/project project remove team/code-review
```

`remove` 从 Library 删除条目后，会检查剩余 Library、Activation、Deployment 和当前
项目依赖。没有任何引用时同时删除 `~/.skm/objects/<hash>/<name>`；相同快照仍被其他
Skill 或项目引用时只删除 Library 条目并保留快照。

Git Source binding 不会被 `remove` 删除。只移除 Git-backed Skill 后再次运行
`source update` 或 `sync`，该 Skill 可能重新导入；需要停止同步时应同时调整或删除
对应 Source binding。

Source 更新产生的旧版本等孤立快照可以统一清理。先使用 dry-run 查看候选路径和空间：

```bash
skm prune --dry-run
skm prune
```

`prune` 只处理标准 `objects/<64位hash>/<name>` 目录，并保留 Library、固定
Activation、Deployment 或当前项目依赖仍引用的快照。其他项目中尚未应用的 Git
依赖不属于当前机器的活动引用；需要时可由项目锁文件和 Git Source 重新恢复。

## 5. 标签

标签只管理个人 Library：

```bash
skm tag list
skm tag add local/code-review backend security
skm tag remove local/code-review security
skm tag rename backend server
```

标签格式为小写字母、数字和连字符，长度 1 到 32。多个查询标签为 AND 关系。

默认配置：

```yaml
version: 2
defaults:
  tags:
    - general
  agents:
    - claude
    - codex
  linkMode: auto
```

显式标签替代默认的 `general`。

## 6. 用户 Activation

### 6.1 启用

```bash
skm enable local/code-review --agent claude,codex
```

按标签批量启用：

```bash
skm enable --tag development --tag review --agent codex
```

默认 Agent 来自 `config.yaml`。部署目标：

| Agent | 用户目标 |
| --- | --- |
| Claude Code | `~/.claude/skills/<name>` |
| Codex | `~/.agents/skills/<name>` |

### 6.2 部署模式

```bash
skm enable local/code-review --mode auto
skm enable local/code-review --mode symlink
skm enable local/code-review --mode copy
```

当前 `auto` 解析为 `symlink`。软链接是推荐方式；`copy` 用于不支持链接的环境，
skm 会用内容 hash 检测外部修改。

只查看计划：

```bash
skm enable local/code-review --dry-run
```

### 6.3 禁用

```bash
skm disable local/code-review
skm disable local/code-review --agent codex
skm disable --tag development
```

禁用删除 skm 管理的链接，但保留 Library Skill。若目标在部署后被外部修改，默认拒绝
删除；确认后可以：

```bash
skm disable local/code-review --force
```

### 6.4 计划和状态

```bash
skm plan
skm apply
skm status
```

常见状态：`create`、`unchanged`、`replace-managed`、`conflict-unmanaged`、
`broken`。

## 7. Git Source

### 7.1 添加

扫描整个仓库：

```bash
skm source add git@github.com:example/team-skills.git \
  --name team \
  --ref main \
  --tag team
```

只绑定指定目录：

```bash
skm source add https://github.com/example/skills.git \
  --name review-pack \
  --path skills/code-review \
  --tag review
```

`--path` 是仓库内相对目录，可重复传入。

本地 Git 仓库路径也可作为个人 Source：

```bash
skm source add "$HOME/src/my-skills" --name local-git
```

但本地路径不能作为团队可恢复的 `project require` 来源；需要 vendor 或发布到远程。

### 7.2 将个人 Library Skill 绑定到远程 Git

skm 的 Library 保存内容快照，`~/.skm/objects/` 不是 Skill 源码工作目录。要长期维护
和同步个人 Skill，应准备一个独立的 Git 工作目录，再将远程仓库作为 Source 导入。

已有 `local/code-review` 时，优先把原始 Skill 目录放入新的 Git 工作目录。如果原始
目录已经不存在，可运行 `skm show local/code-review` 查看当前快照的 `Path`，将该
目录复制到 Git 工作目录后再维护；不要直接编辑 `~/.skm/objects/` 中的快照。

假设源码布局如下：

```text
~/my-skills/
└── skills/
    └── code-review/
        └── SKILL.md
```

对于新建的空远程仓库，先发布本地源码：

```bash
cd "$HOME/my-skills"
git init -b main
git remote add origin git@github.com:your-name/my-skills.git
git add .
git commit -m "add personal skills"
git push -u origin main
```

如果远程仓库已经有内容，应先 `git clone`，再把 Skill 放入 clone 得到的工作目录中
提交，避免覆盖已有历史。

然后绑定远程并导入个人 Library：

```bash
skm source add git@github.com:your-name/my-skills.git \
  --name personal \
  --ref main \
  --path skills/code-review \
  --tag personal

skm source list
skm show personal/code-review
```

这里的 `personal` 是 Source 名，导入后的完整 ID 是 `personal/code-review`。
`source add` 不会把原有的 `local/code-review` 原地改名或绑定；两个条目会暂时同时
保留。若原来的本地版本已经启用，应显式切换，避免同一 Agent 出现同名冲突：

```bash
skm disable local/code-review
skm enable personal/code-review --agent claude,codex

# 确认 Git 版本工作正常后，可选择移除旧快照
skm remove local/code-review
```

后续始终在 `$HOME/my-skills` 工作目录修改、提交并推送，然后更新 skm：

```bash
cd "$HOME/my-skills"
git add skills/code-review
git commit -m "update code-review skill"
git push

skm sync --source personal
```

`sync` 会拉取新快照并刷新已启用的 Git 版本。只想更新 Library、暂不调整部署时使用：

```bash
skm sync --source personal --no-apply
```

当需要在团队项目中锁定这个远程 Skill 时，可以运行：

```bash
skm project require personal/code-review --agent claude,codex
```

### 7.3 更新和移除绑定

```bash
skm source list
skm source update team
skm source update
skm source remove team
```

移除 Source 绑定会同时删除 `~/.skm/sources/<name>` 中的 Git checkout，但不删除已导入
Library 的 Skill 和不可变快照。如果绑定已经不存在但 checkout 仍然存在，同一命令会将
该孤立 checkout 清理掉；两者都不存在时才报告 Source not found。

更新 Source 并刷新当前 Activation：

```bash
skm sync
skm sync --source team
skm sync --no-apply
```

项目 require 锁定版本不会随 `sync` 漂移。要升级项目依赖，在 Source 更新后重新运行
`skm project require <id>`。

## 8. Project Require

Require 适用于“团队项目消费共享 Skill，但不在本仓库修改它”。

```bash
cd "$HOME/Projects/shop-api"
skm project require team/code-review --agent claude,codex
```

项目记录：

```yaml
version: 2
dependencies:
  - id: team/code-review
    name: code-review
    source: team
    url: git@github.com:example/team-skills.git
    ref: main
    sourcePath: skills/code-review
    revision: <commit>
    hash: <content-hash>
    tags: [team, review]
    agents: [claude, codex]
    mode: auto
```

默认立即应用。只写 Manifest：

```bash
skm project require team/code-review --no-apply
```

团队成员恢复：

```bash
git clone <project-url>
cd <project>
skm project apply
```

恢复过程按锁定 revision 获取不可变快照，不要求成员事先执行 `source add`。

如果用户 Activation 已在目标 Agent 提供相同 ID 和 hash，输出：

```text
Satisfied by user activation: team/code-review:codex
```

此时不会创建重复项目链接。

同名但 ID 或版本不同会直接报冲突。skm 不假设 Claude Code 和 Codex 具有相同的
覆盖规则。

## 9. Project Vendor

Vendor 适用于“项目要修改并独立维护这个 Skill”。

```bash
skm project vendor local/release-check --agent claude,codex
```

项目内容：

```text
.skm/
├── project.yaml
├── lock.yaml
└── skills/
    └── release-check/
        └── SKILL.md
```

个人 `local/release-check` 保留；项目副本 ID 是 `project/release-check`。之后可以直接
修改 `.skm/skills/release-check/`，再运行：

```bash
skm project apply
```

skm 会重新验证并刷新项目 hash。

Vendor 时默认继承个人 Library 标签，也可覆盖：

```bash
skm project vendor local/release-check --tag release --tag project
```

如果同名个人 Skill 正在目标 Agent 启用，先禁用它，避免 Agent 同时发现两个同名
版本：

```bash
skm disable local/release-check
skm project vendor local/release-check
```

## 10. 项目管理

```bash
skm project list
skm project apply
skm project remove team/code-review
skm project remove project/release-check
```

项目操作不要求预先运行 `skm project init`。第一次执行 `project require`、
`project vendor` 或 `project apply` 时，skm 会按需创建 `.skm/` 项目状态。

`skm project init` 只是可选的项目根标记：当目录没有 `.git`，但希望从其子目录执行
skm 并自动识别同一个项目根时，可以提前创建空的 `.skm`：

```bash
skm project init
```

`skm init --with-project` 等价于同时初始化个人 Library 和这个可选项目根标记，普通
个人使用和已有 Git 项目均不需要它。

删除 vendored Skill 会删除 `.skm/skills/<name>` 和对应的 skm 管理链接，个人 Library
原版不受影响。

指定其他项目：

```bash
skm --project "$HOME/Projects/shop-api" project list
skm --project "$HOME/Projects/shop-api" project apply
```

未指定时，skm 从当前目录向上查找最近的 `.git` 或 `.skm`。

应提交：

```text
.skm/project.yaml
.skm/lock.yaml
.skm/skills/            # 若有 vendor
```

不应提交由 skm 生成的 `.claude/skills/<name>`、`.agents/skills/<name>` 软链接。

## 11. 冲突和修复

### `conflict-unmanaged`

目标已存在但不属于 skm，或部署后被外部修改。skm 不自动覆盖。先检查目标并决定
迁移或删除，再重新运行命令。

### 同名 Skill

Library 同名时使用完整 ID。两个同名 Skill 不能同时启用到同一个 Agent。项目与用户
同名但版本不同也会失败；先 `disable` 用户版本或调整项目依赖。

### Git 认证

```bash
git ls-remote <git-url>
```

使用 SSH Agent 或 Credential Helper。包含内嵌用户名/Token 的 URL 会被拒绝。

### 检查环境

```bash
skm doctor
```

没有配置 Git Source 时，缺少 Git 只显示为可选；存在 Git Source 时才是错误。

## 12. JSON 和自动化

全局 `--json` 返回稳定 envelope：

```bash
skm --json list --tag development
skm --json status
skm --json project list
```

结构：

```json
{
  "schemaVersion": 1,
  "command": "list",
  "success": true,
  "data": []
}
```

`--home` 或 `SKM_HOME` 可隔离 Library。若 CI 会执行 `enable` 或其他部署命令，
还应同时提供隔离的 `HOME`，因为 Agent 用户目录仍从 `HOME` 解析：

```bash
HOME="$PWD/.tmp/user-home" \
SKM_HOME="$PWD/.tmp/user-home/.skm" \
skm init
```

## 13. v1 迁移

schema v1 会在读取时映射：

```text
global/personal Catalog Skill -> library
project Catalog Skill         -> project
global/personal Installation  -> user Activation
project Installation          -> project Activation
```

下一次写入会保存为 v2。旧 `link/unlink` 仍是 `enable/disable` 的命令别名，但
`--scope` 已移除；项目工作流应改用 `project require` 或 `project vendor`。

## 14. 命令速查

```text
skm init
skm add <skill-path> [--source ...] [--tag ...]
skm list [--tag ...]
skm show <skill>
skm validate <skill-path>
skm remove <skill>

skm enable [skill...] [--tag ...] [--agent ...] [--mode ...] [--dry-run]
skm disable [skill...] [--tag ...] [--agent ...] [--force]
skm plan
skm apply [--digest ...]
skm status
skm doctor

skm source add <git-url> --name <name> [--ref ...] [--path ...] [--tag ...]
skm source list
skm source update [name...]
skm source remove <name>
skm sync [--source ...] [--no-apply]

skm tag list
skm tag add <skill> <tag...>
skm tag remove <skill> <tag...>
skm tag rename <old> <new>

skm project list
skm project require <skill> [--agent ...] [--mode ...] [--no-apply]
skm project vendor <skill> [--agent ...] [--mode ...] [--tag ...] [--no-apply]
skm project remove <skill> [--force]
skm project apply [--force]

# 可选：标记没有 .git 的项目根
skm project init
skm init --with-project
```

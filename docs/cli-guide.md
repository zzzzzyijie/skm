# skm CLI 完整教程

本文介绍如何安装 `skm`、管理本地与 Git Skills、使用标签，以及向 Claude Code 和 Codex 部署 Skills。

## 最短使用流程

下面是一套完整的本地 Skill 示例：

```bash
# 1. 添加本地目录；该目录内必须存在 SKILL.md
skm add "$HOME/my-skills/code-review" --scope personal

# 2. 查看添加后生成的完整 ID，例如 local/code-review
skm list

# 3. 部署到指定项目
skm --project "$HOME/Projects/shop-api" \
  link local/code-review --scope project --agent claude,codex

# 4. 检查结果
skm --project "$HOME/Projects/shop-api" status
```

三个容易混淆的参数：

| 写法 | 实际含义 |
| --- | --- |
| `skm add <skill-path>` | `<skill-path>` 是包含 `SKILL.md` 的本地目录 |
| `skm link <skill>` | `<skill>` 是 Catalog 中的名称或完整 ID，不是路径 |
| `--scope project` | `project` 是固定作用域值；具体路径使用 `--project <path>` |

如果来源是 Git URL，应使用 `skm source add`，详见第 5.6 节。

## 1. 环境要求

- macOS
- Go 1.25 或更高版本
- Git
- zsh、bash 或其他常见 Shell

检查环境：

```bash
go version
git --version
```

## 2. 安装

### 2.1 为什么之前使用 `bin/skm`

`bin/skm` 是放在仓库内的本地构建产物，不是源码入口，并且 `bin/` 已被 `.gitignore` 忽略。`bin/skm`、`./bin/skm` 和 `skm` 的差别只在于 Shell 如何查找可执行文件：

- `./bin/skm` 是仓库内的相对路径，只能在路径正确时使用。
- `skm` 不包含路径，Shell 会按 `$PATH` 从前到后查找名为 `skm` 的可执行文件。
- 把构建结果安装到 `$PATH` 中的目录后，就能在任意目录直接执行 `skm ...`。

可以查看当前搜索路径：

```bash
print -r -- $PATH
```

### 2.2 推荐安装方式

在 skm 仓库根目录构建到 `~/.local/bin`：

```bash
mkdir -p ~/.local/bin
go build -trimpath -o ~/.local/bin/skm ./cmd/skm
export PATH="$HOME/.local/bin:$PATH"
rehash
```

上面的 `export` 只影响当前终端。要让以后打开的 zsh 终端也能找到 `skm`，请在 `~/.zshrc` 中加入以下一行：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

然后重新加载配置：

```bash
source ~/.zshrc
rehash
```

确认安装：

```bash
command -v skm
skm version
```

预期结果类似：

```text
/Users/your-name/.local/bin/skm
0.1.0
```

安装完成后，可以在任意目录直接执行 `skm`，不需要再写 `bin/skm` 或 `./skm`。

### 2.3 使用 Go 安装目录

也可以执行：

```bash
go install ./cmd/skm
```

如果设置了 `GOBIN`，命令会安装到 `$(go env GOBIN)`；否则默认安装到 `$(go env GOPATH)/bin`。可以这样确认实际目录：

```bash
go env GOBIN GOPATH
```

如果终端找不到 `skm`，需要把实际安装目录加入 `~/.zshrc`。未设置 `GOBIN` 时可使用：

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

然后重新加载：

```bash
source ~/.zshrc
```

### 2.4 更新

进入仓库并重新构建即可覆盖旧版本：

```bash
git pull
go test ./...
go build -trimpath -o ~/.local/bin/skm ./cmd/skm
skm version
```

### 2.5 卸载

删除 `~/.local/bin/skm` 即可卸载命令。该操作不会删除 `~/.skm` 中的 Catalog、Git Source、快照和部署状态。

## 3. 初始化

初始化用户级数据目录：

```bash
skm init
```

同时初始化当前项目的 `.skm/`：

```bash
skm init --with-project
```

默认用户数据目录为以下结构；`catalog.yaml`、`sources.yaml` 和 `state/state.yaml` 会在相关功能首次写入时创建：

```text
~/.skm/
├── config.yaml
├── catalog.yaml
├── sources.yaml
├── objects/
├── sources/
├── state/
│   └── state.yaml
└── locks/
```

项目级数据位于：

```text
<project>/.skm/
├── project.yaml
├── lock.yaml
└── skills/
```

## 4. 创建一个 Skill

一个最小 Skill 是包含 `SKILL.md` 的目录：

```text
skills/code-review/
└── SKILL.md
```

`SKILL.md` 示例：

```markdown
---
name: code-review
description: Review code changes for correctness, regressions, and missing tests.
---

# Code Review

1. Inspect the changed files.
2. Identify correctness and regression risks.
3. Report findings with file references.
4. Check whether tests cover the changed behavior.
```

`name` 必须使用小写字母、数字和连字符，最长 64 个字符。`name` 和 `description` 都是必填字段。

Skill 目录还可以包含：

```text
code-review/
├── SKILL.md
├── scripts/
├── references/
└── assets/
```

`skm` 只管理这些文件，不会主动执行 Skill 中的脚本。

## 5. 添加 Skills

Skills 有两种来源，使用不同命令：

| 来源 | 命令 | 参数含义 |
| --- | --- | --- |
| 本地目录 | `skm add <skill-path>` | `<skill-path>` 是本机目录 |
| Git 仓库 | `skm source add <git-url>` | `<git-url>` 是 Git HTTPS 或 SSH 地址 |

`skm add` 不能接收 GitHub 页面或 Git URL，`skm source add` 也不能接收普通本地 Skill 目录。

### 5.1 `<skill-path>` 应该填写什么

`<skill-path>` 必须是包含 `SKILL.md` 的本地目录，而不是 `SKILL.md` 文件本身。例如：

```text
/Users/alice/my-skills/
└── code-review/
    ├── SKILL.md
    └── references/
```

以下写法都可以：

```bash
# 当前目录的相对路径
skm add ./my-skills/code-review

# 绝对路径
skm add /Users/alice/my-skills/code-review

# 使用 HOME，适合教程和脚本
skm add "$HOME/my-skills/code-review"

# 路径包含空格时必须加引号
skm add "$HOME/My Skills/code-review"
```

不要把文件路径传给 `add`：

```text
错误：skm add ./my-skills/code-review/SKILL.md
正确：skm add ./my-skills/code-review
```

建议先验证，再添加：

```bash
skm validate "$HOME/my-skills/code-review"
skm add "$HOME/my-skills/code-review" --scope personal
```

### 5.2 添加个人 Skill

个人 Skill 默认使用 `personal` Catalog Scope，适合自己的通用工作流：

```bash
skm add "$HOME/my-skills/code-review" \
  --scope personal \
  --tag development \
  --tag review
```

未指定 `--source` 时，生成的完整 ID 是：

```text
local/code-review
```

### 5.3 添加当前项目专属 Skill

先进入项目，再使用 `--scope project`：

```bash
cd "$HOME/Projects/shop-api"
skm add ./skills/release-check \
  --scope project \
  --tag release
```

也可以不切换目录，显式指定项目路径：

```bash
skm --project "$HOME/Projects/shop-api" \
  add "$HOME/Projects/shop-api/skills/release-check" \
  --scope project \
  --tag release
```

项目 Skill 的默认 ID 是 `project/release-check`。内容会复制到项目的 `.skm/skills/release-check/`，并记录到 `.skm/project.yaml`。

### 5.4 添加全局基线 Skill

`global` 表示 skm Catalog 中的低优先级公共基线，不表示写入 macOS 的系统目录：

```bash
skm add "$HOME/team-skills/security-review" \
  --scope global \
  --source team-local \
  --tag security
```

完整 ID 是 `team-local/security-review`。个人或项目中存在同名 Skill 时，会按作用域优先级覆盖它。

### 5.5 自定义 Source 名称与 Skill ID

Skill 完整 ID 的格式是：

```text
<source>/<SKILL.md 中的 name>
```

例如：

```bash
skm add "$HOME/my-skills/code-review" \
  --scope personal \
  --source my-skills
```

对应完整 ID：

```text
my-skills/code-review
```

`--source` 是逻辑来源名称，不是文件路径。未指定时，`global` 和 `personal` 使用 `local`，`project` 使用 `project`。

### 5.6 从 Git 仓库添加 Skills

扫描远程仓库内的所有 `SKILL.md`：

```bash
skm source add https://github.com/example/team-skills.git \
  --name team \
  --ref main \
  --scope global \
  --tag team
```

只添加仓库中的指定 Skill：

```bash
skm source add https://github.com/example/team-skills.git \
  --name team-review \
  --ref main \
  --path skills/code-review \
  --scope global \
  --tag review
```

这里的 `--path` 是 Git 仓库内部的相对路径，不是本机路径。Git Source 只支持 `global` 或 `personal` Catalog Scope；需要用于某个项目时，先导入 Catalog，再用 `skm link ... --scope project` 部署。

### 5.7 确认添加结果

```bash
skm list
skm list --scope personal
skm show local/code-review
```

`add` 会把内容复制到 skm 管理的位置，之后修改原始目录不会自动更新已导入版本。修改后应重新执行相同的 `skm add`；Git Source 则使用 `skm source update` 或 `skm sync`。

## 6. 理解作用域

skm 中有两类作用域需要区分。

### 6.1 Catalog Scope：Skill 存在哪里、谁覆盖谁

Catalog Scope 表示 Skill 来自哪一层：

| Scope | 典型用途 | 管理位置 | 默认 Source | 优先级 |
| --- | --- | --- | --- | --- |
| `global` | 团队或公共基线 | `~/.skm/objects/`、`catalog.yaml` | `local` | 低 |
| `personal` | 个人通用 Skill、个人覆盖 | `~/.skm/objects/`、`catalog.yaml` | `local` | 中 |
| `project` | 当前仓库专属 Skill | `<project>/.skm/skills/`、`project.yaml` | `project` | 高 |

同名 Skill 的解析顺序：

```text
project > personal > global
```

同一层出现多个同名 Skill 时，必须使用完整 ID：

```bash
skm show team/code-review
skm link team/code-review --scope project
```

### 6.2 Deployment Scope：Skill 部署到哪里

Deployment Scope 表示 Skill 部署到哪里：

| Agent | 用户级部署 | 项目级部署 |
| --- | --- | --- |
| Claude Code | `~/.claude/skills/<name>` | `<project>/.claude/skills/<name>` |
| Codex | `~/.agents/skills/<name>` | `<project>/.agents/skills/<name>` |

部署命令中的三种 Scope：

| `skm link --scope` | 实际目标 | 适用场景 |
| --- | --- | --- |
| `global` | Agent 用户级目录 | 部署团队或公共基线 |
| `personal` | Agent 用户级目录 | 部署个人 Skill；与 Global 同名时优先 |
| `project` | 当前项目内的 Agent 目录 | 只让当前项目使用 |

`global` 和 `personal` 都写入当前用户的 Agent 目录，不会写入 `/usr/local`、`/Library` 等系统目录。两者的区别是逻辑优先级和状态记录。

Catalog Scope 和 Deployment Scope 是独立概念。例如，团队 Git Source 中的 Global Skill 可以部署到某个项目：

```bash
skm link team/code-review --scope project --agent claude,codex
```

### 6.3 Catalog Scope 与 Deployment Scope 组合示例

个人 Catalog 中的 Skill，只部署给当前项目：

```bash
skm add "$HOME/my-skills/code-review" --scope personal
skm --project "$HOME/Projects/shop-api" \
  link local/code-review --scope project --agent claude,codex
```

团队 Git Source 中的 Global Skill，部署到用户级目录供所有项目发现：

```bash
skm source add https://github.com/example/team-skills.git \
  --name team --scope global --path skills/code-review
skm link team/code-review --scope global --agent claude,codex
```

当前项目原创 Skill，只部署给当前项目：

```bash
cd "$HOME/Projects/shop-api"
skm add ./skills/release-check --scope project
skm link project/release-check --scope project --agent claude,codex
```

### 6.4 如何指定项目路径

`--scope project` 中的 `project` 是固定作用域值，不是路径。具体项目由以下两种方式确定：

```bash
# 方式一：进入项目后执行
cd "$HOME/Projects/shop-api"
skm link local/code-review --scope project

# 方式二：使用全局参数 --project
skm --project "$HOME/Projects/shop-api" \
  link local/code-review --scope project
```

未指定 `--project` 时，skm 从当前目录向上查找最近的 `.git` 或 `.skm`，将其作为项目根目录。

### 6.5 作用域选择建议

| 需求 | 添加时的 Catalog Scope | 部署时的 Deployment Scope |
| --- | --- | --- |
| 自己维护，供所有项目使用 | `personal` | `personal` |
| 自己维护，只给一个项目使用 | `personal` | `project` |
| Skill 本身属于当前仓库 | `project` | `project` |
| 团队公共基线，供所有项目使用 | `global` | `global` |
| 团队公共 Skill，只给一个项目使用 | `global` | `project` |

例如，个人 Skill 用户级部署：

```bash
skm add "$HOME/my-skills/code-review" --scope personal
skm link local/code-review --scope personal --agent claude,codex
```

项目同名 Skill 会优先于 Personal，Personal 又会优先于 Global。需要绕过自动优先级、明确选择某个版本时，始终使用完整 ID。

## 7. 查看 Catalog

列出全部 Skills：

```bash
skm list
```

按作用域过滤：

```bash
skm list --scope global
skm list --scope personal
skm list --scope project
```

查看详情：

```bash
skm show code-review
skm show team/code-review
```

## 8. 标签

### 8.1 默认标签

未指定标签时，Skill 自动使用 `general`：

```bash
skm add ./skills/code-review
skm list --tag general
```

显式指定标签后，不再自动添加 `general`：

```bash
skm add ./skills/code-review \
  --tag development \
  --tag code-review
```

如需更改默认标签，编辑 `~/.skm/config.yaml` 中的 `defaults.tags`。标签只能包含 1 到 32 个小写字母、数字或连字符：

```yaml
version: 1
defaults:
  tags:
    - general
  agents:
    - claude
    - codex
  linkMode: auto
```

### 8.2 标签筛选

```bash
skm list --tag development
skm list --tag development --tag code-review
```

多个 `--tag` 使用 AND 语义，Skill 必须同时包含全部指定标签。

### 8.3 标签管理

```bash
skm tag list
skm tag add code-review backend security
skm tag remove code-review security
skm tag rename backend server
```

删除最后一个标签时，skm 会恢复配置中的默认标签。

### 8.4 按标签批量部署

```bash
skm link \
  --tag development \
  --tag code-review \
  --scope project \
  --agent claude,codex
```

## 9. 部署 Skill

### 9.1 `<skill-name>` 应该填写什么

`skm link` 接收已经添加到 Catalog 的 Skill 名称或完整 ID，不接收本地路径和 Git URL。先用 `skm list` 查看可用值：

```bash
skm list
```

假设输出中有 `local/code-review`：

```bash
# 名称唯一时可以使用短名称
skm link code-review --scope project

# 推荐在脚本和团队文档中使用完整 ID
skm link local/code-review --scope project
```

同一优先级存在多个同名 Skill 时，短名称会产生歧义，必须使用 `team/code-review` 这样的完整 ID。

以下写法是错误的：

```text
错误：skm link ./my-skills/code-review --scope project
错误：skm link https://github.com/example/skills.git --scope project
正确：skm link local/code-review --scope project
```

### 9.2 部署到当前项目

```bash
cd "$HOME/Projects/shop-api"
skm link local/code-review --scope project --agent claude,codex
```

部署结果：

```text
$HOME/Projects/shop-api/.claude/skills/code-review
$HOME/Projects/shop-api/.agents/skills/code-review
```

如果省略 `--agent`，默认同时部署到 Claude Code 和 Codex：

```bash
skm link local/code-review --scope project
```

不切换当前目录时，使用 `--project` 指定项目路径：

```bash
skm --project "$HOME/Projects/shop-api" \
  link local/code-review --scope project --agent claude,codex
```

### 9.3 部署到用户级目录

```bash
# 个人级部署到 Claude Code
skm link local/code-review --scope personal --agent claude

# 个人级部署到 Codex
skm link local/code-review --scope personal --agent codex

# 全局基线部署到两个 Agent
skm link team/security-review --scope global --agent claude,codex
```

`global` 和 `personal` Deployment Scope 都使用 Agent 的用户级目录；同名目标中 Personal 优先于 Global。

### 9.4 一次部署多个 Skill 或按标签部署

一次指定多个 Skill：

```bash
skm link local/code-review team/security-review \
  --scope project \
  --agent claude,codex
```

不写 Skill 名称，按标签批量选择：

```bash
skm link --tag development --tag review \
  --scope project \
  --agent claude,codex
```

多个 `--tag` 是 AND 关系。命令必须至少提供一个 Skill 或一个 `--tag`。

### 9.5 部署模式

默认模式是 `auto`，当前会优先使用软链接：

```bash
skm link local/code-review --scope project --mode auto
```

明确使用软链接：

```bash
skm link local/code-review --scope project --mode symlink
```

需要真实目录副本时使用：

```bash
skm link local/code-review --scope project --mode copy
```

### 9.6 预览变更

在不修改文件和状态的情况下预览本次链接：

```bash
skm link local/code-review --scope project --dry-run
```

查看全部已记录部署的当前计划：

```bash
skm plan
```

常见状态：

| 状态 | 含义 |
| --- | --- |
| `create` | 目标不存在，将创建 |
| `unchanged` | 已经是期望状态 |
| `replace-managed` | skm 管理的旧版本将更新 |
| `broken` | skm 管理的目标丢失，将重建 |
| `conflict-unmanaged` | 目标不是 skm 管理的文件，拒绝覆盖 |

### 9.7 Apply 与 digest

执行当前完整计划：

```bash
skm apply
```

需要确保预览后状态没有变化时，可使用 Plan 输出中的 digest：

```bash
skm plan
skm apply --digest <plan-digest>
```

如果文件状态发生变化，digest 不一致，Apply 会停止。

## 10. 状态与解绑

检查部署状态：

```bash
skm status
skm doctor
```

从当前项目的所有 Agent 解绑：

```bash
skm unlink local/code-review --scope project
```

只从 Codex 解绑：

```bash
skm unlink local/code-review --scope project --agent codex
```

按标签批量解绑：

```bash
skm unlink --tag development --scope project
```

如果受管理的目标被手工修改，默认拒绝删除。确认不需要这些修改时才能使用：

```bash
skm unlink local/code-review --scope project --force
```

从 Catalog 删除 Skill 前，必须先解除所有部署：

```bash
skm unlink local/code-review --scope project
skm unlink local/code-review --scope personal
skm unlink local/code-review --scope global
skm remove local/code-review
```

只需要解除实际使用过的 Deployment Scope；未部署到某个 Scope 时可以跳过对应命令。

## 11. Git Source 自定义绑定

### 11.1 绑定整个仓库

```bash
skm source add git@github.com:example/team-skills.git \
  --name team \
  --ref main \
  --scope global \
  --tag team
```

未指定 `--path` 时，skm 会扫描仓库中的全部 `SKILL.md`。

### 11.2 绑定指定目录

```bash
skm source add https://github.com/example/skills.git \
  --name engineering \
  --ref v1.2.0 \
  --path skills/code-review \
  --path skills/release \
  --tag engineering
```

`--ref` 可以是 branch、tag 或 commit。`--path` 必须是仓库内的相对路径，不能逃逸仓库根目录。

### 11.3 私有仓库

推荐使用 SSH：

```bash
skm source add git@github.com:company/private-skills.git \
  --name company \
  --scope personal \
  --tag private
```

也可以使用系统 Git Credential Helper 管理 HTTPS 凭证。不要把 Token 写入 URL；skm 会拒绝包含内嵌凭证的 URL。

### 11.4 查看与更新 Source

```bash
skm source list
skm source update
skm source update team
```

更新流程会：

1. 克隆指定 ref 到临时目录。
2. 验证绑定范围内的所有 Skills。
3. 计算内容 hash。
4. 写入不可变快照。
5. 更新 Catalog 中的 revision 和 hash。

已部署目标不会因为单独执行 `source update` 而自动切换；可以先查看 Plan，再 Apply：

```bash
skm source update team
skm plan
skm apply
```

### 11.5 一步同步

更新全部 Source 并更新已部署目标：

```bash
skm sync
```

只更新指定 Source：

```bash
skm sync --source team
```

只更新 Git Source 和 Catalog，不修改 Agent 目标目录：

```bash
skm sync --source team --no-apply
```

注意：`--no-apply` 不是完整的网络 Dry Run。它仍会拉取 Git、验证并更新 Catalog，只是不执行部署 Plan。

### 11.6 移除 Source 绑定

```bash
skm source remove team
```

该命令停止后续同步，但保留已经导入的 Catalog 条目和不可变快照，避免破坏现有部署。需要删除某个 Skill 时，再使用 `skm remove <source/name>`。

## 12. 项目 Manifest 与 Lock

项目级 `link` 会维护：

```text
.skm/project.yaml
.skm/lock.yaml
```

`project.yaml` 记录：

- Skill ID
- 标签
- 目标 Agent
- 部署模式
- 项目原创 Skills

`lock.yaml` 记录：

- Source
- Git revision
- 内容 hash
- 标签

这两个文件中的项目 Skill 路径使用项目相对路径，可以提交 Git。Agent 目录中的软链接是本机部署状态，链接目标使用当前机器上的绝对路径，不应提交。

当前版本会记录 Manifest 和 Lock，但还没有仅凭这两个文件执行一键恢复的命令。在另一台机器上，需要先用 `skm source add` 绑定同一个 Git URL，并用 Lock 中的 revision 作为 `--ref`，再按照 `project.yaml` 中的依赖执行 `skm link ... --scope project`。项目内原创 Skill 则可直接从已提交的相对路径重新 `link`。

## 13. JSON 输出

全局 `--json` 可用于脚本和后续 macOS App：

```bash
skm --json version
skm --json list
skm --json show team/code-review
skm --json plan
skm --json source list
```

成功响应结构：

```json
{
  "schemaVersion": 1,
  "command": "version",
  "success": true,
  "data": {
    "version": "0.1.0"
  }
}
```

命令成功时退出码为 `0`，失败时为非零退出码，错误 JSON 写入标准错误输出。

## 14. 自定义数据目录和项目目录

临时覆盖 skm 用户数据目录：

```bash
skm --home /custom/skm-home list
```

或者设置环境变量：

```bash
export SKM_HOME=/custom/skm-home
skm list
```

从其他目录操作指定项目：

```bash
skm --project "$HOME/Projects/shop-api" list
skm --project "$HOME/Projects/shop-api" \
  link local/code-review --scope project
```

默认情况下，skm 从当前目录向上查找 `.git` 或 `.skm`，将最近的目录作为项目根目录。

## 15. Shell Completion

临时启用 zsh Completion：

```bash
source <(skm completion zsh)
```

生成其他 Shell 的 Completion：

```bash
skm completion bash
skm completion fish
skm completion powershell
```

## 16. 常见问题

### 16.1 `command not found: skm`

检查安装位置和 `$PATH`：

```bash
ls -l ~/.local/bin/skm
print -r -- $PATH
command -v skm
```

如果 `~/.local/bin` 不在 `$PATH` 中，在 `~/.zshrc` 添加：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

然后执行：

```bash
source ~/.zshrc
rehash
```

### 16.2 Skill 校验失败

```bash
skm validate /path/to/skill
```

重点检查：

- 文件名必须是 `SKILL.md`
- 文件必须以 YAML Frontmatter 开始
- `name`、`description` 必填
- `name` 只能包含小写字母、数字和连字符
- Skill 内的软链接不能使用绝对路径或逃逸 Skill 根目录

### 16.3 `conflict-unmanaged`

目标目录已经存在，但不是 skm 创建的。skm 不会自动覆盖。先检查目标：

```bash
skm plan
skm doctor
```

手工决定保留、迁移或删除原目标后，再重新执行 `skm link`。`--force` 只提供给 `unlink` 删除已经被 skm 记录、但后来被修改的目标，不会让 `link` 覆盖未知文件。

### 16.4 Git Source 更新失败

先直接检查 Git 认证：

```bash
git ls-remote <git-url>
```

然后检查绑定：

```bash
skm source list
skm source update <name>
```

`--path` 指向的目录必须存在并包含合法的 `SKILL.md`。

### 16.5 Agent 没有识别新 Skill

```bash
skm status
skm doctor
```

确认目标分别位于：

- Claude Code：`.claude/skills/` 或 `~/.claude/skills/`
- Codex：`.agents/skills/` 或 `~/.agents/skills/`

如果 Agent 会话已经启动，创建新会话或重启对应 Agent，让它重新发现 Skill。

## 17. 开发与测试

```bash
go test ./...
go test -race ./...
go vet ./...
go build -trimpath -o /tmp/skm ./cmd/skm
```

测试使用临时 HOME、临时项目和本地 Git 仓库，不会访问真实的 `~/.claude`、`~/.agents` 或 `~/.skm`。

## 18. 命令速查

### 18.1 完整命令形式

```text
skm init [--with-project]
skm add <skill-path> [--scope ...] [--source ...] [--tag ...]
skm list [--scope ...] [--tag ...]
skm show <skill>
skm validate <skill-path>
skm remove <skill>

skm link [skill...] [--tag ...] [--scope ...] [--agent ...] [--mode ...] [--dry-run]
skm unlink [skill...] [--tag ...] [--scope ...] [--agent ...] [--force]
skm plan
skm apply [--digest ...]
skm status
skm doctor

skm source add <git-url> --name <name> [--ref ...] [--path ...] [--scope ...] [--tag ...]
skm source list
skm source update [name...]
skm source remove <name>
skm sync [--source ...] [--no-apply]

skm tag list
skm tag add <skill> <tag...>
skm tag remove <skill> <tag...>
skm tag rename <old> <new>

skm completion <shell>
skm version
```

所有命令都可以附加全局参数：

```text
--home <skm-data-directory>
--project <project-root>
--json
--no-color
```

### 18.2 可直接参照的完整示例

初始化和检查：

```bash
skm version
skm init --with-project
skm doctor
```

验证并添加本地个人 Skill：

```bash
skm validate "$HOME/my-skills/code-review"
skm add "$HOME/my-skills/code-review" \
  --scope personal \
  --source local \
  --tag development \
  --tag review
skm show local/code-review
```

添加项目专属 Skill 并部署：

```bash
skm --project "$HOME/Projects/shop-api" \
  add "$HOME/Projects/shop-api/skills/release-check" \
  --scope project \
  --tag release

skm --project "$HOME/Projects/shop-api" \
  link project/release-check \
  --scope project \
  --agent claude,codex \
  --mode symlink
```

绑定、查看和更新 Git Source：

```bash
skm source add git@github.com:example/team-skills.git \
  --name team \
  --ref main \
  --path skills/code-review \
  --scope global \
  --tag team \
  --tag review
skm source list
skm source update team
```

把 Git Skill 部署到指定项目：

```bash
skm --project "$HOME/Projects/shop-api" \
  link team/code-review \
  --scope project \
  --agent claude,codex \
  --dry-run

skm --project "$HOME/Projects/shop-api" \
  link team/code-review \
  --scope project \
  --agent claude,codex
```

筛选、打标签和批量部署：

```bash
skm list --scope personal --tag development
skm tag add local/code-review backend
skm tag remove local/code-review backend
skm tag rename review code-review
skm link --tag development --tag code-review \
  --scope personal \
  --agent codex
```

同步、查看计划和应用：

```bash
skm sync --source team --no-apply
skm plan
skm apply --digest <plan-digest>
skm status
```

解绑和删除：

```bash
skm --project "$HOME/Projects/shop-api" \
  unlink team/code-review --scope project --agent claude,codex
skm source remove team
skm remove team/code-review
```

JSON 与 Shell Completion：

```bash
skm --json list --scope personal
skm --json status
source <(skm completion zsh)
```

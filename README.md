# skm (AI Skill Manager)

`skm` 管理个人 AI Agent Skill Library，并将选中的 Skill 安全启用到 Claude Code、
Codex 或具体项目。它将三个概念明确分开：

```text
Library     用户拥有和管理哪些 Skill
Activation  哪些 Skill 对哪些 Agent 启用
Project     项目引用或独立维护哪些 Skill
```

## 功能

- 个人 Library：添加、移除、查看和验证本地 Skill。
- Library 标签：分类、组合筛选和批量启用。
- Git Source：导入和更新个人或团队 Skill 仓库。
- Agent Activation：用受控软链接启用到 Claude Code 和 Codex。
- Project Require：记录 Git 来源、revision 和 hash，可在其他机器恢复。
- Project Vendor：复制成项目独立版本，个人原版继续保留。
- 安全部署：不覆盖未知目标，检测被修改的链接或副本。
- 可审计状态：YAML Manifest、Lock 和版本化 JSON 输出。

完整设计见 [Library、Activation 与 Project 设计](docs/library-activation-project-design.md)，
完整命令说明见 [CLI 使用指南](docs/cli-guide.md)。

## 安装

Git 仅在使用 Git Source 或恢复 `project require` 依赖时需要。Homebrew 和 curl
安装的是预编译文件，不要求本机安装 Go。

### Homebrew

```bash
brew install --cask zzzzzyijie/tap/skm
skm version
```

### curl

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | sh
skm version
```

安装器支持 macOS/Linux 的 Intel 和 ARM64，下载 GitHub Release 后会验证 SHA-256。
它优先安装到当前 `PATH` 中可写的目录；如果只能使用 `~/.local/bin`，会输出需要添加
的 PATH 配置。指定版本或目录：

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | \
  sh -s -- --version v0.2.0 --install-dir "$HOME/.local/bin"
```

首次使用再初始化个人 Library：

```bash
skm init
```

### 从源码构建

仅开发项目时需要 Go 1.25+：

```bash
go build -trimpath -o ./bin/skm ./cmd/skm
./bin/skm version
```

发布维护方式见 [发布指南](docs/releasing.md)。

## 快速开始

### 建立个人 Library

```bash
skm add "$HOME/my-skills/code-review" \
  --tag development \
  --tag review

skm list --tag development
skm show local/code-review
```

`add` 只加入 Library，不自动启用。

### 对 Agent 启用或禁用

```bash
skm enable local/code-review --agent claude,codex

# 也可以按个人 Library 标签批量启用
skm enable --tag development --agent codex

skm disable local/code-review --agent claude
```

默认使用软链接：

```text
~/.claude/skills/code-review -> ~/.skm/objects/<hash>/code-review
~/.agents/skills/code-review -> ~/.skm/objects/<hash>/code-review
```

禁用不会删除 Library 内容。

### 添加 Git Skill 库

```bash
skm source add git@github.com:example/team-skills.git \
  --name team \
  --ref main \
  --path skills/code-review \
  --tag team

skm source update team
skm sync
```

Git 凭证由系统 Git、SSH Agent 或 Credential Helper 管理。不要把 Token 写入 URL。
skm 不会自动执行 `git init`、提交或配置远程；纯本地 Library 不需要 Git。

将个人 Skill 发布并绑定到远程 Git：

```bash
cd "$HOME/my-skills"
git init -b main
git remote add origin git@github.com:your-name/my-skills.git
git add .
git commit -m "add personal skills"
git push -u origin main

skm source add git@github.com:your-name/my-skills.git \
  --name personal \
  --ref main \
  --path skills/code-review \
  --tag personal
```

导入后 ID 为 `personal/code-review`。这不会原地修改已有的 `local/code-review`；
确认新 Source 版本后，可先禁用本地版本，再启用 Git 版本。完整迁移与更新流程见
[CLI 使用指南](docs/cli-guide.md#72-将个人-library-skill-绑定到远程-git)。

## 项目使用

无需提前运行 `skm project init`。个人 Skill 经 `enable` 后可供所有项目使用；
`project require`、`project vendor` 和 `project apply` 会在需要时自动创建 `.skm/`
项目状态。

### Require：项目引用团队 Skill

适用于团队消费一个 Git-backed Skill，但不在当前仓库修改它：

```bash
cd "$HOME/Projects/shop-api"
skm project require team/code-review --agent claude,codex
```

这会提交可移植的：

```text
.skm/project.yaml
.skm/lock.yaml
```

其他成员 clone 项目后执行：

```bash
skm project apply
```

如果相同 `ID + hash` 已由用户 Activation 提供，项目不会创建重复链接，
状态显示为 `satisfied-by-user`。本地-only Skill 没有团队可恢复来源，不能 require；
应先发布到 Git Source，或改用 vendor。

### Vendor：项目独立维护 Skill

适用于项目需要修改并随仓库维护 Skill：

```bash
skm project vendor local/release-check --agent claude,codex
```

内容复制到：

```text
<project>/.skm/skills/release-check/
```

个人 Library 中的 `local/release-check` 保留不变。项目副本成为
`project/release-check`，之后独立演化，不进行隐式双向同步。

项目 Agent 目录里的链接是本机生成状态，不应提交 Git；应提交 `.skm/` 中的
Manifest、Lock 和 vendored Skill 内容。

## 标签

标签属于个人 Library Skill：

```bash
skm tag list
skm tag add local/code-review backend security
skm tag remove local/code-review security
skm tag rename backend server

skm list --tag development --tag review
skm enable --tag development --tag review --agent claude
```

多个 `--tag` 使用 AND 语义。未指定标签时使用默认的 `general`。

## 主要命令

```text
Library:
  init, add, list, show, validate, remove
  source add|list|update|remove, sync
  tag list|add|remove|rename

Activation:
  enable, disable, plan, apply, status, doctor

Project:
  project list|require|vendor|remove|apply

Optional:
  project init  # 仅用于标记没有 .git 的项目根
```

`link` 和 `unlink` 暂时作为 `enable` 和 `disable` 的兼容别名，但不再支持旧的
`--scope` 参数。

## 开发与验证

```bash
GOCACHE=/tmp/skm-go-cache go test ./...
GOCACHE=/tmp/skm-go-cache go test -race ./...
GOCACHE=/tmp/skm-go-cache go vet ./...
sh scripts/install_test.sh
```

测试使用临时 HOME、项目目录和本地 Git 仓库，不访问真实 Agent 配置目录。
发布配置的本地验证方式见 [发布指南](docs/releasing.md#4-本地检查发布配置)。

## License

[MIT License](LICENSE)

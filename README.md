# skm (AI Skill Manager)

`skm` 管理个人 AI Agent Skill Library，并将选中的 Skill 安全启用到 Claude Code、
Codex 或具体项目。它将三个概念明确分开：

```text
Library     用户拥有和管理哪些 Skill
Prompt      用户保存、填写变量和复用哪些 Prompt 模板
Activation  哪些 Skill 对哪些 Agent 启用
Project     项目引用或独立维护哪些 Skill
```

## 功能

- 个人 Library：添加、查看、验证、编辑和移除本地 Skill。
- Library 标签：分类、组合筛选和批量启用。
- Prompt Library：创建、导入、校验、编辑和变量化渲染可复用 Prompt。
- Git Source：导入和更新个人或团队 Skill 仓库。
- Agent Activation：用受控软链接启用到 Claude Code 和 Codex。
- 本机项目部署：登记多个项目，并将 Library Skill 软链或复制到项目 Agent 目录。
- 项目 Skill 迁移：将项目中发现的外部 Skill 关联或复制到个人 Library，可安全移动一致的非托管副本。
- Project Require：记录 Git 来源、revision 和 hash，可在其他机器恢复。
- Project Vendor：复制成项目独立版本，个人原版继续保留。
- 安全部署：不覆盖未知目标，检测被修改的链接或副本。
- 可审计状态：YAML Manifest、Lock 和版本化 JSON 输出。

完整设计见 [Library、Activation 与 Project 设计](docs/技能库激活与项目设计.md)，
Prompt 格式与第一版能力见 [Prompt 管理](docs/Prompt管理.md)，
完整命令说明见 [CLI 使用指南](docs/命令行使用指南.md)。当前完成度和待办见
[核心流程验收清单](docs/核心流程验收清单.md)。macOS 原生应用的实现与后续方案见
[Mac 原生版技术设计](docs/Mac原生版技术设计.md)与
[Mac 原生版原型设计](docs/Mac原生版原型设计.md)。

## 安装

Git 仅在使用 Git Source 或恢复 `project require` 依赖时需要。Homebrew 和 curl
安装的是预编译文件，不要求本机安装 Go。

### Homebrew

```bash
brew install zzzzzyijie/tap/skm
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

完整的隔离开发、UI 验证、Snapshot 打包和正式发布流程见
[隔离开发与发布流程](docs/开发与发布流程.md)。发布维护细节见
[发布指南](docs/发布指南.md)。

### macOS 原生 App（0.5.1）

原生工程位于 `macos/`，需要 macOS 14+、Xcode 26+、Go 1.25+；Tuist 只在修改工程结构后
重新生成 Xcode 工程时需要。已生成的 `macos/SKM.xcworkspace` 可以直接打开：

```bash
open macos/SKM.xcworkspace
```

命令行构建与测试：

```bash
xcodebuild -workspace macos/SKM.xcworkspace -scheme SKM build
xcodebuild -workspace macos/SKM.xcworkspace -scheme SKM test
```

构建阶段会把同版本、Universal 2 的 Go Core 作为 `skm-core` 打入 App Bundle。App 不查找
`PATH`，因此不依赖 Homebrew/curl 安装的 CLI；它与 CLI 继续共享 `~/.skm`。开发与架构说明见
[macOS 工程说明](macos/README.md)。

## 快速开始

### Web UI

启动内嵌的本地管理界面：

```bash
skm ui
```

默认监听 `http://localhost:9527` 并打开浏览器。可用 `--port` 更换端口，或用
`--no-browser` 只启动服务。Web UI 与 CLI 使用同一份 Library、Project 注册和 Activation
数据；Prompt 页面可填写名称、描述、标签和正文，校验后保存，并将正文直接复制到设备剪贴板；
Projects 页面可完成本机项目登记、Skill 软链/复制、状态检查和解绑。设置中的“个人工作区”可连接
一个私有 Git 仓库，通过 Logo 右侧同步入口在多台电脑间双向同步独立的本地 Skill 和 Prompt；
同步前会预览上传、下载、删除和冲突，冲突可逐项选择本地或远程状态。

“添加 Skill 到库”提供三种明确的导入方式：本地导入支持 Skill 文件夹和仅包含一个
Skill 的 `.zip`；仓库来源支持 Git URL 与 GitHub `owner/repo` 简写；安装命令支持直接
粘贴 `npx skills add` 命令：

```text
npx skills add jakubkrehel/skills
npx skills add jakubkrehel/skills --skill better-ui
```

SKM 只解析命令中的来源和 `--skill` 选项，不会执行 `npx`；导入仍由内置的 Git Source
流程完成并记录 revision。来源名称可留空，由仓库 owner 和名称自动生成。

### 管理 Prompt

Prompt 使用带 YAML frontmatter 的 `PROMPT.md`，支持本地管理、变量替换和个人工作区同步，
不调用模型 API，也不会把 Prompt 部署到 Agent Skill 目录：

```bash
skm prompt validate ./PROMPT.md
skm prompt add ./PROMPT.md
skm prompt list --tag review
skm prompt render local/code-review \
  --var language=Go \
  --var-file code=./main.go
skm prompt export local/code-review --output ./PROMPT.md
```

### 建立个人 Library

```bash
skm add "$HOME/my-skills/code-review" \
  --tag development \
  --tag review

skm list --tag development
skm show local/code-review
skm update local/code-review "$HOME/my-skills/code-review"
```

`add` 只加入 Library，不自动启用。移除前必须先禁用；`remove` 会删除没有其他引用的
物理快照，共享或被项目固定的快照会保留。历史孤立快照可先预览再清理：

Web 管理界面可以直接编辑独立本地 Skill 的完整 `SKILL.md`。保存时创建新的不可变
快照，并自动刷新用户级 Agent 部署；Git Source Skill 和仍在跟随项目的 Skill 保持只读。

```bash
skm prune --dry-run
skm prune
```

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
~/.codex/skills/code-review -> ~/.skm/objects/<hash>/code-review
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
外部 Git Source 不会执行提交或推送；个人工作区同步会在完整校验和冲突预览后创建普通
fast-forward 提交并推送，绝不使用 force push。纯本地 Library 不需要 Git。

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
[CLI 使用指南](docs/命令行使用指南.md#72-将个人-library-skill-绑定到远程-git)。

## 项目使用

### 本机项目部署：不要求项目运行时依赖 skm

`project add` 只在用户的 `~/.skm/projects.yaml` 中登记项目路径，不会向项目写入
`.skm` 文件。之后可以从个人 Library 直接部署到项目对应的 Agent 目录：

```bash
skm project add "$HOME/Projects/shop-api" --name shop-api
skm project list

skm project link shop-api local/code-review --agent claude
skm project copy shop-api local/release-check --agent codex
skm project status shop-api
```

`link` 会创建指向 `~/.skm/objects/<hash>/<name>` 的软链接；项目运行时不需要启动
skm，但仍依赖本机 Library 快照。`copy` 会把 Skill 内容复制到项目的
`.claude/skills` 或 `.codex/skills`，复制完成后 Agent 可以脱离 skm 运行。
复制后的目录若要让其他机器直接使用，需要由项目自己的 Git 工作流提交。

重复执行相同的部署是幂等的；同一个项目和 Agent 中如果已有不同 ID 或 hash 的同名
Skill，skm 会报告冲突且不会覆盖未知目标。解绑和注销：

```bash
skm project unlink shop-api local/code-review --agent claude
skm project unregister shop-api
```

`unregister` 只删除本机项目登记，不删除项目目录。`project skills` 才是查看当前
项目 `require/vendor` 状态的命令。

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

标签是 Skill 与 Prompt 共用的个人资源。Web UI 可创建、重命名和删除未使用标签，并在添加或
编辑 Skill、Prompt 时从已有标签中选择。CLI 的以下命令用于管理 Skill 的标签关联：

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
  init, add, list, show, validate, update, remove, prune
  source add|list|update|remove, sync
  tag list|add|remove|rename

Activation:
  enable, disable, plan, apply, status, doctor

Project:
  project add|list|show|link|copy|unlink|status|unregister
  project skills|require|vendor|remove|apply

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
一键运行隔离的 Library/Activation 冒烟测试：

```bash
sh scripts/dev_smoke_test.sh
```

使用 `--full` 加入仓库测试、安装器和 Formula 检查；完整开发到发布流程见
[隔离开发与发布流程](docs/开发与发布流程.md)。

发布配置的本地验证方式见 [发布指南](docs/发布指南.md#4-本地检查发布配置)。
需要覆盖测试开发二进制时，请使用 [隔离开发与发布流程](docs/开发与发布流程.md)，
不要直接对真实 `~/.skm` 或 Agent 目录执行 `enable`、`apply`。

## License

[MIT License](LICENSE)

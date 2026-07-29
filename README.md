# 🚀 skm (AI Skill Manager)

**skm** 是一款专为开发者打造的 AI Agent 技能 (Skills) 管理工具。通过统一的作用域隔离与软链接引擎，轻松搞定个人偏好、项目规范与全局技能在 Claude Code、Codex 之间的同步与管理。

---

## ✨ Features

- 🎯 **三层作用域管理**：清晰划分 `Global`（全局）、`Personal`（个人私有）与 `Project`（项目专属）。
- 🏷 **场景标签**：支持默认标签与自定义多标签，用于分类、筛选和批量管理不同场景的 Skills。
- 🔗 **安全部署引擎**：一次编写，通过软链接或受控复制映射至各 Agent 的 Skill 目录。
- 🖥 **macOS 原生体验**：终端极速 CLI。
- 🔄 **Git 同步**：支持个人与团队 Skill 库的远程 Git 拉取与更新。

---

## 📦 Quick Start (CLI)

```bash
# 构建并安装到 ~/.local/bin
mkdir -p ~/.local/bin
go build -trimpath -o ~/.local/bin/skm ./cmd/skm

# 如果尚未配置，在 ~/.zshrc 中加入：
# export PATH="$HOME/.local/bin:$PATH"
export PATH="$HOME/.local/bin:$PATH"
rehash

# 初始化
skm version
skm init

# 添加本地 Skill：参数是包含 SKILL.md 的目录，不是网络 URL
skm add "$HOME/my-skills/code-review" \
  --scope personal \
  --tag development

# 进入目标项目并部署；project 是固定作用域值，不是路径
cd "$HOME/Projects/shop-api"
skm link local/code-review \
  --scope project \
  --agent claude,codex

# 检查部署状态；重复 apply 是幂等的
skm status
skm apply
```

`skm add` 只添加本地目录；Git 仓库使用 `skm source add <git-url> --name <name>`。如需在其他目录操作项目，使用全局参数 `--project /path/to/project`：

```bash
skm --project "$HOME/Projects/shop-api" \
  link local/code-review --scope project --agent claude,codex
```

项目级部署会更新可提交的 `.skm/project.yaml` 和 `.skm/lock.yaml`，记录依赖、Git revision 与内容 hash。

从安装、Skill 编写到 Git 同步和故障排查的完整说明见 [CLI 完整教程](docs/cli-guide.md)。

---

## 🔗 Git Sources

通过自定义名称绑定任意 Git Skill 仓库。可以锁定 branch、tag 或 commit，并通过 `--path` 只导入指定子目录：

```bash
# 扫描仓库内所有 SKILL.md
skm source add git@github.com:example/team-skills.git \
  --name team \
  --ref main \
  --tag team

# 只绑定指定 Skill 目录，可重复传入 --path
skm source add https://github.com/example/skills.git \
  --name review-pack \
  --path skills/code-review \
  --scope global

# 拉取新快照并更新已部署 Skill
skm sync
```

Git 凭证由系统 Git、SSH Agent 或 Credential Helper 管理；`skm` 不接受或保存 URL 内嵌凭证。

---

## 🏷 Tags

```bash
skm list --tag development --tag testing
skm link --tag development --scope project --agent codex
skm tag add local/code-review backend code-review
skm tag remove local/code-review backend
skm tag rename code-review review
```

多个 `--tag` 使用 AND 语义。显式标签替代默认的 `general` 标签。

---

## 🧰 Commands

```text
init, add, list, show, validate, remove
link, unlink, plan, apply, status, doctor
source add|list|update|remove, sync
tag list|add|remove|rename
completion, version
```

所有查询和主要操作均支持 `--json`，输出带有 `schemaVersion` 的稳定 envelope，可供后续 macOS App 调用。使用 `--home` 或 `SKM_HOME` 可以覆盖默认的 `~/.skm` 数据目录。

---

## ✅ Tests

```bash
go test ./...
go test -race ./...
go vet ./...
```

测试使用临时 HOME、临时项目和本地 Git 仓库，不访问真实 Agent 配置目录。

---

## 🛠 Tech Stack

- **CLI Engine**: Go / Cobra
- **Format**: `SKILL.md` (YAML Frontmatter + Markdown)
- **macOS App**: Swift / SwiftUI / AppKit

---

## 📄 License

[MIT License](LICENSE)

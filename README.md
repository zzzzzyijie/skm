# 🚀 skm (AI Skill Manager)

**skm** 是一款专为开发者打造的 AI Agent 技能 (Skills) 管理工具。通过统一的作用域隔离与软链接引擎，轻松搞定个人偏好、项目规范与全局技能在多个 AI Agent（Claude Code, Cursor, Windsurf 等）之间的同步与管理。

---

## ✨ Features

- 🎯 **三层作用域管理**：清晰划分 `Global`（全局）、`Personal`（个人私有）与 `Project`（项目专属）。
- 🔗 **软链接驱动 (Symlink Engine)**：一次编写，自动映射至各类 Agent 的 Skill/Rule 目录。
- 🖥 **macOS 原生体验**：内置 MenuBar 菜单栏 App 与终端极速 CLI。
- 🔄 **Git 同步**：支持个人与团队 Skill 库的远程 Git 拉取与更新。

---

## 📦 Quick Start (CLI)

```bash
# 安装 / 构建
go build -o skm main.go

# 导入/管理 Skill
skm add <skill-path>

# 链接至当前项目或全局 Agent 目录
skm link --scope project
```

---

## 🛠 Tech Stack

- **CLI Engine**: Go / Cobra / Bubbletea (TUI)
- **macOS App**: Swift / SwiftUI / AppKit
- **Format**: `SKILL.md` (YAML Frontmatter + Markdown)

---

## 📄 License

[MIT License](LICENSE)
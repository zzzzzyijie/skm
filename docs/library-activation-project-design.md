# skm Library、Activation 与 Project 设计

## 1. 目标

skm 将 Skill 的“拥有”“启用”和“项目使用”拆成三个独立概念：

```text
Library             用户拥有和管理哪些 Skill
Activation          哪些 Library Skill 对哪些 Agent 启用
Project Requirement 项目声明依赖或独立维护哪些 Skill
```

该模型替代原先同时表达内容归属、覆盖优先级和部署位置的
`global / personal / project` Scope。

## 2. 个人 Library

所有本地和 Git Skill 首先进入用户的个人 Library：

```text
~/.skm/
├── catalog.yaml
├── objects/<hash>/<skill>/
├── sources.yaml
├── sources/
└── state/state.yaml
```

Library 操作：

```bash
skm add <skill-path> [--tag ...]
skm source add <git-url> --name <name> [--tag ...]
skm list [--tag ...]
skm show <skill>
skm remove <skill>
skm prune [--dry-run]
```

添加只表示 Library 拥有该 Skill，不会自动部署。删除正在启用或被当前项目
引用的 Skill 时必须先禁用或解除项目依赖。`remove` 在删除 Library 条目后立即清理
没有其他已知引用的物理快照；仍被其他 Library 条目、固定 Activation、Deployment
或当前项目依赖引用的快照继续保留。`prune` 使用相同的引用规则清理所有历史孤立快照。

### 2.1 标签

标签属于个人 Library Skill，用于分类、筛选和批量启用：

```bash
skm add ./code-review --tag development --tag review
skm list --tag development
skm enable --tag development --agent codex
skm tag add local/code-review backend
```

多个 `--tag` 使用 AND 语义。未显式指定标签时使用配置中的默认标签。

## 3. Activation

Activation 表示 Library Skill 对用户的哪些 Agent 启用：

```bash
skm enable <skill> --agent claude,codex
skm disable <skill> --agent claude
```

用户级部署使用软链接：

```text
~/.claude/skills/<name> -> ~/.skm/objects/<hash>/<name>
~/.agents/skills/<name> -> ~/.skm/objects/<hash>/<name>
```

禁用只移除 skm 管理的链接和 Activation，不删除 Library 内容。两个已启用 Skill
如果会在同一个 Agent 中产生同名目标，skm 必须报告冲突，不猜测覆盖顺序。

## 4. Project Requirement

项目有两种使用 Skill 的模式，必须由用户显式选择。

个人 Library Skill 通过用户级 Activation 已经可以在所有项目中使用，不需要初始化
项目。只有项目需要声明团队依赖或维护独立副本时才产生项目状态；`require`、
`vendor` 和 `apply` 会按需创建该状态。

### 4.1 require：引用可恢复依赖

```bash
skm project require team/code-review --agent claude,codex
skm project apply
```

`require` 将完整 ID、Git URL、仓库内路径、revision、内容 hash、Agent 和部署模式
写入 `.skm/project.yaml` 与 `.skm/lock.yaml`。项目不复制 Skill 源码。

如果用户级 Activation 已经为目标 Agent 提供完全相同的 `ID + hash`，
项目依赖标记为 `satisfied-by-user`，不创建重复项目链接。否则 skm 获取锁定快照，
并在项目 Agent 目录生成本机链接。

只有具有可共享 Git 来源的 Library Skill 可以 `require`。本地-only Skill 无法在其他
机器恢复，必须先发布到 Git Source，或使用 `vendor`。

### 4.2 vendor：项目独立维护副本

```bash
skm project vendor local/code-review --agent claude,codex
```

`vendor` 将当前快照复制到：

```text
<project>/.skm/skills/<name>/
```

该目录是项目版本的唯一真实来源，应提交 Git。skm 再从 Claude Code 和 Codex 的
项目 Skill 目录创建链接。Agent 目录中的生成链接不应提交。

项目副本记录 `forkedFrom` 和 `forkedRevision`，但复制后与个人原版独立演化。
vendor 不会移动、删除或自动修改个人原版，也不执行隐式双向同步。

## 5. 项目文件

```text
<project>/.skm/
├── project.yaml
├── lock.yaml
└── skills/                 # 仅 vendored Skill
```

`project.yaml` 是期望状态，`lock.yaml` 是可复现快照。项目 Agent 目录下的软链接是
本机部署结果，不是项目数据。这些文件不要求预先初始化，由项目命令按需创建。

`project init` 仅用于在没有 `.git` 的目录中提前创建空 `.skm`，使 skm 从子目录运行
时能够识别项目根；它不是个人使用或项目工作流的前置步骤。

### 5.1 Git 策略

Git 是可选的协作与恢复能力，不是个人 Library 的前置条件。skm 不自动执行
`git init`、创建提交或配置远程：

- `skm add` 和用户 Activation 可以完全脱离 Git 使用；
- `skm source add` 由用户显式绑定已有的本地或远程 Git 仓库；
- `project require` 只接受可由团队访问的 Git URL，确保其他机器能恢复锁定快照；
- `project vendor` 不要求项目已经是 Git 仓库，但需要团队共享时，应由项目自身的
  Git 工作流提交 `.skm/` 中的 Manifest、Lock 和 vendored 内容。

因此，个人使用不需要先建立本地 Git；只有要版本化个人 Skill 库时，用户才自行给
该目录初始化 Git。项目已经使用 Git 时，vendor 内容自然跟随项目仓库维护。

个人 Skill 的 Git 工作目录应独立于 `~/.skm/objects/`：后者是 skm 管理的不可变
快照，不是源码工作树。个人 Skill 发布到远程后，通过 `source add` 重新导入为
`<source>/<name>`；已有 `local/<name>` 不会被原地转换，切换 Activation 和删除旧
条目必须由用户显式执行。

## 6. 冲突规则

1. 完整 Skill ID 使用 `<source>/<name>`。
2. Library 中短名称只有唯一时才允许使用。
3. 同一 Agent 的用户级 Activation 不允许两个同名 Skill。
4. 项目依赖与用户级 Skill 同名但 ID 或 hash 不同时报告冲突。
5. skm 不依赖 Claude Code 或 Codex 各自不同的同名覆盖规则。
6. 未由 skm 管理的 Agent 目标永远不会被自动覆盖。

## 7. 命令模型

```text
Library:
  init, add, list, show, validate, remove, prune
  source add|list|update|remove, sync
  tag list|add|remove|rename

Activation:
  enable, disable, plan, apply, status

Project:
  project list
  project require
  project vendor
  project remove
  project apply

Optional project-root marker:
  project init
```

旧 `link/unlink` 命令可作为过渡别名，但新文档和输出统一使用
`enable/disable` 与 `project require/vendor`。

## 8. 持久化与迁移

schema v2 使用：

```text
Skill.location       library | project
Activation.placement user | project
```

读取 schema v1 时：

- `global` 和 `personal` Catalog Skill 迁移为 `library`；
- `project` Catalog Skill 迁移为 `project`；
- `global` 和 `personal` Installation 迁移为用户 Activation；
- `project` Installation 迁移为项目 Activation。

写回后只保存 v2 字段。

# SKM UI MVP 方案

## 背景

`skm` 是一个用 Go 编写的 AI Agent Skill 管理 CLI 工具，核心功能包括：
- **Library 管理**：`add`、`list`、`show`、`remove`、`prune`
- **Activation 管理**：`enable`、`disable`、`plan`、`apply`、`status`、`doctor`
- **Source 管理**：`source add/list/update/remove`、`sync`
- **Tag 管理**：`tag list/add/remove/rename`
- **Project 管理**：Web UI 支持 `project add/list/show/link/copy/unlink/status/unregister` 的
  本机项目部署流程；CLI 仍支持完整的 `project skills/require/vendor/remove/apply`

现在要为这个 CLI 工具增加一个 Web UI 前端，MVP 版本。

## 架构方案选择

### 方案 A：Go 内嵌 HTTP Server + 前端单页面（采用）

在 Go 项目中增加一个 `internal/server` 包，提供 REST API，并用 `embed` 嵌入前端静态文件。用户通过 `skm ui` 命令启动本地 Web 服务。

```text
用户执行: skm ui
         → 启动 HTTP Server (localhost:9527)
         → 浏览器打开 Web UI
         → UI 通过 REST API 调用后端
         → 后端复用现有 store/catalog/planner 逻辑
```

**优势**：
- 零依赖部署，单个二进制文件包含一切
- 完全复用现有 Go 业务逻辑（`store`、`catalog`、`planner` 等）
- 无需额外安装 Node.js 或任何运行时
- 与 CLI 共享同一份数据，天然一致
- `go:embed` 可将前端资产嵌入二进制，分发简单

## 前端技术栈

| 层面 | 选型 | 理由 |
|------|------|------|
| 框架 | 原生 HTML/CSS/JS（单页应用） | MVP 无需框架，减少构建依赖 |
| 构建 | 无（直接 `go:embed`） | 最小化工具链 |
| 样式 | Vanilla CSS + CSS Variables | 可控、轻量 |
| 交互 | Fetch API + DOM 操作 | MVP 足够 |

## MVP 功能范围

### 核心页面（3 个）

#### 1. Dashboard（首页概览）
- Skill 总数、已启用数、Source 数量
- 最近添加的 Skill

#### 2. Library 管理
- Skill 列表（支持按 Tag 筛选）
- Skill 详情查看
- 添加 Skill：本地目录导入或 Git 来源导入
- 移除 Skill
- 标签管理（增/删/改名）
- 每个 Skill 直接切换 Claude / Codex 启用状态
- Git 来源的 Skill 详情中支持更新来源
- 本地导入通过 macOS Finder 选择 Skill 文件夹

Activation 不再作为独立导航页面，启用状态在对应的 Skill 卡片中完成。

#### 3. Projects（本机项目部署）
- 项目注册列表：路径、存在状态和项目级 Activation 数量
- 添加项目：路径必填，名称可选；省略名称时由后端使用项目根文件夹名称
- 选择项目后，从个人 Library 选择 Skill，按 Agent 勾选 Claude/Codex
- 选择软链接或复制模式，执行部署、查看状态、解绑和注销
- 重复部署显示 `unchanged`；冲突、被外部修改和混用部署模式通过错误提示返回

本页面对应的项目注册信息保存到用户侧 `~/.skm/projects.yaml`。`copy` 完成后项目
目录中的 Skill 可以脱离 skm 运行；`link` 仍依赖用户侧 Library 快照。

## 目录结构

```text
skm/
├── cmd/skm/main.go          # 不变
├── internal/
│   ├── cli/
│   │   ├── app.go            # 增加 newUICommand()
│   │   ├── ui.go             # [NEW] skm ui 命令
│   │   └── ...
│   └── server/               # [NEW] HTTP API 服务
│       ├── server.go          # HTTP 路由和服务启动
│       ├── api_library.go     # Library 相关 API
│       ├── api_activation.go  # Activation 相关 API
│       ├── api_source.go      # Source 相关 API
│       └── api_overview.go    # Dashboard/Doctor/Version API
└── web/                      # [NEW] 前端静态资源
    ├── embed.go               # go:embed 指令
    ├── index.html
    ├── app.js
    ├── app.css
    ├── assets/                    # Claude/Codex 图标
    └── components/                # JS 组件模块
        ├── dashboard.js
        ├── library.js
        └── projects.js
```

## REST API 设计

```text
GET    /api/dashboard          # 概览数据
GET    /api/version            # 版本信息

GET    /api/skills             # 列出所有 Skill（?tag=xxx 筛选）
GET    /api/skills/:id         # Skill 详情
POST   /api/skills             # 添加 Skill {path, tags, source}
POST   /api/dialogs/skill-directory # 打开 macOS Finder 选择本地 Skill 目录
DELETE /api/skills/:id         # 移除 Skill

GET    /api/tags               # 所有标签及计数
POST   /api/skill-tags/add     # 为 Skill 添加标签 {skill, tags}
POST   /api/skill-tags/remove  # 移除标签 {skill, tag}
POST   /api/tags/rename        # 重命名标签

GET    /api/status             # 当前 activation 状态
POST   /api/enable             # 启用 {skills, agents, mode}
POST   /api/disable            # 禁用 {skills, agents}
GET    /api/plan               # 查看 plan
POST   /api/apply              # 应用 plan

GET    /api/sources            # Source 列表
POST   /api/sources            # 添加并导入 Git Source {name, url, ref, paths, tags}
POST   /api/sources/:name/update  # 更新 Source
POST   /api/sync               # Sync 所有 Source

GET    /api/doctor             # 健康检查

GET    /api/projects           # 本机注册项目列表
POST   /api/projects           # 登记项目 {path, name}
GET    /api/projects/:id       # 项目详情、Activation 和部署状态
GET    /api/projects/:id/status # 项目部署状态
POST   /api/projects/:id/link # 软链接 {skill, agents, dryRun}
POST   /api/projects/:id/copy # 复制 {skill, agents, dryRun}
POST   /api/projects/:id/unlink # 解绑 {skill, agents, force}
DELETE /api/projects/:id      # 注销项目登记
```

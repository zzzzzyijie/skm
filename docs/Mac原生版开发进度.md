# SKM macOS 原生版开发进度

> 更新日期：2026-09-01
>
> 当前工程版本：`0.5.3 (Build 3)`
>
> Bundle ID：`com.zzzzzyijie.skm`（已确认）
>
> 当前结论：Phase 0 至 Phase 3 的工程实现已完成；Phase 1/2 已统一补强，本地正式签名、公证闭环已验收。剩余事项是 Sparkle 发布密钥、GitHub/Apple 凭据和外部机器上的正式发布动作，不属于本地功能开发缺口。

相关文档：

- [macOS 原生版技术设计](Mac原生版技术设计.md)
- [macOS 原生版原型设计](Mac原生版原型设计.md)
- [macOS 工程构建与发布说明](../macos/README.md)
- [macOS 原生 App 签名、公证与发布流程](Mac原生App签名公证与发布流程.md)

## 1. 总体进度

| 阶段 | 当前状态 | 说明 |
| --- | --- | --- |
| Phase 0：Core 契约 | 已完成 | stdio JSON-RPC、握手、稳定错误模型、读写方法、半包/损坏/超限/并发/崩溃与恢复测试已完成 |
| Phase 1：原生 MVP | 已完成 | Skills、Prompts、Agents、首次启动、双语、键盘、冲突恢复、诊断及本地 Developer ID 发布闭环已完成 |
| Phase 2：项目与同步 | 已完成 | Projects、Sources、Workspace、Diagnostics 和 Homebrew Cask 已实现并接入原生界面与发布工作流 |
| Phase 3：完整 CLI 能力 | 已完成 | 项目 Require/Vendor/Apply、Prompt 渲染、历史回滚、Quick Look 与 Sparkle 2 均已实现 |

## 2. 已完成

### 2.1 工程与运行时

- [x] SwiftUI 原生 macOS 工程、Xcode workspace、Tuist 工程描述和共享 Scheme。
- [x] App 使用内置 `skm-core`，通过私有 stdio JSON-RPC 通信，不监听网络端口。
- [x] 构建阶段分别生成 `arm64`、`x86_64` Core，并合并为 Universal 2。
- [x] App、Core 和发布参数执行版本一致性检查，当前版本统一为 `0.5.3`，构建号为 `3`。
- [x] App、测试 Target、发布脚本均使用最终 Bundle ID 命名空间；主 App 固定为 `com.zzzzzyijie.skm`。
- [x] Xcode 构建脚本可从 Apple Silicon Homebrew、Intel Homebrew 或 `SKM_GO_EXECUTABLE` 找到 Go。
- [x] 最终用户运行 App 不依赖 Go、Homebrew 或已安装的 SKM CLI。

### 2.2 Core Bridge

- [x] `system.handshake`、协议版本、Core 版本、Schema 版本和 capabilities 返回。
- [x] Skills 的列表、详情、添加、编辑、标签和移除方法。
- [x] Agents 的扫描、配置、自定义 Agent 保存与删除方法。
- [x] 用户级 Skill 启用、停用和部署状态读取方法。
- [x] Prompts 的列表、详情、创建、导入、编辑、校验和移除方法。
- [x] Sources 添加、列表、更新、移除与统一同步方法。
- [x] Projects 登记、详情扫描、部署预览/执行、解绑、注销和迁移方法。
- [x] 个人 Git 同步的配置、预览、冲突选择与双向同步方法（Core 内部仍使用 Workspace 协议名）。
- [x] Doctor 覆盖 Library、Git、所有已配置 Agent、Projects、Sources 和 Workspace 基础健康状态。
- [x] 结构化错误包含错误类型和是否可重试信息。
- [x] Core 异常退出后只自动重试安全的只读请求，不自动重放写请求。
- [x] 超限消息返回结构化错误，Bridge 可以继续处理下一条请求。
- [x] Projects 的 Require、Vendor、Apply 与清单条目安全移除方法。
- [x] Prompt 安全渲染与 Skill/Prompt 历史列表、差异和回滚方法。

### 2.3 Phase 1 原生界面

- [x] 主导航聚焦 Skills、Prompts、Projects；技能来源、Git 同步、Agent 管理、诊断与更新迁入独立 Settings 窗口。
- [x] 首次启动欢迎页，并识别现有 `~/.skm` Library。
- [x] Core 启动失败阻断页、重试和诊断信息复制。
- [x] Skills 本地目录、ZIP、Git URL、`owner/repo` 和安装命令导入。
- [x] Skills 搜索、详情、原文编辑、标签、健康状态、删除和 Agent 启用/停用。
- [x] Agents 检测、管理、自定义 Agent 新建、编辑和删除。
- [x] Prompts 搜索、创建、文件导入、编辑、复制、导出和删除。
- [x] Projects、技能来源与个人 Git 同步的完整读写主流程。
- [x] 监听 SKM 数据文件变化，检测 CLI 修改后自动刷新界面。
- [x] `⌘N`、`⌘O`、`⌘R`、`⌘⌫`、`⌘1...3`、`⌘,` 和系统设置菜单。
- [x] 空状态、加载状态、错误状态和操作完成提示。
- [x] Skill、Prompt、Agent 行摘要和主要控件的 VoiceOver 标签。
- [x] 简体中文、英文 String Catalog，可跟随 macOS 系统语言。

### 2.4 Phase 2 原生界面

- [x] 空白项目也可选择受支持 Agent，先预览再执行 Link/Copy 部署，并创建目标目录。
- [x] 未知目标或损坏状态会阻断覆盖；项目支持重新扫描、解绑、注销和 follow/copy/move 迁移。
- [x] Source 支持新增、编辑标签、更新、移除和逐来源失败隔离的统一同步。
- [x] 个人 Git 同步支持 Git URL/ref/root 配置、连接测试、双向差异预览和逐项冲突选择。
- [x] Skill/Prompt 并发编辑保留用户草稿，展示磁盘版本，并提供使用磁盘版、覆盖或另存为副本。
- [x] Diagnostics 提供完整 Doctor、脱敏复制/导出和 GitHub Release 更新检查。
- [x] Homebrew Cask 使用独立 token `skm-app`，只安装 `/Applications/SKM.app`，不覆盖 Formula 的 CLI。
- [x] 主窗口工具栏提供统一同步菜单，可更新所有技能来源、预览个人 Git 同步或直接打开同步设置。

### 2.5 Phase 3 原生界面

- [x] 项目清单支持 Require、Vendor、Apply 与条目移除，沿用 Planner 的冲突保护和原子部署。
- [x] Prompt 编辑器支持 text、multiline、number、boolean、select、secret 变量定义、必填/重复校验和默认值。
- [x] Prompt 填写表单由 Core 渲染，secret 值只在内存中传递，可校验缺失变量并复制最终结果。
- [x] Skill 与 Prompt 编辑前自动保存历史快照，原生界面支持版本列表、行差异和安全回滚。
- [x] Skill/PROMPT.md 支持系统 Quick Look、工具栏入口和空格快捷键。
- [x] Sparkle `2.9.6` 已通过 Swift Package Manager 固定，配置公钥时启用签名 appcast 自动更新；本地未配置时回退到只读版本检查。
- [x] 正式发布脚本生成 EdDSA 签名 `appcast.xml`，GitHub Release 工作流校验并注入独立 Sparkle 公私钥。

### 2.6 构建、测试与本地预览

- [x] 15 个 Swift 单元测试覆盖 Model、主导航/设置分区、Phase 3 响应、首次启动、版本比较及 Core 并发/损坏响应/stderr/超时/崩溃恢复。
- [x] 4 个中英文 XCUITest 覆盖空 Library、快捷键、设置入口、Skill 导入、Agent 管理、Prompt 创建/渲染入口、项目登记和 Git 同步预览。
- [x] 测试使用隔离的 `SKM_HOME`、`SKM_USER_HOME`、`SKM_PROJECT`，不操作真实个人数据。
- [x] CI 执行 Go 测试、Vet、Build、安装器测试和 macOS App 测试。
- [x] `--preview` 构建 Universal 2 ZIP、DMG 和 SHA-256 校验文件。
- [x] 本地预览对内置 Core 和 App 执行 ad-hoc hardened runtime 签名及严格校验。
- [x] 已验证内置 Core 在最小 `PATH=/usr/bin:/bin`、没有 Go 运行环境时完成版本查询和握手。
- [x] 正式发布工作流代码已包含 Developer ID 导入、notarytool、公证、staple、Gatekeeper 和双架构 runner 验证。
- [x] 已使用真实 Developer ID Application 与 App Store Connect Team API Key 跑通 App、DMG 公证与 stapling。
- [x] App 与 DMG 均通过 Gatekeeper，结果为 `accepted / Notarized Developer ID`。
- [x] DMG 在提交公证前执行独立 Developer ID 签名，避免 `source=no usable signature`。
- [x] Release 工作流会在 Apple Silicon 与 Intel 干净 runner 验证签名、公证票据、Gatekeeper、架构、无 Go Core 与 App 启动。
- [x] Tag 发布后自动生成并发布 CLI Formula 与原生 App Cask。

## 3. 部分完成，仍需补强

以下项目已经有基础实现，但还没有达到完整验收标准：

- [ ] 已有 VoiceOver 标签和键盘快捷键；仍需使用 Accessibility Inspector 人工检查焦点顺序、Full Keyboard Access、系统字体缩放和高对比度。
- [ ] 界面使用系统颜色并支持浅色/深色；仍需完成减少动态效果、不同窗口宽度和高对比度快照验收。

## 4. 待完成

### 4.1 Phase 1 发布闭环

- [x] 最终 Bundle ID 已确认并固定为 `com.zzzzzyijie.skm`；本机 Apple Developer Team 已完成真实签名验收。
- [x] 使用真实 `Developer ID Application` 证书完成 Universal 2 验收构建。
- [x] 使用 App Store Connect Team API Key 完成 App 和 DMG 公证、staple 与 Gatekeeper 验证。
- [ ] 在干净的 Apple Silicon Mac 和 Intel Mac 上验证安装、启动、Core 握手和主要读写流程。
- [x] 完成 Skills、Agents、Prompts、Projects、个人 Git 同步的隔离端到端 XCUITest。
- [x] 完成 App/Core 与共享 Store 并发修改的冲突、草稿保留和恢复测试。
- [ ] 完成正式发布前的 VoiceOver、键盘、浅色、深色和窄窗口人工验收。
- [ ] 把 Developer ID `.p12` 与 App Store Connect Team API Key 配置为 GitHub Actions Secrets，并验证 Tag 工作流。
- [ ] 创建 `v0.5.3` Tag 和正式 Release；当前尚未发布。

### 4.2 Phase 2：项目与同步

- [x] 项目登记、重新扫描、注销和解绑。
- [x] 项目级 Link、Copy、迁移、部署预览和冲突阻断。
- [x] Sources 管理、更新和统一同步。
- [x] 个人 Git 同步、差异预览和冲突解决。
- [x] 完整 Diagnostics、日志导出和更新检查。
- [x] Homebrew Cask，且不与 CLI Formula 的安装路径冲突。

### 4.3 Phase 3：完整 CLI 能力

- [x] `project require`、`project vendor`、`project apply`。
- [x] Prompt 变量表单、变量校验和渲染后复制。
- [x] 历史快照、差异查看和回滚。
- [x] Skill/Prompt Quick Look。
- [x] Sparkle 2 自动更新工程与 appcast 发布链路。

### 4.4 macOS 交互体验重构与功能精简清单（TODO）

- [x] **【Skills / Prompts】标签筛选交互重构**：
  - 方案：标签直接成为 Skills / Prompts 列表的分组层级，不再保留独立筛选控件，也不混入承载全局模块导航的主侧边栏；
  - 完成：列表由默认展开的“全部”分组与默认收起的各标签分组组成，每个分组均可独立展开、收起并显示条目数；
  - 交互：同一条目可同时出现在“全部”和多个标签分组中，任一入口都打开同一详情；搜索会同步收窄所有分组。
- [x] **【Skills】评估与精简主工具栏【刷新】按钮**：
  - 现状：macOS App 已具备 `FileChangeMonitor` 文件系统事件监听与 250ms 防抖热重载机制，外部 CLI 改动时会自动静默刷新；
  - 完成：已从主工具栏移除常驻显式“刷新”按钮，释放视觉焦点；同时在系统菜单栏 `视图 -> 刷新` 与全局快捷键（`⌘R`）保留全量重新载入能力。
- [x] **【Skills】评估与精简【历史版本】功能入口**：
  - 现状：Skill 多数为只读或来自 Git 导入，主工具栏常驻“历史”按钮认知负担重；
  - 完成：已从 Skill 详情主工具栏移除常驻“历史”按钮，收纳至「编辑」弹窗左下角（仅在编辑修改场景提供历史快照与 Diff 回滚，回滚后自动重载当前草稿）。
- [x] **【Skills】重构 Skill 详情页交互体验**：
  - 现状：原为"概览 / SKILL.md / 部署"三段 Segmented Control，割裂严重且 Markdown 为纯文本等宽字体；
  - 完成：合并为一体化滚动布局——顶部展示名称、健康 Badge、描述、标签与来源元数据，中部 Agent 激活采用水平卡片式切换（LazyVGrid + AgentToggleCard），正文区域使用 SwiftUI 原生 `AttributedString(markdown:)` 零依赖渲染 Markdown（支持标题层级、粗体/斜体/链接/代码/代码块），底部路径元数据卡片；工具栏新增"在 Finder 中显示"快捷按钮。
- [x] **【添加 Skill】同步 Web 端的两步式扫描预览与勾选导入能力**：
  - 现状：macOS 端原为旧版一步式直接导入，无法在导入前识别仓库内的多子技能；
  - 完成：Core Bridge 接入 `sources.preview` 能力，App 端升级为两步式导入向导（第 1 步输入仓库 URL/简写 -> 异步扫描多子技能与有效性 -> 第 2 步展示候选技能列表与错误 Badge -> 支持全选/反选与多选 -> 确认按需批量导入）。
- [ ] **【Skill 标签管理】集中式标签管理与维护入口**：
  - 现状：当前 Skill 标签仅能在单条 Skill 的编辑弹窗中修改，缺乏全局/集中式的标签管理视图；
  - 待办优化：提供统一的 Skill 标签管理面板/入口，支持集中查看全部标签列表、标签重命名/合并（自动更新所关联 Skill）、清理无引用的孤立标签，并探索列表多选批量打标能力。
- [ ] **【Prompt 标签管理】集中式标签管理与维护入口**：
  - 现状：Prompt 标签目前同样仅散落在单条 Prompt 编辑表单中，缺乏统一的标签池管理；
  - 待办优化：对齐 Skill 标签管理能力，提供 Prompt 标签集中管理面板，支持查看全局标签统计、重命名/合并、清理无用标签及批量管理。
- [ ] **【添加 Skill / Prompt】新建与导入入口流程优化**：
  - 现状：当前新建/添加操作主要依赖工具栏右上角 `＋` 按钮或快捷键（`⌘N` / `⌘O`），入口层级深且引导不够直观；
  - 待办优化：在侧栏分组、列表空状态及右键菜单中增加更显眼的「添加 Skill」/「新建 Prompt」快捷入口；统合本地创建、模板新建、外部导入（Git / 本地目录 / ZIP / 剪贴板）的引导向导流程。
- [ ] **【一键同步】主窗口左下角常驻一键同步与状态展示**：
  - 现状：个人 Git 同步与来源更新入口目前分散在设置窗口（Settings -> Git 同步 / 技能来源）及工具栏菜单中，日常高频同步操作路径长；
  - 待办优化：将「一键同步」快捷按钮与实时同步状态（如上次同步时间、同步中旋转指示、待同步变更计数）常驻置于主侧栏左下角（设置按钮上方/同栏），点击即可直接触发一键同步或快速唤起差异审阅弹窗。
- [ ] **【Projects 列表】修复项目项选中时高亮背景遮盖项目图标问题**：
  - 现状：列表项选中时的高亮色与项目图标颜色相近，导致选中后图标被背景色吞没、辨识度降低；
  - 待办优化：调整选中态下的图标颜色与背景对比（如选中时动态切换为高对比度白/浅色或使用自适应容器底色），确保选中状态下图标清晰可见。
- [ ] **【Project 详情】重构项目 Skills 列表中 Agent 标识 UI 样式**：
  - 现状：Project 详情中 Skill 行下的 Agent 标识带有通用 cpu 图标，排列稍显凌乱；
  - 待办优化：精简移除冗余小图标，重构 Agent 胶囊（Pill Badge）样式与字阶排版，使展示更加轻量、整洁、美观。
- [x] **【Projects 列表】优化「添加项目」按钮位置**：
  - 现状：添加项目按钮原使用 `folder.badge.plus` 图标且未显式指定 placement；
  - 完成：已将「添加项目」按钮对齐 Skills / Prompts 规范，统一使用 `ToolbarItem(placement: .primaryAction)` 与 `plus` 图标紧邻 Projects 标题右侧，保持整体界面布局一致。
- [x] **【项目导入 Skill】规范导入方式文案（软链接 / 复制）**：
  - 现状：弹窗中导入方式文案原为“保持同步/独立副本”，说明偏长；
  - 完成：统一规范模式选项为「软链接」与「复制」，明确软链接跟随个人 Library 自动同步、复制为脱离 SKM 的独立副本，并对齐 Agent 品牌图标展示。
- [x] **【设置 - Agent 管理】引入真实 Agent 品牌图标（对齐 Web UI）**：
  - 现状：原 Agent 列表与详情使用通用 SF Symbols，缺乏各工具识别度；
  - 完成：在 `Assets.xcassets` 引入 12 款真实 Agent 矢量品牌图标（Claude、Codex、Cursor、Copilot、Gemini、Windsurf、Kiro、Cline、OpenCode、Trae、Hermes、OpenClaw），并封装通用 `AgentIconView`，提供平滑圆角自适应展示与自定义 Agent 优雅降级。
- [x] **【设置 - Agent 管理】精简移除「由 SKM 管理」冗余文案**：
  - 现状：界面中存在「由 SKM 管理」等冗长文字标签，增加界面噪音；
  - 完成：移除开关旁冗余的“由 SKM 管理”文本标签，列表状态精简为“已启用 / 已检测 / 未检测”，视觉层次更清晰。

## 5. 后续需要提供的信息

当前本地开发、运行和 ad-hoc 预览不需要额外信息。进入正式分发时需要确认或配置：

| 项目 | 用途 | 当前状态 |
| --- | --- | --- |
| 最终 Bundle ID | App 身份和签名 | 已确认：`com.zzzzzyijie.skm` |
| Apple Developer Team | Developer ID 签名 | 本机已配置并验证 |
| Developer ID Application `.p12` 与密码 | CI 正式签名 | 本机钥匙串已验证；待配置为 GitHub Secrets，不提交到仓库 |
| App Store Connect API `.p8`、Key ID、Issuer ID | 自动公证 | 本机已验证；待配置为 GitHub Secrets，不提交到仓库 |
| Sparkle EdDSA 公钥与私钥 | App 内更新验签与 appcast 签名 | 待生成；公钥配置为 Secret，私钥文件/Secret 永不提交 |
| 正式发布账号与仓库权限 | 创建 Tag、Release 和上传产物 | 发布时确认 |

不要把证书、私钥或密码直接写入文档或提交到 Git。优先在本机钥匙串和 GitHub Actions Secrets 中配置。

## 6. 最近一次验证记录

截至 2026-09-03，已通过：

- `xcodebuild -workspace macos/SKM.xcworkspace -scheme SKM -configuration Debug -derivedDataPath /tmp/skm-macos-derived -only-testing:SKMTests test`：17 个 Swift 单元测试全量通过（包含 AppModelTests、CoreClientResilienceTests、ModelsTests）；
- `xcodebuild -workspace macos/SKM.xcworkspace -scheme SKM -configuration Debug -derivedDataPath /tmp/skm-macos-derived -only-testing:SKMUITests test`：5 个中英文 UI 测试全量通过（包含中英文空库与快捷键、项目登记与工作区同步、多标签分组渲染与技能 Agent / Prompt 写入流）；
- `go test ./...`：全量 Go 单元测试与工作区同步集成测试通过；
- `0.5.3 (Build 3)` Universal 2 ad-hoc 预览打包，包含 Sparkle 2.9.6；
- `com.zzzzzyijie.skm`、版本/构建号、双架构、签名、校验和与无 Go 环境 Core 独立复核；
- App/Core `arm64 + x86_64`、版本、签名、Core 无 Go 环境启动和 SHA-256 校验；
- Sparkle Downloader/Installer/Updater/Framework 嵌套签名严格校验，预览 App 在最小 PATH、无 Go/Homebrew 环境启动通过；
- 使用发布参数 `0.6.0 (Build 3)` 完成真实发布链路验收，但未创建对应 Tag 或 GitHub Release；
- App 公证与 DMG 公证均返回 `Accepted`；
- App 与 DMG 的 stapling、ticket validate 和 Gatekeeper 均通过；
- ZIP 与 DMG SHA-256 复核通过；
- 本次验收过程与 Submission ID 记录在
  [macOS 原生 App 签名、公证与发布流程](Mac原生App签名公证与发布流程.md)。

正式 Release 尚未创建；当前完成的是本地真实凭据发布闭环验证，不等同于发布 `v0.6.0`。

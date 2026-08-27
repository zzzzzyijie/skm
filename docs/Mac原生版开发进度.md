# SKM macOS 原生版开发进度

> 更新日期：2026-08-27  
> 当前工程版本：`0.5.1`  
> 当前结论：Phase 1 功能工程已完成；正式签名、公证与分发验收待完成。

相关文档：

- [macOS 原生版技术设计](Mac原生版技术设计.md)
- [macOS 原生版原型设计](Mac原生版原型设计.md)
- [macOS 工程构建与发布说明](../macos/README.md)

## 1. 总体进度

| 阶段 | 当前状态 | 说明 |
| --- | --- | --- |
| Phase 0：Core 契约 | 主要功能已完成 | stdio JSON-RPC、握手、稳定错误模型和 MVP 读写方法已接通；异常与并发测试矩阵仍可补强 |
| Phase 1：原生 MVP | 功能工程已完成，发布闭环待完成 | Skills、Prompts、Agents、首次启动、双语、键盘与基础 VoiceOver 已实现；Developer ID 签名、公证和人工验收待完成 |
| Phase 2：项目与同步 | 仅完成只读骨架 | Projects、Sources、Workspace 已能读取；项目写操作、同步、冲突处理尚未实现 |
| Phase 3：完整 CLI 能力 | 待开始 | 项目高级能力、Prompt 渲染、历史回滚和自动更新尚未实现 |

## 2. 已完成

### 2.1 工程与运行时

- [x] SwiftUI 原生 macOS 工程、Xcode workspace、Tuist 工程描述和共享 Scheme。
- [x] App 使用内置 `skm-core`，通过私有 stdio JSON-RPC 通信，不监听网络端口。
- [x] 构建阶段分别生成 `arm64`、`x86_64` Core，并合并为 Universal 2。
- [x] App、Core 和发布参数执行版本一致性检查，当前版本统一为 `0.5.1`。
- [x] Xcode 构建脚本可从 Apple Silicon Homebrew、Intel Homebrew 或 `SKM_GO_EXECUTABLE` 找到 Go。
- [x] 最终用户运行 App 不依赖 Go、Homebrew 或已安装的 SKM CLI。

### 2.2 Core Bridge

- [x] `system.handshake`、协议版本、Core 版本、Schema 版本和 capabilities 返回。
- [x] Skills 的列表、详情、添加、编辑、标签和移除方法。
- [x] Agents 的扫描、配置、自定义 Agent 保存与删除方法。
- [x] 用户级 Skill 启用、停用和部署状态读取方法。
- [x] Prompts 的列表、详情、创建、导入、编辑、校验和移除方法。
- [x] Sources 添加和列表、Projects 列表、Workspace 读取、Doctor 方法。
- [x] 结构化错误包含错误类型和是否可重试信息。
- [x] Core 异常退出后只自动重试安全的只读请求，不自动重放写请求。
- [x] 超限消息返回结构化错误，Bridge 可以继续处理下一条请求。

### 2.3 Phase 1 原生界面

- [x] 三栏导航和 Skills、Prompts、Projects、Agents 四个主模块。
- [x] 首次启动欢迎页，并识别现有 `~/.skm` Library。
- [x] Core 启动失败阻断页、重试和诊断信息复制。
- [x] Skills 本地目录、ZIP、Git URL、`owner/repo` 和安装命令导入。
- [x] Skills 搜索、详情、原文编辑、标签、健康状态、删除和 Agent 启用/停用。
- [x] Agents 检测、管理、自定义 Agent 新建、编辑和删除。
- [x] Prompts 搜索、创建、文件导入、编辑、复制、导出和删除。
- [x] Projects 与个人 Workspace 状态只读展示。
- [x] 监听 SKM 数据文件变化，检测 CLI 修改后自动刷新界面。
- [x] `⌘N`、`⌘O`、`⌘R`、`⌘⌫`、`⌘1...4` 和系统设置菜单。
- [x] 空状态、加载状态、错误状态和操作完成提示。
- [x] Skill、Prompt、Agent 行摘要和主要控件的 VoiceOver 标签。
- [x] 简体中文、英文 String Catalog，可跟随 macOS 系统语言。

### 2.4 构建、测试与本地预览

- [x] Swift Model 单元测试和 AppModel 首次启动、错误恢复测试。
- [x] 中英文 XCUITest，覆盖空 Library 和 `⌘N` 新建 Skill 主入口。
- [x] 测试使用隔离的 `SKM_HOME`、`SKM_USER_HOME`、`SKM_PROJECT`，不操作真实个人数据。
- [x] CI 执行 Go 测试、Vet、Build、安装器测试和 macOS App 测试。
- [x] `--preview` 构建 Universal 2 ZIP、DMG 和 SHA-256 校验文件。
- [x] 本地预览对内置 Core 和 App 执行 ad-hoc hardened runtime 签名及严格校验。
- [x] 已验证内置 Core 在最小 `PATH=/usr/bin:/bin`、没有 Go 运行环境时完成版本查询和握手。
- [x] 正式发布工作流代码已包含 Developer ID 导入、notarytool、公证、staple、Gatekeeper 和双架构 runner 验证。

## 3. 部分完成，仍需补强

以下项目已经有基础实现，但还没有达到完整验收标准：

- [ ] Core Bridge 已有握手、读写和超限消息测试；仍需增加半包、损坏响应、stderr 噪声、超时、并发请求和 Core 崩溃恢复的专项测试。
- [ ] Skill 编辑已经使用 `baseHash` 防止覆盖 CLI 并发修改；仍需实现保留用户草稿、展示差异和明确的冲突恢复选择。
- [ ] 已有诊断信息复制；完整 Doctor 界面、日志导出和敏感信息脱敏验收属于后续工作。
- [ ] 已有 VoiceOver 标签和键盘快捷键；仍需使用 Accessibility Inspector 人工检查焦点顺序、Full Keyboard Access、系统字体缩放和高对比度。
- [ ] 界面使用系统颜色并支持浅色/深色；仍需完成减少动态效果、不同窗口宽度和高对比度快照验收。
- [ ] Release 工作流已经实现，但没有真实 Apple 凭据，因此 Developer ID 签名、公证和 Gatekeeper 发布验收尚未实际执行。

## 4. 待完成

### 4.1 Phase 1 发布闭环

- [ ] 确认最终 Bundle ID 和 Apple Developer Team。
- [ ] 使用真实 `Developer ID Application` 证书构建 `0.5.1`。
- [ ] 使用 App Store Connect API Key 完成 App 和 DMG 公证、staple 与 Gatekeeper 验证。
- [ ] 在干净的 Apple Silicon Mac 和 Intel Mac 上验证安装、启动、Core 握手和主要读写流程。
- [ ] 完成 Skills、Agents、Prompts 的端到端 XCUITest，不只测试空状态入口。
- [ ] 完成 App 与 CLI 交替写入同一隔离 Library 的一致性测试。
- [ ] 完成正式发布前的 VoiceOver、键盘、浅色、深色和窄窗口人工验收。
- [ ] 创建 `v0.5.1` Tag 和正式 Release；当前尚未发布。

### 4.2 Phase 2：项目与同步

- [ ] 项目登记、重新扫描、注销和解绑。
- [ ] 项目级 Link、Copy、迁移、部署预览和冲突阻断。
- [ ] Sources 管理、更新和统一同步。
- [ ] 个人 Workspace 双向同步、差异预览和冲突解决。
- [ ] 完整 Diagnostics、日志导出和更新检查。
- [ ] Homebrew Cask，且不与 CLI Formula 的安装路径冲突。

### 4.3 Phase 3：完整 CLI 能力

- [ ] `project require`、`project vendor`、`project apply`。
- [ ] Prompt 变量表单、变量校验和渲染后复制。
- [ ] 历史快照、差异查看和回滚。
- [ ] Skill/Prompt Quick Look。
- [ ] Sparkle 2 自动更新。

## 5. 后续需要提供的信息

当前本地开发、运行和 ad-hoc 预览不需要额外信息。进入正式分发时需要确认或配置：

| 项目 | 用途 | 当前状态 |
| --- | --- | --- |
| 最终 Bundle ID | App 身份和签名 | 当前暂用 `com.zzzzzyijie.skm`，待确认 |
| Apple Developer Team | Developer ID 签名 | 待提供或在本机 Xcode 中配置 |
| Developer ID Application `.p12` 与密码 | CI 正式签名 | 待配置为 GitHub Secrets，不提交到仓库 |
| App Store Connect API `.p8`、Key ID、Issuer ID | 自动公证 | 待配置为 GitHub Secrets，不提交到仓库 |
| 正式发布账号与仓库权限 | 创建 Tag、Release 和上传产物 | 发布时确认 |

不要把证书、私钥或密码直接写入文档或提交到 Git。优先在本机钥匙串和 GitHub Actions Secrets 中配置。

## 6. 最近一次验证记录

截至 2026-08-27，已通过：

- `xcodebuild ... test`：8 个 Swift 单元测试、2 个中英文 UI 测试通过；
- `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`；
- curl 安装器测试和 Homebrew Formula 生成器测试；
- `0.5.1` Universal 2 ad-hoc 预览打包；
- App/Core `arm64 + x86_64`、版本、签名、Core 无 Go 环境启动和 SHA-256 校验。

正式 Developer ID 签名、公证、stapling 和 Gatekeeper 结果不能在缺少 Apple 凭据时标记为完成。

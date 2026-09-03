import AppKit
import Foundation
import Observation

/// 主界面业务分区（左侧主导航栏）
enum AppSection: String, CaseIterable, Identifiable {
    /// 技能库：管理本地与远程导入的 AI Skills
    case skills = "Skills"
    /// 提示词库：管理带参数定义的复用 Prompt 模板
    case prompts = "Prompts"
    /// 项目管理：管理项目维度的 Skill 部署、Require 与 Vendor 状态
    case projects = "Projects"

    var id: String { rawValue }

    var symbol: String {
        switch self {
        case .skills: "square.stack.3d.up"
        case .prompts: "text.bubble"
        case .projects: "folder"
        }
    }
}

/// 偏好设置分区
enum SettingsSection: String, CaseIterable, Identifiable {
    /// 通用：版本信息、存储路径与概览统计
    case general
    /// 权限访问：项目目录读取状态与 macOS 授权恢复
    case fileAccess
    /// Agent 管理：Claude Desktop、Codex 等 AI 客户端配置与自定义路径
    case agents
    /// 技能来源：Git 仓库源列表管理与批量拉取
    case sources
    /// Git 同步：个人工作区（Workspace）远端仓库配置与冲突合并
    case gitSync
    /// 软件更新：Sparkle 签名升级与 GitHub Releases 版本检查
    case updates

    var id: String { rawValue }

    var title: String {
        switch self {
        case .general: String(localized: "通用")
        case .fileAccess: String(localized: "权限访问")
        case .agents: String(localized: "Agent 管理")
        case .sources: String(localized: "技能来源")
        case .gitSync: String(localized: "Git 同步")
        case .updates: String(localized: "软件更新")
        }
    }

    var symbol: String {
        switch self {
        case .general: "gearshape"
        case .fileAccess: "folder.badge.gearshape"
        case .agents: "cpu"
        case .sources: "arrow.triangle.branch"
        case .gitSync: "arrow.triangle.2.circlepath.icloud"
        case .updates: "arrow.triangle.2.circlepath.circle"
        }
    }
}

/// 全局命令类型（由系统菜单快捷键触发，再分发给各子视图消费）
enum AppCommandKind: Equatable, Sendable {
    case create
    case importItem
    case deleteSelection
}

/// 全局命令包装
struct AppCommand: Identifiable, Sendable {
    let id = UUID()
    let kind: AppCommandKind
    let section: AppSection
}

/// AppModel - 主应用程序 ViewModel 状态机
/// 采用 Swift 5.9+ @Observable 宏进行响应式状态跟踪。
/// 核心职责：
/// 1. 驱动与底层 Go Core 子进程（JSON-RPC 2.0）的通信和数据同步；
/// 2. 维护 Skills、Prompts、Projects、Agents、Sources、Workspace 全局数据与当前选中项；
/// 3. 通过 FileChangeMonitor 监听 ~/.skm 本地数据变化并实现自动热重载；
/// 4. 统一处理加载状态、状态提示气泡（StatusPill）、错误弹窗与无障碍语音播报；
/// 5. 集中管理业务增删改查（mutate/mutateResult）与冲突处理逻辑。
@MainActor
@Observable
final class AppModel {
    let core: any CoreServing
    @ObservationIgnored private let fileMonitor = FileChangeMonitor()
    @ObservationIgnored private var isMonitoring = false
    @ObservationIgnored private let preferences: UserDefaults
    @ObservationIgnored private let monitorsFiles: Bool
    @ObservationIgnored private let presentsWelcome: Bool
    var section: AppSection = .skills
    var settingsSection: SettingsSection = .general
    var skills: [SkillSummary] = []
    var prompts: [PromptSummary] = []
    var projects: [ProjectModel] = []
    var agents: [AgentModel] = []
    var sources: [SourceModel] = []
    var workspace: WorkspaceView?
    var projectDetails: ProjectDetails?
    var projectDeploymentPreview: ProjectDeployResponse?
    var workspacePreview: WorkspacePreview?
    var workspaceResolutions: [String: String] = [:]
    var sourceSyncResult: SourceSyncResponse?
    var doctorChecks: [DoctorCheck] = []
    var updateStatus: String?
    var plan = PlanModel(digest: "", operations: [])
    var selectedSkillID: String?
    var selectedPromptID: String?
    var selectedProjectID: String?
    var selectedAgentID: String?
    var selectedSourceID: String?
    var handshake: Handshake?
    var isLoading = false
    var errorMessage: String?
    var startupErrorMessage: String?
    var statusMessage: String?
    var showsWelcome = false
    var hasExistingData = false
    var pendingCommand: AppCommand?
    var lastErrorKind: String?

    init(
        core: any CoreServing = CoreClient(),
        preferences: UserDefaults = .standard,
        monitorsFiles: Bool = true,
        presentsWelcome: Bool = true
    ) {
        self.core = core
        self.preferences = preferences
        self.monitorsFiles = monitorsFiles
        self.presentsWelcome = presentsWelcome
    }

    /// 启动应用：发起握手连接 Core，全量拉取业务数据，开启本地文件监控，并在首次使用时展示欢迎向导
    func start() async {
        guard handshake == nil, !isLoading else { return }
        isLoading = true
        statusMessage = String(localized: "正在连接 Core…")
        startupErrorMessage = nil
        defer { isLoading = false }
        do {
            handshake = try await core.handshake()
            try await self.reload()
            startupErrorMessage = nil
            statusMessage = nil
            startFileMonitoring()
            if presentsWelcome,
               ProcessInfo.processInfo.environment["SKM_SKIP_WELCOME"] != "1",
               !preferences.bool(forKey: Self.welcomePreferenceKey) {
                showsWelcome = true
            }
        } catch {
            startupErrorMessage = error.localizedDescription
            statusMessage = nil
        }
    }

    /// 重启并重新连接 Core（用于启动失败后的用户手动重试）
    func retryStart() async {
        await core.stop()
        handshake = nil
        await start()
    }

    /// 手动全量刷新所有业务数据
    func refresh() async {
        await perform(String(localized: "正在刷新…")) { try await self.reload() }
    }

    /// 发布状态提示文案，并在 2.5 秒后自动清除，同时触发 macOS VoiceOver 辅助功能播报
    func announce(_ message: String) {
        statusMessage = message
        NSAccessibility.post(
            element: NSApp as Any,
            notification: .announcementRequested,
            userInfo: [
                .announcement: message,
                .priority: NSAccessibilityPriorityLevel.high.rawValue,
            ]
        )
        Task { @MainActor in
            try? await Task.sleep(for: .seconds(2.5))
            if self.statusMessage == message { self.statusMessage = nil }
        }
    }

    /// 停止文件监听并关闭 Core 进程连接
    func stop() async {
        fileMonitor.stop()
        isMonitoring = false
        await core.stop()
    }

    /// 完成欢迎向导并记录到 UserDefaults 避免重复弹出
    func completeWelcome() {
        preferences.set(true, forKey: Self.welcomePreferenceKey)
        showsWelcome = false
    }

    /// 分发系统级菜单命令
    func request(_ kind: AppCommandKind) {
        pendingCommand = AppCommand(kind: kind, section: section)
    }

    /// 消费已处理的菜单命令
    func consumeCommand(_ id: UUID) {
        guard pendingCommand?.id == id else { return }
        pendingCommand = nil
    }

    var canCreate: Bool { true }

    var canImport: Bool { section == .skills || section == .prompts || section == .projects }

    var canDeleteSelection: Bool {
        switch section {
        case .skills: return selectedSkillID != nil
        case .prompts: return selectedPromptID != nil
        case .projects: return selectedProjectID != nil
        }
    }

    var diagnosticText: String {
        let appVersion = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
        let systemVersion = ProcessInfo.processInfo.operatingSystemVersionString
        let report = [
            "SKM diagnostics",
            "App: \(appVersion)",
            "Core: \(handshake?.coreVersion ?? "unavailable")",
            "Protocol: \(handshake?.protocolVersion.description ?? "unavailable")",
            "macOS: \(systemVersion)",
            "Architecture: \(architectureName)",
            "Error: \(startupErrorMessage ?? errorMessage ?? "none")",
            "Doctor:",
            doctorChecks.map { "- \($0.name): \($0.status) — \($0.message)" }.joined(separator: "\n"),
        ].joined(separator: "\n")
        return redactDiagnostics(report)
    }

    private func redactDiagnostics(_ input: String) -> String {
        var result = input.replacingOccurrences(of: FileManager.default.homeDirectoryForCurrentUser.path, with: "~")
        if let expression = try? NSRegularExpression(pattern: #"(https?://)[^\s/@]+@"#, options: [.caseInsensitive]) {
            let range = NSRange(result.startIndex..., in: result)
            result = expression.stringByReplacingMatches(in: result, range: range, withTemplate: "$1***@")
        }
        return result
    }

    func copyDiagnostics() {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(diagnosticText, forType: .string)
        announce(String(localized: "诊断信息已复制"))
    }

    func exportDiagnostics() {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "SKM-diagnostics.txt"
        guard panel.runModal() == .OK, let url = panel.url else { return }
        do {
            try diagnosticText.write(to: url, atomically: true, encoding: .utf8)
            announce(String(localized: "诊断信息已导出"))
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// 查询指定 Skill 的详细信息（包含完整 Frontmatter 与正文）
    func skillDetails(_ id: String) async throws -> SkillDetails {
        try await core.call("skills.get", params: IDParams(id: id))
    }

    /// 查询指定 Prompt 的详细信息
    func promptDetails(_ id: String) async throws -> PromptDetails {
        try await core.call("prompts.get", params: IDParams(id: id))
    }

    /// 导入本地目录或 ZIP 压缩包作为 Skill
    func addLocalSkill(path: String, tags: [String]) async {
        await mutate(success: String(localized: "Skill 已导入")) {
            let _: MutationSkill = try await self.core.call("skills.add", params: AddSkillParams(path: path, tags: tags, source: "local"))
        }
    }

    /// 预览 Git 仓库中的候选 Skill 列表与合法性状态
    func previewSource(input: String, name: String? = nil, ref: String? = nil) async throws -> SourcePreviewResult {
        let params = AddSourceParams(input: input, name: name, ref: ref, paths: nil, tags: [])
        return try await core.call("sources.preview", params: params)
    }

    /// 从 Git 远程仓库导入选中的 Skills 并注册 Source
    @discardableResult
    func addRemoteSkill(
        input: String,
        name: String? = nil,
        ref: String? = nil,
        paths: [String]? = nil,
        tags: [String] = []
    ) async -> Bool {
        await mutateResult(success: String(localized: "Skills 已从 Git 仓库导入")) {
            let _: AddSourceResponse = try await self.core.call(
                "sources.add",
                params: AddSourceParams(input: input, name: name, ref: ref, paths: paths, tags: tags)
            )
        }
    }

    /// 在线更新 Skill 内容（带基于 baseHash 的乐观并发冲突检测）
    @discardableResult
    func updateSkill(id: String, content: String, baseHash: String, tags: [String]) async -> Bool {
        await mutateResult(success: String(localized: "Skill 已保存并重新部署"), handlesConflict: true) {
            let _: SkillUpdateResponse = try await self.core.call("skills.update", params: UpdateSkillParams(id: id, content: content, baseHash: baseHash, tags: tags))
        }
    }

    /// 移除指定 Skill（若有已生效部署将阻止或提示）
    func removeSkill(id: String) async {
        await mutate(success: String(localized: "Skill 已移除")) {
            let _: MutationSkill = try await self.core.call("skills.remove", params: IDParams(id: id))
            self.selectedSkillID = nil
        }
    }

    /// 针对指定 Agent 启用或停用全局 Skill
    func setSkill(_ skillID: String, agentID: String, enabled: Bool) async {
        await mutate(success: enabled ? String(localized: "已为 Agent 启用") : String(localized: "已停用")) {
            if enabled {
                let _: PlanModel = try await self.core.call("activations.enable", params: ActivationParams(skills: [skillID], agents: [agentID], mode: "auto"))
            } else {
                let _: StatusResponse = try await self.core.call("activations.disable", params: ActivationParams(skills: [skillID], agents: [agentID], mode: nil))
            }
        }
    }

    /// 检查指定 Skill 是否已对某 Agent 全局启用
    func isEnabled(_ skillID: String, for agentID: String) -> Bool {
        plan.operations.contains { $0.skillId == skillID && $0.agent == agentID && $0.placement == "user" }
    }

    /// 勾选/取消勾选受管 Agent 列表
    func configureAgent(_ id: String, enabled: Bool) async {
        var selected = Set(agents.filter(\.configured).map(\.id))
        if enabled { selected.insert(id) } else { selected.remove(id) }
        await mutate(success: String(localized: "Agent 配置已更新")) {
            let _: [AgentModel] = try await self.core.call("agents.configure", params: ConfigureAgentsParams(agents: Array(selected).sorted()))
        }
    }

    /// 保存自定义 Agent 适配器路径配置
    func saveCustomAgent(id: String, name: String, path: String) async {
        await mutate(success: String(localized: "自定义 Agent 已保存")) {
            let _: [AgentModel] = try await self.core.call("agents.custom.save", params: CustomAgentParams(id: id, name: name, skillsPath: path))
        }
    }

    /// 删除自定义 Agent 适配器
    func deleteCustomAgent(id: String) async {
        await mutate(success: String(localized: "自定义 Agent 已删除")) {
            let _: StatusResponse = try await self.core.call("agents.custom.delete", params: IDParams(id: id))
            self.selectedAgentID = nil
        }
    }

    /// 新建或更新 Prompt 模板
    @discardableResult
    func savePrompt(
        id: String?,
        name: String,
        description: String,
        body: String,
        tags: [String],
        variables: [PromptVariable],
        baseHash: String?
    ) async -> Bool {
        await mutateResult(success: id == nil ? String(localized: "Prompt 已创建") : String(localized: "Prompt 已保存"), handlesConflict: true) {
            let params = PromptWriteParams(id: id, content: nil, name: name, description: description, tags: tags, body: body, variables: variables, source: "local", baseHash: baseHash)
            let _: PromptSummary = try await self.core.call(id == nil ? "prompts.create" : "prompts.update", params: params)
        }
    }

    /// 导入外部 Markdown 文本为 Prompt
    func importPrompt(content: String) async {
        await mutate(success: String(localized: "Prompt 已导入")) {
            let params = PromptWriteParams(id: nil, content: content, name: "", description: "", tags: [], body: "", variables: [], source: "local", baseHash: nil)
            let _: PromptSummary = try await self.core.call("prompts.create", params: params)
        }
    }

    /// 参数化实时渲染 Prompt 模板
    func renderPrompt(id: String, values: [String: String]) async throws -> PromptRenderResponse {
        try await core.call("prompts.render", params: PromptRenderParams(id: id, values: values))
    }

    /// 获取项目/Prompt/Skill 的历史版本记录列表
    func history(kind: String, itemID: String) async throws -> [HistoryEntryModel] {
        try await core.call("history.list", params: HistoryParams(kind: kind, itemId: itemID))
    }

    /// 对比两个历史快照版本间的 Diff
    func historyDiff(kind: String, itemID: String, from: String, to: String = "current") async throws -> HistoryDiffResponse {
        try await core.call("history.diff", params: HistoryDiffParams(kind: kind, itemId: itemID, from: from, to: to))
    }

    /// 回滚到指定的历史版本
    @discardableResult
    func rollbackHistory(kind: String, itemID: String, entryID: String) async -> Bool {
        await mutateResult(success: String(localized: "历史版本已恢复"), handlesConflict: true) {
            let _: HistoryRollbackResponse = try await self.core.call(
                "history.rollback",
                params: HistoryEntryParams(kind: kind, itemId: itemID, entryId: entryID)
            )
        }
    }

    /// 移除 Prompt 模板
    func removePrompt(id: String) async {
        await mutate(success: String(localized: "Prompt 已移除")) {
            let _: PromptSummary = try await self.core.call("prompts.remove", params: IDParams(id: id))
            self.selectedPromptID = nil
        }
    }

    /// 加载并扫描指定项目的内部 Skills 与各 Agent 部署情况
    func loadProjectDetails(_ id: String) async {
        await perform(String(localized: "正在扫描项目…")) {
            self.projectDetails = try await self.core.call("projects.get", params: IDParams(id: id))
        }
    }

    /// 登记本机项目目录
    func addProject(path: String, name: String) async {
        await mutate(success: String(localized: "项目已登记")) {
            let _: RegisteredProject = try await self.core.call("projects.add", params: AddProjectParams(path: path, name: name))
        }
    }

    /// 注销项目登记（不删除项目源码文件）
    func unregisterProject(id: String) async {
        await mutate(success: String(localized: "项目已注销")) {
            let _: RegisteredProject = try await self.core.call("projects.unregister", params: IDParams(id: id))
            self.selectedProjectID = nil
            self.projectDetails = nil
        }
    }

    /// 部署 Skill 到指定项目（支持 dryRun 预演预览与真实应用）
    func deployProject(project: String, skill: String, agents: [String], mode: String, dryRun: Bool) async {
        let succeeded = await mutateResult(
            success: dryRun ? String(localized: "部署预览已生成") : String(localized: "项目部署已完成"),
            reloads: !dryRun
        ) {
            let response: ProjectDeployResponse = try await self.core.call(
                "projects.deploy",
                params: ProjectDeployParams(project: project, skill: skill, agents: agents, mode: mode, dryRun: dryRun)
            )
            self.projectDeploymentPreview = response
        }
        if succeeded && !dryRun {
            projectDeploymentPreview = nil
            await loadProjectDetails(project)
        }
    }

    /// 解除项目内 Skill 与各 Agent 的部署绑定
    func unlinkProject(project: String, skill: String, agents: [String]) async {
        let succeeded = await mutateResult(success: String(localized: "项目 Skill 已解绑")) {
            let _: StatusResponse = try await self.core.call(
                "projects.unlink",
                params: ProjectUnlinkParams(project: project, skill: skill, agents: agents, force: false)
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    /// 迁移项目内已有的散装 Skill 为 SKM 受管格式
    func migrateProjectSkill(project: String, skill: String, agent: String, mode: String, removeSource: Bool) async {
        let succeeded = await mutateResult(success: String(localized: "项目 Skill 已迁移")) {
            let _: ProjectMigrateResponse = try await self.core.call(
                "projects.migrate",
                params: ProjectMigrateParams(project: project, skill: skill, agent: agent, mode: mode, removeSource: removeSource, tags: [])
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    /// 固定项目依赖（在项目清单中声明 Require）
    func requireProjectSkill(project: String, skill: String, agents: [String], mode: String) async {
        let succeeded = await mutateResult(success: String(localized: "项目依赖已固定")) {
            let _: ProjectAdvancedResponse = try await self.core.call(
                "projects.require",
                params: ProjectRequireParams(project: project, skill: skill, agents: agents, mode: mode, apply: false)
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    /// 将 Skill 源码直接 Vendor 拷贝到项目中
    func vendorProjectSkill(project: String, skill: String, agents: [String], mode: String) async {
        let succeeded = await mutateResult(success: String(localized: "Skill 已 Vendor 到项目")) {
            let _: ProjectAdvancedResponse = try await self.core.call(
                "projects.vendor",
                params: ProjectVendorParams(project: project, skill: skill, agents: agents, mode: mode, tags: [], apply: false)
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    /// 根据项目清单（Manifest）应用并修复所有依赖部署
    func applyProjectManifest(project: String, force: Bool = false) async {
        let succeeded = await mutateResult(success: String(localized: "项目清单已应用")) {
            let _: ProjectAdvancedResponse = try await self.core.call(
                "projects.apply",
                params: ProjectApplyParams(project: project, force: force)
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    /// 移除项目清单中的特定条目
    func removeProjectEntry(project: String, entry: String, force: Bool = false) async {
        let succeeded = await mutateResult(success: String(localized: "项目清单条目已移除")) {
            let _: ProjectAdvancedResponse = try await self.core.call(
                "projects.entry.remove",
                params: ProjectEntryRemoveParams(project: project, entry: entry, force: force)
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    /// 查找当前选中项的预览文件路径（SKILL.md 或 PROMPT.md）供 QuickLook 使用
    func quickLookURL() async throws -> URL? {
        switch section {
        case .skills:
            guard let selectedSkillID else { return nil }
            let details = try await skillDetails(selectedSkillID)
            return URL(fileURLWithPath: details.effectivePath).appendingPathComponent("SKILL.md")
        case .prompts:
            guard let selectedPromptID else { return nil }
            let details = try await promptDetails(selectedPromptID)
            let path = URL(fileURLWithPath: details.path)
            return path.pathExtension.lowercased() == "md" ? path : path.appendingPathComponent("PROMPT.md")
        default:
            return nil
        }
    }

    /// 添加 Git 技能源并支持筛选路径与预设标签
    @discardableResult
    func addSource(
        input: String,
        name: String? = nil,
        ref: String? = nil,
        paths: [String]? = nil,
        tags: [String] = []
    ) async -> Bool {
        await mutateResult(success: String(localized: "Git Source 已添加")) {
            let _: AddSourceResponse = try await self.core.call(
                "sources.add",
                params: AddSourceParams(input: input, name: name, ref: ref, paths: paths, tags: tags)
            )
        }
    }

    /// 更新指定的单个 Git 技能源
    func updateSource(name: String) async {
        await mutate(success: String(localized: "Git Source 已更新")) {
            let _: SourceUpdateResponse = try await self.core.call("sources.update", params: SourceNamesParams(names: [name]))
        }
    }

    /// 移除 Git 技能源
    func removeSource(name: String) async {
        await mutate(success: String(localized: "Git Source 已移除")) {
            let _: SourceRemovalResponse = try await self.core.call("sources.remove", params: IDParams(id: name))
            self.selectedSourceID = nil
        }
    }

    /// 批量拉取并同步所有 Git 来源
    func syncSources() async {
        await mutate(success: String(localized: "所有 Git Sources 已同步")) {
            self.sourceSyncResult = try await self.core.call("sources.sync", params: EmptyParams())
        }
    }

    /// 配置个人 Git 工作区（绑定同步远端）
    func configureWorkspace(url: String, ref: String, root: String) async {
        await mutate(success: String(localized: "个人工作区已配置")) {
            let _: WorkspaceView = try await self.core.call(
                "workspace.configure",
                params: WorkspaceConfigureParams(url: url, ref: ref, root: root.isEmpty ? nil : root)
            )
        }
    }

    /// 预演个人工作区双向同步差异
    func previewWorkspace() async {
        await perform(String(localized: "正在预览同步…")) {
            self.workspacePreview = try await self.core.call("workspace.preview", params: EmptyParams())
            self.workspaceResolutions = [:]
        }
    }

    /// 执行个人工作区同步（应用冲突决议方案并提交推拉）
    func syncWorkspace() async {
        let succeeded = await mutateResult(success: String(localized: "个人工作区同步完成")) {
            let _: WorkspaceSyncResponse = try await self.core.call(
                "workspace.sync",
                params: WorkspaceSyncParams(resolutions: self.workspaceResolutions)
            )
        }
        if succeeded {
            workspacePreview = nil
            workspaceResolutions = [:]
        }
    }

    /// 一键全量同步：根据已配置的个人工作区与技能来源，执行一键拉取推演或批量同步
    func oneClickSync() async {
        if workspace?.configured == true {
            await previewWorkspace()
            if let preview = workspacePreview {
                if preview.changes.isEmpty {
                    announce(String(localized: "已与远端保持完全同步"))
                } else if preview.conflicts == 0 {
                    await syncWorkspace()
                } else {
                    self.settingsSection = .gitSync
                }
            }
        } else if !sources.isEmpty {
            await syncSources()
        } else {
            self.settingsSection = .gitSync
        }
    }

    /// 批量重命名或合并 Skill 标签
    func renameSkillTag(from oldTag: String, to newTag: String) async {
        let trimmedNew = newTag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedNew.isEmpty, trimmedNew != oldTag else { return }
        let affected = skills.filter { $0.tags.contains(oldTag) }
        guard !affected.isEmpty else { return }

        await perform(String(format: String(localized: "正在更新 %lld 个 Skill 的标签…"), affected.count)) {
            for summary in affected {
                if let details: SkillDetails = try? await self.core.call("skills.get", params: IDParams(id: summary.id)) {
                    var newTags = details.tags.map { $0 == oldTag ? trimmedNew : $0 }
                    newTags = Array(NSOrderedSet(array: newTags)).compactMap { $0 as? String }
                    let _: SkillUpdateResponse = try await self.core.call(
                        "skills.update",
                        params: UpdateSkillParams(id: summary.id, content: details.content, baseHash: details.hash, tags: newTags)
                    )
                }
            }
            await self.refresh()
        }
        announce(String(localized: "标签已更新"))
    }

    /// 批量从所有 Skill 中移除指定标签
    func removeSkillTag(_ tag: String) async {
        let affected = skills.filter { $0.tags.contains(tag) }
        guard !affected.isEmpty else { return }

        await perform(String(format: String(localized: "正在从 %lld 个 Skill 中移除标签…"), affected.count)) {
            for summary in affected {
                if let details: SkillDetails = try? await self.core.call("skills.get", params: IDParams(id: summary.id)) {
                    let newTags = details.tags.filter { $0 != tag }
                    let _: SkillUpdateResponse = try await self.core.call(
                        "skills.update",
                        params: UpdateSkillParams(id: summary.id, content: details.content, baseHash: details.hash, tags: newTags)
                    )
                }
            }
            await self.refresh()
        }
        announce(String(localized: "标签已移除"))
    }

    /// 批量重命名或合并 Prompt 标签
    func renamePromptTag(from oldTag: String, to newTag: String) async {
        let trimmedNew = newTag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedNew.isEmpty, trimmedNew != oldTag else { return }
        let affected = prompts.filter { $0.tags.contains(oldTag) }
        guard !affected.isEmpty else { return }

        await perform(String(format: String(localized: "正在更新 %lld 个 Prompt 的标签…"), affected.count)) {
            for summary in affected {
                if let details: PromptDetails = try? await self.core.call("prompts.get", params: IDParams(id: summary.id)) {
                    var newTags = details.tags.map { $0 == oldTag ? trimmedNew : $0 }
                    newTags = Array(NSOrderedSet(array: newTags)).compactMap { $0 as? String }
                    let _: PromptSummary = try await self.core.call(
                        "prompts.update",
                        params: PromptWriteParams(
                            id: summary.id,
                            content: nil,
                            name: details.name,
                            description: details.description,
                            tags: newTags,
                            body: details.body,
                            variables: details.variables ?? [],
                            source: "local",
                            baseHash: details.hash
                        )
                    )
                }
            }
            await self.refresh()
        }
        announce(String(localized: "标签已更新"))
    }

    /// 批量从所有 Prompt 中移除指定标签
    func removePromptTag(_ tag: String) async {
        let affected = prompts.filter { $0.tags.contains(tag) }
        guard !affected.isEmpty else { return }

        await perform(String(format: String(localized: "正在从 %lld 个 Prompt 中移除标签…"), affected.count)) {
            for summary in affected {
                if let details: PromptDetails = try? await self.core.call("prompts.get", params: IDParams(id: summary.id)) {
                    let newTags = details.tags.filter { $0 != tag }
                    let _: PromptSummary = try await self.core.call(
                        "prompts.update",
                        params: PromptWriteParams(
                            id: summary.id,
                            content: nil,
                            name: details.name,
                            description: details.description,
                            tags: newTags,
                            body: details.body,
                            variables: details.variables ?? [],
                            source: "local",
                            baseHash: details.hash
                        )
                    )
                }
            }
            await self.refresh()
        }
        announce(String(localized: "标签已移除"))
    }

    private static let customTagsStorageKey = "skm.custom.tags"

    /// 用户主动添加并持久化的标签集合
    var customTags: Set<String> {
        get {
            let list = preferences.stringArray(forKey: Self.customTagsStorageKey) ?? []
            return Set(list)
        }
        set {
            preferences.set(Array(newValue).sorted(), forKey: Self.customTagsStorageKey)
        }
    }

    /// 注册一个自定义标签（即便尚未绑定任何条目）
    func registerCustomTag(_ tag: String) {
        let trimmed = tag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        var current = customTags
        current.insert(trimmed)
        customTags = current
    }

    /// 批量为指定 Skill 打上标签
    func addTagToSkills(_ tag: String, skillIDs: Set<String>) async {
        let trimmed = tag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        registerCustomTag(trimmed)
        guard !skillIDs.isEmpty else { return }

        await perform(String(format: String(localized: "正在为 %lld 个 Skill 添加标签…"), skillIDs.count)) {
            for id in skillIDs {
                if let details: SkillDetails = try? await self.core.call("skills.get", params: IDParams(id: id)) {
                    var newTags = details.tags
                    if !newTags.contains(trimmed) {
                        newTags.append(trimmed)
                    }
                    newTags = Array(NSOrderedSet(array: newTags)).compactMap { $0 as? String }
                    let _: SkillUpdateResponse = try await self.core.call(
                        "skills.update",
                        params: UpdateSkillParams(id: id, content: details.content, baseHash: details.hash, tags: newTags)
                    )
                }
            }
            await self.refresh()
        }
        announce(String(localized: "标签已添加并应用"))
    }

    /// 批量为指定 Prompt 打上标签
    func addTagToPrompts(_ tag: String, promptIDs: Set<String>) async {
        let trimmed = tag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        registerCustomTag(trimmed)
        guard !promptIDs.isEmpty else { return }

        await perform(String(format: String(localized: "正在为 %lld 个 Prompt 添加标签…"), promptIDs.count)) {
            for id in promptIDs {
                if let details: PromptDetails = try? await self.core.call("prompts.get", params: IDParams(id: id)) {
                    var newTags = details.tags
                    if !newTags.contains(trimmed) {
                        newTags.append(trimmed)
                    }
                    newTags = Array(NSOrderedSet(array: newTags)).compactMap { $0 as? String }
                    let _: PromptSummary = try await self.core.call(
                        "prompts.update",
                        params: PromptWriteParams(
                            id: id,
                            content: nil,
                            name: details.name,
                            description: details.description,
                            tags: newTags,
                            body: details.body,
                            variables: details.variables ?? [],
                            source: "local",
                            baseHash: details.hash
                        )
                    )
                }
            }
            await self.refresh()
        }
        announce(String(localized: "标签已添加并应用"))
    }

    /// 运行系统 Doctor 健康诊断
    func runDoctor() async {
        await perform(String(localized: "正在运行诊断…")) {
            self.doctorChecks = try await self.core.call("system.doctor", params: EmptyParams())
        }
    }

    /// 检查软件更新（优先 Sparkle，次选 GitHub Releases API）
    func checkForUpdates() async {
        if SparkleUpdater.shared.isConfigured {
            updateStatus = String(localized: "已打开安全更新检查窗口")
            SparkleUpdater.shared.checkForUpdates()
            return
        }
        updateStatus = String(localized: "正在检查更新…")
        do {
            let url = URL(string: "https://api.github.com/repos/zzzzzyijie/skm/releases/latest")!
            var request = URLRequest(url: url)
            request.setValue("application/vnd.github+json", forHTTPHeaderField: "Accept")
            let (data, response) = try await URLSession.shared.data(for: request)
            guard (response as? HTTPURLResponse)?.statusCode == 200 else {
                throw URLError(.badServerResponse)
            }
            let release = try JSONDecoder().decode(GitHubRelease.self, from: data)
            let current = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "0.0.0"
            updateStatus = Self.isVersion(release.tagName, newerThan: current)
                ? String(format: String(localized: "发现新版本 %@"), locale: .current, release.tagName)
                : String(localized: "当前已是最新版本")
        } catch {
            updateStatus = String(format: String(localized: "检查更新失败：%@"), locale: .current, error.localizedDescription)
        }
    }

    /// 语义化版本比对助手（如 v1.2.0 vs 1.1.9）
    static func isVersion(_ candidate: String, newerThan current: String) -> Bool {
        let lhs = candidate.trimmingCharacters(in: CharacterSet(charactersIn: "vV")).split(separator: ".").map { Int($0) ?? 0 }
        let rhs = current.trimmingCharacters(in: CharacterSet(charactersIn: "vV")).split(separator: ".").map { Int($0) ?? 0 }
        for index in 0..<max(lhs.count, rhs.count) {
            let left = index < lhs.count ? lhs[index] : 0
            let right = index < rhs.count ? rhs[index] : 0
            if left != right { return left > right }
        }
        return false
    }

    /// 从 Core 重新拉取所有全量状态，并尽可能维持各业务分区的选中项
    private func reload() async throws {
        let previousSkillID = selectedSkillID
        let previousPromptID = selectedPromptID
        let previousProjectID = selectedProjectID
        let previousAgentID = selectedAgentID
        let previousSourceID = selectedSourceID

        let loadedSkills: [SkillSummary] = try await core.call("skills.list", params: EmptyParams())
        let loadedPrompts: [PromptSummary] = try await core.call("prompts.list", params: EmptyParams())
        let loadedAgents: [AgentModel] = try await core.call("agents.list", params: EmptyParams())
        let loadedPlan: PlanModel = try await core.call("activations.status", params: EmptyParams())
        let loadedSources: [SourceModel] = try await core.call("sources.list", params: EmptyParams())
        let loadedProjects: [ProjectModel] = try await core.call("projects.list", params: EmptyParams())
        let loadedWorkspace: WorkspaceView = try await core.call("workspace.get", params: EmptyParams())
        let loadedDoctor: [DoctorCheck] = try await core.call("system.doctor", params: EmptyParams())

        skills = loadedSkills
        prompts = loadedPrompts
        agents = loadedAgents
        plan = loadedPlan
        sources = loadedSources
        projects = loadedProjects
        workspace = loadedWorkspace
        doctorChecks = loadedDoctor
        hasExistingData = !loadedSkills.isEmpty || !loadedPrompts.isEmpty || !loadedProjects.isEmpty ||
            !loadedSources.isEmpty || loadedWorkspace.configured
        selectedSkillID = loadedSkills.contains(where: { $0.id == previousSkillID }) ? previousSkillID : loadedSkills.first?.id
        selectedPromptID = loadedPrompts.contains(where: { $0.id == previousPromptID }) ? previousPromptID : loadedPrompts.first?.id
        selectedProjectID = loadedProjects.contains(where: { $0.id == previousProjectID }) ? previousProjectID : loadedProjects.first?.id
        selectedAgentID = loadedAgents.contains(where: { $0.id == previousAgentID }) ? previousAgentID : loadedAgents.first?.id
        selectedSourceID = loadedSources.contains(where: { $0.id == previousSourceID }) ? previousSourceID : loadedSources.first?.id
    }

    private func startFileMonitoring() {
        guard monitorsFiles, !isMonitoring else { return }
        let environment = ProcessInfo.processInfo.environment
        let home = environment["SKM_HOME"].flatMap { $0.isEmpty ? nil : $0 }
            ?? FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".skm").path
        fileMonitor.start(paths: [home, (home as NSString).appendingPathComponent("state"), (home as NSString).appendingPathComponent("workspace")]) { [weak self] in
            guard let self, !self.isLoading else { return }
            Task { @MainActor in
                do {
                    try await self.reload()
                    self.announce(String(localized: "检测到 CLI 变更，已刷新"))
                } catch {
                    self.errorMessage = error.localizedDescription
                }
            }
        }
        isMonitoring = true
    }

    private func perform(_ status: String, operation: () async throws -> Void) async {
        isLoading = true
        statusMessage = status
        errorMessage = nil
        defer { isLoading = false }
        do {
            try await operation()
            if statusMessage == status { statusMessage = nil }
        } catch {
            errorMessage = error.localizedDescription
            statusMessage = nil
        }
    }

    private func mutate(success: String, operation: () async throws -> Void) async {
        await perform(String(localized: "正在应用更改…")) {
            try await operation()
            try await self.reload()
            self.announce(success)
        }
    }

    @discardableResult
    private func mutateResult(
        success: String,
        reloads: Bool = true,
        handlesConflict: Bool = false,
        operation: () async throws -> Void
    ) async -> Bool {
        isLoading = true
        statusMessage = String(localized: "正在应用更改…")
        errorMessage = nil
        lastErrorKind = nil
        defer { isLoading = false }
        do {
            try await operation()
            if reloads { try await reload() }
            announce(success)
            return true
        } catch {
            if let coreError = error as? CoreClientError {
                lastErrorKind = coreError.kind
                if handlesConflict && coreError.kind == "conflict" {
                    statusMessage = nil
                    return false
                }
            }
            errorMessage = error.localizedDescription
            statusMessage = nil
            return false
        }
    }

    private var architectureName: String {
#if arch(arm64)
        "arm64"
#elseif arch(x86_64)
        "x86_64"
#else
        "unknown"
#endif
    }

    private static let welcomePreferenceKey = "SKMHasCompletedWelcome"
}

struct IDParams: Codable, Sendable { let id: String }
struct AddSkillParams: Codable, Sendable { let path: String; let tags: [String]; let source: String }
struct AddSourceParams: Codable, Sendable {
    let input: String
    var name: String? = nil
    var ref: String? = nil
    var paths: [String]? = nil
    var tags: [String] = []
}
struct UpdateSkillParams: Codable, Sendable { let id: String; let content: String; let baseHash: String; let tags: [String] }
struct ConfigureAgentsParams: Codable, Sendable { let agents: [String] }
struct CustomAgentParams: Codable, Sendable { let id: String; let name: String; let skillsPath: String }
struct ActivationParams: Codable, Sendable { let skills: [String]; let agents: [String]; let mode: String? }
struct PromptWriteParams: Codable, Sendable {
    let id: String?
    let content: String?
    let name: String
    let description: String
    let tags: [String]
    let body: String
    let variables: [PromptVariable]
    let source: String
    let baseHash: String?
}

struct AddProjectParams: Codable, Sendable { let path: String; let name: String }
struct ProjectDeployParams: Codable, Sendable { let project: String; let skill: String; let agents: [String]; let mode: String; let dryRun: Bool }
struct ProjectUnlinkParams: Codable, Sendable { let project: String; let skill: String; let agents: [String]; let force: Bool }
struct ProjectMigrateParams: Codable, Sendable { let project: String; let skill: String; let agent: String; let mode: String; let removeSource: Bool; let tags: [String] }
struct ProjectRequireParams: Codable, Sendable { let project: String; let skill: String; let agents: [String]; let mode: String; let apply: Bool }
struct ProjectVendorParams: Codable, Sendable { let project: String; let skill: String; let agents: [String]; let mode: String; let tags: [String]; let apply: Bool }
struct ProjectApplyParams: Codable, Sendable { let project: String; let force: Bool }
struct ProjectEntryRemoveParams: Codable, Sendable { let project: String; let entry: String; let force: Bool }
struct PromptRenderParams: Codable, Sendable { let id: String; let values: [String: String] }
struct HistoryParams: Codable, Sendable { let kind: String; let itemId: String }
struct HistoryEntryParams: Codable, Sendable { let kind: String; let itemId: String; let entryId: String }
struct HistoryDiffParams: Codable, Sendable { let kind: String; let itemId: String; let from: String; let to: String }
struct SourceNamesParams: Codable, Sendable { let names: [String] }
struct WorkspaceConfigureParams: Codable, Sendable { let url: String; let ref: String; let root: String? }
struct WorkspaceSyncParams: Codable, Sendable { let resolutions: [String: String] }

private struct GitHubRelease: Codable, Sendable {
    let tagName: String

    enum CodingKeys: String, CodingKey { case tagName = "tag_name" }
}

struct SkillUpdateResponse: Codable, Sendable {
    let skill: MutationSkill
    let plan: PlanModel
    let applied: Bool
    let deploymentError: String?
    let warning: String?
}

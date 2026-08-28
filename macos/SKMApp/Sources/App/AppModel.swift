import AppKit
import Foundation
import Observation

enum AppSection: String, CaseIterable, Identifiable {
    case skills = "Skills"
    case prompts = "Prompts"
    case projects = "Projects"
    case sources = "Sources"
    case workspace = "Workspace"
    case agents = "Agents"
    case diagnostics = "Diagnostics"

    var id: String { rawValue }

    var symbol: String {
        switch self {
        case .skills: "square.stack.3d.up"
        case .prompts: "text.bubble"
        case .projects: "folder"
        case .sources: "arrow.triangle.branch"
        case .workspace: "icloud"
        case .agents: "cpu"
        case .diagnostics: "stethoscope"
        }
    }
}

enum AppCommandKind: Equatable, Sendable {
    case create
    case importItem
    case deleteSelection
}

struct AppCommand: Identifiable, Sendable {
    let id = UUID()
    let kind: AppCommandKind
    let section: AppSection
}

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

    func retryStart() async {
        await core.stop()
        handshake = nil
        await start()
    }

    func refresh() async {
        await perform(String(localized: "正在刷新…")) { try await self.reload() }
    }

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

    func stop() async {
        fileMonitor.stop()
        isMonitoring = false
        await core.stop()
    }

    func completeWelcome(openAgents: Bool = false) {
        preferences.set(true, forKey: Self.welcomePreferenceKey)
        showsWelcome = false
        if openAgents { section = .agents }
    }

    func request(_ kind: AppCommandKind) {
        pendingCommand = AppCommand(kind: kind, section: section)
    }

    func consumeCommand(_ id: UUID) {
        guard pendingCommand?.id == id else { return }
        pendingCommand = nil
    }

    var canCreate: Bool { section != .workspace && section != .diagnostics }

    var canImport: Bool { section == .skills || section == .prompts || section == .projects }

    var canDeleteSelection: Bool {
        switch section {
        case .skills: return selectedSkillID != nil
        case .prompts: return selectedPromptID != nil
        case .projects: return selectedProjectID != nil
        case .sources: return selectedSourceID != nil
        case .workspace, .diagnostics: return false
        case .agents:
            guard let selectedAgentID else { return false }
            return agents.first(where: { $0.id == selectedAgentID })?.custom == true
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

    func skillDetails(_ id: String) async throws -> SkillDetails {
        try await core.call("skills.get", params: IDParams(id: id))
    }

    func promptDetails(_ id: String) async throws -> PromptDetails {
        try await core.call("prompts.get", params: IDParams(id: id))
    }

    func addLocalSkill(path: String, tags: [String]) async {
        await mutate(success: String(localized: "Skill 已导入")) {
            let _: MutationSkill = try await self.core.call("skills.add", params: AddSkillParams(path: path, tags: tags, source: "local"))
        }
    }

    func addRemoteSkill(input: String) async {
        await mutate(success: String(localized: "Git Source 已导入")) {
            let _: AddSourceResponse = try await self.core.call("sources.add", params: AddSourceParams(input: input, tags: []))
        }
    }

    @discardableResult
    func updateSkill(id: String, content: String, baseHash: String, tags: [String]) async -> Bool {
        await mutateResult(success: String(localized: "Skill 已保存并重新部署"), handlesConflict: true) {
            let _: SkillUpdateResponse = try await self.core.call("skills.update", params: UpdateSkillParams(id: id, content: content, baseHash: baseHash, tags: tags))
        }
    }

    func removeSkill(id: String) async {
        await mutate(success: String(localized: "Skill 已移除")) {
            let _: MutationSkill = try await self.core.call("skills.remove", params: IDParams(id: id))
            self.selectedSkillID = nil
        }
    }

    func setSkill(_ skillID: String, agentID: String, enabled: Bool) async {
        await mutate(success: enabled ? String(localized: "已为 Agent 启用") : String(localized: "已停用")) {
            if enabled {
                let _: PlanModel = try await self.core.call("activations.enable", params: ActivationParams(skills: [skillID], agents: [agentID], mode: "auto"))
            } else {
                let _: StatusResponse = try await self.core.call("activations.disable", params: ActivationParams(skills: [skillID], agents: [agentID], mode: nil))
            }
        }
    }

    func isEnabled(_ skillID: String, for agentID: String) -> Bool {
        plan.operations.contains { $0.skillId == skillID && $0.agent == agentID && $0.placement == "user" }
    }

    func configureAgent(_ id: String, enabled: Bool) async {
        var selected = Set(agents.filter(\.configured).map(\.id))
        if enabled { selected.insert(id) } else { selected.remove(id) }
        await mutate(success: String(localized: "Agent 配置已更新")) {
            let _: [AgentModel] = try await self.core.call("agents.configure", params: ConfigureAgentsParams(agents: Array(selected).sorted()))
        }
    }

    func saveCustomAgent(id: String, name: String, path: String) async {
        await mutate(success: String(localized: "自定义 Agent 已保存")) {
            let _: [AgentModel] = try await self.core.call("agents.custom.save", params: CustomAgentParams(id: id, name: name, skillsPath: path))
        }
    }

    func deleteCustomAgent(id: String) async {
        await mutate(success: String(localized: "自定义 Agent 已删除")) {
            let _: StatusResponse = try await self.core.call("agents.custom.delete", params: IDParams(id: id))
            self.selectedAgentID = nil
        }
    }

    @discardableResult
    func savePrompt(id: String?, name: String, description: String, body: String, tags: [String], baseHash: String?) async -> Bool {
        await mutateResult(success: id == nil ? String(localized: "Prompt 已创建") : String(localized: "Prompt 已保存"), handlesConflict: true) {
            let params = PromptWriteParams(id: id, content: nil, name: name, description: description, tags: tags, body: body, source: "local", baseHash: baseHash)
            let _: PromptSummary = try await self.core.call(id == nil ? "prompts.create" : "prompts.update", params: params)
        }
    }

    func importPrompt(content: String) async {
        await mutate(success: String(localized: "Prompt 已导入")) {
            let params = PromptWriteParams(id: nil, content: content, name: "", description: "", tags: [], body: "", source: "local", baseHash: nil)
            let _: PromptSummary = try await self.core.call("prompts.create", params: params)
        }
    }

    func removePrompt(id: String) async {
        await mutate(success: String(localized: "Prompt 已移除")) {
            let _: PromptSummary = try await self.core.call("prompts.remove", params: IDParams(id: id))
            self.selectedPromptID = nil
        }
    }

    func loadProjectDetails(_ id: String) async {
        await perform(String(localized: "正在扫描项目…")) {
            self.projectDetails = try await self.core.call("projects.get", params: IDParams(id: id))
        }
    }

    func addProject(path: String, name: String) async {
        await mutate(success: String(localized: "项目已登记")) {
            let _: RegisteredProject = try await self.core.call("projects.add", params: AddProjectParams(path: path, name: name))
        }
    }

    func unregisterProject(id: String) async {
        await mutate(success: String(localized: "项目已注销")) {
            let _: RegisteredProject = try await self.core.call("projects.unregister", params: IDParams(id: id))
            self.selectedProjectID = nil
            self.projectDetails = nil
        }
    }

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

    func unlinkProject(project: String, skill: String, agents: [String]) async {
        let succeeded = await mutateResult(success: String(localized: "项目 Skill 已解绑")) {
            let _: StatusResponse = try await self.core.call(
                "projects.unlink",
                params: ProjectUnlinkParams(project: project, skill: skill, agents: agents, force: false)
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    func migrateProjectSkill(project: String, skill: String, agent: String, mode: String, removeSource: Bool) async {
        let succeeded = await mutateResult(success: String(localized: "项目 Skill 已迁移")) {
            let _: ProjectMigrateResponse = try await self.core.call(
                "projects.migrate",
                params: ProjectMigrateParams(project: project, skill: skill, agent: agent, mode: mode, removeSource: removeSource, tags: [])
            )
        }
        if succeeded { await loadProjectDetails(project) }
    }

    func addSource(input: String) async {
        await mutate(success: String(localized: "Git Source 已添加")) {
            let _: AddSourceResponse = try await self.core.call("sources.add", params: AddSourceParams(input: input, tags: []))
        }
    }

    func updateSource(name: String) async {
        await mutate(success: String(localized: "Git Source 已更新")) {
            let _: SourceUpdateResponse = try await self.core.call("sources.update", params: SourceNamesParams(names: [name]))
        }
    }

    func removeSource(name: String) async {
        await mutate(success: String(localized: "Git Source 已移除")) {
            let _: SourceRemovalResponse = try await self.core.call("sources.remove", params: IDParams(id: name))
            self.selectedSourceID = nil
        }
    }

    func syncSources() async {
        await mutate(success: String(localized: "所有 Git Sources 已同步")) {
            self.sourceSyncResult = try await self.core.call("sources.sync", params: EmptyParams())
        }
    }

    func configureWorkspace(url: String, ref: String, root: String) async {
        await mutate(success: String(localized: "个人工作区已配置")) {
            let _: WorkspaceView = try await self.core.call(
                "workspace.configure",
                params: WorkspaceConfigureParams(url: url, ref: ref, root: root.isEmpty ? nil : root)
            )
        }
    }

    func previewWorkspace() async {
        await perform(String(localized: "正在预览同步…")) {
            self.workspacePreview = try await self.core.call("workspace.preview", params: EmptyParams())
            self.workspaceResolutions = [:]
        }
    }

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

    func runDoctor() async {
        await perform(String(localized: "正在运行诊断…")) {
            self.doctorChecks = try await self.core.call("system.doctor", params: EmptyParams())
        }
    }

    func checkForUpdates() async {
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
struct AddSourceParams: Codable, Sendable { let input: String; let tags: [String] }
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
    let source: String
    let baseHash: String?
}

struct AddProjectParams: Codable, Sendable { let path: String; let name: String }
struct ProjectDeployParams: Codable, Sendable { let project: String; let skill: String; let agents: [String]; let mode: String; let dryRun: Bool }
struct ProjectUnlinkParams: Codable, Sendable { let project: String; let skill: String; let agents: [String]; let force: Bool }
struct ProjectMigrateParams: Codable, Sendable { let project: String; let skill: String; let agent: String; let mode: String; let removeSource: Bool; let tags: [String] }
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

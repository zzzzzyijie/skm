import AppKit
import Foundation
import Observation

enum AppSection: String, CaseIterable, Identifiable {
    case skills = "Skills"
    case prompts = "Prompts"
    case projects = "Projects"
    case agents = "Agents"

    var id: String { rawValue }

    var symbol: String {
        switch self {
        case .skills: "square.stack.3d.up"
        case .prompts: "text.bubble"
        case .projects: "folder"
        case .agents: "cpu"
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
    var plan = PlanModel(digest: "", operations: [])
    var selectedSkillID: String?
    var selectedPromptID: String?
    var selectedProjectID: String?
    var selectedAgentID: String?
    var handshake: Handshake?
    var isLoading = false
    var errorMessage: String?
    var startupErrorMessage: String?
    var statusMessage: String?
    var showsWelcome = false
    var hasExistingData = false
    var pendingCommand: AppCommand?

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

    var canCreate: Bool { section != .projects }

    var canImport: Bool { section == .skills || section == .prompts }

    var canDeleteSelection: Bool {
        switch section {
        case .skills: return selectedSkillID != nil
        case .prompts: return selectedPromptID != nil
        case .projects: return false
        case .agents:
            guard let selectedAgentID else { return false }
            return agents.first(where: { $0.id == selectedAgentID })?.custom == true
        }
    }

    var diagnosticText: String {
        let appVersion = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
        let systemVersion = ProcessInfo.processInfo.operatingSystemVersionString
        return [
            "SKM diagnostics",
            "App: \(appVersion)",
            "Core: \(handshake?.coreVersion ?? "unavailable")",
            "Protocol: \(handshake?.protocolVersion.description ?? "unavailable")",
            "macOS: \(systemVersion)",
            "Architecture: \(architectureName)",
            "Error: \(startupErrorMessage ?? errorMessage ?? "none")",
        ].joined(separator: "\n")
    }

    func copyDiagnostics() {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(diagnosticText, forType: .string)
        announce(String(localized: "诊断信息已复制"))
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

    func updateSkill(id: String, content: String, baseHash: String, tags: [String]) async {
        await mutate(success: String(localized: "Skill 已保存并重新部署")) {
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

    func savePrompt(id: String?, name: String, description: String, body: String, tags: [String], baseHash: String?) async {
        await mutate(success: id == nil ? String(localized: "Prompt 已创建") : String(localized: "Prompt 已保存")) {
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

    private func reload() async throws {
        let previousSkillID = selectedSkillID
        let previousPromptID = selectedPromptID
        let previousProjectID = selectedProjectID
        let previousAgentID = selectedAgentID

        let loadedSkills: [SkillSummary] = try await core.call("skills.list", params: EmptyParams())
        let loadedPrompts: [PromptSummary] = try await core.call("prompts.list", params: EmptyParams())
        let loadedAgents: [AgentModel] = try await core.call("agents.list", params: EmptyParams())
        let loadedPlan: PlanModel = try await core.call("activations.status", params: EmptyParams())
        let loadedSources: [SourceModel] = try await core.call("sources.list", params: EmptyParams())
        let loadedProjects: [ProjectModel] = try await core.call("projects.list", params: EmptyParams())
        let loadedWorkspace: WorkspaceView = try await core.call("workspace.get", params: EmptyParams())

        skills = loadedSkills
        prompts = loadedPrompts
        agents = loadedAgents
        plan = loadedPlan
        sources = loadedSources
        projects = loadedProjects
        workspace = loadedWorkspace
        hasExistingData = !loadedSkills.isEmpty || !loadedPrompts.isEmpty || !loadedProjects.isEmpty ||
            !loadedSources.isEmpty || loadedWorkspace.configured
        selectedSkillID = loadedSkills.contains(where: { $0.id == previousSkillID }) ? previousSkillID : loadedSkills.first?.id
        selectedPromptID = loadedPrompts.contains(where: { $0.id == previousPromptID }) ? previousPromptID : loadedPrompts.first?.id
        selectedProjectID = loadedProjects.contains(where: { $0.id == previousProjectID }) ? previousProjectID : loadedProjects.first?.id
        selectedAgentID = loadedAgents.contains(where: { $0.id == previousAgentID }) ? previousAgentID : loadedAgents.first?.id
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

struct SkillUpdateResponse: Codable, Sendable {
    let skill: MutationSkill
    let plan: PlanModel
    let applied: Bool
    let deploymentError: String?
    let warning: String?
}

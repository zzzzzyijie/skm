import Foundation

/// 空参数载荷
struct EmptyParams: Codable, Sendable {}

/// 与 Go Core 初始握手响应
/// 包含协议版本、核心版本、各类 Schema 架构版本以及支持的能力集
struct Handshake: Codable, Sendable {
    let protocolVersion: Int
    let coreVersion: String
    let schemaVersion: Int
    let promptSchemaVersion: Int
    let workspaceSchemaVersion: Int
    let capabilities: [String]
}

/// Skill 概要模型（列表展示用）
struct SkillSummary: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let description: String
    let tags: [String]
    let source: String
    let location: String
    let hash: String
    let path: String
    /// 健康状态："ok"、"warning"、"error"
    let health: String
    let healthDetail: String?
    let usingFallback: Bool?
    let effectivePath: String
    let editable: Bool
    let editReason: String?
}

/// Skill 详情模型（包含完整 Markdown 正文）
struct SkillDetails: Codable, Identifiable, Sendable {
    let id: String
    let name: String
    let description: String
    let tags: [String]
    let source: String
    let hash: String
    let path: String
    let health: String
    let healthDetail: String?
    let effectivePath: String
    let editable: Bool
    let editReason: String?
    /// 完整 SKILL.md 内容（含 Frontmatter）
    let content: String
    /// 去除 Frontmatter 后的正文
    let body: String
}

/// Agent 目标适配器模型（如 Claude Desktop、Codex 等）
struct AgentModel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let path: String?
    let format: String?
    /// 是否在 SKM 中已勾选配置
    let configured: Bool
    /// 是否在本机检测到对应客户端已安装
    let detected: Bool
    let supported: Bool
    let note: String?
    let icon: String?
    /// 是否为用户手动添加的自定义 Agent
    let custom: Bool
}

/// Prompt 模板中的参数化变量定义
struct PromptVariable: Codable, Hashable, Sendable {
    let name: String
    let label: String?
    let type: String?
    let required: Bool?
    let `default`: String?
    let options: [String]?
    let description: String?
}

/// Prompt 概要模型
struct PromptSummary: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let description: String
    let tags: [String]
    let source: String
    let hash: String
    let path: String
    let variables: [PromptVariable]?
}

/// Prompt 详情模型
struct PromptDetails: Codable, Identifiable, Sendable {
    let id: String
    let name: String
    let description: String
    let tags: [String]
    let source: String
    let hash: String
    let path: String
    let variables: [PromptVariable]?
    let content: String
    let body: String
}

/// 部署或同步计划中的单项操作
struct OperationModel: Codable, Hashable, Sendable {
    let status: String
    let skillId: String
    let name: String
    let agent: String
    let placement: String
    let projectRoot: String?
    let target: String
    let sourcePath: String
    let mode: String
    let hash: String
    let message: String?
}

/// 部署与激活计划模型
struct PlanModel: Codable, Sendable {
    let digest: String
    let operations: [OperationModel]
}

/// Git 技能来源模型
struct SourceModel: Codable, Identifiable, Hashable, Sendable {
    var id: String { name }
    let name: String
    let url: String
    let ref: String?
    let paths: [String]?
    let tags: [String]
    let revision: String?
}

/// 已登记的项目模型
struct ProjectModel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let path: String
    let exists: Bool
    let access: String?
    let accessMessage: String?
    let activationCount: Int
    let skillCount: Int
    let agentCounts: [String: Int]
}

/// 项目/全局激活关联
struct ActivationModel: Codable, Hashable, Sendable {
    let skillId: String
    let name: String
    let placement: String
    let projectRoot: String?
    let agents: [String]
    let mode: String
}

/// 项目扫描到的 Agent 状态
struct ProjectScanAgent: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let label: String
    let skillCount: Int
    let available: Bool
}

/// 项目扫描到的单个 Skill 状态及关联
struct ProjectScanSkill: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let description: String?
    let agents: [String]
    let paths: [String: String]
    let hash: String?
    let librarySkillId: String?
    let status: String
    let issues: [String]?
}

/// 项目扫描报告（包含 Agent 分布与技能清单）
struct ProjectScan: Codable, Sendable {
    let scannedAt: String
    let skillCount: Int
    let agentCounts: [String: Int]
    let agents: [ProjectScanAgent]
    let skills: [ProjectScanSkill]
    let errors: [String]?
}

/// 项目详情（含注册信息、清单 Manifest、激活状态和扫描结果）
struct ProjectDetails: Codable, Sendable {
    let project: RegisteredProject
    let exists: Bool
    let activations: [ActivationModel]
    let manifest: ProjectManifestModel
    let scan: ProjectScan
    let plan: PlanModel
}

/// 项目清单模型（对应项目内 skm.project.json 或 lock 状态）
struct ProjectManifestModel: Codable, Sendable {
    let version: Int
    let skills: [MutationSkill]
    let dependencies: [ProjectDependencyModel]?
}

/// 项目显式固定的依赖模型（Require 模式）
struct ProjectDependencyModel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let source: String
    let url: String
    let ref: String?
    let sourcePath: String?
    let revision: String
    let hash: String
    let tags: [String]
    let agents: [String]
    let mode: String
}

/// 已注册的项目基本信息
struct RegisteredProject: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let path: String
}

/// 部署到项目的返回结果
struct ProjectDeployResponse: Codable, Sendable {
    let project: RegisteredProject
    let skill: MutationSkill
    let plan: PlanModel
    let applied: Bool
}

/// 项目 Skill 迁移结果
struct ProjectMigrateResponse: Codable, Sendable {
    let project: RegisteredProject
    let skill: MutationSkill
    let mode: String
    let removedPaths: [String]
}

/// 项目高级操作（Require/Vendor/Apply）返回结果
struct ProjectAdvancedResponse: Codable, Sendable {
    let project: RegisteredProject
    let manifest: ProjectManifestModel
    let dependency: ProjectDependencyModel?
    let skill: MutationSkill?
    let plan: PlanModel
    let satisfiedByUser: [String]
    let applied: Bool
    let removedId: String?
}

/// Prompt 动态渲染结果（包含填充后的 Markdown 及缺失变量列表）
struct PromptRenderResponse: Codable, Sendable {
    let content: String
    let missingVariables: [String]
}

/// 历史版本快照条目（Skill 或 Prompt 的修改记录）
struct HistoryEntryModel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let itemId: String
    let kind: String
    let hash: String
    let createdAt: String
    let reason: String
    let current: Bool?
    let content: String?
}

/// 两个版本之间的 Diff 结果
struct HistoryDiffResponse: Codable, Sendable {
    let from: String
    let to: String
    let diff: String
}

/// 历史版本回滚结果
struct HistoryRollbackResponse: Codable, Sendable {
    let entry: HistoryEntryModel
    let skill: SkillUpdateResponse?
    let prompt: PromptSummary?
}

/// 个人 Git 工作区（Workspace）视图状态
struct WorkspaceView: Codable, Sendable {
    let configured: Bool
    let config: WorkspaceConfig?
    let state: WorkspaceState?
}

/// Workspace 配置参数（绑定远端仓库 URL、分支与子目录）
struct WorkspaceConfig: Codable, Sendable {
    let version: Int
    let url: String
    let ref: String
    let root: String?
}

/// Workspace 本地状态（记录最新提交 Revision 与同步时间）
struct WorkspaceState: Codable, Sendable {
    let version: Int
    let revision: String?
    let lastSyncedAt: String?
}

/// Workspace 同步检测到的单项变更（包含增删改与冲突）
struct WorkspaceChange: Codable, Identifiable, Hashable, Sendable {
    var id: String { "\(kind):\(itemID)" }
    let kind: String
    private let itemID: String
    let name: String
    let action: String
    let reason: String?
    let localHash: String?
    let remoteHash: String?
    let baseHash: String?
    let detail: String?

    enum CodingKeys: String, CodingKey {
        case kind, name, action, reason, localHash, remoteHash, baseHash, detail
        case itemID = "id"
    }
}

/// Workspace 同步预览模型
struct WorkspacePreview: Codable, Sendable {
    let configured: Bool
    let config: WorkspaceConfig?
    let baseRevision: String?
    let remoteRevision: String?
    let skills: Int
    let prompts: Int
    let sources: Int
    let uploads: Int
    let downloads: Int
    let deletes: Int
    let conflicts: Int
    let changes: [WorkspaceChange]
    let lastSyncedAt: String?
    let remoteAvailable: Bool
}

/// Workspace 执行同步后的响应
struct WorkspaceSyncResponse: Codable, Sendable {
    let preview: WorkspacePreview
    let revision: String
    let committed: Bool
    let applied: Bool
    let deploymentError: String?
    let sourceWarnings: [String]?
    let syncedAt: String
    let plan: PlanModel?
}

/// 更新 Git 来源的响应
struct SourceUpdateResponse: Codable, Sendable {
    let sources: [SourceModel]
    let skills: [MutationSkill]
}

/// 移除 Git 来源的响应
struct SourceRemovalResponse: Codable, Sendable {
    let name: String
    let source: SourceModel?
    let bindingRemoved: Bool
    let checkoutPath: String?
    let checkoutRemoved: Bool
}

/// 批量同步单个 Source 的结果条目
struct SourceSyncItem: Codable, Identifiable, Sendable {
    var id: String { name }
    let name: String
    let status: String
    let source: SourceModel?
    let skillCount: Int
    let error: String?
}

/// 批量同步所有 Git 来源的总体响应
struct SourceSyncResponse: Codable, Sendable {
    let configured: Int
    let updated: Int
    let failed: Int
    let skillCount: Int
    let results: [SourceSyncItem]
    let plan: PlanModel?
    let applied: Bool
    let deploymentError: String?
    let syncedAt: String
}

/// 系统 Doctor 检查项
struct DoctorCheck: Codable, Identifiable, Hashable, Sendable {
    var id: String { name }
    let name: String
    let status: String
    let message: String
}

/// 通用状态响应
struct StatusResponse: Codable, Sendable {
    let status: String
}

/// 变更 Skill 的基本标识
struct MutationSkill: Codable, Identifiable, Sendable {
    let id: String
    let name: String
}

/// 添加 Git 来源的响应
struct AddSourceResponse: Codable, Sendable {
    let source: SourceModel
    let skills: [MutationSkill]
}

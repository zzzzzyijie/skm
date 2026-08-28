import Foundation

struct EmptyParams: Codable, Sendable {}

struct Handshake: Codable, Sendable {
    let protocolVersion: Int
    let coreVersion: String
    let schemaVersion: Int
    let promptSchemaVersion: Int
    let workspaceSchemaVersion: Int
    let capabilities: [String]
}

struct SkillSummary: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let description: String
    let tags: [String]
    let source: String
    let location: String
    let hash: String
    let path: String
    let health: String
    let healthDetail: String?
    let usingFallback: Bool?
    let effectivePath: String
    let editable: Bool
    let editReason: String?
}

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
    let content: String
    let body: String
}

struct AgentModel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let name: String
    let path: String?
    let format: String?
    let configured: Bool
    let detected: Bool
    let supported: Bool
    let note: String?
    let icon: String?
    let custom: Bool
}

struct PromptVariable: Codable, Hashable, Sendable {
    let name: String
    let label: String?
    let type: String?
    let required: Bool?
    let `default`: String?
    let options: [String]?
    let description: String?
}

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

struct PlanModel: Codable, Sendable {
    let digest: String
    let operations: [OperationModel]
}

struct SourceModel: Codable, Identifiable, Hashable, Sendable {
    var id: String { name }
    let name: String
    let url: String
    let ref: String?
    let paths: [String]?
    let tags: [String]
    let revision: String?
}

struct ProjectModel: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let path: String
    let exists: Bool
    let activationCount: Int
    let skillCount: Int
    let agentCounts: [String: Int]
}

struct ActivationModel: Codable, Hashable, Sendable {
    let skillId: String
    let name: String
    let placement: String
    let projectRoot: String?
    let agents: [String]
    let mode: String
}

struct ProjectScanAgent: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let label: String
    let skillCount: Int
    let available: Bool
}

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

struct ProjectScan: Codable, Sendable {
    let scannedAt: String
    let skillCount: Int
    let agentCounts: [String: Int]
    let agents: [ProjectScanAgent]
    let skills: [ProjectScanSkill]
    let errors: [String]?
}

struct ProjectDetails: Codable, Sendable {
    let project: RegisteredProject
    let exists: Bool
    let activations: [ActivationModel]
    let scan: ProjectScan
    let plan: PlanModel
}

struct RegisteredProject: Codable, Identifiable, Hashable, Sendable {
    let id: String
    let path: String
}

struct ProjectDeployResponse: Codable, Sendable {
    let project: RegisteredProject
    let skill: MutationSkill
    let plan: PlanModel
    let applied: Bool
}

struct ProjectMigrateResponse: Codable, Sendable {
    let project: RegisteredProject
    let skill: MutationSkill
    let mode: String
    let removedPaths: [String]
}

struct WorkspaceView: Codable, Sendable {
    let configured: Bool
    let config: WorkspaceConfig?
    let state: WorkspaceState?
}

struct WorkspaceConfig: Codable, Sendable {
    let version: Int
    let url: String
    let ref: String
    let root: String?
}

struct WorkspaceState: Codable, Sendable {
    let version: Int
    let revision: String?
    let lastSyncedAt: String?
}

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

struct SourceUpdateResponse: Codable, Sendable {
    let sources: [SourceModel]
    let skills: [MutationSkill]
}

struct SourceRemovalResponse: Codable, Sendable {
    let name: String
    let source: SourceModel?
    let bindingRemoved: Bool
    let checkoutPath: String?
    let checkoutRemoved: Bool
}

struct SourceSyncItem: Codable, Identifiable, Sendable {
    var id: String { name }
    let name: String
    let status: String
    let source: SourceModel?
    let skillCount: Int
    let error: String?
}

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

struct DoctorCheck: Codable, Identifiable, Hashable, Sendable {
    var id: String { name }
    let name: String
    let status: String
    let message: String
}

struct StatusResponse: Codable, Sendable {
    let status: String
}

struct MutationSkill: Codable, Identifiable, Sendable {
    let id: String
    let name: String
}

struct AddSourceResponse: Codable, Sendable {
    let source: SourceModel
    let skills: [MutationSkill]
}

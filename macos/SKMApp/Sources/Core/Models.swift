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

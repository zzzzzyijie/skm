import SwiftUI

/// AgentsSettingsView - Agent 管理设置页面
/// 控制允许 SKM 部署 Skills 的 AI 客户端（如 Claude Desktop、Codex、Cursor、Windsurf 等），
/// 并支持添加、编辑与删除指向特定目录的自定义 Agent 适配器。
struct AgentsSettingsView: View {
    @Bindable var model: AppModel
    @State private var showsEditor = false
    @State private var editingAgent: AgentModel?
    @State private var agentToDelete: AgentModel?
    @State private var confirmsDelete = false

    private var configuredCount: Int {
        model.agents.filter(\.configured).count
    }

    private var sortedAgents: [AgentModel] {
        model.agents.sorted { lhs, rhs in
            if lhs.configured != rhs.configured {
                return lhs.configured && !rhs.configured
            }
            if lhs.detected != rhs.detected {
                return lhs.detected && !rhs.detected
            }
            return lhs.name.localizedStandardCompare(rhs.name) == .orderedAscending
        }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                HStack(alignment: .firstTextBaseline) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(String(format: String(localized: "Agent 管理 (%lld)"), locale: .current, configuredCount))
                            .font(.largeTitle.bold())
                        Text("选择 SKM 可以部署 Skill 的工具，并管理自定义 Agent 路径。已勾选启用的 Agent 会置顶显示。")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("添加自定义 Agent", systemImage: "plus") {
                        editingAgent = nil
                        showsEditor = true
                    }
                    .buttonStyle(.borderedProminent)
                }

                if model.agents.isEmpty {
                    ContentUnavailableView(
                        "没有可用的 Agent",
                        systemImage: "cpu",
                        description: Text("可以添加一个自定义 Agent，或安装受支持的工具后重新刷新。")
                    )
                    .frame(minHeight: 260)
                } else {
                    LazyVStack(spacing: 10) {
                        ForEach(sortedAgents) { agent in
                            AgentSettingsRow(
                                agent: agent,
                                onToggle: { enabled in
                                    Task { await model.configureAgent(agent.id, enabled: enabled) }
                                },
                                onEdit: {
                                    editingAgent = agent
                                    showsEditor = true
                                },
                                onDelete: {
                                    agentToDelete = agent
                                    confirmsDelete = true
                                }
                            )
                        }
                    }
                }
            }
            .padding(26)
            .frame(maxWidth: 820, alignment: .leading)
        }
        .navigationTitle("Agent 管理")
        .sheet(isPresented: $showsEditor, onDismiss: { editingAgent = nil }) {
            CustomAgentSheet(model: model, agent: editingAgent)
        }
        .confirmationDialog(
            String(format: String(localized: "删除 %@？"), locale: .current, agentToDelete?.name ?? ""),
            isPresented: $confirmsDelete
        ) {
            if let agentToDelete {
                Button("删除自定义 Agent", role: .destructive) {
                    Task { await model.deleteCustomAgent(id: agentToDelete.id) }
                }
            }
        } message: {
            Text("如果仍有 Skill 在此 Agent 中启用，Core 会拒绝删除。")
        }
    }
}

private struct AgentSettingsRow: View {
    let agent: AgentModel
    let onToggle: (Bool) -> Void
    let onEdit: () -> Void
    let onDelete: () -> Void

    var body: some View {
        HStack(spacing: 14) {
            Image(systemName: agent.custom ? "cpu.fill" : "cpu")
                .font(.title3)
                .foregroundStyle(agent.detected ? Color.accentColor : .secondary)
                .frame(width: 38, height: 38)
                .background(.quaternary, in: RoundedRectangle(cornerRadius: 9))

            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 7) {
                    Text(agent.name).fontWeight(.semibold)
                    if agent.custom {
                        Text("自定义")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
                Text(agent.path ?? String(localized: "未提供 Skill 路径"))
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                Text(agent.detected ? String(localized: "已检测") : String(localized: "未检测"))
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }

            Spacer(minLength: 12)

            Toggle("允许 SKM 向此 Agent 部署 Skill", isOn: Binding(
                get: { agent.configured },
                set: onToggle
            ))
            .labelsHidden()
            .accessibilityLabel("允许 SKM 向此 Agent 部署 Skill")
            .accessibilityIdentifier("agent-management-\(agent.id)")
            .disabled(!agent.supported)

            if agent.custom {
                Menu("更多", systemImage: "ellipsis.circle") {
                    Button("编辑", systemImage: "pencil", action: onEdit)
                    Button("删除", systemImage: "trash", role: .destructive, action: onDelete)
                }
                .menuStyle(.borderlessButton)
                .fixedSize()
            }
        }
        .padding(14)
        .background(.quaternary.opacity(0.45), in: RoundedRectangle(cornerRadius: 12))
    }
}

struct AgentsListView: View {
    @Bindable var model: AppModel
    @State private var showsCustomAgent = false

    private var sortedAgents: [AgentModel] {
        model.agents.sorted { lhs, rhs in
            if lhs.configured != rhs.configured {
                return lhs.configured && !rhs.configured
            }
            if lhs.detected != rhs.detected {
                return lhs.detected && !rhs.detected
            }
            return lhs.name.localizedStandardCompare(rhs.name) == .orderedAscending
        }
    }

    var body: some View {
        List(sortedAgents, selection: $model.selectedAgentID) { agent in
            HStack(spacing: 10) {
                Image(systemName: agent.custom ? "cpu.fill" : "cpu")
                    .foregroundStyle(agent.detected ? Color.accentColor : .secondary)
                    .frame(width: 20)
                VStack(alignment: .leading, spacing: 3) {
                    Text(agent.name).fontWeight(.medium)
                    Text(agent.configured
                         ? String(localized: "已管理")
                         : (agent.detected ? String(localized: "已检测") : String(localized: "未检测")))
                        .font(.caption).foregroundStyle(.secondary)
                }
                Spacer()
                if agent.configured { Image(systemName: "checkmark.circle.fill").foregroundStyle(.green) }
            }
            .padding(.vertical, 4)
            .tag(agent.id)
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(agentAccessibilityLabel(agent))
        }
        .navigationTitle("Agents")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("添加自定义 Agent", systemImage: "plus") { showsCustomAgent = true }
            }
        }
        .sheet(isPresented: $showsCustomAgent) { CustomAgentSheet(model: model, agent: nil) }
    }
}

struct AgentDetailView: View {
    @Bindable var model: AppModel
    @State private var showsEditor = false
    @State private var confirmsDelete = false

    var body: some View {
        if let id = model.selectedAgentID, let agent = model.agents.first(where: { $0.id == id }) {
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    HStack(spacing: 16) {
                        Image(systemName: agent.custom ? "cpu.fill" : "cpu")
                            .font(.system(size: 42, weight: .medium))
                            .foregroundStyle(Color.accentColor)
                            .frame(width: 64, height: 64)
                            .background(.quaternary, in: RoundedRectangle(cornerRadius: 14))
                        VStack(alignment: .leading, spacing: 5) {
                            Text(agent.name).font(.largeTitle.bold())
                            Text(agent.custom ? String(localized: "自定义 Agent") : String(localized: "内置 Agent"))
                                .foregroundStyle(.secondary)
                        }
                    }
                    GroupBox("管理") {
                        Toggle("允许 SKM 向此 Agent 部署 Skill", isOn: Binding(
                            get: { agent.configured },
                            set: { value in Task { await model.configureAgent(agent.id, enabled: value) } }
                        ))
                        .disabled(model.isLoading)
                        .padding(4)
                    }
                    GroupBox("路径") {
                        VStack(alignment: .leading, spacing: 10) {
                            LabeledContent("Skill 目录", value: agent.path ?? String(localized: "未提供"))
                            LabeledContent("格式", value: agent.format ?? "—")
                            LabeledContent(
                                "本机检测",
                                value: agent.detected ? String(localized: "已检测") : String(localized: "未检测")
                            )
                        }
                        .textSelection(.enabled)
                        .padding(4)
                    }
                    if let note = agent.note { Label(note, systemImage: "info.circle").foregroundStyle(.secondary) }
                }
                .padding(26)
                .frame(maxWidth: 760, alignment: .leading)
            }
            .toolbar {
                if agent.custom {
                    ToolbarItemGroup(placement: .primaryAction) {
                        Button("编辑", systemImage: "pencil") { showsEditor = true }
                        Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    }
                }
            }
            .sheet(isPresented: $showsEditor) { CustomAgentSheet(model: model, agent: agent) }
            .confirmationDialog("删除 \(agent.name)？", isPresented: $confirmsDelete) {
                Button("删除自定义 Agent", role: .destructive) { Task { await model.deleteCustomAgent(id: id) } }
            } message: {
                Text("如果仍有 Skill 在此 Agent 中启用，Core 会拒绝删除。")
            }
        } else {
            ContentUnavailableView("选择一个 Agent", systemImage: "cpu")
        }
    }
}

struct CustomAgentSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let agent: AgentModel?
    @State private var id: String
    @State private var name: String
    @State private var path: String

    init(model: AppModel, agent: AgentModel?) {
        self.model = model
        self.agent = agent
        _id = State(initialValue: agent?.id ?? "")
        _name = State(initialValue: agent?.name ?? "")
        _path = State(initialValue: agent?.path?.replacingOccurrences(of: "/<skill-name>", with: "") ?? "~/.agent/skills")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text(agent == nil ? String(localized: "添加自定义 Agent") : String(localized: "编辑自定义 Agent"))
                .font(.title2.bold())
            Form {
                TextField("标识", text: $id, prompt: Text("my-agent"))
                    .disabled(agent != nil)
                TextField("名称", text: $name)
                TextField("Skill 根目录", text: $path, prompt: Text("~/.agent/skills"))
            }
            .formStyle(.columns)
            Text("标识使用 2–32 位小写字母、数字或连字符；路径必须以 ~/ 开头。")
                .font(.caption).foregroundStyle(.secondary)
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("保存") {
                    Task {
                        await model.saveCustomAgent(id: id, name: name, path: path)
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(id.isEmpty || name.isEmpty || !path.hasPrefix("~/"))
            }
        }
        .padding(24)
        .frame(width: 540, height: 300)
    }
}

private func agentAccessibilityLabel(_ agent: AgentModel) -> String {
    let kind = agent.custom ? String(localized: "自定义 Agent") : String(localized: "内置 Agent")
    let status = agent.configured
        ? String(localized: "已管理")
        : (agent.detected ? String(localized: "已检测但未管理") : String(localized: "未检测"))
    return String(
        format: String(localized: "%1$@，%2$@，%3$@"),
        locale: .current,
        agent.name,
        kind,
        status
    )
}

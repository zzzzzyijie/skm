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

    private var detectedCount: Int {
        model.agents.filter(\.detected).count
    }

    private var configuredAgents: [AgentModel] {
        sortedAgents.filter(\.configured)
    }

    private var availableAgents: [AgentModel] {
        sortedAgents.filter { !$0.configured }
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
            VStack(alignment: .leading, spacing: 26) {
                HStack(alignment: .center, spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Agents")
                            .font(.largeTitle.bold())
                        Text("选择要由 SKM 管理的 AI 工具。启用后，你可以从 Skill 详情中一键部署。")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("添加自定义 Agent", systemImage: "plus") {
                        editingAgent = nil
                        showsEditor = true
                    }
                    .buttonStyle(.borderedProminent)
                }

                HStack(spacing: 10) {
                    AgentSummaryPill(
                        title: String(localized: "已启用"),
                        value: configuredCount,
                        systemImage: "checkmark.circle.fill",
                        tint: .accentColor
                    )
                    AgentSummaryPill(
                        title: String(localized: "本机已检测"),
                        value: detectedCount,
                        systemImage: "desktopcomputer",
                        tint: .green
                    )
                    AgentSummaryPill(
                        title: String(localized: "全部 Agent"),
                        value: model.agents.count,
                        systemImage: "cpu",
                        tint: .secondary
                    )
                }

                if model.agents.isEmpty {
                    ContentUnavailableView(
                        "没有可用的 Agent",
                        systemImage: "cpu",
                        description: Text("可以添加一个自定义 Agent，或安装受支持的工具后重新刷新。")
                    )
                    .frame(minHeight: 260)
                } else {
                    LazyVStack(alignment: .leading, spacing: 22) {
                        if !configuredAgents.isEmpty {
                            agentSection(
                                title: String(localized: "已启用"),
                                subtitle: String(localized: "这些 Agent 可以使用我的 Skill"),
                                agents: configuredAgents
                            )
                        }
                        if !availableAgents.isEmpty {
                            agentSection(
                                title: String(localized: "其他 Agent"),
                                subtitle: String(localized: "启用后即可由 SKM 统一部署和更新"),
                                agents: availableAgents
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

    private func agentSection(title: String, subtitle: String, agents: [AgentModel]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                Text(title).font(.title3.bold())
                Text(agents.count.description)
                    .font(.callout.monospacedDigit())
                    .foregroundStyle(.secondary)
                Spacer()
                Text(subtitle)
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }

            VStack(spacing: 10) {
                ForEach(agents) { agent in
                    AgentSettingsRow(
                        agent: agent,
                        isLoading: model.isLoading,
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
}

private struct AgentSummaryPill: View {
    let title: String
    let value: Int
    let systemImage: String
    let tint: Color

    var body: some View {
        Label {
            Text("\(value) \(title)")
                .monospacedDigit()
        } icon: {
            Image(systemName: systemImage)
        }
        .font(.callout)
        .foregroundStyle(tint)
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .background(tint.opacity(0.09), in: Capsule())
        .accessibilityElement(children: .combine)
    }
}

private struct AgentSettingsRow: View {
    let agent: AgentModel
    let isLoading: Bool
    let onToggle: (Bool) -> Void
    let onEdit: () -> Void
    let onDelete: () -> Void

    var body: some View {
        HStack(spacing: 16) {
            Image(systemName: agentSymbol)
                .font(.title3)
                .foregroundStyle(agent.configured ? Color.accentColor : .secondary)
                .frame(width: 44, height: 44)
                .background(
                    agent.configured ? Color.accentColor.opacity(0.1) : Color.primary.opacity(0.045),
                    in: RoundedRectangle(cornerRadius: 11)
                )
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(agent.name).font(.headline)
                    AgentStateBadge(agent: agent)
                    if agent.custom {
                        Text("自定义")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 2)
                            .background(.quaternary, in: Capsule())
                    }
                }
                Text(agent.path ?? String(localized: "未提供 Skill 路径"))
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                if let note = agent.note, !note.isEmpty {
                    Text(note)
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }
            }

            Spacer(minLength: 12)

            VStack(alignment: .trailing, spacing: 4) {
                Toggle("由 SKM 管理", isOn: Binding(
                    get: { agent.configured },
                    set: { value in onToggle(value) }
                ))
                .toggleStyle(.switch)
                .controlSize(.small)
                .accessibilityLabel("允许 SKM 向此 Agent 部署 Skill")
                .accessibilityIdentifier("agent-management-\(agent.id)")
                .disabled(!agent.supported || isLoading)

                if !agent.detected && !agent.configured {
                    Text("未检测到，也可预先启用")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }

            if agent.custom {
                Menu("更多", systemImage: "ellipsis.circle") {
                    Button("编辑", systemImage: "pencil", action: onEdit)
                    Button("删除", systemImage: "trash", role: .destructive, action: onDelete)
                }
                .menuStyle(.borderlessButton)
                .fixedSize()
            }
        }
        .padding(16)
        .background(.quaternary.opacity(0.38), in: RoundedRectangle(cornerRadius: 14))
        .overlay {
            if agent.configured {
                RoundedRectangle(cornerRadius: 14)
                    .stroke(Color.accentColor.opacity(0.22))
            }
        }
    }

    private var agentSymbol: String {
        if agent.custom { return "cpu.fill" }
        switch agent.id {
        case "claude": return "sparkles"
        case "codex": return "terminal"
        case "cursor": return "cursorarrow.rays"
        case "copilot": return "person.wave.2"
        case "gemini": return "diamond"
        default: return "cpu"
        }
    }
}

private struct AgentStateBadge: View {
    let agent: AgentModel

    var body: some View {
        Label(title, systemImage: symbol)
            .font(.caption)
            .foregroundStyle(color)
            .padding(.horizontal, 7)
            .padding(.vertical, 2)
            .background(color.opacity(0.09), in: Capsule())
    }

    private var title: String {
        if agent.configured { return String(localized: "已启用") }
        return agent.detected ? String(localized: "已检测") : String(localized: "未检测")
    }

    private var symbol: String {
        if agent.configured { return "checkmark.circle.fill" }
        return agent.detected ? "circle.dotted.circle" : "circle.dashed"
    }

    private var color: Color {
        if agent.configured { return .accentColor }
        return agent.detected ? .green : .secondary
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

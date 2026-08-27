import SwiftUI

struct AgentsListView: View {
    @Bindable var model: AppModel
    @State private var showsCustomAgent = false

    var body: some View {
        List(model.agents, selection: $model.selectedAgentID) { agent in
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
        .onChange(of: model.pendingCommand?.id) { _, _ in
            guard let command = model.pendingCommand,
                  command.section == .agents,
                  command.kind == .create else { return }
            showsCustomAgent = true
            model.consumeCommand(command.id)
        }
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
            .onChange(of: model.pendingCommand?.id) { _, _ in
                guard let command = model.pendingCommand,
                      command.section == .agents,
                      command.kind == .deleteSelection,
                      agent.custom else { return }
                confirmsDelete = true
                model.consumeCommand(command.id)
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

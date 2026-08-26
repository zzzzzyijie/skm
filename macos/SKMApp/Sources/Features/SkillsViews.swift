import AppKit
import SwiftUI

struct SkillsListView: View {
    @Bindable var model: AppModel
    @State private var search = ""
    @State private var showsAdd = false

    private var filtered: [SkillSummary] {
        guard !search.isEmpty else { return model.skills }
        return model.skills.filter {
            $0.name.localizedStandardContains(search) ||
            $0.description.localizedCaseInsensitiveContains(search) ||
            $0.tags.contains(where: { $0.localizedCaseInsensitiveContains(search) })
        }
    }

    var body: some View {
        Group {
            if model.skills.isEmpty && !model.isLoading {
                ContentUnavailableView("还没有 Skill", systemImage: "square.stack.3d.up", description: Text("从本地目录、ZIP 或 Git Source 导入第一个 Skill。"))
            } else {
                List(filtered, selection: $model.selectedSkillID) { skill in
                    VStack(alignment: .leading, spacing: 5) {
                        HStack {
                            Text(skill.name).fontWeight(.medium)
                            Spacer()
                            HealthBadge(health: skill.health)
                        }
                        Text(skill.description.isEmpty ? "无描述" : skill.description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                        if !skill.tags.isEmpty {
                            Text(skill.tags.joined(separator: " · "))
                                .font(.caption2)
                                .foregroundStyle(.tertiary)
                        }
                    }
                    .padding(.vertical, 4)
                    .tag(skill.id)
                }
            }
        }
        .searchable(text: $search, prompt: "搜索名称、描述或标签")
        .navigationTitle("Skills")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("添加 Skill", systemImage: "plus") { showsAdd = true }
                    .keyboardShortcut("n", modifiers: .command)
            }
        }
        .sheet(isPresented: $showsAdd) { AddSkillSheet(model: model) }
    }
}

struct SkillDetailView: View {
    @Bindable var model: AppModel
    @State private var details: SkillDetails?
    @State private var tab = 0
    @State private var showsEditor = false
    @State private var confirmsDelete = false

    var body: some View {
        Group {
            if let id = model.selectedSkillID, let summary = model.skills.first(where: { $0.id == id }) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 22) {
                        header(summary)
                        Picker("内容", selection: $tab) {
                            Text("概览").tag(0)
                            Text("SKILL.md").tag(1)
                            Text("部署").tag(2)
                        }
                        .pickerStyle(.segmented)
                        switch tab {
                        case 1: sourceView
                        case 2: deployments(summary)
                        default: overview(summary)
                        }
                    }
                    .padding(26)
                    .frame(maxWidth: 820, alignment: .leading)
                }
                .task(id: "\(id):\(summary.hash)") { await loadDetails(id) }
                .toolbar {
                    ToolbarItemGroup(placement: .primaryAction) {
                        Button("编辑", systemImage: "pencil") { showsEditor = true }
                            .disabled(details?.editable != true)
                        Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    }
                }
                .sheet(isPresented: $showsEditor, onDismiss: { Task { await loadDetails(id) } }) {
                    if let details { SkillEditorSheet(model: model, details: details) }
                }
                .confirmationDialog("移除 \(summary.name)？", isPresented: $confirmsDelete) {
                    Button("移除 Skill", role: .destructive) { Task { await model.removeSkill(id: id) } }
                } message: {
                    Text("已经启用的 Skill 必须先停用。不可变快照只会在没有引用时清理。")
                }
            } else {
                ContentUnavailableView("选择一个 Skill", systemImage: "square.stack.3d.up")
            }
        }
    }

    private func header(_ skill: SkillSummary) -> some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack(alignment: .firstTextBaseline) {
                Text(skill.name).font(.largeTitle.bold())
                HealthBadge(health: skill.health)
            }
            Text(skill.description.isEmpty ? "无描述" : skill.description)
                .font(.title3)
                .foregroundStyle(.secondary)
            if skill.usingFallback == true {
                Label("源目录当前不可用，正在读取安全快照", systemImage: "exclamationmark.triangle.fill")
                    .foregroundStyle(.orange)
            }
        }
    }

    private func overview(_ skill: SkillSummary) -> some View {
        VStack(alignment: .leading, spacing: 16) {
            GroupBox("信息") {
                VStack(spacing: 10) {
                    LabeledContent("来源", value: skill.source.isEmpty ? "local" : skill.source)
                    LabeledContent("标签", value: skill.tags.isEmpty ? "—" : skill.tags.joined(separator: ", "))
                    LabeledContent("Hash", value: String(skill.hash.prefix(12)))
                    LabeledContent("有效路径", value: skill.effectivePath)
                }
                .textSelection(.enabled)
                .padding(4)
            }
            if let reason = skill.editReason, !skill.editable {
                Label(reason, systemImage: "lock.fill")
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private var sourceView: some View {
        GroupBox {
            ScrollView(.horizontal) {
                Text(details?.content ?? "正在读取…")
                    .font(.system(.body, design: .monospaced))
                    .textSelection(.enabled)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(8)
            }
        }
    }

    private func deployments(_ skill: SkillSummary) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("选择要加载此 Skill 的 Agent。变更由 Go Planner 校验后原子应用。")
                .foregroundStyle(.secondary)
            ForEach(model.agents.filter(\.configured)) { agent in
                Toggle(isOn: Binding(
                    get: { model.isEnabled(skill.id, for: agent.id) },
                    set: { enabled in Task { await model.setSkill(skill.id, agentID: agent.id, enabled: enabled) } }
                )) {
                    Label(agent.name, systemImage: agent.detected ? "checkmark.circle" : "circle.dashed")
                }
                .disabled(model.isLoading)
            }
            if model.agents.allSatisfy({ !$0.configured }) {
                ContentUnavailableView("没有已管理的 Agent", systemImage: "cpu", description: Text("先在 Agents 中添加或启用一个 Agent。"))
            }
        }
    }

    private func loadDetails(_ id: String) async {
        do { details = try await model.skillDetails(id) }
        catch { model.errorMessage = error.localizedDescription }
    }
}

struct AddSkillSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    @State private var mode = 0
    @State private var path = ""
    @State private var remote = ""
    @State private var tags = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("添加 Skill").font(.title2.bold())
            Picker("来源", selection: $mode) {
                Text("本地").tag(0)
                Text("Git / 命令").tag(1)
            }
            .pickerStyle(.segmented)
            if mode == 0 {
                HStack {
                    TextField("Skill 目录或 ZIP", text: $path)
                    Button("选择…") { chooseLocalSkill() }
                }
                TextField("标签，以逗号分隔（可选）", text: $tags)
                Text("本地内容会被验证并写入 SKM 的不可变对象库。")
                    .font(.caption).foregroundStyle(.secondary)
            } else {
                TextField("Git URL、owner/repo 或 npx skills add …", text: $remote)
                Text("凭据由系统 Git、SSH Agent 或 Credential Helper 管理，SKM 不保存 Token。")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("导入") {
                    Task {
                        if mode == 0 { await model.addLocalSkill(path: path, tags: parseTags(tags)) }
                        else { await model.addRemoteSkill(input: remote) }
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(mode == 0 ? path.trimmingCharacters(in: .whitespaces).isEmpty : remote.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
        .padding(24)
        .frame(width: 560, height: 300)
    }

    private func chooseLocalSkill() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = true
        panel.allowsMultipleSelection = false
        panel.allowedContentTypes = [.zip]
        if panel.runModal() == .OK { path = panel.url?.path ?? path }
    }
}

struct SkillEditorSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let details: SkillDetails
    @State private var content: String
    @State private var tags: String

    init(model: AppModel, details: SkillDetails) {
        self.model = model
        self.details = details
        _content = State(initialValue: details.content)
        _tags = State(initialValue: details.tags.joined(separator: ", "))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("编辑 \(details.name)").font(.title2.bold())
            TextEditor(text: $content)
                .font(.system(.body, design: .monospaced))
                .border(.separator)
            TextField("标签，以逗号分隔", text: $tags)
            HStack {
                Text("保存时会校验 baseHash，防止覆盖来自 CLI 的并发修改。")
                    .font(.caption).foregroundStyle(.secondary)
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("保存") {
                    Task {
                        await model.updateSkill(id: details.id, content: content, baseHash: details.hash, tags: parseTags(tags))
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding(24)
        .frame(minWidth: 720, minHeight: 560)
    }
}

struct HealthBadge: View {
    let health: String

    var body: some View {
        Label(label, systemImage: symbol)
            .font(.caption)
            .foregroundStyle(color)
            .labelStyle(.iconOnly)
            .help(label)
            .accessibilityLabel(label)
    }

    private var label: String {
        switch health {
        case "available": "可用"
        case "changed": "已变更"
        case "missing": "缺失"
        case "unreachable": "不可访问"
        default: "无效"
        }
    }

    private var symbol: String { health == "available" ? "checkmark.circle.fill" : "exclamationmark.triangle.fill" }
    private var color: Color { health == "available" ? .green : .orange }
}

func parseTags(_ value: String) -> [String] {
    Array(Set(value.split(separator: ",").map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty })).sorted()
}

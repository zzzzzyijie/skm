import AppKit
import SwiftUI

/// SkillsListView - 技能列表视图
/// 支持基于名称/描述/标签的本地化模糊搜索、按标签展开/收起的分组列表、
/// 空数据向导以及无障碍屏幕阅读适配。
struct SkillsListView: View {
    @Bindable var model: AppModel
    @State private var search = ""
    @State private var showsAdd = false
    @State private var addMode = 0
    @State private var isAllGroupExpanded = true
    @State private var expandedTags: Set<String> = []

    private var filtered: [SkillSummary] {
        return model.skills.filter {
            search.isEmpty ||
                $0.name.localizedStandardContains(search) ||
                $0.description.localizedStandardContains(search) ||
                $0.tags.contains(where: { $0.localizedStandardContains(search) })
        }
    }

    var body: some View {
        let visibleSkills = filtered
        let tagGroups = itemsGroupedByTag(visibleSkills, tags: \.tags)

        Group {
            if model.skills.isEmpty && !model.isLoading {
                ContentUnavailableView {
                    Label("还没有 Skill", systemImage: "square.stack.3d.up")
                } description: {
                    Text("从本地目录、ZIP 或 Git Source 导入第一个 Skill。")
                } actions: {
                    Button("添加 Skill") {
                        addMode = 0
                        showsAdd = true
                    }
                    .buttonStyle(.borderedProminent)
                }
            } else if visibleSkills.isEmpty {
                ContentUnavailableView.search(text: search)
            } else {
                List(selection: $model.selectedSkillID) {
                    TagGroupHeader(
                        title: String(localized: "全部"),
                        systemImage: "square.stack.3d.up",
                        count: visibleSkills.count,
                        isExpanded: isAllGroupExpanded
                    ) {
                        isAllGroupExpanded.toggle()
                    }
                    .accessibilityIdentifier("skills-group-all")

                    if isAllGroupExpanded {
                        ForEach(visibleSkills) { skill in
                            SkillSummaryRow(skill: skill)
                                .padding(.leading, 28)
                                .listRowInsets(EdgeInsets(top: 3, leading: 12, bottom: 3, trailing: 12))
                                .accessibilityIdentifier("skill-row-\(skill.id)")
                                .tag(skill.id)
                        }
                    }

                    ForEach(tagGroups, id: \.tag) { group in
                        TagGroupHeader(
                            title: group.tag,
                            count: group.items.count,
                            isExpanded: expandedTags.contains(group.tag)
                        ) {
                            toggleTag(group.tag)
                        }
                        .accessibilityIdentifier("skills-group-\(group.tag)")

                        if expandedTags.contains(group.tag) {
                            ForEach(group.items) { skill in
                                SkillSummaryRow(skill: skill)
                                    .padding(.leading, 28)
                                    .listRowInsets(EdgeInsets(top: 3, leading: 12, bottom: 3, trailing: 12))
                                    .accessibilityIdentifier("skill-row-\(skill.id)")
                                    .tag(skill.id)
                            }
                        }
                    }
                }
            }
        }
        .searchable(text: $search, prompt: "搜索名称、描述或标签")
        .navigationTitle("Skills")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("添加 Skill", systemImage: "plus") {
                    addMode = 0
                    showsAdd = true
                }
            }
        }
        .sheet(isPresented: $showsAdd) { AddSkillSheet(model: model, initialMode: addMode) }
        .onChange(of: model.pendingCommand?.id) { _, _ in
            guard let command = model.pendingCommand, command.section == .skills else { return }
            if command.kind == .create || command.kind == .importItem {
                addMode = command.kind == .importItem ? 1 : 0
                showsAdd = true
                model.consumeCommand(command.id)
            }
        }
    }

    private func toggleTag(_ tag: String) {
        if expandedTags.contains(tag) {
            expandedTags.remove(tag)
        } else {
            expandedTags.insert(tag)
        }
    }
}

/// SkillDetailView - 技能详情视图
/// 包含“概览”、“SKILL.md 源码”、“Agent 部署矩阵”三段内容，
/// 提供 QuickLook 快捷预览、修改历史查看（HistorySheet）、在线编辑（SkillEditorSheet）与删除安全确认。
struct SkillDetailView: View {
    @Bindable var model: AppModel
    @State private var details: SkillDetails?
    @State private var tab = 0
    @State private var showsEditor = false
    @State private var showsHistory = false
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
                        Button("快速查看", systemImage: "eye") { Task { await showQuickLook() } }
                        Button("历史", systemImage: "clock.arrow.circlepath") { showsHistory = true }
                        Button("编辑", systemImage: "pencil") { showsEditor = true }
                            .disabled(details?.editable != true)
                        Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    }
                }
                .sheet(isPresented: $showsEditor, onDismiss: { Task { await loadDetails(id) } }) {
                    if let details { SkillEditorSheet(model: model, details: details) }
                }
                .sheet(isPresented: $showsHistory, onDismiss: { Task { await loadDetails(id) } }) {
                    HistorySheet(model: model, kind: "skill", itemID: id, title: summary.name)
                }
                .confirmationDialog("移除 \(summary.name)？", isPresented: $confirmsDelete) {
                    Button("移除 Skill", role: .destructive) { Task { await model.removeSkill(id: id) } }
                } message: {
                    Text("已经启用的 Skill 必须先停用。不可变快照只会在没有引用时清理。")
                }
                .onChange(of: model.pendingCommand?.id) { _, _ in
                    guard let command = model.pendingCommand,
                          command.section == .skills,
                          command.kind == .deleteSelection else { return }
                    confirmsDelete = true
                    model.consumeCommand(command.id)
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
            Text(skill.description.isEmpty ? String(localized: "无描述") : skill.description)
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
                Text(details?.content ?? String(localized: "正在读取…"))
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
                .accessibilityLabel(String(
                    format: String(localized: "为 %1$@ 启用 %2$@"),
                    locale: .current,
                    agent.name,
                    skill.name
                ))
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

    private func showQuickLook() async {
        do {
            guard let url = try await model.quickLookURL() else { return }
            QuickLookPresenter.shared.show(url)
        } catch { model.errorMessage = error.localizedDescription }
    }
}

/// AddSkillSheet - 添加/导入 Skill 弹窗
/// 支持两类导入模式：
/// 1. 本地导入：选取本地目录或 ZIP 压缩包，校验结构后写入 SKM 不可变对象存储；
/// 2. Git/命令行导入：支持输入 Git URL、GitHub 简写或 npx skills add 命令，直接调用系统的 Git/SSH 认证链。
struct AddSkillSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    @State private var mode = 0
    @State private var path = ""
    @State private var remote = ""
    @State private var tags = ""

    init(model: AppModel, initialMode: Int = 0) {
        self.model = model
        _mode = State(initialValue: initialMode)
    }

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

/// SkillEditorSheet - 在线编辑 Skill 弹窗
/// 包含 Markdown 源码编辑器与标签输入。
/// 内置乐观并发控制：保存时向 Core 发送 baseHash，当检测到底层文件在编辑期间被 CLI 修改时，
/// 呈现左右分栏冲突对比（ConflictPreview），允许用户选择“保留草稿覆盖”或“采用磁盘版本”。
struct SkillEditorSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let details: SkillDetails
    @State private var content: String
    @State private var tags: String
    @State private var baseHash: String
    @State private var latest: SkillDetails?

    init(model: AppModel, details: SkillDetails) {
        self.model = model
        self.details = details
        _content = State(initialValue: details.content)
        _tags = State(initialValue: details.tags.joined(separator: ", "))
        _baseHash = State(initialValue: details.hash)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("编辑 \(details.name)").font(.title2.bold())
            TextEditor(text: $content)
                .font(.system(.body, design: .monospaced))
                .border(.separator)
            TextField("标签，以逗号分隔", text: $tags)
            if let latest {
                GroupBox("检测到并发修改") {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("磁盘版本在编辑期间发生了变化。你的草稿仍保留，请比较后选择恢复方式。")
                            .foregroundStyle(.orange)
                        HStack(alignment: .top, spacing: 12) {
                            ConflictPreview(title: "你的草稿", content: content)
                            ConflictPreview(title: "磁盘版本", content: latest.content)
                        }
                        HStack {
                            Button("使用磁盘版本") {
                                content = latest.content
                                tags = latest.tags.joined(separator: ", ")
                                baseHash = latest.hash
                                self.latest = nil
                            }
                            Spacer()
                            Button("保留草稿并覆盖") {
                                baseHash = latest.hash
                                self.latest = nil
                                Task { await save() }
                            }
                            .buttonStyle(.borderedProminent)
                        }
                    }
                    .padding(6)
                }
            }
            HStack {
                Text("保存时会校验 baseHash，防止覆盖来自 CLI 的并发修改。")
                    .font(.caption).foregroundStyle(.secondary)
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("保存") {
                    Task { await save() }
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding(24)
        .frame(minWidth: 720, minHeight: 560)
    }

    private func save() async {
        let saved = await model.updateSkill(id: details.id, content: content, baseHash: baseHash, tags: parseTags(tags))
        if saved {
            dismiss()
        } else if model.lastErrorKind == "conflict" {
            do { latest = try await model.skillDetails(details.id) }
            catch { model.errorMessage = error.localizedDescription }
        }
    }
}

struct ConflictPreview: View {
    let title: LocalizedStringKey
    let content: String

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title).font(.headline)
            ScrollView {
                Text(content)
                    .font(.caption.monospaced())
                    .textSelection(.enabled)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            .frame(minHeight: 90, maxHeight: 150)
            .padding(8)
            .background(.quaternary.opacity(0.45), in: RoundedRectangle(cornerRadius: 8))
        }
        .frame(maxWidth: .infinity, alignment: .leading)
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

    private var label: String { healthLabel(health) }

    private var symbol: String { health == "available" ? "checkmark.circle.fill" : "exclamationmark.triangle.fill" }
    private var color: Color { health == "available" ? .green : .orange }
}

func healthLabel(_ health: String) -> String {
    switch health {
    case "available": String(localized: "可用")
    case "changed": String(localized: "已变更")
    case "missing": String(localized: "缺失")
    case "unreachable": String(localized: "不可访问")
    default: String(localized: "无效")
    }
}

func parseTags(_ value: String) -> [String] {
    Array(Set(value.split(separator: ",").map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty })).sorted()
}

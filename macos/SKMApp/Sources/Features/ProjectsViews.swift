import AppKit
import SwiftUI

struct ProjectsListView: View {
    @Bindable var model: AppModel
    @State private var showsAddProject = false
    @State private var projectToRemove: ProjectModel?

    var body: some View {
        Group {
            if model.projects.isEmpty {
                ContentUnavailableView {
                    Label("没有已注册项目", systemImage: "folder")
                } description: {
                    Text("添加一个本机项目，扫描各 Agent 的 Skills，或把 Library Skill 部署到项目。")
                } actions: {
                    Button("添加项目…") { showsAddProject = true }
                        .buttonStyle(.borderedProminent)
                }
            } else {
                List(model.projects, selection: $model.selectedProjectID) { project in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(project.id).fontWeight(.medium)
                            Spacer()
                            Image(systemName: project.exists ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                                .foregroundStyle(project.exists ? .green : .orange)
                                .accessibilityLabel(project.exists ? String(localized: "项目可用") : String(localized: "项目路径缺失"))
                        }
                        Text(project.path).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                        Text(String(format: String(localized: "%lld 个 Skills · %lld 个部署"), locale: .current, project.skillCount, project.activationCount))
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
                    .padding(.vertical, 4)
                    .tag(project.id)
                    .contextMenu {
                        Button("在 Finder 中显示") { NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: project.path) }
                        Divider()
                        Button("注销项目", role: .destructive) { projectToRemove = project }
                    }
                }
            }
        }
        .navigationTitle("Projects")
        .toolbar {
            Button("添加项目", systemImage: "folder.badge.plus") { showsAddProject = true }
                .help("登记本机项目目录")
        }
        .sheet(isPresented: $showsAddProject) { AddProjectSheet(model: model) }
        .confirmationDialog("注销项目？", isPresented: Binding(
            get: { projectToRemove != nil },
            set: { if !$0 { projectToRemove = nil } }
        )) {
            Button("注销", role: .destructive) {
                guard let projectToRemove else { return }
                Task { await model.unregisterProject(id: projectToRemove.id) }
            }
        } message: {
            Text("只移除登记信息，不删除项目文件；仍有受管部署时会安全阻止。")
        }
        .onChange(of: model.pendingCommand?.id) { _, _ in
            guard let command = model.pendingCommand, command.section == .projects else { return }
            if command.kind == .create || command.kind == .importItem { showsAddProject = true }
            if command.kind == .deleteSelection, let id = model.selectedProjectID {
                projectToRemove = model.projects.first(where: { $0.id == id })
            }
            model.consumeCommand(command.id)
        }
    }
}

struct AddProjectSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    @State private var path = ""
    @State private var name = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("添加项目").font(.title2.bold())
            Form {
                HStack {
                    TextField("项目目录", text: $path)
                        .accessibilityIdentifier("project-path-field")
                    Button("选择…") { chooseProject() }
                }
                TextField("显示名称（可选）", text: $name)
            }
            .formStyle(.grouped)
            Text("SKM 只登记目录，不会修改项目；部署操作仍会单独预览和确认。")
                .font(.caption)
                .foregroundStyle(.secondary)
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("添加") {
                    Task {
                        await model.addProject(path: path, name: name)
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(path.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24)
        .frame(width: 560, height: 300)
    }

    private func chooseProject() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK {
            path = panel.url?.path ?? path
            if name.isEmpty { name = panel.url?.lastPathComponent ?? "" }
        }
    }
}

struct ProjectDetailView: View {
    @Bindable var model: AppModel
    @State private var selectedLibrarySkill = ""
    @State private var selectedAgents = Set<String>()
    @State private var mode = "symlink"

    var body: some View {
        if let id = model.selectedProjectID,
           let project = model.projects.first(where: { $0.id == id }) {
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    header(project)
                    if let details = model.projectDetails, details.project.id == id {
                        deploymentSection(details)
                        planSection
                        scannedSkills(details)
                    } else {
                        ProgressView("正在扫描项目…")
                            .frame(maxWidth: .infinity, minHeight: 220)
                    }
                }
                .padding(26)
                .frame(maxWidth: 900, alignment: .leading)
            }
            .task(id: id) {
                selectedAgents = Set(model.agents.filter(\.configured).map(\.id))
                if selectedAgents.isEmpty { selectedAgents = ["codex"] }
                selectedLibrarySkill = model.skills.first?.id ?? ""
                await model.loadProjectDetails(id)
            }
        } else {
            ContentUnavailableView("选择一个项目", systemImage: "folder")
        }
    }

    private func header(_ project: ProjectModel) -> some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 6) {
                Text(project.id).font(.largeTitle.bold())
                Text(project.path).foregroundStyle(.secondary).textSelection(.enabled)
            }
            Spacer()
            Button("重新扫描", systemImage: "arrow.clockwise") { Task { await model.loadProjectDetails(project.id) } }
                .disabled(model.isLoading)
        }
    }

    private func deploymentSection(_ details: ProjectDetails) -> some View {
        GroupBox("从 Library 部署") {
            VStack(alignment: .leading, spacing: 14) {
                if model.skills.isEmpty {
                    ContentUnavailableView("Library 中没有 Skill", systemImage: "square.stack.3d.up")
                } else {
                    Picker("Skill", selection: $selectedLibrarySkill) {
                        ForEach(model.skills) { skill in Text(skill.name).tag(skill.id) }
                    }
                    Picker("模式", selection: $mode) {
                        Text("Link（跟随 Library）").tag("symlink")
                        Text("Copy（独立副本）").tag("copy")
                    }
                    .pickerStyle(.segmented)
                    Text("目标 Agents").font(.headline)
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 150), alignment: .leading)], alignment: .leading) {
                        ForEach(model.agents.filter(\.supported)) { agent in
                            Toggle(agent.name, isOn: Binding(
                                get: { selectedAgents.contains(agent.id) },
                                set: { enabled in
                                    if enabled { selectedAgents.insert(agent.id) }
                                    else { selectedAgents.remove(agent.id) }
                                }
                            ))
                        }
                    }
                    Text("空白项目也可选择 Agent；应用后 SKM 会创建对应的隐藏 Skills 目录。")
                        .font(.caption).foregroundStyle(.secondary)
                    HStack {
                        Spacer()
                        Button("预览部署") {
                            Task { await model.deployProject(project: details.project.id, skill: selectedLibrarySkill, agents: selectedAgents.sorted(), mode: mode, dryRun: true) }
                        }
                        .disabled(selectedLibrarySkill.isEmpty || selectedAgents.isEmpty)
                    }
                }
            }
            .padding(8)
        }
    }

    @ViewBuilder
    private var planSection: some View {
        if let preview = model.projectDeploymentPreview {
            let hasConflict = preview.plan.operations.contains { $0.status == "conflict-unmanaged" || $0.status == "broken" }
            GroupBox("部署预览") {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(Array(preview.plan.operations.enumerated()), id: \.offset) { _, operation in
                        HStack(alignment: .top) {
                            Image(systemName: operation.status == "conflict-unmanaged" ? "exclamationmark.triangle.fill" : "arrow.right.circle")
                                .foregroundStyle(operation.status == "conflict-unmanaged" ? .orange : .secondary)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("\(operation.agent) · \(operation.status)").fontWeight(.medium)
                                Text(operation.target).font(.caption.monospaced()).foregroundStyle(.secondary).textSelection(.enabled)
                                if let message = operation.message { Text(message).font(.caption).foregroundStyle(.orange) }
                            }
                        }
                    }
                    if hasConflict {
                        Label("检测到未知目标或损坏状态。SKM 不会覆盖它；请先在 Finder 或终端中处理。", systemImage: "hand.raised.fill")
                            .foregroundStyle(.orange)
                    }
                    HStack {
                        Spacer()
                        Button("取消") { model.projectDeploymentPreview = nil }
                        Button("应用部署") {
                            Task {
                                await model.deployProject(
                                    project: preview.project.id,
                                    skill: preview.skill.id,
                                    agents: preview.plan.operations.map(\.agent),
                                    mode: preview.plan.operations.first?.mode ?? "symlink",
                                    dryRun: false
                                )
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(hasConflict || preview.plan.operations.isEmpty)
                    }
                }
                .padding(8)
            }
        }
    }

    private func scannedSkills(_ details: ProjectDetails) -> some View {
        GroupBox("项目扫描") {
            VStack(alignment: .leading, spacing: 0) {
                if details.scan.skills.isEmpty {
                    ContentUnavailableView("项目中没有 Agent Skill", systemImage: "doc.text.magnifyingglass")
                        .frame(minHeight: 150)
                } else {
                    ForEach(details.scan.skills) { skill in
                        ProjectSkillRow(model: model, project: details.project, skill: skill, activations: details.activations)
                        if skill.id != details.scan.skills.last?.id { Divider() }
                    }
                }
            }
            .padding(8)
        }
    }
}

private struct ProjectSkillRow: View {
    let model: AppModel
    let project: RegisteredProject
    let skill: ProjectScanSkill
    let activations: [ActivationModel]
    @State private var showsMigration = false

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: skill.status == "ok" ? "checkmark.circle" : "exclamationmark.triangle")
                .foregroundStyle(skill.status == "ok" ? .green : .orange)
            VStack(alignment: .leading, spacing: 4) {
                Text(skill.name).fontWeight(.medium)
                if let description = skill.description, !description.isEmpty { Text(description).foregroundStyle(.secondary) }
                Text(skill.agents.joined(separator: ", ")).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            if let activation = activations.first(where: { $0.name == skill.id || $0.skillId == skill.librarySkillId }) {
                Button("解绑") { Task { await model.unlinkProject(project: project.id, skill: activation.skillId, agents: activation.agents) } }
            } else if skill.librarySkillId == nil {
                Button("迁移到 Library") { showsMigration = true }
            } else {
                Label("已在 Library", systemImage: "checkmark")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 10)
        .sheet(isPresented: $showsMigration) { ProjectMigrationSheet(model: model, project: project, skill: skill) }
        .accessibilityElement(children: .contain)
    }
}

private struct ProjectMigrationSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let project: RegisteredProject
    let skill: ProjectScanSkill
    @State private var agent: String
    @State private var mode = "copy"
    @State private var removeSource = false

    init(model: AppModel, project: RegisteredProject, skill: ProjectScanSkill) {
        self.model = model
        self.project = project
        self.skill = skill
        _agent = State(initialValue: skill.agents.first ?? "")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("迁移 \(skill.name)").font(.title2.bold())
            Picker("来源 Agent", selection: $agent) {
                ForEach(skill.agents, id: \.self) { Text($0).tag($0) }
            }
            Picker("Library 模式", selection: $mode) {
                Text("复制到 Library").tag("copy")
                Text("跟随项目").tag("symlink")
            }
            .pickerStyle(.segmented)
            if mode == "copy" {
                Toggle("复制成功后移除项目原件", isOn: $removeSource)
                Text("只有所有 Agent 副本内容一致且不受 SKM 管理时才允许移除。")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("迁移") {
                    Task {
                        await model.migrateProjectSkill(project: project.id, skill: skill.id, agent: agent, mode: mode, removeSource: removeSource)
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding(24)
        .frame(width: 540, height: 340)
        .onChange(of: mode) { _, newValue in if newValue != "copy" { removeSource = false } }
    }
}

struct MetricCard: View {
    let title: String
    let value: String
    let symbol: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(title, systemImage: symbol).foregroundStyle(.secondary)
            Text(value).font(.title.bold()).monospacedDigit()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(.quaternary.opacity(0.5), in: RoundedRectangle(cornerRadius: 12))
    }
}

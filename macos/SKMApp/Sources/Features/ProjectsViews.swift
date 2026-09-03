import AppKit
import SwiftUI

/// ProjectsListView - 项目列表视图
/// 管理在本机登记的开发项目仓库，实时展示各项目的 Skill 总数与部署生效数，
/// 支持右键在 Finder 中快速定位或安全注销项目登记。
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
                    Text("添加一个本机项目，扫描各 Agent 的 Skills，或把我的 Skill 导入项目。")
                } actions: {
                    Button("添加项目…") { showsAddProject = true }
                        .buttonStyle(.borderedProminent)
                }
            } else {
                List(model.projects, selection: $model.selectedProjectID) { project in
                    let access = ProjectAccessStatus(project: project)
                    let isSelected = model.selectedProjectID == project.id
                    HStack(alignment: .top, spacing: 11) {
                        Image(systemName: access.canRead ? "folder.fill" : access.symbol)
                            .font(.title3)
                            .foregroundStyle(isSelected ? Color.white : (access.canRead ? Color.accentColor : access.color))
                            .frame(width: 32, height: 32)
                            .background(
                                isSelected ? Color.white.opacity(0.22) : (access.canRead ? Color.accentColor.opacity(0.1) : Color.primary.opacity(0.06)),
                                in: RoundedRectangle(cornerRadius: 8)
                            )
                            .accessibilityHidden(true)

                        VStack(alignment: .leading, spacing: 5) {
                            HStack(spacing: 8) {
                                Text(project.id).bold()
                                Spacer()
                                Label(access.title, systemImage: access.symbol)
                                    .font(.caption)
                                    .foregroundStyle(access.color)
                            }
                            Text(project.path)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .lineLimit(1)
                                .truncationMode(.middle)
                            Text(String(format: String(localized: "%lld 个 Skills · %lld 个部署"), locale: .current, project.skillCount, project.activationCount))
                                .font(.caption)
                                .foregroundStyle(.tertiary)
                        }
                    }
                    .padding(.vertical, 6)
                    .tag(project.id)
                    .accessibilityElement(children: .combine)
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
            ToolbarItem(placement: .primaryAction) {
                Button("添加项目", systemImage: "plus") {
                    showsAddProject = true
                }
                .help("登记本机项目目录")
                .accessibilityIdentifier("add-project-button")
            }
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

/// AddProjectSheet - 登记本机开发项目弹窗
/// 选取本地文件夹目录并登记，SKM 仅记录路径信息，不会主动修改任何现有代码文件。
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

/// ProjectDetailView - 项目技能详情与部署管理视图
/// 包含项目扫描概览、从个人 Skill 库导入、解绑与迁移操作。
struct ProjectDetailView: View {
    @Environment(\.openSettings) private var openSettings
    @Bindable var model: AppModel
    @State private var showsSkillImporter = false

    var body: some View {
        Group {
            if let id = model.selectedProjectID,
               let project = model.projects.first(where: { $0.id == id }) {
                let access = ProjectAccessStatus(project: project)
                if access.canRead {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            if let details = model.projectDetails, details.project.id == id {
                                header(project, details: details)
                                projectOverview(details)
                                planSection
                                scannedSkills(details)
                            } else {
                                header(project, details: nil)
                                ProgressView("正在扫描项目…")
                                    .frame(maxWidth: .infinity, minHeight: 220)
                            }
                        }
                        .padding(26)
                        .frame(maxWidth: 900, alignment: .leading)
                    }
                    .task(id: id) {
                        await model.loadProjectDetails(id)
                    }
                } else {
                    ContentUnavailableView {
                        Label(access.title, systemImage: access.symbol)
                    } description: {
                        Text(access.detail)
                    } actions: {
                        Button("打开权限访问设置…", systemImage: "folder.badge.gearshape", action: openFileAccessSettings)
                            .buttonStyle(.borderedProminent)
                        if project.exists {
                            Button("在 Finder 中显示") {
                                NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: project.path)
                            }
                        }
                    }
                }
            } else {
                ContentUnavailableView("选择一个项目", systemImage: "folder")
            }
        }
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                if let id = model.selectedProjectID,
                   let project = model.projects.first(where: { $0.id == id }) {
                    Button("重新扫描", systemImage: "arrow.clockwise") {
                        Task { await model.loadProjectDetails(project.id) }
                    }
                    .help("重新扫描项目目录中的 Skills")
                    .disabled(model.isLoading)

                    Button("在 Finder 中显示", systemImage: "folder") {
                        NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: project.path)
                    }
                    .help("在 Finder 中查看此项目目录")
                }
            }
        }
    }

    private func openFileAccessSettings() {
        model.settingsSection = .fileAccess
        openSettings()
    }

    private func header(_ project: ProjectModel, details: ProjectDetails?) -> some View {
        HStack(alignment: .center, spacing: 20) {
            VStack(alignment: .leading, spacing: 6) {
                Text(project.id).font(.largeTitle.bold())
                Label(project.path, systemImage: "folder")
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .textSelection(.enabled)
            }
            Spacer()
            Button("重新扫描", systemImage: "arrow.clockwise") {
                Task { await model.loadProjectDetails(project.id) }
            }
                .disabled(model.isLoading)
            Button("从我的 Skill 里导入", systemImage: "square.and.arrow.down") {
                showsSkillImporter = true
            }
            .buttonStyle(.borderedProminent)
            .disabled(details == nil)
        }
        .sheet(isPresented: $showsSkillImporter) {
            if let details {
                ProjectSkillImportSheet(model: model, project: details.project)
            }
        }
    }

    private func projectOverview(_ details: ProjectDetails) -> some View {
        HStack(spacing: 12) {
            MetricCard(
                title: String(localized: "项目 Skills"),
                value: details.scan.skillCount.description,
                symbol: "square.stack.3d.up"
            )
            MetricCard(
                title: String(localized: "使用中的 Agent"),
                value: details.scan.agents.filter { $0.skillCount > 0 }.count.description,
                symbol: "cpu"
            )
            MetricCard(
                title: String(localized: "受管 Skill"),
                value: details.activations.count.description,
                symbol: "checkmark.shield"
            )
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
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Label("项目 Skills", systemImage: "shippingbox")
                    .font(.title2.bold())
                Text(details.scan.skillCount.description)
                    .font(.callout.monospacedDigit())
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(.quaternary, in: Capsule())
                Spacer()
                Text("自动扫描各 Agent 的项目目录")
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }

            VStack(alignment: .leading, spacing: 0) {
                if details.scan.skills.isEmpty {
                    ContentUnavailableView {
                        Label("项目中还没有 Skill", systemImage: "shippingbox")
                    } description: {
                        Text("从我的 Skill 中导入，或重新扫描项目里的 Agent 目录。")
                    } actions: {
                        Button("从我的 Skill 里导入", systemImage: "square.and.arrow.down") {
                            showsSkillImporter = true
                        }
                        .buttonStyle(.borderedProminent)
                    }
                        .frame(minHeight: 150)
                } else {
                    ForEach(details.scan.skills) { skill in
                        ProjectSkillRow(
                            model: model,
                            project: details.project,
                            skill: skill,
                            agents: details.scan.agents,
                            activations: details.activations
                        )
                        if skill.id != details.scan.skills.last?.id { Divider() }
                    }
                }
            }
            .padding(.horizontal, 16)
            .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 14))
        }
    }
}

private struct ProjectSkillRow: View {
    let model: AppModel
    let project: RegisteredProject
    let skill: ProjectScanSkill
    let agents: [ProjectScanAgent]
    let activations: [ActivationModel]
    @State private var showsMigration = false

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: skill.status == "ok" ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                .foregroundStyle(skill.status == "ok" ? .green : .orange)
                .font(.title3)
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 7) {
                Text(skill.name).font(.headline)
                if let description = skill.description, !description.isEmpty {
                    Text(description)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }
                HStack(spacing: 6) {
                    ForEach(skill.agents, id: \.self) { agentID in
                        Text(agentName(agentID))
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 2.5)
                            .background(Color.primary.opacity(0.05), in: Capsule())
                            .overlay(
                                Capsule()
                                    .stroke(Color.primary.opacity(0.08), lineWidth: 0.5)
                            )
                    }
                }
            }
            Spacer()
            if let activation = activations.first(where: { $0.name == skill.id || $0.skillId == skill.librarySkillId }) {
                Button("从项目移除") {
                    Task { await model.unlinkProject(project: project.id, skill: activation.skillId, agents: activation.agents) }
                }
            } else if skill.librarySkillId == nil {
                Button("存到我的 Skill") { showsMigration = true }
            } else {
                Label("已在我的 Skill", systemImage: "checkmark")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 10)
        .sheet(isPresented: $showsMigration) { ProjectMigrationSheet(model: model, project: project, skill: skill) }
        .accessibilityElement(children: .contain)
    }

    private func agentName(_ id: String) -> String {
        agents.first(where: { $0.id == id })?.label ?? id
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
            Label(title, systemImage: symbol)
                .font(.callout)
                .foregroundStyle(.secondary)
            Text(value).font(.title.bold()).monospacedDigit()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(.quaternary.opacity(0.45), in: RoundedRectangle(cornerRadius: 12))
    }
}

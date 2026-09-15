import AppKit
import SwiftUI

/// ProjectsListView - 项目列表视图
/// 管理在本机登记的开发项目仓库，实时展示各项目的 Skill 总数与部署生效数，
/// 支持右键在 Finder 中快速定位或安全注销项目登记。
struct ProjectsListView: View {
    @Bindable var model: AppModel
    @State private var showsAddProject = false
    @State private var projectToRemove: ProjectModel?
    @State private var search = ""

    private var filteredProjects: [ProjectModel] {
        model.projects.filter {
            search.isEmpty || $0.id.localizedStandardContains(search) || $0.path.localizedStandardContains(search)
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            CollectionSearchField(title: "搜索项目", text: $search)
                .accessibilityIdentifier("projects-search-field")
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
                } else if filteredProjects.isEmpty {
                    ContentUnavailableView.search(text: search)
                } else {
                    List(filteredProjects) { project in
                        let isSelected = model.selectedProjectID == project.id

                        Button {
                            model.selectedProjectID = project.id
                        } label: {
                            HStack(alignment: .center, spacing: 10) {
                                Image(systemName: "folder")
                                    .font(.body)
                                    .foregroundStyle(isSelected ? SKMDesign.libraryGreen : Color.orange)
                                    .frame(width: 16)
                                    .accessibilityHidden(true)
                                Text(project.id)
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(isSelected ? Color(red: 0.10, green: 0.49, blue: 0.16) : Color.primary)
                                    .lineLimit(1)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 10)
                            .frame(height: 48)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .listRowInsets(EdgeInsets())
                        .listRowSeparator(.hidden)
                        .listRowBackground(isSelected ? SKMDesign.libraryGreen.opacity(0.11) : Color.clear)
                        .help(project.path)
                        .accessibilityElement(children: .combine)
                        .accessibilityAddTraits(isSelected ? .isSelected : [])
                        .contextMenu {
                            Button("在 Finder 中显示") { NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: project.path) }
                            Divider()
                            Button("注销项目", role: .destructive) { projectToRemove = project }
                        }
                    }
                    .listStyle(.plain)
                    .contentMargins(.horizontal, 0, for: .scrollContent)
                    .scrollContentBackground(.hidden)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            CollectionFooter(count: filteredProjects.count, symbol: "folder")
        }
        .safeAreaInset(edge: .top, spacing: 0) {
            CollectionHeader(title: "Projects") {
                Button("添加项目", systemImage: "plus") {
                    showsAddProject = true
                }
                .help("登记本机项目目录")
                .accessibilityIdentifier("add-project-button")
                .topToolbarActionStyle()
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
            PanelHeader(title: AppLocalization.string("添加项目"), subtitle: AppLocalization.string("连接本机项目，集中查看和部署它的 Skills。"), symbol: "folder.badge.plus")
            Form {
                HStack {
                    TextField("项目目录", text: $path)
                        .accessibilityIdentifier("project-path-field")
                    Button("选择…") { chooseProject() }
                }
                TextField("显示名称（可选）", text: $name)
            }
            .formStyle(.columns)
            .padding(16)
            .skmSurface()
            Text("SKM 只登记目录，不会修改项目；部署操作仍会单独预览和确认。")
                .font(.caption)
                .foregroundStyle(.secondary)
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                    .keyboardShortcut(.cancelAction)
                    .disabled(model.isLoading)
                Button("添加") {
                    Task {
                        await model.addProject(path: path, name: name)
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .keyboardShortcut(.defaultAction)
                .disabled(path.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || model.isLoading)
            }
        }
        .padding(24)
        .frame(width: 580, height: 340)
        .sheetChrome(model: model)
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
                        VStack(alignment: .leading, spacing: SKMDesign.librarySectionSpacing) {
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
                        .libraryReadingLayout()
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
                ContentUnavailableView("选择一个项目", systemImage: "folder", description: Text("选择本机项目，查看 Skills 和 Agent 部署状态。"))
            }
        }
        .safeAreaInset(edge: .top, spacing: 0) {
            DetailToolbar(isLoading: model.isLoading) {
                if let id = model.selectedProjectID,
                   let project = model.projects.first(where: { $0.id == id }) {
                    Group {
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
                    .topToolbarActionStyle()
                }
            }
        }
    }

    private func openFileAccessSettings() {
        model.settingsSection = .fileAccess
        openSettings()
    }

    private func header(_ project: ProjectModel, details: ProjectDetails?) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(project.id)
                .font(.system(size: 30, weight: .semibold))
                .textSelection(.enabled)

            Label(project.path, systemImage: "folder")
                .font(.callout)
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .truncationMode(.middle)
                .textSelection(.enabled)

            HStack(spacing: 10) {
                Button("从我的 Skill 里导入", systemImage: "arrow.down") {
                    showsSkillImporter = true
                }
                .buttonStyle(LibraryActionButtonStyle(prominent: true))
                .disabled(details == nil)
            }
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
                title: AppLocalization.string("项目 Skills"),
                value: details.scan.skillCount.description,
                symbol: "tag"
            )
            MetricCard(
                title: AppLocalization.string("使用中的 Agent"),
                value: details.scan.agents.filter { $0.skillCount > 0 }.count.description,
                symbol: "desktopcomputer"
            )
            MetricCard(
                title: AppLocalization.string("受管 Skill"),
                value: details.activations.count.description,
                symbol: "checkmark.circle"
            )
        }
    }

    @ViewBuilder
    private var planSection: some View {
        if let preview = model.projectDeploymentPreview, preview.project.id == model.selectedProjectID {
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
                        .disabled(hasConflict || preview.plan.operations.isEmpty || model.isLoading)
                    }
                }
                .padding(8)
            }
        }
    }

    private func scannedSkills(_ details: ProjectDetails) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(spacing: 8) {
                Label("项目 Skills", systemImage: "globe")
                    .font(.system(size: 16, weight: .semibold))
                Text(details.scan.skillCount.description)
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                    .monospacedDigit()
                    .padding(.horizontal, 8)
                    .frame(height: 24)
                    .background(SKMDesign.librarySelection, in: Capsule())
                Spacer()
            }

            LazyVStack(alignment: .leading, spacing: 12) {
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
                        .frame(maxWidth: .infinity)
                        .skmSurface(radius: SKMDesign.compactCardRadius)
                } else {
                    ForEach(details.scan.skills) { skill in
                        ProjectSkillRow(
                            model: model,
                            project: details.project,
                            skill: skill,
                            agents: details.scan.agents,
                            activations: details.activations
                        )
                        .padding(16)
                        .frame(minHeight: 126, alignment: .topLeading)
                        .libraryCard()
                    }
                }
            }
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
    @State private var confirmsUnlink = false

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 6) {
                    Label(skill.name, systemImage: skill.status == "ok" ? "checkmark.circle" : "exclamationmark.triangle")
                        .font(.system(size: 14, weight: .semibold))
                        .labelStyle(ProjectSkillLabelStyle(isHealthy: skill.status == "ok"))
                    if let description = skill.description, !description.isEmpty {
                        Text(description)
                            .font(.system(size: 13))
                            .lineSpacing(3)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                }
                Spacer(minLength: 0)
                if let activation = activations.first(where: { $0.name == skill.id || $0.skillId == skill.librarySkillId }) {
                    Button("从项目移除", role: .destructive) { confirmsUnlink = true }
                        .buttonStyle(LibraryActionButtonStyle())
                        .fixedSize()
                        .disabled(model.isLoading)
                        .confirmationDialog("从项目移除此 Skill？", isPresented: $confirmsUnlink) {
                            Button("从项目移除", role: .destructive) {
                                Task { await model.unlinkProject(project: project.id, skill: activation.skillId, agents: activation.agents) }
                            }
                        } message: {
                            Text("移除这个项目中的受管部署，个人资料库中的 Skill 会保留。")
                        }
                } else if skill.librarySkillId == nil {
                    Button("存到我的 Skill") { showsMigration = true }
                        .buttonStyle(LibraryActionButtonStyle())
                        .fixedSize()
                } else {
                    Label("已在我的 Skill", systemImage: "checkmark")
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
            ViewThatFits(in: .horizontal) {
                HStack(spacing: 8) { agentBadges }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 80), alignment: .leading)], alignment: .leading, spacing: 6) {
                    agentBadges
                }
            }
        }
        .sheet(isPresented: $showsMigration) { ProjectMigrationSheet(model: model, project: project, skill: skill) }
        .accessibilityElement(children: .contain)
    }

    private func agentName(_ id: String) -> String {
        agents.first(where: { $0.id == id })?.label ?? id
    }

    private var agentBadges: some View {
        ForEach(skill.agents, id: \.self) { agentID in
            Text(agentName(agentID))
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(SKMDesign.librarySelection, in: Capsule())
                .overlay { Capsule().strokeBorder(SKMDesign.libraryBorder, lineWidth: 0.5) }
                .fixedSize()
        }
    }
}

private struct ProjectSkillLabelStyle: LabelStyle {
    let isHealthy: Bool

    func makeBody(configuration: Configuration) -> some View {
        HStack(spacing: 8) {
            configuration.icon.foregroundStyle(isHealthy ? SKMDesign.libraryGreen : .orange)
            configuration.title.foregroundStyle(.primary)
        }
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
            PanelHeader(title: AppLocalization.string("迁移 \(skill.name)"), subtitle: AppLocalization.string("把项目中的技能加入个人资料库。"), symbol: "tray.and.arrow.down")
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
            } else {
                Text("将在个人 Library 中建立指向项目 Skill 的软链接；项目内的修改会实时同步到我的 Skill。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                    .keyboardShortcut(.cancelAction)
                    .disabled(model.isLoading)
                Button("迁移") {
                    Task {
                        await model.migrateProjectSkill(project: project.id, skill: skill.id, agent: agent, mode: mode, removeSource: removeSource)
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .keyboardShortcut(.defaultAction)
                .disabled(agent.isEmpty || model.isLoading)
            }
        }
        .padding(24)
        .frame(width: 560, height: 380)
        .sheetChrome(model: model)
        .onChange(of: mode) { _, newValue in if newValue != "copy" { removeSource = false } }
    }
}

struct MetricCard: View {
    let title: String
    let value: String
    let symbol: String

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label(title, systemImage: symbol)
                .font(.system(size: 13))
                .foregroundStyle(.secondary)
            Text(value)
                .font(.system(size: 30, weight: .semibold))
                .monospacedDigit()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 16)
        .frame(height: 88)
        .libraryCard()
        .accessibilityElement(children: .combine)
    }
}

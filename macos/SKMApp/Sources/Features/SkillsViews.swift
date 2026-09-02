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
            } else {
                List(selection: $model.selectedSkillID) {
                    if !visibleSkills.isEmpty {
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
        }
        .safeAreaInset(edge: .top, spacing: 0) {
            HStack(spacing: 6) {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
                    .font(.system(size: 12, weight: .medium))
                TextField("搜索 Skill", text: $search)
                    .textFieldStyle(.plain)
                    .font(.system(size: 13))
                if !search.isEmpty {
                    Button {
                        search = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.secondary)
                            .font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .stroke(Color(nsColor: .separatorColor).opacity(0.6), lineWidth: 0.5)
            )
            .padding(.horizontal, 10)
            .padding(.top, 8)
            .padding(.bottom, 6)
            .accessibilityIdentifier("skills-search-field")
        }
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
/// 一体化滚动布局：顶部标题与元数据、Agent 激活卡片区、Markdown 正文渲染、底部路径与来源信息。
/// 提供 QuickLook 快捷预览、在线编辑（SkillEditorSheet，内含历史版本回滚）与删除安全确认。
struct SkillDetailView: View {
    @Bindable var model: AppModel
    @State private var details: SkillDetails?
    @State private var showsEditor = false
    @State private var confirmsDelete = false

    var body: some View {
        Group {
            if let id = model.selectedSkillID, let summary = model.skills.first(where: { $0.id == id }) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        // ── 顶部：标题、描述、标签与元数据 ──
                        headerSection(summary)

                        // ── Agent 激活卡片区 ──
                        agentSection(summary)
                            .padding(.top, 20)

                        // ── 分隔线 ──
                        Divider().padding(.vertical, 20)

                        // ── Markdown 正文 ──
                        markdownSection

                        // ── 底部元数据 ──
                        footerSection(summary)
                            .padding(.top, 20)
                    }
                    .padding(26)
                    .frame(maxWidth: 820, alignment: .leading)
                }
                .task(id: "\(id):\(summary.hash)") { await loadDetails(id) }
                .toolbar {
                    ToolbarItemGroup(placement: .primaryAction) {
                        Button("快速查看", systemImage: "eye") { Task { await showQuickLook() } }
                        if details?.editable ?? summary.editable {
                            Button("编辑", systemImage: "pencil") { showsEditor = true }
                        }
                        Button("在 Finder 中显示", systemImage: "folder") { revealInFinder(summary) }
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

    // MARK: - 顶部：标题、描述、标签、元数据

    private func headerSection(_ skill: SkillSummary) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            // 标题行
            HStack(alignment: .firstTextBaseline) {
                Text(skill.name).font(.largeTitle.bold())
                HealthBadge(health: skill.health)
            }

            // 描述
            Text(skill.description.isEmpty ? String(localized: "无描述") : skill.description)
                .font(.title3)
                .foregroundStyle(.secondary)

            // Fallback 警告
            if skill.usingFallback == true {
                Label("源目录当前不可用，正在读取安全快照", systemImage: "exclamationmark.triangle.fill")
                    .font(.callout)
                    .foregroundStyle(.orange)
            }

            // 标签（capsule 样式）
            if !skill.tags.isEmpty {
                HStack(spacing: 5) {
                    ForEach(skill.tags, id: \.self) { tag in
                        Text(tag)
                            .font(.caption)
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(Color.accentColor.opacity(0.1), in: Capsule())
                            .foregroundStyle(Color.accentColor)
                    }
                }
            }

            // 元信息（来源 · hash · 字数）
            HStack(spacing: 16) {
                Label(skill.source.isEmpty ? "local" : skill.source, systemImage: skill.source == "git" ? "arrow.triangle.branch" : "externaldrive")
                Text(String(skill.hash.prefix(8)))
                    .font(.callout.monospaced())
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 4))
                if let body = details?.body {
                    Label("\(body.count) 字符", systemImage: "textformat.size")
                }
            }
            .font(.callout)
            .foregroundStyle(.secondary)
        }
    }

    // MARK: - Agent 激活卡片区

    private func agentSection(_ skill: SkillSummary) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Agent 激活")
                .font(.headline)
                .foregroundStyle(Color.secondary)

            let configuredAgents = model.agents.filter(\.configured)

            if configuredAgents.isEmpty {
                HStack {
                    Image(systemName: "cpu")
                        .foregroundStyle(Color.secondary)
                    Text("还没有已管理的 Agent，请先在设置中启用。")
                        .font(.callout)
                        .foregroundStyle(Color.secondary)
                }
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.primary.opacity(0.03), in: RoundedRectangle(cornerRadius: 10))
            } else {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 160), spacing: 10)], spacing: 10) {
                    ForEach(configuredAgents) { agent in
                        AgentToggleCard(
                            agent: agent,
                            isEnabled: model.isEnabled(skill.id, for: agent.id),
                            isLoading: model.isLoading
                        ) { enabled in
                            Task { await model.setSkill(skill.id, agentID: agent.id, enabled: enabled) }
                        }
                        .accessibilityLabel(String(
                            format: String(localized: "为 %1$@ 启用 %2$@"),
                            locale: .current,
                            agent.name,
                            skill.name
                        ))
                    }
                }
            }
        }
    }

    // MARK: - Markdown 正文

    private var markdownSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let details {
                if details.body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    Text("此 Skill 没有正文内容。")
                        .foregroundStyle(Color.secondary)
                        .italic()
                } else {
                    MarkdownBodyView(markdown: details.body)
                }
            } else {
                HStack(spacing: 8) {
                    ProgressView().controlSize(.small)
                    Text("正在读取文档…")
                        .foregroundStyle(Color.secondary)
                }
            }
        }
    }

    // MARK: - 底部元数据

    private func footerSection(_ skill: SkillSummary) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            if let reason = skill.editReason, !skill.editable {
                Label(reason, systemImage: "lock.fill")
                    .font(.callout)
                    .foregroundStyle(Color.secondary)
            }

            HStack(spacing: 16) {
                Label(skill.effectivePath, systemImage: "folder")
                    .font(.caption)
                    .foregroundStyle(Color.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .textSelection(.enabled)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.primary.opacity(0.03), in: RoundedRectangle(cornerRadius: 8))
    }

    // MARK: - 辅助方法

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

    private func revealInFinder(_ skill: SkillSummary) {
        let url = URL(fileURLWithPath: skill.effectivePath)
        NSWorkspace.shared.activateFileViewerSelecting([url])
    }
}

// MARK: - AgentToggleCard

/// Agent 激活卡片组件 — 水平排列的可点击卡片，替代原 Toggle 列表
private struct AgentToggleCard: View {
    let agent: AgentModel
    let isEnabled: Bool
    let isLoading: Bool
    let onToggle: (Bool) -> Void

    var body: some View {
        Button {
            onToggle(!isEnabled)
        } label: {
            HStack(spacing: 10) {
                Image(systemName: isEnabled ? "checkmark.circle.fill" : "circle")
                    .font(.title3)
                    .foregroundStyle(isEnabled ? Color.accentColor : Color.secondary.opacity(0.5))
                    .symbolEffect(.bounce, value: isEnabled)

                VStack(alignment: .leading, spacing: 2) {
                    Text(agent.name)
                        .font(.callout.weight(.medium))
                        .foregroundStyle(isEnabled ? Color.primary : Color.secondary)

                    HStack(spacing: 4) {
                        Circle()
                            .fill(agent.detected ? Color.green : Color.orange)
                            .frame(width: 6, height: 6)
                        Text(agent.detected ? String(localized: "已安装") : String(localized: "未检测到"))
                            .font(.caption2)
                            .foregroundStyle(Color.secondary)
                    }
                }

                Spacer()
            }
            .padding(12)
            .background(
                isEnabled ? Color.accentColor.opacity(0.08) : Color.primary.opacity(0.03),
                in: RoundedRectangle(cornerRadius: 10)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 10)
                    .stroke(isEnabled ? Color.accentColor.opacity(0.35) : Color.primary.opacity(0.08), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .disabled(isLoading)
    }
}

// MARK: - MarkdownBodyView

/// Markdown 正文渲染组件 — 使用 SwiftUI 原生 AttributedString 解析渲染
struct MarkdownBodyView: View {
    let markdown: String

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            ForEach(Array(blocks.enumerated()), id: \.offset) { _, block in
                switch block {
                case .heading(let level, let text):
                    Text(text)
                        .font(headingFont(level))
                        .fontWeight(.bold)
                        .padding(.top, level <= 2 ? 8 : 4)
                case .codeBlock(let code):
                    ScrollView(.horizontal, showsIndicators: false) {
                        Text(code)
                            .font(.system(.callout, design: .monospaced))
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    .padding(12)
                    .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 8))
                case .paragraph(let text):
                    Text(attributedString(from: text))
                        .font(.body)
                        .textSelection(.enabled)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    // MARK: - 块级解析

    private enum Block {
        case heading(Int, String)
        case codeBlock(String)
        case paragraph(String)
    }

    private var blocks: [Block] {
        var result: [Block] = []
        let lines = markdown.components(separatedBy: "\n")
        var i = 0
        var paragraphBuffer: [String] = []

        func flushParagraph() {
            let text = paragraphBuffer.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
            if !text.isEmpty {
                result.append(.paragraph(text))
            }
            paragraphBuffer.removeAll()
        }

        while i < lines.count {
            let line = lines[i]

            // 代码块
            if line.hasPrefix("```") {
                flushParagraph()
                var codeLines: [String] = []
                i += 1
                while i < lines.count && !lines[i].hasPrefix("```") {
                    codeLines.append(lines[i])
                    i += 1
                }
                i += 1 // 跳过结尾 ```
                result.append(.codeBlock(codeLines.joined(separator: "\n")))
                continue
            }

            // 标题（# ~ ######）
            if let match = line.wholeMatch(of: /^(#{1,6})\s+(.+)$/) {
                flushParagraph()
                let level = match.1.count
                let text = String(match.2)
                result.append(.heading(level, text))
                i += 1
                continue
            }

            // 空行分段
            if line.trimmingCharacters(in: .whitespaces).isEmpty {
                flushParagraph()
                i += 1
                continue
            }

            // 普通文本行
            paragraphBuffer.append(line)
            i += 1
        }

        flushParagraph()
        return result
    }

    // MARK: - 行内 Markdown → AttributedString

    private func attributedString(from text: String) -> AttributedString {
        // 使用 SwiftUI 原生 Markdown 解析（支持粗体、斜体、代码、链接）
        if let attributed = try? AttributedString(markdown: text, options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)) {
            return attributed
        }
        // 解析失败时回退为纯文本
        return AttributedString(text)
    }

    private func headingFont(_ level: Int) -> Font {
        switch level {
        case 1: .title
        case 2: .title2
        case 3: .title3
        default: .headline
        }
    }
}

/// AddSkillSheet - 添加/导入 Skill 弹窗
/// 支持两类导入模式：
/// 1. 本地导入：选取本地目录或 ZIP 压缩包，校验结构后写入 SKM 不可变对象存储；
/// 2. Git/命令行两步式导入向导：
///    - 第一步：输入 Git URL、GitHub 简写或 npx skills add 命令；
///    - 第二步：调用 Core sources.preview 扫描仓库候选技能，展示合法性与冲突提示，由用户勾选所需技能后批量导入。
struct AddSkillSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    @State private var mode = 0
    @State private var path = ""
    @State private var remote = ""
    @State private var sourceName = ""
    @State private var tags = ""

    // Git 两步向导状态
    @State private var wizardStep = 0
    @State private var isScanning = false
    @State private var previewResult: SourcePreviewResult?
    @State private var selectedPaths: Set<String> = []

    init(model: AppModel, initialMode: Int = 0) {
        self.model = model
        _mode = State(initialValue: initialMode)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            headerView

            if mode == 0 {
                localImportView
            } else {
                if wizardStep == 0 {
                    gitInputStepView
                } else {
                    gitPreviewStepView
                }
            }
        }
        .padding(24)
        .frame(
            minWidth: mode == 1 && wizardStep == 1 ? 660 : 560,
            minHeight: mode == 1 && wizardStep == 1 ? 480 : 300
        )
    }

    private var headerView: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(mode == 1 && wizardStep == 1 ? String(localized: "选择要导入的 Skill") : String(localized: "添加 Skill"))
                    .font(.title2.bold())
                Spacer()
                if mode == 1 {
                    Text(wizardStep == 0 ? "1/2 步：输入来源" : "2/2 步：勾选技能")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(.quaternary, in: Capsule())
                }
            }

            if wizardStep == 0 {
                Picker("来源", selection: $mode) {
                    Text("本地").tag(0)
                    Text("Git / 命令").tag(1)
                }
                .pickerStyle(.segmented)
            }
        }
    }

    private var localImportView: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                TextField("Skill 目录或 ZIP", text: $path)
                Button("选择…") { chooseLocalSkill() }
            }
            TextField("标签，以逗号分隔（可选）", text: $tags)
            Text("本地内容会被验证并写入 SKM 的不可变对象库。")
                .font(.caption).foregroundStyle(.secondary)

            Spacer()

            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("导入") {
                    Task {
                        await model.addLocalSkill(path: path, tags: parseTags(tags))
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(path.trimmingCharacters(in: .whitespaces).isEmpty)
            }
        }
    }

    private var gitInputStepView: some View {
        VStack(alignment: .leading, spacing: 14) {
            TextField("Git URL、owner/repo 或 npx skills add …", text: $remote)
            TextField("来源名称（可选，留空自动提取）", text: $sourceName)
            TextField("标签，以逗号分隔（可选）", text: $tags)
            Text("凭据由系统 Git、SSH Agent 或 Credential Helper 管理，SKM 不保存 Token。")
                .font(.caption).foregroundStyle(.secondary)

            Spacer()

            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button {
                    Task { await startPreview() }
                } label: {
                    HStack(spacing: 6) {
                        if isScanning {
                            ProgressView().controlSize(.small)
                            Text("正在扫描…")
                        } else {
                            Text("下一步：扫描技能")
                            Image(systemName: "arrow.right")
                        }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(remote.trimmingCharacters(in: .whitespaces).isEmpty || isScanning)
            }
        }
    }

    private var gitPreviewStepView: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let preview = previewResult {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(preview.source.name).fontWeight(.semibold)
                        Text(preview.source.url)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                    Spacer()
                    if let rev = preview.source.revision {
                        Text(String(rev.prefix(8)))
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(.quaternary, in: RoundedRectangle(cornerRadius: 4))
                    }
                }
                .padding(10)
                .background(.quaternary.opacity(0.4), in: RoundedRectangle(cornerRadius: 8))

                HStack {
                    let validCandidates = preview.skills.filter(\.valid)
                    Text(String(format: String(localized: "已选择 %lld / %lld 个可用技能"), locale: .current, selectedPaths.count, validCandidates.count))
                        .font(.callout)
                        .foregroundStyle(.secondary)
                    Spacer()
                    Button("全选") {
                        selectedPaths = Set(validCandidates.map(\.path))
                    }
                    .buttonStyle(.link)
                    .disabled(selectedPaths.count == validCandidates.count)

                    Text("·").foregroundStyle(.secondary)

                    Button("取消全选") {
                        selectedPaths.removeAll()
                    }
                    .buttonStyle(.link)
                    .disabled(selectedPaths.isEmpty)
                }

                ScrollView {
                    LazyVStack(spacing: 8) {
                        ForEach(preview.skills) { candidate in
                            SkillCandidateRow(
                                candidate: candidate,
                                isSelected: selectedPaths.contains(candidate.path),
                                onToggle: {
                                    if selectedPaths.contains(candidate.path) {
                                        selectedPaths.remove(candidate.path)
                                    } else {
                                        selectedPaths.insert(candidate.path)
                                    }
                                }
                            )
                        }
                    }
                    .padding(.vertical, 2)
                }
                .frame(maxHeight: 260)
            }

            Spacer()

            HStack {
                Button("上一步", systemImage: "arrow.left") {
                    wizardStep = 0
                }
                .disabled(model.isLoading)

                Spacer()

                Button("取消", role: .cancel) { dismiss() }
                Button(String(format: String(localized: "导入所选技能 (%lld)"), locale: .current, selectedPaths.count)) {
                    Task { await confirmImport() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(selectedPaths.isEmpty || model.isLoading)
            }
        }
    }

    private func chooseLocalSkill() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = true
        panel.allowsMultipleSelection = false
        panel.allowedContentTypes = [.zip]
        if panel.runModal() == .OK { path = panel.url?.path ?? path }
    }

    private func startPreview() async {
        isScanning = true
        defer { isScanning = false }
        do {
            let result = try await model.previewSource(
                input: remote,
                name: sourceName.trimmingCharacters(in: .whitespaces).isEmpty ? nil : sourceName
            )
            previewResult = result
            // 默认勾选所有有效技能；如果有 requestedSkills 则按 requestedSkills 过滤
            let valid = result.skills.filter(\.valid)
            if let requested = result.requestedSkills, !requested.isEmpty {
                let requestedSet = Set(requested)
                selectedPaths = Set(valid.filter { requestedSet.contains($0.name) || requestedSet.contains($0.path) }.map(\.path))
                if selectedPaths.isEmpty {
                    selectedPaths = Set(valid.map(\.path))
                }
            } else {
                selectedPaths = Set(valid.map(\.path))
            }
            wizardStep = 1
        } catch {
            model.errorMessage = error.localizedDescription
        }
    }

    private func confirmImport() async {
        guard let preview = previewResult else { return }
        let success = await model.addRemoteSkill(
            input: preview.source.url,
            name: preview.source.name,
            ref: preview.source.ref,
            paths: Array(selectedPaths),
            tags: parseTags(tags)
        )
        if success {
            dismiss()
        }
    }
}

/// SkillCandidateRow - 候选技能列表项
private struct SkillCandidateRow: View {
    let candidate: SkillCandidate
    let isSelected: Bool
    let onToggle: () -> Void

    var body: some View {
        Button(action: {
            if candidate.valid { onToggle() }
        }) {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: candidate.valid ? (isSelected ? "checkmark.square.fill" : "square") : "xmark.square")
                    .font(.title3)
                    .foregroundStyle(candidate.valid ? (isSelected ? Color.accentColor : Color.secondary) : Color.secondary.opacity(0.4))
                    .frame(width: 20)

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 8) {
                        Text(candidate.name)
                            .fontWeight(.semibold)
                            .foregroundStyle(candidate.valid ? Color.primary : Color.secondary)
                        if !candidate.path.isEmpty && candidate.path != candidate.name {
                            Text(candidate.path)
                                .font(.caption.monospaced())
                                .foregroundStyle(Color.secondary)
                        }
                    }

                    if let desc = candidate.description, !desc.isEmpty {
                        Text(desc)
                            .font(.caption)
                            .foregroundStyle(Color.secondary)
                            .lineLimit(2)
                    }

                    if !candidate.valid, let error = candidate.error {
                        Label(error, systemImage: "exclamationmark.triangle.fill")
                            .font(.caption2)
                            .foregroundStyle(Color.orange)
                    }
                }

                Spacer()
            }
            .padding(10)
            .background(
                isSelected ? Color.accentColor.opacity(0.08) : Color.primary.opacity(0.04),
                in: RoundedRectangle(cornerRadius: 8)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(isSelected ? Color.accentColor.opacity(0.35) : Color.clear, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .disabled(!candidate.valid)
    }
}

/// SkillEditorSheet - 在线编辑 Skill 弹窗
/// 包含 Markdown 源码编辑器、标签输入与历史版本快照回滚入口。
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
    @State private var showsHistory = false

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
                Button("历史版本…", systemImage: "clock.arrow.circlepath") {
                    showsHistory = true
                }
                .help("查看修改历史快照、Diff 与回滚")

                Spacer()

                Text("保存时校验 baseHash 防止并发冲突。")
                    .font(.caption).foregroundStyle(.secondary)
                Button("取消", role: .cancel) { dismiss() }
                Button("保存") {
                    Task { await save() }
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding(24)
        .frame(minWidth: 720, minHeight: 560)
        .sheet(isPresented: $showsHistory, onDismiss: { Task { await reloadDetails() } }) {
            HistorySheet(model: model, kind: "skill", itemID: details.id, title: details.name)
        }
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

    private func reloadDetails() async {
        do {
            let refreshed = try await model.skillDetails(details.id)
            content = refreshed.content
            tags = refreshed.tags.joined(separator: ", ")
            baseHash = refreshed.hash
            latest = nil
        } catch {
            model.errorMessage = error.localizedDescription
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

import AppKit
import SwiftUI

/// PromptsListView - 提示词列表视图
/// 展示所有已创建/导入的 Prompt 模板，支持按标签展开/收起、搜索、导入外部 Markdown 及新建提示词。
struct PromptsListView: View {
    @Bindable var model: AppModel
    @State private var search = ""
    @State private var showsNewPrompt = false
    @State private var isAllGroupExpanded = true
    @State private var expandedTags: Set<String> = []
    @State private var showsTagManager = false

    private var filtered: [PromptSummary] {
        return model.prompts.filter {
            search.isEmpty ||
                $0.name.localizedStandardContains(search) ||
                $0.description.localizedStandardContains(search) ||
                $0.tags.contains(where: { $0.localizedStandardContains(search) })
        }
    }

    var body: some View {
        let visiblePrompts = filtered
        let tagGroups = itemsGroupedByTag(visiblePrompts, tags: \.tags)

        VStack(spacing: 0) {
            CollectionSearchField(title: "搜索 Prompt", text: $search)
                .accessibilityIdentifier("prompt-search-field")
            Group {
                if model.prompts.isEmpty && !model.isLoading {
                    ContentUnavailableView {
                        Label("还没有 Prompt", systemImage: "text.bubble")
                    } description: {
                        Text("创建可复用、带变量定义的提示词。")
                    } actions: {
                        Button("新建 Prompt") { showsNewPrompt = true }
                            .buttonStyle(.borderedProminent)
                    }
                } else if visiblePrompts.isEmpty && !search.isEmpty {
                    ContentUnavailableView.search(text: search)
                } else {
                    List(selection: $model.selectedPromptID) {
                        if !visiblePrompts.isEmpty {
                            TagGroupHeader(
                                title: AppLocalization.string("全部"),
                                systemImage: "text.bubble",
                                count: visiblePrompts.count,
                                isExpanded: isAllGroupExpanded
                            ) {
                                isAllGroupExpanded.toggle()
                            }
                            .accessibilityIdentifier("prompts-group-all")

                            if isAllGroupExpanded {
                                ForEach(visiblePrompts) { prompt in
                                    PromptSummaryRow(prompt: prompt)
                                        .libraryListRow(isSelected: model.selectedPromptID == prompt.id)
                                        .accessibilityIdentifier("prompt-row-\(prompt.id)")
                                        .tag(prompt.id)
                                        .contextMenu {
                                            Button("新建 Prompt…") {
                                                showsNewPrompt = true
                                            }
                                            Button("导入 Prompt…") {
                                                importPrompt()
                                            }
                                            Divider()
                                            Button("管理 Prompt 标签…") {
                                                showsTagManager = true
                                            }
                                        }
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
                                .accessibilityIdentifier("prompts-group-\(group.tag)")

                                if expandedTags.contains(group.tag) {
                                    ForEach(group.items) { prompt in
                                        PromptSummaryRow(prompt: prompt)
                                            .libraryListRow(isSelected: model.selectedPromptID == prompt.id)
                                            .accessibilityIdentifier("prompt-row-\(prompt.id)")
                                            .tag(prompt.id)
                                            .contextMenu {
                                                Button("新建 Prompt…") {
                                                    showsNewPrompt = true
                                                }
                                                Button("导入 Prompt…") {
                                                    importPrompt()
                                                }
                                                Divider()
                                                Button("管理 Prompt 标签…") {
                                                    showsTagManager = true
                                                }
                                            }
                                    }
                                }
                            }
                        }
                    }
                    .listStyle(.plain)
                    .contentMargins(.horizontal, 0, for: .scrollContent)
                    .scrollContentBackground(.hidden)
                    .environment(\.defaultMinListRowHeight, 28)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            CollectionFooter(count: visiblePrompts.count, symbol: "text.bubble")
        }
        .safeAreaInset(edge: .top, spacing: 0) {
            CollectionHeader(title: "Prompts") {
                Group {
                    Button("标签管理", systemImage: "tag") {
                        showsTagManager = true
                    }
                    .help("集中管理 Prompt 标签")
                    .accessibilityIdentifier("prompts-manage-tags-button")

                    Button("导入 Prompt", systemImage: "bubble") { importPrompt() }
                    Button("新建 Prompt", systemImage: "plus") { showsNewPrompt = true }
                }
                .topToolbarActionStyle()
            }
        }
        .sheet(isPresented: $showsNewPrompt) { PromptEditorSheet(model: model, details: nil) }
        .sheet(isPresented: $showsTagManager) { TagManagementSheet(model: model, target: .prompts) }
        .onChange(of: model.pendingCommand?.id) { _, _ in
            guard let command = model.pendingCommand, command.section == .prompts else { return }
            switch command.kind {
            case .create:
                showsNewPrompt = true
            case .importItem:
                importPrompt()
            case .deleteSelection:
                return
            }
            model.consumeCommand(command.id)
        }
    }

    private func toggleTag(_ tag: String) {
        if expandedTags.contains(tag) {
            expandedTags.remove(tag)
        } else {
            expandedTags.insert(tag)
        }
    }

    private func importPrompt() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = false
        panel.canChooseFiles = true
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK, let url = panel.url {
            do {
                let content = try String(contentsOf: url, encoding: .utf8)
                Task { await model.importPrompt(content: content) }
            } catch {
                model.errorMessage = error.localizedDescription
            }
        }
    }
}

/// PromptDetailView - 提示词详情视图
/// 展示提示词变量列表与 Markdown 正文，提供 QuickLook 预览、一键复制与导出 .md 文件。
struct PromptDetailView: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Bindable var model: AppModel
    @State private var details: PromptDetails?
    @State private var showsEditor = false
    @State private var confirmsDelete = false
    @State private var showsRender = false
    @State private var showsCopiedFeedback = false
    @State private var copyFeedbackTask: Task<Void, Never>?

    var body: some View {
        Group {
            if let id = model.selectedPromptID, let prompt = model.prompts.first(where: { $0.id == id }) {
                ScrollView {
                    VStack(alignment: .leading, spacing: SKMDesign.librarySectionSpacing) {
                        promptHeader(prompt)

                        if let variables = prompt.variables, !variables.isEmpty {
                            variablesSection(variables)
                        }

                        contentSection
                    }
                    .libraryReadingLayout()
                }
                .task(id: "\(id):\(prompt.hash)") { await loadDetails(id) }
                .safeAreaInset(edge: .top, spacing: 0) {
                    DetailToolbar(isLoading: model.isLoading) {
                        Group {
                            Button("快速查看", systemImage: "eye.slash") { Task { await showQuickLook() } }
                            Button("复制", systemImage: "paperclip", action: copyBody)
                                .disabled(details == nil)
                            Button("填写变量", systemImage: "tag") { showsRender = true }
                                .disabled(details == nil)
                            Button("导出", systemImage: "bubble") { exportPrompt(prompt.name) }
                                .disabled(details == nil)
                            Button("编辑", systemImage: "tag") { showsEditor = true }
                                .disabled(details == nil)
                            Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                        }
                        .topToolbarActionStyle()
                    }
                }
                .sheet(isPresented: $showsEditor, onDismiss: { Task { await loadDetails(id) } }) {
                    if let details { PromptEditorSheet(model: model, details: details) }
                }
                .sheet(isPresented: $showsRender) {
                    if let details { PromptRenderSheet(model: model, details: details) }
                }
                .confirmationDialog("移除 \(prompt.name)？", isPresented: $confirmsDelete) {
                    Button("移除 Prompt", role: .destructive) { Task { await model.removePrompt(id: id) } }
                }
                .onChange(of: model.pendingCommand?.id) { _, _ in
                    guard let command = model.pendingCommand,
                          command.section == .prompts,
                          command.kind == .deleteSelection else { return }
                    confirmsDelete = true
                    model.consumeCommand(command.id)
                }
            } else {
                ContentUnavailableView("选择一个 Prompt", systemImage: "text.bubble", description: Text("在左侧选择模板，填写变量或复制到你的工作流程。"))
            }
        }
        .onDisappear {
            copyFeedbackTask?.cancel()
        }
    }

    private func promptHeader(_ prompt: PromptSummary) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(prompt.name)
                .font(.system(size: 30, weight: .semibold))
                .textSelection(.enabled)

            if !prompt.description.isEmpty {
                Text(prompt.description)
                    .font(.system(size: 15))
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }

            ViewThatFits(in: .horizontal) {
                HStack(spacing: 18) {
                    Label(prompt.source, systemImage: "folder").libraryMetadataPill()
                    ForEach(prompt.tags, id: \.self) { tag in
                        Label(tag, systemImage: "tag")
                            .libraryMetadataPill()
                            .fixedSize()
                    }
                }
                VStack(alignment: .leading, spacing: 10) {
                    Label(prompt.source, systemImage: "folder").libraryMetadataPill()
                    ForEach(prompt.tags, id: \.self) { tag in
                        Label(tag, systemImage: "tag").libraryMetadataPill()
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func variablesSection(_ variables: [PromptVariable]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("变量")
                    .font(.headline)
                Text(variables.count, format: .number)
                    .monospacedDigit()
                    .skmMetadataPill(tint: .secondary)
                Spacer()
            }

            VStack(alignment: .leading, spacing: 0) {
                ForEach(variables, id: \.name) { variable in
                    HStack(spacing: 12) {
                        Text("{{\(variable.name)}}")
                            .font(.body.monospaced())
                        Spacer()
                        if variable.required == true {
                            Text("必填")
                                .skmMetadataPill()
                        }
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)

                    if variable.name != variables.last?.name {
                        Divider()
                    }
                }
            }
            .skmSurface(radius: SKMDesign.compactCardRadius)
        }
    }

    private var contentSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            if showsCopiedFeedback {
                Label("已复制", systemImage: "checkmark.circle")
                    .font(.caption)
                    .foregroundStyle(SKMDesign.successTint)
                    .transition(.opacity)
            }

            Group {
                if let body = details?.body {
                    if !body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                        MarkdownBodyView(markdown: body)
                    }
                } else {
                    HStack(spacing: 8) {
                        ProgressView().controlSize(.small)
                        Text("正在读取…").foregroundStyle(.secondary)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .topLeading)
        }
        .animation(reduceMotion ? nil : .easeOut(duration: 0.15), value: showsCopiedFeedback)
    }

    private func loadDetails(_ id: String) async {
        details = nil
        do {
            let loaded = try await model.promptDetails(id)
            guard !Task.isCancelled, model.selectedPromptID == id else { return }
            details = loaded
        } catch {
            guard !Task.isCancelled, model.selectedPromptID == id else { return }
            model.errorMessage = error.localizedDescription
        }
    }

    private func copyBody() {
        guard let body = details?.body else { return }
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(body, forType: .string)
        model.announce(AppLocalization.string("Prompt 已复制"))
        copyFeedbackTask?.cancel()
        showsCopiedFeedback = true
        copyFeedbackTask = Task {
            try? await Task.sleep(for: .seconds(1.5))
            guard !Task.isCancelled else { return }
            showsCopiedFeedback = false
        }
    }

    private func showQuickLook() async {
        do {
            guard let url = try await model.quickLookURL() else { return }
            QuickLookPresenter.shared.show(url)
        } catch { model.errorMessage = error.localizedDescription }
    }

    private func exportPrompt(_ name: String) {
        guard let content = details?.content else { return }
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "\(name).md"
        if panel.runModal() == .OK, let url = panel.url {
            do {
                try content.write(to: url, atomically: true, encoding: .utf8)
                model.announce(AppLocalization.string("Prompt 已导出"))
            } catch {
                model.errorMessage = error.localizedDescription
            }
        }
    }
}

/// PromptEditorSheet - 提示词创建与编辑弹窗
/// 支持配置 Prompt 名称、描述、标签、正文模板以及动态参数变量（PromptVariableDraft）。
/// 具备 baseHash 乐观锁并发冲突处理，若检测到冲突可选择“使用磁盘版本”、“另存为新副本”或“保留草稿覆盖”。
struct PromptEditorSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let details: PromptDetails?
    @State private var name: String
    @State private var description: String
    @State private var promptBody: String
    @State private var tags: [String]
    @State private var baseHash: String?
    @State private var latest: PromptDetails?
    @State private var variables: [PromptVariableDraft]
    @State private var confirmsDiscard = false
    @FocusState private var nameFocused: Bool

    init(model: AppModel, details: PromptDetails?) {
        self.model = model
        self.details = details
        _name = State(initialValue: details?.name ?? "")
        _description = State(initialValue: details?.description ?? "")
        _promptBody = State(initialValue: details?.body ?? "")
        _tags = State(initialValue: details?.tags ?? ["general"])
        _baseHash = State(initialValue: details?.hash)
        _variables = State(initialValue: (details?.variables ?? []).map(PromptVariableDraft.init))
    }

    var body: some View {
        VStack(spacing: 0) {
            SheetHeader(
                title: details == nil ? AppLocalization.string("新建 Prompt") : AppLocalization.string("编辑 Prompt"),
                subtitle: AppLocalization.string("把好用的提示词保存为可复用模板。"),
                symbol: details == nil ? "text.badge.plus" : "square.and.pencil",
                tint: .purple
            )
            Divider()

            HStack(alignment: .top, spacing: 0) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 24) {
                        SheetSection("基本信息", symbol: "info.circle") {
                            VStack(alignment: .leading, spacing: 16) {
                                VStack(alignment: .leading, spacing: 6) {
                                    Text("名称").font(.caption).foregroundStyle(.secondary)
                                    TextField("名称", text: $name)
                                        .focused($nameFocused)
                                        .accessibilityIdentifier("prompt-name-field")
                                }
                                VStack(alignment: .leading, spacing: 6) {
                                    Text("描述").font(.caption).foregroundStyle(.secondary)
                                    TextField("描述", text: $description, axis: .vertical)
                                        .lineLimit(3...5)
                                        .accessibilityIdentifier("prompt-description-field")
                                }
                            }
                        }
                        Divider()
                        TagSelector(model: model, selectedTags: $tags, accessibilityIdentifier: "prompt-tags")
                    }
                    .padding(22)
                }
                .frame(width: 280)
                .frame(maxHeight: .infinity)
                .background(SKMDesign.canvas)

                Divider()

                VStack(spacing: 0) {
                    HStack(spacing: 8) {
                        Image(systemName: "doc.plaintext")
                            .foregroundStyle(.secondary)
                        Text("内容")
                            .fontWeight(.semibold)
                        Spacer()
                        Text("Markdown")
                            .font(.system(size: 10, weight: .medium, design: .monospaced))
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: 5))
                    }
                    .font(.system(size: 12))
                    .padding(.horizontal, 22)
                    .padding(.vertical, 14)

                    TextEditor(text: $promptBody)
                        .font(.system(size: 14, design: .monospaced))
                        .lineSpacing(5)
                        .scrollContentBackground(.hidden)
                        .padding(.horizontal, 18)
                        .padding(.bottom, 18)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .accessibilityIdentifier("prompt-body-editor")

                    Divider()
                    variableSection

                    if latest != nil {
                        Divider()
                        ScrollView { conflictSection.padding(16) }
                            .frame(maxHeight: 220)
                    }
                }
                .background(SKMDesign.detailCanvas)
            }

            SheetActionBar {
                Text(variableHint)
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
                    .fixedSize(horizontal: false, vertical: true)
            } actions: {
                Button("取消", role: .cancel) { requestDismiss() }
                    .keyboardShortcut(.cancelAction)
                    .disabled(model.isLoading)
                Button("保存") {
                    Task { await save() }
                }
                .buttonStyle(.borderedProminent)
                .keyboardShortcut("s", modifiers: .command)
                .disabled(!canSave || model.isLoading)
            }
        }
        .frame(width: 920, height: 640)
        .sheetChrome(model: model)
        .interactiveDismissDisabled(hasChanges || model.isLoading)
        .onAppear { nameFocused = details == nil }
        .confirmationDialog("放弃未保存的更改？", isPresented: $confirmsDiscard) {
            Button("放弃更改", role: .destructive) { dismiss() }
            Button("继续编辑", role: .cancel) { }
        } message: {
            Text("关闭后，本次未保存的编辑将丢失。")
        }
    }

    private var variableSection: some View {
        DisclosureGroup("变量（\(variables.count)）") {
            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if variables.isEmpty {
                        Text("暂无变量。需要复用动态内容时，可添加名称、类型和默认值。")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    ForEach($variables) { $variable in
                        PromptVariableEditor(variable: $variable) {
                            variables.removeAll { $0.id == variable.id }
                        }
                        .padding(12)
                        .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: 8))
                    }
                    Button("添加变量", systemImage: "plus") {
                        variables.append(PromptVariableDraft())
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
                .padding(.top, 10)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .frame(maxHeight: 170)
        }
        .font(.system(size: 12, weight: .medium))
        .padding(.horizontal, 22)
        .padding(.vertical, 14)
    }

    @ViewBuilder
    private var conflictSection: some View {
        if let latest {
            GroupBox("检测到并发修改") {
                VStack(alignment: .leading, spacing: 10) {
                    Text("磁盘版本在编辑期间发生了变化。你的草稿没有丢失。")
                        .foregroundStyle(.orange)
                    HStack(alignment: .top, spacing: 12) {
                        ConflictPreview(title: "你的草稿", content: promptBody)
                        ConflictPreview(title: "磁盘版本", content: latest.body)
                    }
                    HStack {
                        Button("使用磁盘版本") {
                            name = latest.name
                            description = latest.description
                            promptBody = latest.body
                            tags = latest.tags
                            variables = (latest.variables ?? []).map(PromptVariableDraft.init)
                            baseHash = latest.hash
                            self.latest = nil
                        }
                        Button("另存为新 Prompt") {
                            self.latest = nil
                            Task { await save(asCopy: true) }
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
    }

    private var hasChanges: Bool {
        name != (details?.name ?? "") || description != (details?.description ?? "") ||
        promptBody != (details?.body ?? "") || tags != (details?.tags ?? ["general"]) ||
        variables.map(\.model) != (details?.variables ?? []).map { PromptVariableDraft($0).model }
    }

    private func requestDismiss() {
        if hasChanges { confirmsDiscard = true } else { dismiss() }
    }

    private func save(asCopy: Bool = false) async {
        let saved = await model.savePrompt(
            id: asCopy ? nil : details?.id,
            name: asCopy ? "\(name)-copy" : name,
            description: description,
            body: promptBody,
            tags: tags,
            variables: variables.map(\.model),
            baseHash: asCopy ? nil : baseHash
        )
        if saved {
            dismiss()
        } else if model.lastErrorKind == "conflict", let id = details?.id {
            do { latest = try await model.promptDetails(id) }
            catch { model.errorMessage = error.localizedDescription }
        }
    }

    private var canSave: Bool {
        let names = variables.map { $0.name.trimmingCharacters(in: .whitespacesAndNewlines) }
        return !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            !description.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            !promptBody.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            names.allSatisfy { !$0.isEmpty } && Set(names).count == names.count
    }

    private var variableHint: String {
        if description.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return AppLocalization.string("请填写名称、描述和正文后保存。")
        }
        let names = variables.map { $0.name.trimmingCharacters(in: .whitespacesAndNewlines) }
        if Set(names).count != names.count { return AppLocalization.string("变量名不能重复。") }
        return AppLocalization.string("变量可在正文中使用 {{name}}。secret 类型只在内存中参与渲染。")
    }
}

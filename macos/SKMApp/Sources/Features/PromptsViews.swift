import AppKit
import SwiftUI

struct PromptsListView: View {
    @Bindable var model: AppModel
    @State private var search = ""
    @State private var selectedTag: String?
    @State private var showsNewPrompt = false

    private var availableTags: [String] {
        availableFilterTags(from: model.prompts.map(\.tags))
    }

    private var filtered: [PromptSummary] {
        return model.prompts.filter {
            let matchesSearch = search.isEmpty ||
                $0.name.localizedStandardContains(search) ||
                $0.description.localizedStandardContains(search) ||
                $0.tags.contains(where: { $0.localizedStandardContains(search) })
            return matchesSearch && matchesSelectedTag($0.tags, selectedTag: selectedTag)
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            if !availableTags.isEmpty {
                TagFilterBar(
                    tags: availableTags,
                    filteredCount: filtered.count,
                    totalCount: model.prompts.count,
                    selectedTag: $selectedTag
                )
            }

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
                } else if filtered.isEmpty, let tag = selectedTag, search.isEmpty {
                    ContentUnavailableView {
                        Label("没有匹配标签的 Prompt", systemImage: "tag.slash")
                    } description: {
                        Text("当前没有标记为“\(tag)”的 Prompt。")
                    } actions: {
                        Button("显示全部标签") { selectedTag = nil }
                    }
                } else if filtered.isEmpty {
                    ContentUnavailableView.search(text: search)
                } else {
                    List(filtered, selection: $model.selectedPromptID) { prompt in
                        VStack(alignment: .leading, spacing: 5) {
                            Text(prompt.name).fontWeight(.medium)
                            Text(prompt.description.isEmpty ? String(localized: "无描述") : prompt.description)
                                .font(.caption).foregroundStyle(.secondary).lineLimit(2)
                            if !prompt.tags.isEmpty {
                                Text(prompt.tags.joined(separator: " · "))
                                    .font(.caption).foregroundStyle(.tertiary)
                            }
                        }
                        .padding(.vertical, 4)
                        .tag(prompt.id)
                        .accessibilityElement(children: .ignore)
                        .accessibilityLabel(promptAccessibilityLabel(prompt))
                    }
                }
            }
        }
        .searchable(text: $search, prompt: "搜索 Prompt")
        .navigationTitle("Prompts")
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                Button("导入 Prompt", systemImage: "square.and.arrow.down") { importPrompt() }
                Button("新建 Prompt", systemImage: "plus") { showsNewPrompt = true }
            }
        }
        .sheet(isPresented: $showsNewPrompt) { PromptEditorSheet(model: model, details: nil) }
        .onChange(of: selectedTag) { _, _ in reconcileSelection() }
        .onChange(of: availableTags) { _, tags in
            if let selectedTag, !tags.contains(selectedTag) {
                self.selectedTag = nil
            }
        }
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

    private func promptAccessibilityLabel(_ prompt: PromptSummary) -> String {
        let tags = prompt.tags.isEmpty
            ? String(localized: "无标签")
            : String(
                format: String(localized: "标签 %@"),
                locale: .current,
                prompt.tags.joined(separator: String(localized: "、"))
            )
        return String(
            format: String(localized: "%1$@，来源 %2$@，%3$@"),
            locale: .current,
            prompt.name,
            prompt.source,
            tags
        )
    }

    private func reconcileSelection() {
        guard let selectedPromptID = model.selectedPromptID,
              !filtered.contains(where: { $0.id == selectedPromptID }) else { return }
        model.selectedPromptID = filtered.first?.id
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

struct PromptDetailView: View {
    @Bindable var model: AppModel
    @State private var details: PromptDetails?
    @State private var showsEditor = false
    @State private var showsRenderer = false
    @State private var showsHistory = false
    @State private var confirmsDelete = false

    var body: some View {
        Group {
            if let id = model.selectedPromptID, let prompt = model.prompts.first(where: { $0.id == id }) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 22) {
                        VStack(alignment: .leading, spacing: 8) {
                            Text(prompt.name).font(.largeTitle.bold())
                            Text(prompt.description.isEmpty ? String(localized: "无描述") : prompt.description)
                                .font(.title3).foregroundStyle(.secondary)
                        }
                        if let variables = prompt.variables, !variables.isEmpty {
                            GroupBox("变量") {
                                VStack(alignment: .leading, spacing: 10) {
                                    ForEach(variables, id: \.name) { variable in
                                        HStack {
                                            Text("{{\(variable.name)}}").font(.body.monospaced())
                                            Spacer()
                                            if variable.required == true { Text("必填").foregroundStyle(.secondary) }
                                        }
                                    }
                                }.padding(4)
                            }
                        }
                        GroupBox("内容") {
                            Text(details?.body ?? String(localized: "正在读取…"))
                                .font(.system(.body, design: .monospaced))
                                .textSelection(.enabled)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(8)
                        }
                    }
                    .padding(26)
                    .frame(maxWidth: 820, alignment: .leading)
                }
                .task(id: "\(id):\(prompt.hash)") { await loadDetails(id) }
                .toolbar {
                    ToolbarItemGroup(placement: .primaryAction) {
                        Button("快速查看", systemImage: "eye") { Task { await showQuickLook() } }
                        Button("填写变量", systemImage: "text.badge.checkmark") { showsRenderer = true }
                            .disabled(details == nil)
                        Button("历史", systemImage: "clock.arrow.circlepath") { showsHistory = true }
                        Button("复制", systemImage: "doc.on.doc") {
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(details?.body ?? "", forType: .string)
                            model.announce(String(localized: "Prompt 已复制"))
                        }
                        Button("导出", systemImage: "square.and.arrow.up") { exportPrompt(prompt.name) }
                        Button("编辑", systemImage: "pencil") { showsEditor = true }
                        Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    }
                }
                .sheet(isPresented: $showsEditor, onDismiss: { Task { await loadDetails(id) } }) {
                    if let details { PromptEditorSheet(model: model, details: details) }
                }
                .sheet(isPresented: $showsRenderer) {
                    if let details { PromptRenderSheet(model: model, details: details) }
                }
                .sheet(isPresented: $showsHistory, onDismiss: { Task { await loadDetails(id) } }) {
                    HistorySheet(model: model, kind: "prompt", itemID: id, title: prompt.name)
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
                ContentUnavailableView("选择一个 Prompt", systemImage: "text.bubble")
            }
        }
    }

    private func loadDetails(_ id: String) async {
        do { details = try await model.promptDetails(id) }
        catch { model.errorMessage = error.localizedDescription }
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
                model.announce(String(localized: "Prompt 已导出"))
            } catch {
                model.errorMessage = error.localizedDescription
            }
        }
    }
}

struct PromptEditorSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let details: PromptDetails?
    @State private var name: String
    @State private var description: String
    @State private var promptBody: String
    @State private var tags: String
    @State private var baseHash: String?
    @State private var latest: PromptDetails?
    @State private var variables: [PromptVariableDraft]

    init(model: AppModel, details: PromptDetails?) {
        self.model = model
        self.details = details
        _name = State(initialValue: details?.name ?? "")
        _description = State(initialValue: details?.description ?? "")
        _promptBody = State(initialValue: details?.body ?? "")
        _tags = State(initialValue: details?.tags.joined(separator: ", ") ?? "general")
        _baseHash = State(initialValue: details?.hash)
        _variables = State(initialValue: (details?.variables ?? []).map(PromptVariableDraft.init))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(details == nil ? String(localized: "新建 Prompt") : String(localized: "编辑 Prompt"))
                .font(.title2.bold())
            Form {
                TextField("名称", text: $name)
                    .accessibilityIdentifier("prompt-name-field")
                TextField("描述", text: $description)
                    .accessibilityIdentifier("prompt-description-field")
                TextField("标签，以逗号分隔", text: $tags)
                    .accessibilityIdentifier("prompt-tags-field")
            }
            .formStyle(.columns)
            TextEditor(text: $promptBody)
                .font(.system(.body, design: .monospaced))
                .border(.separator)
                .accessibilityIdentifier("prompt-body-editor")
            DisclosureGroup("变量（\(variables.count)）") {
                VStack(alignment: .leading, spacing: 12) {
                    ForEach($variables) { $variable in
                        PromptVariableEditor(variable: $variable) {
                            variables.removeAll { $0.id == variable.id }
                        }
                        if variable.id != variables.last?.id { Divider() }
                    }
                    Button("添加变量", systemImage: "plus") {
                        variables.append(PromptVariableDraft())
                    }
                }
                .padding(.vertical, 8)
            }
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
                                tags = latest.tags.joined(separator: ", ")
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
            HStack {
                Text(variableHint)
                    .font(.caption).foregroundStyle(.secondary)
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("保存") {
                    Task { await save() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(!canSave)
            }
        }
        .padding(24)
        .frame(minWidth: 680, minHeight: 520)
    }

    private func save(asCopy: Bool = false) async {
        let saved = await model.savePrompt(
            id: asCopy ? nil : details?.id,
            name: asCopy ? "\(name)-copy" : name,
            description: description,
            body: promptBody,
            tags: parseTags(tags),
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
            !promptBody.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            names.allSatisfy { !$0.isEmpty } && Set(names).count == names.count
    }

    private var variableHint: String {
        let names = variables.map { $0.name.trimmingCharacters(in: .whitespacesAndNewlines) }
        if Set(names).count != names.count { return String(localized: "变量名不能重复。") }
        return String(localized: "变量可在正文中使用 {{name}}。secret 类型只在内存中参与渲染。")
    }
}

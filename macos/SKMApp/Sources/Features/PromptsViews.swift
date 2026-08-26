import AppKit
import SwiftUI

struct PromptsListView: View {
    @Bindable var model: AppModel
    @State private var search = ""
    @State private var showsNewPrompt = false

    private var filtered: [PromptSummary] {
        guard !search.isEmpty else { return model.prompts }
        return model.prompts.filter {
            $0.name.localizedStandardContains(search) ||
            $0.description.localizedCaseInsensitiveContains(search) ||
            $0.tags.contains(where: { $0.localizedCaseInsensitiveContains(search) })
        }
    }

    var body: some View {
        Group {
            if model.prompts.isEmpty && !model.isLoading {
                ContentUnavailableView("还没有 Prompt", systemImage: "text.bubble", description: Text("创建可复用、带变量定义的提示词。"))
            } else {
                List(filtered, selection: $model.selectedPromptID) { prompt in
                    VStack(alignment: .leading, spacing: 5) {
                        Text(prompt.name).fontWeight(.medium)
                        Text(prompt.description.isEmpty ? "无描述" : prompt.description)
                            .font(.caption).foregroundStyle(.secondary).lineLimit(2)
                        if !prompt.tags.isEmpty {
                            Text(prompt.tags.joined(separator: " · "))
                                .font(.caption2).foregroundStyle(.tertiary)
                        }
                    }
                    .padding(.vertical, 4)
                    .tag(prompt.id)
                }
            }
        }
        .searchable(text: $search, prompt: "搜索 Prompt")
        .navigationTitle("Prompts")
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                Button("导入 Prompt", systemImage: "square.and.arrow.down") { importPrompt() }
                Button("新建 Prompt", systemImage: "plus") { showsNewPrompt = true }
                    .keyboardShortcut("n", modifiers: [.command, .shift])
            }
        }
        .sheet(isPresented: $showsNewPrompt) { PromptEditorSheet(model: model, details: nil) }
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
    @State private var confirmsDelete = false

    var body: some View {
        Group {
            if let id = model.selectedPromptID, let prompt = model.prompts.first(where: { $0.id == id }) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 22) {
                        VStack(alignment: .leading, spacing: 8) {
                            Text(prompt.name).font(.largeTitle.bold())
                            Text(prompt.description.isEmpty ? "无描述" : prompt.description)
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
                            Text(details?.body ?? "正在读取…")
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
                        Button("复制", systemImage: "doc.on.doc") {
                            NSPasteboard.general.clearContents()
                            NSPasteboard.general.setString(details?.body ?? "", forType: .string)
                            model.announce("Prompt 已复制")
                        }
                        Button("导出", systemImage: "square.and.arrow.up") { exportPrompt(prompt.name) }
                        Button("编辑", systemImage: "pencil") { showsEditor = true }
                        Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    }
                }
                .sheet(isPresented: $showsEditor, onDismiss: { Task { await loadDetails(id) } }) {
                    if let details { PromptEditorSheet(model: model, details: details) }
                }
                .confirmationDialog("移除 \(prompt.name)？", isPresented: $confirmsDelete) {
                    Button("移除 Prompt", role: .destructive) { Task { await model.removePrompt(id: id) } }
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

    private func exportPrompt(_ name: String) {
        guard let content = details?.content else { return }
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "\(name).md"
        if panel.runModal() == .OK, let url = panel.url {
            do {
                try content.write(to: url, atomically: true, encoding: .utf8)
                model.announce("Prompt 已导出")
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

    init(model: AppModel, details: PromptDetails?) {
        self.model = model
        self.details = details
        _name = State(initialValue: details?.name ?? "")
        _description = State(initialValue: details?.description ?? "")
        _promptBody = State(initialValue: details?.body ?? "")
        _tags = State(initialValue: details?.tags.joined(separator: ", ") ?? "general")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(details == nil ? "新建 Prompt" : "编辑 Prompt").font(.title2.bold())
            Form {
                TextField("名称", text: $name)
                TextField("描述", text: $description)
                TextField("标签，以逗号分隔", text: $tags)
            }
            .formStyle(.columns)
            TextEditor(text: $promptBody)
                .font(.system(.body, design: .monospaced))
                .border(.separator)
            HStack {
                Text("变量可在正文中使用 {{name}}；高级变量定义可直接通过 CLI 编辑。")
                    .font(.caption).foregroundStyle(.secondary)
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("保存") {
                    Task {
                        await model.savePrompt(
                            id: details?.id,
                            name: name,
                            description: description,
                            body: promptBody,
                            tags: parseTags(tags),
                            baseHash: details?.hash
                        )
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || promptBody.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24)
        .frame(minWidth: 680, minHeight: 520)
    }
}

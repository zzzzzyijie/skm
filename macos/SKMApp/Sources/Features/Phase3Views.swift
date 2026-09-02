import AppKit
import SwiftUI

/// PromptVariableDraft - 提示词变量草稿模型
/// 用于在 SwiftUI 表单中安全编辑 PromptVariable，负责将类型字符串、默认值、必填标记及以逗号分隔的 options 进行双向映射。
struct PromptVariableDraft: Identifiable, Hashable {
    let id: UUID
    var name: String
    var label: String
    var type: String
    var required: Bool
    var defaultValue: String
    var options: String
    var description: String

    init(
        id: UUID = UUID(),
        name: String = "",
        label: String = "",
        type: String = "text",
        required: Bool = false,
        defaultValue: String = "",
        options: String = "",
        description: String = ""
    ) {
        self.id = id
        self.name = name
        self.label = label
        self.type = type
        self.required = required
        self.defaultValue = defaultValue
        self.options = options
        self.description = description
    }

    init(_ value: PromptVariable) {
        self.init(
            name: value.name,
            label: value.label ?? "",
            type: value.type ?? "text",
            required: value.required ?? false,
            defaultValue: value.default ?? "",
            options: (value.options ?? []).joined(separator: ", "),
            description: value.description ?? ""
        )
    }

    var model: PromptVariable {
        PromptVariable(
            name: name.trimmingCharacters(in: .whitespacesAndNewlines),
            label: label.nilIfBlank,
            type: type,
            required: required,
            default: type == "secret" ? nil : defaultValue.nilIfBlank,
            options: type == "select" ? parseTags(options) : nil,
            description: description.nilIfBlank
        )
    }
}

private extension String {
    var nilIfBlank: String? {
        let value = trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
    }
}

/// 提示词单个变量编辑表单行组件
struct PromptVariableEditor: View {
    @Binding var variable: PromptVariableDraft
    let remove: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                TextField("变量名", text: $variable.name)
                    .textFieldStyle(.roundedBorder)
                    .accessibilityLabel("变量名")
                TextField("显示名称（可选）", text: $variable.label)
                    .textFieldStyle(.roundedBorder)
                Button("移除变量", systemImage: "minus.circle", role: .destructive, action: remove)
                    .labelStyle(.iconOnly)
            }
            HStack {
                Picker("类型", selection: $variable.type) {
                    Text("单行文本").tag("text")
                    Text("多行文本").tag("multiline")
                    Text("数字").tag("number")
                    Text("开关").tag("boolean")
                    Text("选项").tag("select")
                    Text("密码").tag("secret")
                }
                Toggle("必填", isOn: $variable.required)
                if variable.type != "secret" {
                    TextField("默认值（可选）", text: $variable.defaultValue)
                }
            }
            if variable.type == "select" {
                TextField("选项，以逗号分隔", text: $variable.options)
            }
            TextField("说明（可选）", text: $variable.description)
        }
    }
}

/// PromptRenderSheet - 提示词变量填充与动态渲染测试弹窗
/// 允许用户在应用内输入实际参数并即时测试渲染结果；secret 变量在内存中加密处理，不落盘。
struct PromptRenderSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let details: PromptDetails
    @State private var values: [String: String]
    @State private var rendered = ""
    @State private var missing: [String] = []
    @State private var errorMessage: String?
    @State private var isRendering = false

    init(model: AppModel, details: PromptDetails) {
        self.model = model
        self.details = details
        _values = State(initialValue: Dictionary(uniqueKeysWithValues: (details.variables ?? []).map {
            ($0.name, $0.type == "secret" ? "" : ($0.default ?? ""))
        }))
    }

    var body: some View {
        NavigationSplitView {
            Form {
                if variables.isEmpty {
                    ContentUnavailableView("没有变量", systemImage: "text.badge.checkmark", description: Text("可以直接复制 Prompt 内容。"))
                } else {
                    ForEach(variables, id: \.name) { variable in
                        input(for: variable)
                    }
                }
                if let errorMessage {
                    Label(errorMessage, systemImage: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                }
            }
            .formStyle(.grouped)
            .navigationTitle("填写变量")
            .toolbar {
                Button("渲染") { Task { await render() } }
                    .buttonStyle(.borderedProminent)
                    .disabled(isRendering)
            }
        } detail: {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Spacer()
                    Button("复制", systemImage: "doc.on.doc") { copyRendered() }
                        .disabled(rendered.isEmpty || !missing.isEmpty)
                }
                if !missing.isEmpty {
                    Label("缺少必填变量：\(missing.joined(separator: ", "))", systemImage: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                }
                ScrollView {
                    Text(rendered.isEmpty ? details.body : rendered)
                        .font(.body.monospaced())
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(12)
                }
                .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 10))
                HStack {
                    Text("secret 值不会写入磁盘、日志或历史记录。")
                        .font(.caption).foregroundStyle(.secondary)
                    Spacer()
                    Button("关闭", role: .cancel) { dismiss() }
                }
            }
            .padding(20)
        }
        .frame(minWidth: 850, minHeight: 560)
    }

    private var variables: [PromptVariable] { details.variables ?? [] }

    @ViewBuilder
    private func input(for variable: PromptVariable) -> some View {
        let title = variable.label?.isEmpty == false ? variable.label! : variable.name
        let binding = Binding(
            get: { values[variable.name, default: ""] },
            set: { values[variable.name] = $0 }
        )
        VStack(alignment: .leading, spacing: 5) {
            switch variable.type ?? "text" {
            case "multiline":
                Text(title)
                TextEditor(text: binding).frame(minHeight: 72)
            case "boolean":
                Toggle(title, isOn: Binding(
                    get: { ["true", "1", "yes", "on"].contains(binding.wrappedValue.lowercased()) },
                    set: { binding.wrappedValue = $0 ? "true" : "false" }
                ))
            case "select":
                Picker(title, selection: binding) {
                    Text("请选择").tag("")
                    ForEach(variable.options ?? [], id: \.self) { Text($0).tag($0) }
                }
            case "secret":
                SecureField(title, text: binding)
            default:
                TextField(title, text: binding)
            }
            if let description = variable.description, !description.isEmpty {
                Text(description).font(.caption).foregroundStyle(.secondary)
            }
        }
    }

    private func render() async {
        isRendering = true
        errorMessage = nil
        defer { isRendering = false }
        do {
            let response = try await model.renderPrompt(id: details.id, values: values)
            rendered = response.content
            missing = response.missingVariables
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func copyRendered() {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(rendered, forType: .string)
        model.announce(String(localized: "渲染结果已复制"))
    }
}

/// HistorySheet - 历史版本快照列表与 Diff 对比回滚弹窗
/// 支持查看 Skill 或 Prompt 的修改时间线（编辑前快照与回滚前快照），
/// 点击历史条目实时请求 Core 计算 Diff 差异并展示，支持一键恢复到指定版本。
struct HistorySheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let kind: String
    let itemID: String
    let title: String
    @State private var entries: [HistoryEntryModel] = []
    @State private var selectedID: String?
    @State private var diff = ""
    @State private var errorMessage: String?
    @State private var confirmsRestore = false

    var body: some View {
        NavigationSplitView {
            List(entries, selection: $selectedID) { entry in
                VStack(alignment: .leading, spacing: 4) {
                    Text(entry.current == true ? String(localized: "当前版本") : historyReason(entry.reason))
                        .fontWeight(.medium)
                    Text(historyDate(entry.createdAt))
                        .font(.caption).foregroundStyle(.secondary)
                    Text(String(entry.hash.prefix(12)))
                        .font(.caption2.monospaced()).foregroundStyle(.tertiary)
                }
                .padding(.vertical, 4)
                .tag(entry.id)
            }
            .navigationTitle("历史")
        } detail: {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    VStack(alignment: .leading) {
                        Text(title).font(.title2.bold())
                        Text("选择历史版本后与当前内容比较。")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("恢复此版本", systemImage: "arrow.uturn.backward") { confirmsRestore = true }
                        .buttonStyle(.borderedProminent)
                        .disabled(selectedID == nil || selectedID == "current")
                }
                if let errorMessage {
                    Label(errorMessage, systemImage: "exclamationmark.triangle.fill")
                        .foregroundStyle(.orange)
                }
                ScrollView([.horizontal, .vertical]) {
                    Text(diff.isEmpty ? String(localized: "请选择一个历史版本。") : diff)
                        .font(.body.monospaced())
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(12)
                }
                .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 10))
                HStack { Spacer(); Button("关闭", role: .cancel) { dismiss() } }
            }
            .padding(20)
        }
        .frame(minWidth: 860, minHeight: 540)
        .task { await load() }
        .onChange(of: selectedID) { _, value in
            guard let value, value != "current" else { diff = ""; return }
            Task { await loadDiff(value) }
        }
        .confirmationDialog("恢复这个历史版本？", isPresented: $confirmsRestore) {
            Button("恢复", role: .destructive) { Task { await restore() } }
        } message: {
            Text("当前内容会先保存为新的历史版本，然后恢复所选内容。")
        }
    }

    private func load() async {
        do {
            entries = try await model.history(kind: kind, itemID: itemID)
            selectedID = entries.first(where: { $0.current != true })?.id
        } catch { errorMessage = error.localizedDescription }
    }

    private func loadDiff(_ entryID: String) async {
        do {
            diff = try await model.historyDiff(kind: kind, itemID: itemID, from: entryID).diff
        } catch { errorMessage = error.localizedDescription }
    }

    private func restore() async {
        guard let selectedID, selectedID != "current" else { return }
        if await model.rollbackHistory(kind: kind, itemID: itemID, entryID: selectedID) {
            dismiss()
        } else {
            errorMessage = model.errorMessage
        }
    }

    private func historyReason(_ reason: String) -> String {
        reason == "rollback" ? String(localized: "回滚前") : String(localized: "编辑前")
    }

    private func historyDate(_ value: String) -> String {
        guard let date = ISO8601DateFormatter().date(from: value) else { return value }
        return date.formatted(date: .abbreviated, time: .shortened)
    }
}

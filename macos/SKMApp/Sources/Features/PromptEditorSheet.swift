import SwiftUI

struct PromptEditorSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let details: PromptDetails?
    @State private var draft: PromptDraft
    @State private var original: PromptDraft
    @State private var baseHash: String?
    @State private var latest: PromptDetails?
    @State private var showsPreview = false
    @State private var confirmsDiscard = false
    @State private var propertiesExpanded = true
    @State private var validates = false
    @FocusState private var focused: Field?

    private enum Field: Hashable { case name, description }

    init(model: AppModel, details: PromptDetails?, initialTags: [String] = []) {
        self.model = model
        self.details = details
        let draft = PromptDraft(details: details, initialTags: initialTags)
        _draft = State(initialValue: draft)
        _original = State(initialValue: draft)
        _baseHash = State(initialValue: details?.hash)
    }

    private var changed: Bool { draft != original }
    private var duplicateName: Bool { details == nil && model.prompts.contains { $0.name == draft.name } }
    private var canSave: Bool { draft.isValid && !duplicateName && (details == nil || changed) && latest == nil }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text(details == nil ? "新建 Prompt" : "编辑 Prompt").font(.title3.weight(.semibold))
                Spacer()
                Text("⌘S").font(.caption).foregroundStyle(.secondary).accessibilityHidden(true)
            }
            .padding(.horizontal, 24).frame(height: 56)
            Divider()
            GeometryReader { geometry in
                if geometry.size.width < 800 {
                    VStack(spacing: 0) {
                        DisclosureGroup("基本信息", isExpanded: $propertiesExpanded) {
                            ScrollView { properties }.frame(maxHeight: 170)
                        }.padding(.horizontal, 24).padding(.vertical, 12)
                        Divider()
                        writing
                    }
                } else {
                    HStack(spacing: 0) {
                        ScrollView { properties.padding(24) }
                            .frame(width: PromptDesign.propertiesWidth)
                            .frame(maxHeight: .infinity).background(SKMDesign.canvas)
                        Divider()
                        writing
                    }
                }
            }
            .disabled(model.isLoading)
            if latest != nil { conflict.disabled(model.isLoading) }
            SheetActionBar {
                if changed {
                    Text("未保存的更改").font(.callout).foregroundStyle(.secondary)
                }
            } actions: {
                Button("取消", role: .cancel, action: requestDismiss)
                    .keyboardShortcut(.cancelAction).disabled(model.isLoading)
                Button("保存") { Task { await save() } }
                    .buttonStyle(.borderedProminent).keyboardShortcut("s", modifiers: .command)
                    .disabled(!canSave || model.isLoading)
            }
        }
        .frame(minWidth: 720, idealWidth: PromptDesign.editorWidth, maxWidth: PromptDesign.editorWidth,
               minHeight: 480, idealHeight: PromptDesign.editorHeight, maxHeight: 600)
        .fixedSize()
        .sheetChrome(model: model)
        .interactiveDismissDisabled(changed || model.isLoading)
        .onAppear { if details == nil { focused = .name } }
        .onChange(of: focused) { old, _ in if old != nil { validates = true } }
        .confirmationDialog("放弃未保存的更改？", isPresented: $confirmsDiscard) {
            Button("放弃更改", role: .destructive) { dismiss() }
            Button("继续编辑", role: .cancel) { }
        } message: { Text("关闭后，本次未保存的编辑将丢失。") }
    }

    private var properties: some View {
        VStack(alignment: .leading, spacing: 24) {
            VStack(alignment: .leading, spacing: 7) {
                Text(details == nil ? "名称 *" : "名称").font(.callout).foregroundStyle(.secondary)
                if details == nil {
                    TextField("名称", text: $draft.name).focused($focused, equals: .name)
                        .accessibilityIdentifier("prompt-name-field")
                    Text("1–64 位小写字母、数字和连字符，首尾须为字母或数字。")
                        .font(.caption).foregroundStyle(.secondary)
                    if validates && !draft.validName { fieldError("请检查名称格式。") }
                    if duplicateName { fieldError("这个名称已存在。") }
                } else {
                    Text(draft.name).font(.body.monospaced()).textSelection(.enabled)
                        .accessibilityIdentifier("prompt-name-readonly")
                    Text("创建后名称固定").font(.caption).foregroundStyle(.secondary)
                }
            }
            VStack(alignment: .leading, spacing: 7) {
                Text("描述 *").font(.callout).foregroundStyle(.secondary)
                TextField("描述", text: $draft.description, axis: .vertical).lineLimit(3...5)
                    .focused($focused, equals: .description)
                    .accessibilityIdentifier("prompt-description-field")
                if validates && draft.description.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    fieldError("请填写描述。")
                } else if !draft.description.isEmpty && !draft.validDescription {
                    fieldError("描述不能超过 1024 字节。")
                }
            }
            PromptTagPicker(model: model, tags: $draft.tags)
            if details == nil && draft.tags.isEmpty {
                Text("未选择标签时使用资料库默认标签。")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var writing: some View {
        VStack(spacing: 0) {
            HStack {
                Text("正文 *").font(.callout.weight(.semibold))
                Spacer()
                Picker("正文显示方式", selection: $showsPreview) {
                    Text("编辑").tag(false)
                    Text("预览").tag(true)
                }.pickerStyle(.segmented).labelsHidden().fixedSize()
                    .accessibilityIdentifier("prompt-editor-mode")
            }.padding(.horizontal, 24).padding(.top, 16).padding(.bottom, 4)
            ZStack(alignment: .topLeading) {
                PromptTextEditor(text: $draft.body, focusOnAppear: details != nil)
                    .opacity(showsPreview ? 0 : 1).allowsHitTesting(!showsPreview)
                    .disabled(showsPreview)
                    .accessibilityHidden(showsPreview)
                if showsPreview {
                    ScrollView { MarkdownBodyView(markdown: draft.body).padding(24) }
                        .background(SKMDesign.detailCanvas)
                }
            }
            if draft.body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                Text("请填写正文，支持 Markdown。")
                    .font(.callout).foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading).padding(.horizontal, 24).padding(.bottom, 12)
            }
        }.background(SKMDesign.detailCanvas)
    }

    private func fieldError(_ text: LocalizedStringKey) -> some View {
        Label(text, systemImage: "exclamationmark.circle").font(.caption).foregroundStyle(.red)
    }

    @ViewBuilder private var conflict: some View {
        if let latest {
            Divider()
            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    Text("检测到并发修改").font(.headline)
                    Text("磁盘版本在编辑期间发生了变化。你的草稿没有丢失。").font(.callout).foregroundStyle(.secondary)
                    HStack(alignment: .top, spacing: 16) {
                        ConflictPreview(title: "你的草稿", content: draft.body)
                        ConflictPreview(title: "磁盘版本", content: latest.body)
                    }
                    HStack {
                        Button("使用磁盘版本") {
                            draft = PromptDraft(details: latest)
                            original = draft
                            baseHash = latest.hash
                            self.latest = nil
                        }
                        Button("另存为新 Prompt") { Task { await save(asCopy: true) } }
                        Spacer()
                        Button("保留草稿并覆盖") {
                            baseHash = latest.hash
                            self.latest = nil
                            Task { await save() }
                        }
                    }
                }.padding(20)
            }.frame(maxHeight: 200)
        }
    }

    private func requestDismiss() {
        if changed { confirmsDiscard = true } else { dismiss() }
    }

    private func save(asCopy: Bool = false) async {
        validates = true
        guard draft.isValid, !model.isLoading else { return }
        let saved = await model.savePrompt(
            id: asCopy ? nil : details?.id,
            name: asCopy ? draft.copyName(existing: model.prompts) : draft.name,
            description: draft.description, body: draft.body, tags: draft.tags,
            variables: draft.variables, baseHash: asCopy ? nil : baseHash
        )
        if saved {
            dismiss()
        } else if model.lastErrorKind == "conflict", let id = details?.id {
            do { latest = try await model.promptDetails(id) }
            catch { model.errorMessage = error.localizedDescription }
        }
    }
}

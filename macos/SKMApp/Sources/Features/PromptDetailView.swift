import SwiftUI

struct PromptDetailView: View {
    @Bindable var model: AppModel
    @State private var details: PromptDetails?
    @State private var loadedForHash: String?
    @State private var loadError: String?
    @State private var retry = 0
    @State private var editorDetails: PromptDetails?
    @State private var confirmsDelete = false
    @State private var showsSource = false
    @State private var copied = false

    private var selected: PromptSummary? { model.prompts.first { $0.id == model.selectedPromptID } }
    private var loaded: PromptDetails? {
        guard let selected, details?.id == selected.id, loadedForHash == selected.hash else { return nil }
        return details
    }

    var body: some View {
        Group {
            if let prompt = selected {
                VStack(spacing: 0) {
                    toolbar(prompt)
                    ScrollView {
                        VStack(alignment: .leading, spacing: PromptDesign.sectionSpacing) {
                            header(prompt)
                            content
                        }
                        .libraryReadingLayout()
                    }
                }
                .task(id: "\(prompt.id):\(prompt.hash):\(retry)") { await load(prompt) }
                .confirmationDialog("移除 \(prompt.name)？", isPresented: $confirmsDelete) {
                    Button("移除 Prompt", role: .destructive) { Task { await model.removePrompt(id: prompt.id) } }
                }
            } else {
                ContentUnavailableView("选择一个 Prompt", systemImage: "text.bubble", description: Text("在左侧选择模板，阅读、复制或导出到你的工作流程。"))
            }
        }
        .sheet(item: $editorDetails) { PromptEditorSheet(model: model, details: $0) }
        .task(id: copied) {
            guard copied else { return }
            do { try await Task.sleep(for: .seconds(1.5)) } catch { return }
            copied = false
        }
        .onChange(of: model.selectedPromptID) { _, _ in copied = false }
        .onChange(of: model.pendingCommand?.id) { _, _ in
            guard let command = model.pendingCommand, command.section == .prompts,
                  command.kind == .deleteSelection, selected != nil else { return }
            confirmsDelete = true
            model.consumeCommand(command.id)
        }
    }

    private func toolbar(_ prompt: PromptSummary) -> some View {
        HStack(spacing: 10) {
            Spacer(minLength: 0)
            Button("编辑", systemImage: "square.and.pencil") { editorDetails = loaded }
                .frame(minHeight: SKMDesign.toolbarActionSize)
                .disabled(loaded == nil || model.isLoading)
                .accessibilityIdentifier("prompt-edit-button")
            Button(copied ? "已复制" : "复制正文", systemImage: copied ? "checkmark" : "doc.on.doc") {
                guard let loaded else { return }
                PromptActions.copy(loaded.body, announces: true)
                copied = true
            }
            .buttonStyle(.borderedProminent)
            .disabled(loaded == nil)
            .accessibilityIdentifier("prompt-copy-button")
            Menu("更多操作", systemImage: "ellipsis") {
                Button("导出 Markdown…", systemImage: "square.and.arrow.up") {
                    guard let loaded else { return }
                    do { try PromptActions.export(loaded) }
                    catch { model.errorMessage = error.localizedDescription }
                }
                .disabled(loaded == nil)
                Button("快速查看", systemImage: "eye") { Task { await quickLook() } }
                Divider()
                Button("删除", systemImage: "trash", role: .destructive) { confirmsDelete = true }
                    .disabled(model.isLoading)
            }
            .menuStyle(.borderlessButton).menuIndicator(.hidden).fixedSize()
            .libraryToolbarIcon()
            .accessibilityIdentifier("prompt-more-menu")
        }
        .controlSize(.regular).buttonStyle(.borderless)
        .padding(.horizontal, SKMDesign.libraryListInset).frame(height: SKMDesign.libraryToolbarHeight)
        .overlay(alignment: .bottom) { Divider() }
    }

    private func header(_ prompt: PromptSummary) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(prompt.name).font(SKMDesign.libraryTitleFont)
                .textSelection(.enabled).fixedSize(horizontal: false, vertical: true)
            Text(prompt.description).font(SKMDesign.libraryDescriptionFont).foregroundStyle(.secondary)
                .textSelection(.enabled).fixedSize(horizontal: false, vertical: true)
            TagWrapLayout(spacing: 8) {
                Text(prompt.source == "local" || prompt.source.isEmpty ? AppLocalization.string("本地") : prompt.source)
                    .font(.callout).foregroundStyle(.secondary).padding(.vertical, 3)
                if prompt.tags.isEmpty {
                    Text("未分类").font(.callout).foregroundStyle(.secondary).padding(.vertical, 3)
                }
                ForEach(prompt.tags, id: \.self) { tag in
                    Button {
                        model.promptBrowser.filter = .tag(tag)
                        model.reconcilePromptSelection()
                    } label: {
                        Text(tag).font(.callout).lineLimit(1).padding(.horizontal, 8).padding(.vertical, 3)
                            .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: 5))
                    }
                    .buttonStyle(.plain).help(tag)
                }
            }
        }
    }

    @ViewBuilder private var content: some View {
        if let loaded {
            VStack(alignment: .leading, spacing: 20) {
                HStack {
                    Picker("正文显示方式", selection: $showsSource) {
                        Text("预览").tag(false)
                        Text("源文").tag(true)
                    }
                    .pickerStyle(.segmented).labelsHidden().fixedSize()
                    .accessibilityIdentifier("prompt-content-mode")
                    Spacer()
                    Text("Markdown").font(.caption).foregroundStyle(.secondary)
                }
                Divider()
                if showsSource {
                    Text(loaded.body).font(.system(size: 14, design: .monospaced))
                        .lineSpacing(5).textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .accessibilityIdentifier("prompt-source-body")
                } else {
                    MarkdownBodyView(markdown: loaded.body)
                }
            }
        } else if let loadError {
            ContentUnavailableView {
                Label("无法读取 Prompt", systemImage: "exclamationmark.triangle")
            } description: { Text(loadError) } actions: {
                Button("重试") { retry += 1 }.accessibilityIdentifier("prompt-retry-button")
            }
        } else {
            ProgressView("正在读取…").frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func load(_ prompt: PromptSummary) async {
        details = nil
        loadedForHash = nil
        loadError = nil
        do {
            let value = try await model.promptDetails(prompt.id)
            guard !Task.isCancelled, selected?.id == prompt.id, selected?.hash == prompt.hash else { return }
            loadedForHash = prompt.hash
            details = value
        } catch {
            guard !Task.isCancelled, selected?.id == prompt.id, selected?.hash == prompt.hash else { return }
            loadError = error.localizedDescription
        }
    }

    private func quickLook() async {
        do {
            guard let url = try await model.quickLookURL() else { return }
            QuickLookPresenter.shared.show(url)
        } catch { model.errorMessage = error.localizedDescription }
    }
}

import AppKit
import SwiftUI

struct PromptsListView: View {
    @Bindable var model: AppModel
    @State private var showsNewPrompt = false
    @State private var showsTagManager = false
    @State private var editorDetails: PromptDetails?
    @State private var deletion: PromptSummary?
    @State private var confirmsDelete = false

    var body: some View {
        @Bindable var browser = model.promptBrowser
        let visible = model.visiblePrompts
        VStack(spacing: 0) {
            header
            CollectionSearchField(title: "搜索名称、描述、标签", text: $browser.query, horizontalInset: PromptDesign.listInset)
                .accessibilityIdentifier("prompt-search-field")
            Group {
                if model.prompts.isEmpty && !model.isLoading {
                    ContentUnavailableView {
                        Label("还没有 Prompt", systemImage: "text.bubble")
                    } description: {
                        Text("保存提示词，随时查找、阅读和复制。")
                    } actions: {
                        Button("新建 Prompt") { showsNewPrompt = true }.buttonStyle(.borderedProminent)
                    }
                } else if visible.isEmpty && !model.isLoading {
                    ContentUnavailableView {
                        Label("没有匹配的 Prompt", systemImage: "magnifyingglass")
                    } description: {
                        Text("试试其他关键词或分类。")
                    } actions: {
                        Button("查看全部 Prompt") {
                            browser.query = ""
                            browser.filter = .all
                        }
                    }
                } else {
                    ScrollViewReader { proxy in
                        List(selection: $model.selectedPromptID) {
                            ForEach(visible) { prompt in
                                PromptSummaryRow(prompt: prompt)
                                    .libraryListRow(isSelected: model.selectedPromptID == prompt.id,
                                                    horizontalPadding: PromptDesign.listRowPadding)
                                    .tag(prompt.id).id(prompt.id)
                                    .accessibilityIdentifier("prompt-row-\(prompt.id)")
                                    .contextMenu { rowMenu(prompt) }
                            }
                        }
                        .listStyle(.plain)
                        .contentMargins(.horizontal, 0, for: .scrollContent)
                        .scrollContentBackground(.hidden)
                        .onChange(of: model.promptBrowser.revealToken) { _, _ in
                            if let id = model.selectedPromptID { proxy.scrollTo(id) }
                        }
                        .onChange(of: model.selectedPromptID) { _, id in
                            if let id { proxy.scrollTo(id) }
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            HStack {
                if browser.query.isEmpty {
                    Text("共 \(visible.count) 项")
                } else {
                    Text("找到 \(visible.count) 项 · 当前类共 \(model.prompts.filter { browser.filter.includes($0) }.count) 项")
                }
                Spacer()
            }
            .libraryFooterStyle()
        }
        .onChange(of: visible, initial: true) { _, _ in model.reconcilePromptSelection() }
        .sheet(isPresented: $showsNewPrompt) {
            PromptEditorSheet(model: model, details: nil, initialTags: browser.initialTags)
        }
        .sheet(item: $editorDetails) { PromptEditorSheet(model: model, details: $0) }
        .sheet(isPresented: $showsTagManager) { TagManagementSheet(model: model, target: .prompts) }
        .confirmationDialog("移除 Prompt？", isPresented: $confirmsDelete, presenting: deletion) { prompt in
            Button("移除 Prompt", role: .destructive) { Task { await model.removePrompt(id: prompt.id) } }
        } message: { Text($0.name) }
        .onChange(of: model.pendingCommand?.id) { _, _ in
            guard let command = model.pendingCommand, command.section == .prompts else { return }
            switch command.kind {
            case .create: showsNewPrompt = true
            case .importItem: importPrompt()
            case .deleteSelection: return
            }
            model.consumeCommand(command.id)
        }
    }

    private var header: some View {
        HStack(spacing: 8) {
            Text(model.promptBrowser.filter.title).font(.headline).lineLimit(1).help(model.promptBrowser.filter.title)
            Spacer(minLength: 4)
            Button("新建 Prompt", systemImage: "plus") { showsNewPrompt = true }
                .help("新建 Prompt").accessibilityIdentifier("prompts-create-button")
                .libraryToolbarIcon()
            Menu("列表选项", systemImage: "ellipsis") {
                Button("导入 Prompt…", systemImage: "square.and.arrow.down", action: importPrompt)
                Button("管理 Prompt 标签…", systemImage: "tag") { showsTagManager = true }
                Divider()
                Picker("排序", selection: Bindable(model.promptBrowser).projectedValue.ascending) {
                    Text("名称升序").tag(true)
                    Text("名称降序").tag(false)
                }
            }
            .menuStyle(.borderlessButton).menuIndicator(.hidden).fixedSize()
            .libraryToolbarIcon()
        }
        .labelStyle(.iconOnly).buttonStyle(.borderless)
        .padding(.horizontal, PromptDesign.listInset)
        .frame(height: SKMDesign.libraryToolbarHeight)
        .overlay(alignment: .bottom) { Divider() }
    }

    @ViewBuilder private func rowMenu(_ prompt: PromptSummary) -> some View {
        Button("复制正文", systemImage: "doc.on.doc") {
            Task {
                do {
                    let details = try await model.promptDetails(prompt.id)
                    PromptActions.copy(details.body)
                    model.announce(AppLocalization.string("Prompt 已复制"))
                } catch { model.errorMessage = error.localizedDescription }
            }
        }
        Button("编辑", systemImage: "square.and.pencil") {
            Task {
                do { editorDetails = try await model.promptDetails(prompt.id) }
                catch { model.errorMessage = error.localizedDescription }
            }
        }
        Button("导出 Markdown…", systemImage: "square.and.arrow.up") {
            Task {
                do { try PromptActions.export(try await model.promptDetails(prompt.id)) }
                catch { model.errorMessage = error.localizedDescription }
            }
        }
        Divider()
        Button("删除", systemImage: "trash", role: .destructive) {
            deletion = prompt
            confirmsDelete = true
        }
    }

    private func importPrompt() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK, let url = panel.url {
            do {
                let content = try String(contentsOf: url, encoding: .utf8)
                Task { await model.importPrompt(content: content) }
            } catch { model.errorMessage = error.localizedDescription }
        }
    }
}

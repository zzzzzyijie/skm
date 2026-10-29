import SwiftUI

/// New tags stay in the draft until the Prompt is saved.
struct PromptTagPicker: View {
    let model: AppModel
    @Binding var tags: [String]
    @State private var showsPicker = false
    @State private var query = ""

    private var available: [String] {
        mergedTagPool(customTags: model.promptCustomTags, tagGroups: model.prompts.map(\.tags), selectedTags: tags)
    }
    private var filteredTags: [String] { available.filter { trimmed.isEmpty || $0.localizedStandardContains(trimmed) } }
    private var trimmed: String { query.trimmingCharacters(in: .whitespacesAndNewlines) }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("标签").font(.callout).foregroundStyle(.secondary)
            TagWrapLayout(spacing: 6) {
                ForEach(tags, id: \.self) { tag in
                    Button { tags.removeAll { tagNamesEqual($0, tag) } } label: {
                        HStack(spacing: 5) {
                            Text(tag).lineLimit(1)
                            Image(systemName: "xmark").imageScale(.small)
                        }
                        .font(.callout).padding(.horizontal, 8).padding(.vertical, 5)
                        .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: 6))
                    }
                    .buttonStyle(.plain).help(tag)
                    .accessibilityLabel(Text("移除标签 \(tag)"))
                    .accessibilityIdentifier("prompt-remove-tag-\(tag)")
                }
                Button("添加标签", systemImage: "plus") { showsPicker = true }
                    .buttonStyle(.borderless)
                    .accessibilityIdentifier("prompt-add-tag-button")
                    .popover(isPresented: $showsPicker) {
                        VStack(alignment: .leading, spacing: 12) {
                            TextField("搜索或新建标签", text: $query).textFieldStyle(.roundedBorder)
                                .accessibilityIdentifier("prompt-tag-search")
                            ScrollView {
                                LazyVStack(alignment: .leading, spacing: 4) {
                                    ForEach(filteredTags, id: \.self) { tag in
                                        Button { toggle(tag) } label: {
                                            HStack {
                                                Text(tag).lineLimit(1)
                                                Spacer()
                                                if tags.contains(where: { tagNamesEqual($0, tag) }) { Image(systemName: "checkmark") }
                                            }
                                            .frame(maxWidth: .infinity, minHeight: 28).contentShape(Rectangle())
                                        }.buttonStyle(.plain)
                                    }
                                    if !trimmed.isEmpty && !available.contains(where: { tagNamesEqual($0, trimmed) }) {
                                        Button("创建标签“\(trimmed)”", systemImage: "plus") {
                                            tags.append(trimmed)
                                            query = ""
                                        }
                                        .buttonStyle(.borderless)
                                        .disabled(!PromptDraft.validTagName(trimmed))
                                        .accessibilityIdentifier("prompt-create-tag-button")
                                    }
                                }
                            }
                            // Keep the search field in place while the result count changes.
                            .frame(height: 180)
                            if !trimmed.isEmpty && !PromptDraft.validTagName(trimmed) {
                                Label("标签限 1–32 个字母、数字或内部连字符。", systemImage: "exclamationmark.circle")
                                    .font(.caption).foregroundStyle(.red)
                            }
                            HStack { Spacer(); Button("完成") { showsPicker = false } }
                        }.padding(16).frame(width: 260)
                    }
            }
        }
    }

    private func toggle(_ tag: String) {
        if tags.contains(where: { tagNamesEqual($0, tag) }) { tags.removeAll { tagNamesEqual($0, tag) } }
        else { tags.append(tag) }
    }
}

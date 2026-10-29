import SwiftUI

struct PromptFilterSidebar: View {
    @Bindable var model: AppModel
    @State private var tagSearch = ""

    private var tags: [String] {
        mergedTagPool(customTags: model.promptCustomTags, tagGroups: model.prompts.map(\.tags), selectedTags: [])
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            filterRow(.all, symbol: "square.stack", identifier: "prompts-filter-all")
            filterRow(.untagged, symbol: "tray", identifier: "prompts-filter-untagged")
            if tags.count > 8 {
                TextField("查找标签", text: $tagSearch)
                    .textFieldStyle(.roundedBorder).padding(.horizontal, 14)
            }
            ForEach(tags.filter { tagSearch.isEmpty || $0.localizedStandardContains(tagSearch) }, id: \.self) { tag in
                filterRow(.tag(tag), symbol: "tag", identifier: "prompts-filter-tag-\(tag)")
            }
        }
        .onChange(of: tags) { _, _ in model.reconcilePromptSelection() }
    }

    private func filterRow(_ filter: PromptFilter, symbol: String, identifier: String) -> some View {
        let isSelected = model.section == .prompts && model.promptBrowser.filter == filter
        return Button {
            model.section = .prompts
            model.promptBrowser.filter = filter
            model.reconcilePromptSelection()
        } label: {
            HStack(spacing: 9) {
                Image(systemName: symbol).frame(width: 20).accessibilityHidden(true)
                Text(filter.title).lineLimit(1).truncationMode(.tail)
                Spacer(minLength: 4)
                Text(model.prompts.filter { filter.includes($0) }.count, format: .number)
                    .font(.caption.monospacedDigit()).foregroundStyle(.secondary)
            }
            .font(.callout)
            .padding(.horizontal, 14).frame(minHeight: 32)
            .frame(maxWidth: .infinity, alignment: .leading)
            .contentShape(Rectangle())
        }
        .background(isSelected ? Color.accentColor.opacity(0.12) : .clear, in: RoundedRectangle(cornerRadius: 6))
        .fontWeight(isSelected ? .semibold : .regular)
        .buttonStyle(SelectionCardButtonStyle(isStatic: true))
        .help(filter.title)
        .accessibilityAddTraits(isSelected ? .isSelected : [])
        .accessibilityIdentifier(identifier)
    }
}

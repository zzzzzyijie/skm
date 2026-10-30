import SwiftUI

struct SkillFilterSidebar: View {
    @Bindable var model: AppModel
    @State private var tagSearch = ""

    private var tags: [String] {
        mergedTagPool(customTags: model.skillCustomTags, tagGroups: model.skills.map(\.tags), selectedTags: [])
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            filterRow(.all, symbol: "square.stack.3d.up", identifier: "skills-filter-all")
            filterRow(.untagged, symbol: "tray", identifier: "skills-filter-untagged")
            if tags.count > 8 {
                TextField("查找标签", text: $tagSearch)
                    .sidebarSearchStyle()
            }
            ForEach(tags.filter { tagSearch.isEmpty || $0.localizedStandardContains(tagSearch) }, id: \.self) { tag in
                filterRow(.tag(tag), symbol: "tag", identifier: "skills-filter-tag-\(tag)")
            }
        }
        .onChange(of: tags) { _, _ in model.reconcileSkillSelection() }
    }

    private func filterRow(_ filter: SkillFilter, symbol: String, identifier: String) -> some View {
        let isSelected = model.section == .skills && model.skillBrowser.filter == filter
        return Button {
            model.section = .skills
            model.skillBrowser.filter = filter
            model.reconcileSkillSelection()
        } label: {
            HStack(spacing: 9) {
                Image(systemName: symbol).frame(width: SKMDesign.sidebarIconWidth).foregroundStyle(.secondary).accessibilityHidden(true)
                Text(filter.title).lineLimit(1).truncationMode(.tail)
                Spacer(minLength: 4)
                Text(model.skills.filter { filter.includes($0) }.count, format: .number)
                    .font(.system(size: 12).monospacedDigit()).foregroundStyle(.secondary)
                    .frame(minWidth: SKMDesign.sidebarCountWidth, alignment: .trailing)
            }
            .font(.callout)
            .padding(.horizontal, SKMDesign.sidebarRowHorizontalPadding)
            .frame(minHeight: SKMDesign.sidebarChildRowHeight)
            .frame(maxWidth: .infinity, alignment: .leading)
            .contentShape(Rectangle())
        }
        .background(isSelected ? SKMDesign.libraryRowSelection : .clear, in: RoundedRectangle(cornerRadius: 6))
        .fontWeight(isSelected ? .semibold : .regular)
        .buttonStyle(SelectionCardButtonStyle(isStatic: true))
        .help(filter.title)
        .accessibilityAddTraits(isSelected ? .isSelected : [])
        .accessibilityIdentifier(identifier)
    }
}

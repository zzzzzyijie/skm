import SwiftUI

/// Skills / Prompts 列表中的可折叠标签分组标题。
///
/// 标题与条目必须作为 `List` 的独立行渲染，避免 `DisclosureGroup` 把所有条目
/// 压进同一个列表行后产生高度计算错误、选中态错位或文本重叠。
struct TagGroupHeader: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let title: String
    let systemImage: String
    let count: Int
    let isExpanded: Bool
    let action: () -> Void

    init(
        title: String,
        systemImage: String = "tag",
        count: Int,
        isExpanded: Bool,
        action: @escaping () -> Void
    ) {
        self.title = title
        self.systemImage = systemImage
        self.count = count
        self.isExpanded = isExpanded
        self.action = action
    }

    var body: some View {
        Button(action: action) {
            HStack(spacing: 6) {
                Image(systemName: "chevron.right")
                    .font(.caption.bold())
                    .foregroundStyle(.secondary)
                    .frame(width: 12)
                    .rotationEffect(.degrees(isExpanded ? 90 : 0))
                    .accessibilityHidden(true)

                Text(title)
                    .font(.system(size: 13))
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .layoutPriority(1)

                Spacer(minLength: 8)

                Text(count, format: .number)
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 10)
            .frame(height: 28)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .listRowInsets(EdgeInsets())
        .listRowSeparator(.hidden)
        .accessibilityLabel(title)
        .accessibilityValue(isExpanded ? Text("已展开") : Text("已收起"))
        .animation(reduceMotion ? nil : .easeOut(duration: 0.15), value: isExpanded)
    }
}

/// 按本地化标签顺序生成分组；多标签条目会出现在每个对应分组中。
func itemsGroupedByTag<Item>(
    _ items: [Item],
    tags: (Item) -> [String]
) -> [(tag: String, items: [Item])] {
    availableFilterTags(from: items.map(tags)).map { tag in
        (tag, items.filter { tags($0).contains(where: { tagNamesEqual($0, tag) }) })
    }
}

/// 从所有项目的标签数组集合中提取去重并按本地化规则排序的可用标签列表。
func availableFilterTags(from tagGroups: [[String]]) -> [String] {
    uniqueTagNamesPreservingCase(tagGroups.flatMap { $0 }).sorted {
        $0.localizedStandardCompare($1) == .orderedAscending
    }
}

/// 集中标签管理的目标实体类型
enum TagManagementTarget {
    case skills
    case prompts

    var scope: TagScope {
        switch self {
        case .skills: return .skills
        case .prompts: return .prompts
        }
    }

    var title: String {
        switch self {
        case .skills: return AppLocalization.string("Skill 标签管理")
        case .prompts: return AppLocalization.string("Prompt 标签管理")
        }
    }

    var itemNoun: String {
        switch self {
        case .skills: return AppLocalization.string("个 Skill")
        case .prompts: return AppLocalization.string("个 Prompt")
        }
    }

    var symbol: String {
        switch self {
        case .skills: return "tag.fill"
        case .prompts: return "text.bubble.fill"
        }
    }

    var tint: Color {
        switch self {
        case .skills: return SKMDesign.tagManagerSkillTint
        case .prompts: return SKMDesign.tagManagerPromptTint
        }
    }
}

/// 集中式标签管理面板：支持全局标签统计浏览、重命名/合并与批量解绑移除
struct TagManagementSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @Bindable var model: AppModel
    let target: TagManagementTarget

    @State private var search = ""
    @State private var editingTag: String?
    @State private var newTagName = ""
    @State private var tagToDelete: String?
    @State private var showsRenameSheet = false
    @State private var showsDeleteConfirmation = false
    @State private var showsAddTagSheet = false
    @State private var newCreatedTag = ""
    @State private var hoveredTag: String?
    @State private var isAddButtonHovered = false
    @State private var isDoneButtonHovered = false
    @FocusState private var renameFieldFocused: Bool
    @FocusState private var addFieldFocused: Bool

    private var allTagCounts: [(tag: String, count: Int)] {
        let activeTags: [String]
        switch target {
        case .skills:
            activeTags = availableFilterTags(from: model.skills.map(\.tags))
        case .prompts:
            activeTags = availableFilterTags(from: model.prompts.map(\.tags))
        }
        let allUnique = uniqueTagNamesPreservingCase(activeTags + model.customTags(for: target.scope).sorted()).sorted {
            $0.localizedStandardCompare($1) == .orderedAscending
        }
        return allUnique.map { tag in
            let count: Int
            switch target {
            case .skills:
                count = model.skills.filter { $0.tags.contains(where: { tagNamesEqual($0, tag) }) }.count
            case .prompts:
                count = model.prompts.filter { $0.tags.contains(where: { tagNamesEqual($0, tag) }) }.count
            }
            return (tag, count)
        }
    }

    private var filteredTagCounts: [(tag: String, count: Int)] {
        if search.isEmpty { return allTagCounts }
        return allTagCounts.filter { $0.tag.localizedStandardContains(search) }
    }

    private var totalItems: Int {
        switch target {
        case .skills: return model.skills.count
        case .prompts: return model.prompts.count
        }
    }

    private var summary: String {
        let tagCount = String(format: AppLocalization.string("共 %lld 个标签"), allTagCounts.count)
        let itemCount = String(
            format: AppLocalization.string("%lld %@"),
            totalItems,
            target.itemNoun
        )
        return "\(tagCount) · \(itemCount)"
    }

    private var subtleTintOpacity: Double {
        colorScheme == .dark ? 0.18 : 0.10
    }

    private var hoverTintOpacity: Double {
        colorScheme == .dark ? 0.18 : 0.08
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            content
            footer
        }
        .background(SKMDesign.detailCanvas)
        .sheetChrome(model: model)
        .frame(width: SKMDesign.tagManagerWidth, height: SKMDesign.tagManagerHeight)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16)
                .strokeBorder(SKMDesign.libraryBorder, lineWidth: 1)
                .allowsHitTesting(false)
        }
        .shadow(color: .black.opacity(colorScheme == .dark ? 0.42 : 0.14), radius: 24, y: 10)
        .onExitCommand { if search.isEmpty { dismiss() } else { search = "" } }
        .confirmationDialog(
            String(format: AppLocalization.string("确定要移除标签“%@”吗？"), tagToDelete ?? ""),
            isPresented: $showsDeleteConfirmation,
            presenting: tagToDelete
        ) {
            tag in
            Button("移除标签", role: .destructive) { removeTag(tag) }
        } message: {
            _ in Text("标签将从所有关联条目中移除，但不会删除条目本身。")
        }
        .sheet(isPresented: $showsRenameSheet, onDismiss: { editingTag = nil }) {
            renameSheet
        }
        .sheet(isPresented: $showsAddTagSheet) {
            addTagSheet
        }
    }

    private var header: some View {
        HStack(spacing: 12) {
            Image(systemName: target.symbol)
                .font(.system(size: 16, weight: .semibold))
                .symbolRenderingMode(.hierarchical)
                .foregroundStyle(colorScheme == .dark ? Color.black.opacity(0.85) : Color.white)
                .frame(width: 36, height: 36)
                .background(target.tint.gradient, in: RoundedRectangle(cornerRadius: 9))
                .overlay {
                    RoundedRectangle(cornerRadius: 9)
                        .strokeBorder(.white.opacity(0.18), lineWidth: 1)
                }
                .shadow(color: target.tint.opacity(0.18), radius: 4, y: 2)
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 2) {
                Text(target.title)
                    .font(.headline)
                Text(String(format: AppLocalization.string("共 %lld 个标签"), allTagCounts.count))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer(minLength: 16)

            Button("添加标签", systemImage: "plus", action: beginAddingTag)
                .buttonStyle(.plain)
                .font(.callout)
                .foregroundStyle(target.tint)
                .padding(.horizontal, 12)
                .frame(height: 30)
                .background(
                    target.tint.opacity(isAddButtonHovered ? subtleTintOpacity + 0.05 : subtleTintOpacity),
                    in: RoundedRectangle(cornerRadius: 8)
                )
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(target.tint.opacity(colorScheme == .dark ? 0.26 : 0.16), lineWidth: 1)
                        .allowsHitTesting(false)
                }
                .contentShape(RoundedRectangle(cornerRadius: 8))
                .onHover { isAddButtonHovered = $0 }
        }
        .padding(.horizontal, 24)
        .padding(.top, 22)
        .padding(.bottom, 14)
    }

    private var content: some View {
        VStack(spacing: 0) {
            CollectionSearchField(title: "搜索标签", text: $search, tint: target.tint, controlHeight: 32)
                .padding(.horizontal, 8)
                .padding(.bottom, 4)

            if filteredTagCounts.isEmpty {
                ContentUnavailableView {
                    Label(search.isEmpty ? "暂无标签" : "无匹配标签", systemImage: target.symbol)
                } description: {
                    Text(search.isEmpty ? "在编辑详情中为条目添加标签后，将在此集中展示。" : "尝试其他关键词。")
                }
                .frame(maxHeight: .infinity)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(filteredTagCounts, id: \.tag) { item in
                            HStack(spacing: 12) {
                                Image(systemName: "tag.fill")
                                    .font(.callout)
                                    .foregroundStyle(target.tint)
                                    .frame(width: 28, height: 28)
                                    .background(target.tint.opacity(subtleTintOpacity), in: RoundedRectangle(cornerRadius: 7))
                                    .accessibilityHidden(true)

                                Text(item.tag)
                                    .font(.callout)
                                    .lineLimit(1)
                                    .help(item.tag)

                                Spacer(minLength: 12)

                                Text(String(format: AppLocalization.string("%lld %@"), item.count, target.itemNoun))
                                    .font(.caption.monospacedDigit())
                                    .foregroundStyle(.secondary)

                                Button("重命名", systemImage: "square.and.pencil") { beginRenaming(item.tag) }
                                    .labelStyle(.iconOnly)
                                    .buttonStyle(.borderless)
                                    .foregroundStyle(hoveredTag == item.tag ? target.tint : Color.secondary)
                                    .frame(width: 28, height: 28)
                                    .background(
                                        hoveredTag == item.tag ? target.tint.opacity(subtleTintOpacity) : Color.clear,
                                        in: RoundedRectangle(cornerRadius: 7)
                                    )
                                    .help("重命名")

                                Button("移除标签", systemImage: "trash", role: .destructive) {
                                    requestTagRemoval(item.tag)
                                }
                                .labelStyle(.iconOnly)
                                .buttonStyle(.borderless)
                                .foregroundStyle(hoveredTag == item.tag ? Color.red : Color.secondary)
                                .frame(width: 28, height: 28)
                                .background(
                                    hoveredTag == item.tag ? Color.red.opacity(subtleTintOpacity) : Color.clear,
                                    in: RoundedRectangle(cornerRadius: 7)
                                )
                                .accessibilityLabel(AppLocalization.string("移除标签") + " " + item.tag)
                                .help("移除标签")
                            }
                            .padding(.horizontal, 24)
                            .frame(height: SKMDesign.tagManagerRowHeight)
                            .contentShape(Rectangle())
                            .background(hoveredTag == item.tag ? target.tint.opacity(hoverTintOpacity) : Color.clear)
                            .overlay(alignment: .leading) {
                                if hoveredTag == item.tag {
                                    Capsule()
                                        .fill(target.tint)
                                        .frame(width: 3, height: 24)
                                        .padding(.leading, 8)
                                }
                            }
                            .overlay(alignment: .bottom) {
                                Divider().padding(.leading, 24)
                            }
                            .onHover { isHovering in
                                if isHovering {
                                    hoveredTag = item.tag
                                } else if hoveredTag == item.tag {
                                    hoveredTag = nil
                                }
                            }
                        }
                    }
                }
            }
        }
        .background(SKMDesign.detailCanvas)
        .frame(maxHeight: .infinity)
    }

    private var footer: some View {
        HStack(spacing: 16) {
            Text(summary)
                .font(.caption)
                .foregroundStyle(.secondary)
                .monospacedDigit()

            Spacer(minLength: 16)

            Button("完成", action: dismiss.callAsFunction)
                .buttonStyle(.plain)
                .font(.callout.bold())
                .foregroundStyle(colorScheme == .dark ? Color.black.opacity(0.85) : Color.white)
                .padding(.horizontal, 16)
                .frame(height: 32)
                .background(target.tint.gradient, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(.white.opacity(colorScheme == .dark ? 0.16 : 0.24), lineWidth: 1)
                        .allowsHitTesting(false)
                }
                .shadow(color: target.tint.opacity(isDoneButtonHovered ? 0.30 : 0.18), radius: 5, y: 2)
                .opacity(isDoneButtonHovered ? 0.94 : 1)
                .contentShape(RoundedRectangle(cornerRadius: 8))
                .onHover { isDoneButtonHovered = $0 }
                .keyboardShortcut(.defaultAction)
        }
        .padding(.horizontal, 24)
        .padding(.vertical, 14)
        .background(SKMDesign.detailCanvas)
        .overlay(alignment: .top) { Divider() }
    }

    private func beginAddingTag() {
        newCreatedTag = ""
        showsAddTagSheet = true
    }

    private func beginRenaming(_ tag: String) {
        editingTag = tag
        newTagName = tag
        showsRenameSheet = true
    }

    private func dismissRename() {
        showsRenameSheet = false
    }

    private func commitRename() {
        guard let oldTag = editingTag else { return }
        let targetName = newTagName.trimmingCharacters(in: .whitespacesAndNewlines)
        showsRenameSheet = false
        Task {
            if target == .skills {
                await model.renameSkillTag(from: oldTag, to: targetName)
            } else {
                await model.renamePromptTag(from: oldTag, to: targetName)
            }
        }
    }

    private func requestTagRemoval(_ tag: String) {
        tagToDelete = tag
        showsDeleteConfirmation = true
    }

    private func removeTag(_ tag: String) {
        Task {
            if target == .skills {
                await model.removeSkillTag(tag)
            } else {
                await model.removePromptTag(tag)
            }
        }
    }

    private func addTag(_ tag: String) {
        model.registerCustomTag(tag, for: target.scope)
        showsAddTagSheet = false
    }

    private var renameSheet: some View {
        VStack(spacing: 0) {
            SheetHeader(
                title: AppLocalization.string("重命名 / 合并标签"),
                subtitle: String(format: AppLocalization.string("将原标签“%@”更新为新名称："), editingTag ?? ""),
                symbol: "tag",
                tint: target.tint
            )

            VStack(alignment: .leading, spacing: 12) {
                SheetSection("标签名称", symbol: "text.cursor") {
                    HStack(spacing: 10) {
                        Image(systemName: "tag").foregroundStyle(target.tint)
                        TextField("新标签名称", text: $newTagName)
                            .focused($renameFieldFocused)
                            .textFieldStyle(.plain)
                    }
                    .padding(11)
                    .background(SKMDesign.detailCanvas, in: RoundedRectangle(cornerRadius: 8))
                    .overlay {
                        RoundedRectangle(cornerRadius: 8)
                            .strokeBorder(renameFieldFocused ? Color.accentColor.opacity(0.65) : SKMDesign.libraryBorder, lineWidth: 1)
                            .allowsHitTesting(false)
                    }
                }

                if allTagCounts.contains(where: {
                    tagNamesEqual($0.tag, newTagName) && $0.tag != editingTag
                }) {
                    Label("新标签已存在，保存后将自动合并。", systemImage: "arrow.triangle.merge")
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
            }
            .padding(SKMDesign.sheetVerticalPadding)
            .frame(maxHeight: .infinity, alignment: .top)

            SheetActionBar {
                Button("取消", role: .cancel, action: dismissRename)
                    .keyboardShortcut(.cancelAction)
                Button("确认更新", action: commitRename)
                    .buttonStyle(.borderedProminent)
                    .keyboardShortcut(.defaultAction)
                    .disabled(newTagName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || newTagName == editingTag)
            }
        }
        .frame(width: 440, height: 280)
        .sheetChrome(model: model)
        .onAppear { renameFieldFocused = true }
    }

    private var addTagSheet: some View {
        let trimmed = newCreatedTag.trimmingCharacters(in: .whitespacesAndNewlines)
        let isDuplicate = allTagCounts.contains(where: { tagNamesEqual($0.tag, trimmed) })

        return VStack(spacing: 0) {
            SheetHeader(
                title: AppLocalization.string("添加新标签"),
                subtitle: AppLocalization.string("添加后可在录入或编辑条目时直接选择。"),
                symbol: "tag.fill",
                tint: target.tint
            )

            VStack(alignment: .leading, spacing: 12) {
                SheetSection("标签名称", symbol: "text.cursor") {
                    HStack(spacing: 10) {
                        Image(systemName: "tag").foregroundStyle(target.tint)
                        TextField("标签名称", text: $newCreatedTag)
                            .focused($addFieldFocused)
                            .textFieldStyle(.plain)
                    }
                    .padding(11)
                    .background(SKMDesign.detailCanvas, in: RoundedRectangle(cornerRadius: 8))
                    .overlay {
                        RoundedRectangle(cornerRadius: 8)
                            .strokeBorder(addFieldFocused ? Color.accentColor.opacity(0.65) : SKMDesign.libraryBorder, lineWidth: 1)
                            .allowsHitTesting(false)
                    }
                }

                if isDuplicate {
                    Label("该标签已存在，无需重复添加。", systemImage: "exclamationmark.circle.fill")
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
            }
            .padding(SKMDesign.sheetVerticalPadding)
            .frame(maxHeight: .infinity, alignment: .top)

            SheetActionBar {
                Button("取消", role: .cancel) { showsAddTagSheet = false }
                    .keyboardShortcut(.cancelAction)
                Button("确认添加") { addTag(trimmed) }
                    .buttonStyle(.borderedProminent)
                    .keyboardShortcut(.defaultAction)
                    .disabled(trimmed.isEmpty || isDuplicate)
            }
        }
        .frame(width: 440, height: 280)
        .sheetChrome(model: model)
        .onAppear { addFieldFocused = true }
    }
}

/// Skill 或 Prompt 专属的标签选择器：可多选当前业务标签池中的已有标签，也可即时创建并选中新标签。
struct TagSelector: View {
    let model: AppModel
    let scope: TagScope
    @Binding var selectedTags: [String]
    let accessibilityIdentifier: String

    @State private var newTagName = ""

    private var availableTags: [String] {
        mergedTagPool(
            customTags: model.customTags(for: scope),
            tagGroups: scope == .skills ? model.skills.map(\.tags) : model.prompts.map(\.tags),
            selectedTags: selectedTags
        )
    }

    private var trimmedNewTag: String {
        newTagName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var isDuplicate: Bool {
        availableTags.contains(where: { tagNamesEqual($0, trimmedNewTag) })
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 7) {
                Text("标签")
                    .font(.system(size: 13, weight: .semibold))
                Spacer()
                if !selectedTags.isEmpty {
                    Text(selectedTags.count, format: .number)
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 7)
                        .padding(.vertical, 2)
                        .background(Color.secondary.opacity(0.1), in: Capsule())
                }
            }

            if availableTags.isEmpty {
                Text("暂无可用标签，可在下方创建。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ScrollView {
                    TagWrapLayout(spacing: 6) {
                        ForEach(availableTags, id: \.self) { tag in
                            let isSelected = selectedTags.contains(where: { tagNamesEqual($0, tag) })
                            Button {
                                toggle(tag)
                            } label: {
                                HStack(spacing: 5) {
                                    Image(systemName: isSelected ? "checkmark" : "plus")
                                        .font(.system(size: 9, weight: .semibold))
                                        .accessibilityHidden(true)
                                    Text(tag)
                                        .lineLimit(1)
                                        .truncationMode(.middle)
                                }
                                .font(.caption)
                                .foregroundStyle(isSelected ? Color.white : Color.secondary)
                                .padding(.horizontal, 9)
                                .padding(.vertical, 5)
                                .background(
                                    isSelected ? Color.accentColor : Color.primary.opacity(0.06),
                                    in: Capsule()
                                )
                                .overlay {
                                    Capsule()
                                        .strokeBorder(
                                            isSelected ? Color.clear : Color.primary.opacity(0.06),
                                            lineWidth: 0.5
                                        )
                                }
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel(tag)
                            .accessibilityValue(isSelected ? Text("已选择") : Text("未选择"))
                            .help(tag)
                            .accessibilityIdentifier("\(accessibilityIdentifier)-option-\(tag)")
                        }
                    }
                    .padding(1)
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .frame(height: 82)
            }

            HStack(spacing: 8) {
                TextField("输入新标签", text: $newTagName)
                    .textFieldStyle(.roundedBorder)
                    .onSubmit(addNewTag)
                    .accessibilityIdentifier("\(accessibilityIdentifier)-new-field")
                Button("添加并选中", systemImage: "plus", action: addNewTag)
                    .labelStyle(.iconOnly)
                    .help("添加并选中")
                    .disabled(trimmedNewTag.isEmpty || isDuplicate)
                    .accessibilityIdentifier("\(accessibilityIdentifier)-add-button")
            }

            if !trimmedNewTag.isEmpty && isDuplicate {
                Text("该标签已存在，可直接在上方选择。")
                    .font(.caption)
                    .foregroundStyle(.orange)
            }
        }
    }

    private func toggle(_ tag: String) {
        if let index = selectedTags.firstIndex(where: { tagNamesEqual($0, tag) }) {
            selectedTags.remove(at: index)
        } else {
            selectedTags.append(tag)
        }
    }

    private func addNewTag() {
        guard !trimmedNewTag.isEmpty, !isDuplicate else { return }
        model.registerCustomTag(trimmedNewTag, for: scope)
        selectedTags.append(trimmedNewTag)
        newTagName = ""
    }
}

/// Tags use their natural width and wrap like Finder's tag tokens.
struct TagWrapLayout: Layout {
    var spacing: CGFloat = 6

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        arrangement(width: proposal.width ?? 300, subviews: subviews).size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let layout = arrangement(width: bounds.width, subviews: subviews)
        for (index, subview) in subviews.enumerated() {
            subview.place(
                at: CGPoint(x: bounds.minX + layout.points[index].x, y: bounds.minY + layout.points[index].y),
                anchor: .topLeading,
                proposal: ProposedViewSize(width: min(subview.sizeThatFits(.unspecified).width, bounds.width), height: nil)
            )
        }
    }

    private func arrangement(width: CGFloat, subviews: Subviews) -> (size: CGSize, points: [CGPoint]) {
        let width = max(width, 1)
        var points: [CGPoint] = []
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let ideal = subview.sizeThatFits(.unspecified)
            let size = subview.sizeThatFits(ProposedViewSize(width: min(ideal.width, width), height: nil))
            if x > 0 && x + size.width > width {
                x = 0
                y += rowHeight + spacing
                rowHeight = 0
            }
            points.append(CGPoint(x: x, y: y))
            x += min(size.width, width) + spacing
            rowHeight = max(rowHeight, size.height)
        }
        return (CGSize(width: width, height: y + rowHeight), points)
    }
}

/// 合并自定义标签、已有条目标签和当前选择，去重后按本地化顺序排序。
func mergedTagPool(customTags: Set<String>, tagGroups: [[String]], selectedTags: [String]) -> [String] {
    uniqueTagNamesPreservingCase(tagGroups.flatMap { $0 } + selectedTags + customTags.sorted())
        .sorted { $0.localizedStandardCompare($1) == .orderedAscending }
}

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
            HStack(spacing: 9) {
                Image(systemName: "chevron.right")
                    .font(.caption.bold())
                    .foregroundStyle(.secondary)
                    .frame(width: 12)
                    .rotationEffect(.degrees(isExpanded ? 90 : 0))
                    .accessibilityHidden(true)

                Image(systemName: systemImage)
                    .symbolRenderingMode(.hierarchical)
                    .foregroundStyle(.secondary)
                    .frame(width: 16)
                    .accessibilityHidden(true)

                Text(title)
                    .font(.headline)
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .layoutPriority(1)

                Spacer(minLength: 8)

                Text(count, format: .number)
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(Color.secondary.opacity(0.11), in: Capsule())
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            .contentShape(Rectangle())
            .background(Color.secondary.opacity(0.07), in: RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
        .buttonStyle(.plain)
        .listRowInsets(EdgeInsets(top: 6, leading: 8, bottom: 3, trailing: 8))
        .listRowSeparator(.hidden)
        .accessibilityLabel(title)
        .accessibilityValue(isExpanded ? Text("已展开") : Text("已收起"))
        .animation(reduceMotion ? nil : .easeOut(duration: 0.16), value: isExpanded)
    }
}

/// 按本地化标签顺序生成分组；多标签条目会出现在每个对应分组中。
func itemsGroupedByTag<Item>(
    _ items: [Item],
    tags: (Item) -> [String]
) -> [(tag: String, items: [Item])] {
    availableFilterTags(from: items.map(tags)).map { tag in
        (tag, items.filter { tags($0).contains(tag) })
    }
}

/// 从所有项目的标签数组集合中提取去重并按本地化规则排序的可用标签列表。
func availableFilterTags(from tagGroups: [[String]]) -> [String] {
    Set(tagGroups.joined()).sorted {
        $0.localizedStandardCompare($1) == .orderedAscending
    }
}

/// 集中标签管理的目标实体类型
enum TagManagementTarget {
    case skills
    case prompts

    var title: String {
        switch self {
        case .skills: return String(localized: "Skill 标签管理")
        case .prompts: return String(localized: "Prompt 标签管理")
        }
    }

    var itemNoun: String {
        switch self {
        case .skills: return String(localized: "个 Skill")
        case .prompts: return String(localized: "个 Prompt")
        }
    }
}

/// 集中式标签管理面板：支持全局标签统计浏览、重命名/合并与批量解绑移除
struct TagManagementSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Bindable var model: AppModel
    let target: TagManagementTarget

    @State private var search = ""
    @State private var editingTag: String?
    @State private var newTagName = ""
    @State private var tagToDelete: String?
    @State private var showsAddTagSheet = false
    @State private var newCreatedTag = ""
    @State private var selectedItemIDs: Set<String> = []

    private var allTagCounts: [(tag: String, count: Int)] {
        let activeTags: [String]
        switch target {
        case .skills:
            activeTags = availableFilterTags(from: model.skills.map(\.tags))
        case .prompts:
            activeTags = availableFilterTags(from: model.prompts.map(\.tags))
        }
        let allUnique = Set(activeTags).union(model.customTags).sorted {
            $0.localizedStandardCompare($1) == .orderedAscending
        }
        return allUnique.map { tag in
            let count: Int
            switch target {
            case .skills:
                count = model.skills.filter { $0.tags.contains(tag) }.count
            case .prompts:
                count = model.prompts.filter { $0.tags.contains(tag) }.count
            }
            return (tag, count)
        }
    }

    private var filteredTagCounts: [(tag: String, count: Int)] {
        if search.isEmpty { return allTagCounts }
        return allTagCounts.filter { $0.tag.localizedStandardContains(search) }
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider()
            content
            Divider()
            footer
        }
        .frame(width: 500, height: 460)
        .confirmationDialog(
            String(format: String(localized: "确定要移除标签“%@”吗？"), tagToDelete ?? ""),
            isPresented: Binding(
                get: { tagToDelete != nil },
                set: { if !$0 { tagToDelete = nil } }
            )
        ) {
            Button("移除标签", role: .destructive) {
                if let tag = tagToDelete {
                    Task {
                        if target == .skills {
                            await model.removeSkillTag(tag)
                        } else {
                            await model.removePromptTag(tag)
                        }
                    }
                }
            }
        } message: {
            Text("标签将从所有关联条目中移除，但不会删除条目本身。")
        }
        .sheet(isPresented: Binding(
            get: { editingTag != nil },
            set: { if !$0 { editingTag = nil } }
        )) {
            renameSheet
        }
        .sheet(isPresented: $showsAddTagSheet) {
            addTagSheet
        }
    }

    private var header: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                Text(target.title).font(.headline)
                Text(String(format: String(localized: "共 %lld 个标签"), allTagCounts.count))
                    .font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            Button("添加标签", systemImage: "plus") {
                newCreatedTag = ""
                selectedItemIDs = []
                showsAddTagSheet = true
            }
            .buttonStyle(.bordered)
            .controlSize(.small)

            Button("完成") { dismiss() }
                .keyboardShortcut(.defaultAction)
        }
        .padding(16)
    }

    private var content: some View {
        VStack(spacing: 8) {
            HStack(spacing: 6) {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
                    .font(.system(size: 11))
                TextField("搜索标签", text: $search)
                    .textFieldStyle(.plain)
                    .font(.system(size: 12))
                if !search.isEmpty {
                    Button { search = "" } label: {
                        Image(systemName: "xmark.circle.fill").font(.system(size: 11)).foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6))
            .overlay(RoundedRectangle(cornerRadius: 6).stroke(Color(nsColor: .separatorColor).opacity(0.6), lineWidth: 0.5))
            .padding(.horizontal, 16)
            .padding(.top, 10)

            if filteredTagCounts.isEmpty {
                ContentUnavailableView {
                    Label(search.isEmpty ? "暂无标签" : "无匹配标签", systemImage: "tag")
                } description: {
                    Text(search.isEmpty ? "在编辑详情中为条目添加标签后，将在此集中展示。" : "尝试其他关键词。")
                }
                .frame(maxHeight: .infinity)
            } else {
                List {
                    ForEach(filteredTagCounts, id: \.tag) { item in
                        HStack(spacing: 10) {
                            Image(systemName: "tag.fill")
                                .foregroundStyle(Color.accentColor)
                                .font(.system(size: 12))

                            Text(item.tag)
                                .font(.system(size: 13, weight: .medium))

                            Spacer()

                            Text(String(format: String(localized: "%lld %@"), item.count, target.itemNoun))
                                .font(.caption.monospacedDigit())
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 7)
                                .padding(.vertical, 2)
                                .background(Color.secondary.opacity(0.1), in: Capsule())

                            Button("重命名") {
                                editingTag = item.tag
                                newTagName = item.tag
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)

                            Button(role: .destructive) {
                                tagToDelete = item.tag
                            } label: {
                                Image(systemName: "trash")
                                    .font(.system(size: 11))
                                    .foregroundStyle(.red)
                            }
                            .buttonStyle(.plain)
                            .padding(.leading, 4)
                        }
                        .padding(.vertical, 3)
                    }
                }
                .listStyle(.inset(alternatesRowBackgrounds: true))
            }
        }
    }

    private var footer: some View {
        HStack {
            Text("重命名若与现有标签相同将自动合并；删除仅解绑标签，不删除条目。")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Spacer()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(Color(nsColor: .windowBackgroundColor).opacity(0.5))
    }

    private var renameSheet: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("重命名 / 合并标签").font(.headline)
            Text(String(format: String(localized: "将原标签“%@”更新为新名称："), editingTag ?? ""))
                .font(.callout).foregroundStyle(.secondary)
            TextField("新标签名称", text: $newTagName)
                .textFieldStyle(.roundedBorder)

            if allTagCounts.contains(where: { $0.tag == newTagName.trimmingCharacters(in: .whitespacesAndNewlines) && $0.tag != editingTag }) {
                Label("新标签已存在，保存后将自动合并。", systemImage: "info.circle")
                    .font(.caption)
                    .foregroundStyle(.orange)
            }

            HStack {
                Spacer()
                Button("取消") { editingTag = nil }
                Button("确认更新") {
                    if let oldTag = editingTag {
                        let targetName = newTagName.trimmingCharacters(in: .whitespacesAndNewlines)
                        editingTag = nil
                        Task {
                            if target == .skills {
                                await model.renameSkillTag(from: oldTag, to: targetName)
                            } else {
                                await model.renamePromptTag(from: oldTag, to: targetName)
                            }
                        }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(newTagName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || newTagName == editingTag)
            }
        }
        .padding(20)
        .frame(width: 360)
    }

    private var addTagSheet: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("添加新标签").font(.headline)
            Text(String(localized: "输入标签名称。添加后将进入全局标签池，并可选择性立即应用到条目："))
                .font(.callout).foregroundStyle(.secondary)

            TextField("标签名称", text: $newCreatedTag)
                .textFieldStyle(.roundedBorder)

            let trimmed = newCreatedTag.trimmingCharacters(in: .whitespacesAndNewlines)
            let isDuplicate = allTagCounts.contains(where: { $0.tag == trimmed })

            if isDuplicate {
                Label("该标签已存在，无需重复添加。", systemImage: "exclamationmark.circle")
                    .font(.caption)
                    .foregroundStyle(.orange)
            }

            let candidateItems: [(id: String, name: String)] = {
                switch target {
                case .skills:
                    return model.skills.map { ($0.id, $0.name) }
                case .prompts:
                    return model.prompts.map { ($0.id, $0.name) }
                }
            }()

            if !candidateItems.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text(String(format: String(localized: "可选：同时应用到以下条目（已选 %lld 项）"), selectedItemIDs.count))
                        .font(.caption.bold())
                        .foregroundStyle(.secondary)

                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 4) {
                            ForEach(candidateItems, id: \.id) { item in
                                Toggle(isOn: Binding(
                                    get: { selectedItemIDs.contains(item.id) },
                                    set: { isChecked in
                                        if isChecked { selectedItemIDs.insert(item.id) }
                                        else { selectedItemIDs.remove(item.id) }
                                    }
                                )) {
                                    Text(item.name).font(.callout).lineLimit(1)
                                }
                                .toggleStyle(.checkbox)
                            }
                        }
                        .padding(6)
                    }
                    .frame(height: 120)
                    .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 6))
                    .overlay(RoundedRectangle(cornerRadius: 6).stroke(Color(nsColor: .separatorColor).opacity(0.6), lineWidth: 0.5))
                }
            }

            HStack {
                Spacer()
                Button("取消") { showsAddTagSheet = false }
                Button("确认添加") {
                    let tag = trimmed
                    showsAddTagSheet = false
                    Task {
                        if target == .skills {
                            await model.addTagToSkills(tag, skillIDs: selectedItemIDs)
                        } else {
                            await model.addTagToPrompts(tag, promptIDs: selectedItemIDs)
                        }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(trimmed.isEmpty || isDuplicate)
            }
        }
        .padding(20)
        .frame(width: 400)
    }
}



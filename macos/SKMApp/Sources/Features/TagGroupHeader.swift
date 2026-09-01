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

import SwiftUI

/// 通用标签过滤栏组件
/// 用于在 Skills 和 Prompts 列表顶部提供基于 Tag 的菜单筛选、一键清除与项数统计展示（如 3/10）
struct TagFilterBar: View {
    /// 当前可用的全部不重复标签列表（已排序）
    let tags: [String]
    /// 筛选后命中的条目数
    let filteredCount: Int
    /// 总条目数
    let totalCount: Int
    /// 当前选中的标签绑定（nil 表示显示全部）
    @Binding var selectedTag: String?

    var body: some View {
        HStack(spacing: 10) {
            Label("标签", systemImage: "tag")
                .font(.callout)
                .foregroundStyle(.secondary)

            Picker("标签筛选", selection: $selectedTag) {
                Text("全部标签").tag(String?.none)
                ForEach(tags, id: \.self) { tag in
                    Text(tag).tag(Optional(tag))
                }
            }
            .labelsHidden()
            .pickerStyle(.menu)
            .fixedSize()

            if selectedTag != nil {
                Button("清除标签筛选", systemImage: "xmark.circle.fill") {
                    selectedTag = nil
                }
                .labelStyle(.iconOnly)
                .buttonStyle(.plain)
                .foregroundStyle(.secondary)
                .help("显示全部标签")
            }

            Spacer(minLength: 8)

            Text("\(filteredCount)/\(totalCount)")
                .font(.caption.monospacedDigit())
                .foregroundStyle(.secondary)
                .accessibilityLabel("显示 \(filteredCount) 项，共 \(totalCount) 项")
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(.ultraThinMaterial)
        .overlay(alignment: .bottom) {
            Divider()
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("标签筛选")
    }
}

/// 从所有项目的标签数组集合中提取去重并按本地化规则排序的可用标签列表
func availableFilterTags(from tagGroups: [[String]]) -> [String] {
    Set(tagGroups.joined()).sorted {
        $0.localizedStandardCompare($1) == .orderedAscending
    }
}

/// 判断指定项目的标签列表是否满足当前选中的标签过滤条件
func matchesSelectedTag(_ tags: [String], selectedTag: String?) -> Bool {
    guard let selectedTag else { return true }
    return tags.contains(selectedTag)
}

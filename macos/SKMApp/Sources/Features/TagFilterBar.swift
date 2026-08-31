import SwiftUI

struct TagFilterBar: View {
    let tags: [String]
    let filteredCount: Int
    let totalCount: Int
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

func availableFilterTags(from tagGroups: [[String]]) -> [String] {
    Set(tagGroups.joined()).sorted {
        $0.localizedStandardCompare($1) == .orderedAscending
    }
}

func matchesSelectedTag(_ tags: [String], selectedTag: String?) -> Bool {
    guard let selectedTag else { return true }
    return tags.contains(selectedTag)
}

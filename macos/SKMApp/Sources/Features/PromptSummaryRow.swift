import SwiftUI

/// Prompts 标签分组中复用的单个提示词摘要行。
struct PromptSummaryRow: View {
    let prompt: PromptSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(prompt.name)
                .font(.body.bold())
                .lineLimit(1)
                .truncationMode(.middle)

            Text(prompt.description.isEmpty ? String(localized: "无描述") : prompt.description)
                .font(.callout)
                .foregroundStyle(.secondary)
                .lineLimit(2)

            if !prompt.tags.isEmpty {
                Label(prompt.tags.joined(separator: " · "), systemImage: "tag")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
                    .truncationMode(.tail)
            }
        }
        .padding(.vertical, 5)
        .frame(maxWidth: .infinity, alignment: .leading)
        .contentShape(Rectangle())
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(accessibilityLabel)
    }

    private var accessibilityLabel: String {
        let tags = prompt.tags.isEmpty
            ? String(localized: "无标签")
            : String(
                format: String(localized: "标签 %@"),
                locale: .current,
                prompt.tags.joined(separator: String(localized: "、"))
            )
        return String(
            format: String(localized: "%1$@，来源 %2$@，%3$@"),
            locale: .current,
            prompt.name,
            prompt.source,
            tags
        )
    }
}

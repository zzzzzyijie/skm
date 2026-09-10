import SwiftUI

/// Prompts 标签分组中复用的单个提示词摘要行。
struct PromptSummaryRow: View {
    let prompt: PromptSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(prompt.name)
                .font(.body.weight(.semibold))
                .lineLimit(1)
                .truncationMode(.middle)

            Text(prompt.description.isEmpty ? AppLocalization.string("无描述") : prompt.description)
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
        .padding(.vertical, 8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .contentShape(Rectangle())
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(accessibilityLabel)
    }

    private var accessibilityLabel: String {
        let tags = prompt.tags.isEmpty
            ? AppLocalization.string("无标签")
            : String(
                format: AppLocalization.string("标签 %@"),
                locale: .current,
                prompt.tags.joined(separator: AppLocalization.string("、"))
            )
        return String(
            format: AppLocalization.string("%1$@，来源 %2$@，%3$@"),
            locale: .current,
            prompt.name,
            prompt.source,
            tags
        )
    }
}

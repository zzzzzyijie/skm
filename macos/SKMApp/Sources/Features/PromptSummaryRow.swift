import SwiftUI

/// Prompts 标签分组中复用的单个提示词摘要行。
struct PromptSummaryRow: View {
    let prompt: PromptSummary

    var body: some View {
        let variableCount = prompt.variables?.count ?? 0

        VStack(alignment: .leading, spacing: 6) {
            Text(prompt.name)
                .font(.body.weight(.semibold))
                .lineLimit(1)
                .truncationMode(.middle)

            Text(prompt.description.isEmpty ? AppLocalization.string("无描述") : prompt.description)
                .font(.callout)
                .foregroundStyle(.secondary)
                .lineLimit(2)

            HStack(spacing: 10) {
                Label(prompt.source, systemImage: "archivebox")
                    .lineLimit(1)

                Spacer(minLength: 4)

                Label {
                    if variableCount == 0 {
                        Text("没有变量")
                    } else {
                        HStack(spacing: 3) {
                            Text(variableCount, format: .number)
                            Text("变量")
                        }
                    }
                } icon: {
                    Image(systemName: "slider.horizontal.3")
                }
                .fixedSize()
            }
            .font(.caption)
            .foregroundStyle(.tertiary)
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

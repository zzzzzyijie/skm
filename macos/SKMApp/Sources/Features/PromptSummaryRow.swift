import SwiftUI

/// Prompts 标签分组中复用的单个提示词摘要行。
struct PromptSummaryRow: View {
    let prompt: PromptSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(prompt.name)
                .font(.system(size: 14, weight: .semibold))
                .lineLimit(1)
                .truncationMode(.middle)

            Text(prompt.description.isEmpty ? AppLocalization.string("无描述") : prompt.description)
                .font(.system(size: 12))
                .foregroundStyle(.secondary)
                .lineLimit(2)

            if !prompt.tags.isEmpty {
                Label(prompt.tags.joined(separator: " · "), systemImage: "tag")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
        }
        .padding(.top, 13)
        .padding(.bottom, 16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .overlay(alignment: .bottom) {
            Rectangle().fill(SKMDesign.libraryBorder.opacity(0.45)).frame(height: 1)
        }
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

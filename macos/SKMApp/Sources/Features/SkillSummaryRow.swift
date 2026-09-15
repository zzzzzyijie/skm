import SwiftUI

/// Skills 标签分组中复用的单个技能摘要行。
struct SkillSummaryRow: View {
    let skill: SkillSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(skill.name)
                    .font(.system(size: 14, weight: .semibold))
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .layoutPriority(1)

                Spacer(minLength: 8)

                HealthBadge(health: skill.health)
                    .fixedSize()
            }

            Text(skill.description.isEmpty ? AppLocalization.string("无描述") : skill.description)
                .font(.system(size: 12))
                .foregroundStyle(.secondary)
                .lineLimit(2)

            if !skill.tags.isEmpty {
                Label(skill.tags.joined(separator: " · "), systemImage: "tag")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.tail)
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
        let source = skill.source.isEmpty ? "local" : skill.source
        let tags = skill.tags.isEmpty
            ? AppLocalization.string("无标签")
            : String(
                format: AppLocalization.string("标签 %@"),
                locale: .current,
                skill.tags.joined(separator: AppLocalization.string("、"))
            )
        return String(
            format: AppLocalization.string("%1$@，来源 %2$@，%3$@，健康状态 %4$@"),
            locale: .current,
            skill.name,
            source,
            tags,
            healthLabel(skill.health)
        )
    }
}

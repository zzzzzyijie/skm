import SwiftUI

/// Skills 标签分组中复用的单个技能摘要行。
struct SkillSummaryRow: View {
    let skill: SkillSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(skill.name)
                    .font(.body.weight(.semibold))
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .layoutPriority(1)

                Spacer(minLength: 8)

                HealthBadge(health: skill.health)
                    .fixedSize()
            }

            Text(skill.description.isEmpty ? AppLocalization.string("无描述") : skill.description)
                .font(.callout)
                .foregroundStyle(.secondary)
                .lineLimit(2)

            if !skill.tags.isEmpty {
                Label(skill.tags.joined(separator: " · "), systemImage: "tag")
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

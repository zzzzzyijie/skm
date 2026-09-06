import SwiftUI

/// Skills 标签分组中复用的单个技能摘要行。
struct SkillSummaryRow: View {
    let skill: SkillSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(skill.name)
                    .font(.system(size: 13, weight: .semibold))
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .layoutPriority(1)

                Spacer(minLength: 8)

                HealthBadge(health: skill.health)
                    .fixedSize()
            }

            Text(skill.description.isEmpty ? String(localized: "无描述") : skill.description)
                .font(.system(size: 12))
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
            ? String(localized: "无标签")
            : String(
                format: String(localized: "标签 %@"),
                locale: .current,
                skill.tags.joined(separator: String(localized: "、"))
            )
        return String(
            format: String(localized: "%1$@，来源 %2$@，%3$@，健康状态 %4$@"),
            locale: .current,
            skill.name,
            source,
            tags,
            healthLabel(skill.health)
        )
    }
}

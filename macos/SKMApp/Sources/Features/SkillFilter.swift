import Foundation

enum SkillFilter: Hashable {
    case all
    case untagged
    case tag(String)

    var title: String {
        switch self {
        case .all: AppLocalization.string("全部 Skill")
        case .untagged: AppLocalization.string("未分类")
        case .tag(let tag): tag
        }
    }

    func includes(_ skill: SkillSummary) -> Bool {
        switch self {
        case .all: true
        case .untagged: skill.tags.isEmpty
        case .tag(let tag): skill.tags.contains { tagNamesEqual($0, tag) }
        }
    }
}

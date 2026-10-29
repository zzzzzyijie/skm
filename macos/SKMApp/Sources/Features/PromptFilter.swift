import Foundation

enum PromptFilter: Hashable {
    case all
    case untagged
    case tag(String)

    var title: String {
        switch self {
        case .all: AppLocalization.string("全部 Prompt")
        case .untagged: AppLocalization.string("未分类")
        case .tag(let tag): tag
        }
    }

    func includes(_ prompt: PromptSummary) -> Bool {
        switch self {
        case .all: true
        case .untagged: prompt.tags.isEmpty
        case .tag(let tag): prompt.tags.contains { tagNamesEqual($0, tag) }
        }
    }
}

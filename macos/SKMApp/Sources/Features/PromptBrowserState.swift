import Foundation
import Observation

/// Presentation-only state. Tags and Prompt contents remain owned by Core.
@MainActor @Observable
final class PromptBrowserState {
    var query = ""
    var filter: PromptFilter = .all
    var ascending = true
    var revealToken = UUID()

    var initialTags: [String] {
        if case .tag(let tag) = filter { return [tag] }
        return []
    }

    func results(in prompts: [PromptSummary]) -> [PromptSummary] {
        let search = query.trimmingCharacters(in: .whitespacesAndNewlines)
        return prompts.filter { prompt in
            filter.includes(prompt) && (search.isEmpty ||
                prompt.name.localizedStandardContains(search) ||
                prompt.description.localizedStandardContains(search) ||
                prompt.tags.contains { $0.localizedStandardContains(search) })
        }.sorted {
            let order = $0.name.localizedStandardCompare($1.name)
            if order == .orderedSame { return $0.id < $1.id }
            return ascending ? order == .orderedAscending : order == .orderedDescending
        }
    }
}

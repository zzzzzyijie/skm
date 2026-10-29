import Foundation
import Observation

/// Presentation state survives sidebar folding and navigation between modules.
@MainActor @Observable
final class SkillBrowserState {
    var query = ""
    var filter: SkillFilter = .all

    func results(in skills: [SkillSummary]) -> [SkillSummary] {
        let search = query.trimmingCharacters(in: .whitespacesAndNewlines)
        return skills.filter { skill in
            filter.includes(skill) && (search.isEmpty ||
                skill.name.localizedStandardContains(search) ||
                skill.description.localizedStandardContains(search) ||
                skill.tags.contains { $0.localizedStandardContains(search) })
        }
    }
}

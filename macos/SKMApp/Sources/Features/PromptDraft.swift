import Foundation

struct PromptDraft: Equatable {
    var name: String
    var description: String
    var body: String
    var tags: [String]
    var variables: [PromptVariable]

    init(details: PromptDetails?, initialTags: [String] = []) {
        name = details?.name ?? ""
        description = details?.description ?? ""
        body = details?.body ?? ""
        tags = details?.tags ?? initialTags
        variables = details?.variables ?? []
    }

    var validName: Bool {
        name.range(of: "^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?\\z", options: .regularExpression) != nil
    }

    var validDescription: Bool {
        let value = description.trimmingCharacters(in: .whitespacesAndNewlines)
        return !value.isEmpty && value.utf8.count <= 1024
    }

    static func validTagName(_ input: String) -> Bool {
        let value = input.trimmingCharacters(in: .whitespacesAndNewlines).precomposedStringWithCompatibilityMapping
        return value.unicodeScalars.count <= 32 &&
            value.range(of: "^[\\p{L}\\p{N}](?:[\\p{L}\\p{N}-]*[\\p{L}\\p{N}])?\\z", options: .regularExpression) != nil
    }

    var isValid: Bool {
        validName && validDescription &&
        !body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func copyName(existing: [PromptSummary]) -> String {
        let base = String(name.prefix(54)).trimmingCharacters(in: CharacterSet(charactersIn: "-"))
        var candidate = "\(base)-copy"
        var suffix = 2
        while existing.contains(where: { $0.name == candidate }) {
            candidate = "\(base)-copy-\(suffix)"
            suffix += 1
        }
        return candidate
    }
}

import Foundation

/// Small, deterministic block parser shared by Skill and Prompt reading surfaces.
/// Unsupported syntax stays visible as text; source mode always preserves the original.
enum MarkdownBlock: Equatable {
    case heading(Int, String)
    case code(String)
    case paragraph(String)
    case listItem(marker: String, text: String, depth: Int)
    case quote(String)

    static func parse(_ markdown: String) -> [Self] {
        let lines = markdown.components(separatedBy: "\n")
        var result: [Self] = []
        var paragraph: [String] = []
        var index = 0
        func flush() {
            let text = paragraph.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
            if !text.isEmpty { result.append(.paragraph(text)) }
            paragraph.removeAll()
        }
        while index < lines.count {
            let line = lines[index]
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("```") || trimmed.hasPrefix("~~~") {
                flush()
                let character = trimmed.first!
                let fence = String(trimmed.prefix(while: { $0 == character }))
                var code: [String] = []
                index += 1
                while index < lines.count {
                    let candidate = lines[index].trimmingCharacters(in: .whitespaces)
                    if candidate.count >= fence.count && candidate.allSatisfy({ $0 == character }) { break }
                    code.append(lines[index])
                    index += 1
                }
                result.append(.code(code.joined(separator: "\n")))
            } else if let match = trimmed.wholeMatch(of: /^(#{1,6})\s+(.+)$/) {
                flush()
                result.append(.heading(match.1.count, String(match.2)))
            } else if let match = line.wholeMatch(of: /^(\s*)([-+*]|\d+[.)])\s+(.+)$/) {
                flush()
                let token = String(match.2)
                let marker = ["-", "+", "*"].contains(token) ? "•" : token
                result.append(.listItem(marker: marker, text: String(match.3), depth: min(match.1.count / 2, 6)))
            } else if trimmed.hasPrefix(">") {
                flush()
                var quoted: [String] = []
                while index < lines.count {
                    let candidate = lines[index].trimmingCharacters(in: .whitespaces)
                    guard candidate.hasPrefix(">") else { break }
                    quoted.append(String(candidate.dropFirst()).trimmingCharacters(in: .whitespaces))
                    index += 1
                }
                result.append(.quote(quoted.joined(separator: "\n")))
                continue
            } else if trimmed.isEmpty {
                flush()
            } else {
                paragraph.append(line)
            }
            index += 1
        }
        flush()
        return result
    }
}

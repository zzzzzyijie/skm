import SwiftUI

struct MarkdownBodyView: View {
    let markdown: String
    @State private var blocks: [MarkdownBlock] = []

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            ForEach(Array(blocks.enumerated()), id: \.offset) { _, block in
                switch block {
                case .heading(let level, let text):
                    Text(inline(text)).font(headingFont(level)).fontWeight(.semibold)
                        .padding(.top, 8).textSelection(.enabled)
                case .code(let code):
                    CodeBlockView(code: code)
                case .paragraph(let text):
                    prose(text).padding(.bottom, 4)
                case .listItem(let marker, let text, let depth):
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Text(marker).monospacedDigit().frame(minWidth: 16, alignment: .trailing)
                        prose(text).frame(maxWidth: .infinity, alignment: .leading)
                    }
                    .padding(.leading, CGFloat(depth) * 16)
                    .accessibilityElement(children: .combine)
                case .quote(let text):
                    prose(text).foregroundStyle(.secondary)
                        .padding(.leading, 14).padding(.vertical, 4)
                        .overlay(alignment: .leading) { Rectangle().fill(SKMDesign.libraryBorder).frame(width: 2) }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .task(id: markdown) { blocks = MarkdownBlock.parse(markdown) }
    }

    private func prose(_ text: String) -> some View {
        Text(inline(text)).font(.system(size: 14)).lineSpacing(5)
            .textSelection(.enabled).fixedSize(horizontal: false, vertical: true)
    }

    private func inline(_ text: String) -> AttributedString {
        (try? AttributedString(markdown: text, options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace))) ?? AttributedString(text)
    }

    private func headingFont(_ level: Int) -> Font {
        switch level {
        case 1: .title2
        case 2: .title3
        default: .headline
        }
    }
}

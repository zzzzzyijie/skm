import AppKit
import SwiftUI

struct CodeBlockView: View {
    let code: String
    @State private var copied = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Image(systemName: "chevron.left.forwardslash.chevron.right")
                    .accessibilityHidden(true)
                Spacer()
                Button(copied ? AppLocalization.string("已复制") : AppLocalization.string("复制"), systemImage: copied ? "checkmark" : "doc.on.doc", action: copy)
                    .buttonStyle(.borderless)
                    .help("复制代码")
            }
            .font(.caption)
            .foregroundStyle(.secondary)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            Divider()
            ScrollView(.horizontal) {
                Text(code)
                    .font(.callout.monospaced())
                    .lineSpacing(4)
                    .textSelection(.enabled)
                    .padding(12)
            }
        }
        .background(Color.primary.opacity(0.035), in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8).strokeBorder(.separator, lineWidth: 0.5)
        }
        .task(id: copied) {
            guard copied else { return }
            do { try await Task.sleep(for: .seconds(2)) } catch { return }
            copied = false
        }
    }

    private func copy() {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(code, forType: .string)
        copied = true
    }
}

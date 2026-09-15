import AppKit
import SwiftUI

struct CodeBlockView: View {
    let code: String
    @State private var copied = false
    @State private var isHovered = false

    var body: some View {
        ScrollView(.horizontal) {
            Text(code)
                .font(.system(size: 13, design: .monospaced))
                .foregroundStyle(.secondary)
                .lineSpacing(4)
                .textSelection(.enabled)
                .padding(.horizontal, 14)
                .padding(.vertical, 12)
        }
        .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: 6))
        .overlay(alignment: .topTrailing) {
            Button(copied ? AppLocalization.string("已复制") : AppLocalization.string("复制"), systemImage: copied ? "checkmark" : "doc.on.doc", action: copy)
                .labelStyle(.iconOnly)
                .buttonStyle(.borderless)
                .help("复制代码")
                .padding(8)
                .opacity(isHovered || copied ? 1 : 0)
        }
        .onHover { isHovered = $0 }
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

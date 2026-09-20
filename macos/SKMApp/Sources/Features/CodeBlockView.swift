import AppKit
import SwiftUI

struct CodeBlockView: View {
    let code: String
    @State private var copied = false

    var body: some View {
        HStack(alignment: .top, spacing: 0) {
            ScrollView(.horizontal) {
                Text(code)
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundStyle(.primary)
                    .lineSpacing(4)
                    .textSelection(.enabled)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 12)
            }
            Button(copied ? AppLocalization.string("已复制") : AppLocalization.string("复制"), systemImage: copied ? "checkmark" : "doc.on.doc", action: copy)
                .labelStyle(.iconOnly)
                .buttonStyle(QuietIconButtonStyle())
                .foregroundStyle(copied ? SKMDesign.projectSelectionText : Color.secondary)
                .help("复制代码")
                .padding(6)
        }
        .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: SKMDesign.controlRadius))
        .overlay {
            RoundedRectangle(cornerRadius: SKMDesign.controlRadius)
                .strokeBorder(SKMDesign.libraryBorder.opacity(0.6), lineWidth: 0.5)
                .allowsHitTesting(false)
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

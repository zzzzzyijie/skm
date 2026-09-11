import SwiftUI

struct PanelHeader: View {
    let title: String
    let subtitle: String
    var symbol: String? = nil
    var tint: Color = .accentColor

    var body: some View {
        HStack(alignment: .top, spacing: 14) {
            if let symbol {
                Image(systemName: symbol)
                    .font(.title2.weight(.medium))
                    .symbolRenderingMode(.hierarchical)
                    .foregroundStyle(tint)
                    .frame(width: 48, height: 48)
                    .background(tint.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                    .accessibilityHidden(true)
            }
            VStack(alignment: .leading, spacing: 5) {
                Text(title).font(.title2.bold()).textSelection(.enabled)
                if !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

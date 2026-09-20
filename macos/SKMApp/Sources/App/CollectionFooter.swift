import SwiftUI

struct CollectionFooter: View {
    let count: Int
    let symbol: String

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: symbol).accessibilityHidden(true)
            Text("共 \(count) 项").monospacedDigit()
            Spacer()
        }
        .font(.caption)
        .foregroundStyle(.secondary)
        .padding(.horizontal, 18)
        .frame(height: 40)
        .background(SKMDesign.detailCanvas)
        .overlay(alignment: .top) { Divider() }
        .accessibilityElement(children: .combine)
    }
}

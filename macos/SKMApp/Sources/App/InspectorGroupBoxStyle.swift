import SwiftUI

struct InspectorGroupBoxStyle: GroupBoxStyle {
    func makeBody(configuration: Configuration) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            configuration.label
                .font(.headline)
                .foregroundStyle(.secondary)
            configuration.content
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(16)
        .skmSurface()
    }
}

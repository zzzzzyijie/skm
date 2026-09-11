import SwiftUI

struct SurfaceStyle: ViewModifier {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.colorSchemeContrast) private var contrast
    var radius: CGFloat = SKMDesign.cardRadius

    func body(content: Content) -> some View {
        content
            .background(SKMDesign.surface, in: RoundedRectangle(cornerRadius: radius))
            .overlay {
                RoundedRectangle(cornerRadius: radius)
                    .strokeBorder(Color.primary.opacity(contrast == .increased ? 0.4 : 0.075), lineWidth: 0.5)
                    .allowsHitTesting(false)
            }
            .shadow(color: .black.opacity(colorScheme == .dark ? 0 : 0.035), radius: 2, y: 1)
    }
}

extension View {
    func skmSurface(radius: CGFloat = SKMDesign.cardRadius) -> some View {
        modifier(SurfaceStyle(radius: radius))
    }

    func readingLayout() -> some View {
        padding(SKMDesign.pagePadding)
            .frame(maxWidth: SKMDesign.readingWidth, alignment: .leading)
            .frame(maxWidth: .infinity, alignment: .center)
    }

    func editorSurface() -> some View {
        scrollContentBackground(.hidden)
            .padding(10)
            .background(Color(nsColor: .textBackgroundColor), in: RoundedRectangle(cornerRadius: SKMDesign.controlRadius))
            .overlay {
                RoundedRectangle(cornerRadius: SKMDesign.controlRadius)
                    .strokeBorder(.separator, lineWidth: 1)
                    .allowsHitTesting(false)
            }
    }

    func skmMetadataPill(tint: Color = SKMDesign.tagTint) -> some View {
        font(.caption)
            .foregroundStyle(tint)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(tint.opacity(0.11), in: Capsule())
    }
}

import SwiftUI

/// A precisely aligned, keyboard-focusable row shared by the library and Settings sidebars.
struct SidebarNavigationButton: View {
    let title: String
    let systemImage: String
    let isSelected: Bool
    var count: Int?
    let action: () -> Void

    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.colorSchemeContrast) private var contrast
    @State private var isHovered = false
    @State private var hasTriggeredPointerDown = false
    @FocusState private var isFocused: Bool

    var body: some View {
        Button(action: action) {
            HStack(spacing: 11) {
                Image(systemName: systemImage)
                    .symbolRenderingMode(.hierarchical)
                    .font(.body)
                    .foregroundStyle(isSelected ? .primary : .secondary)
                    .frame(width: SKMDesign.sidebarIconWidth, height: SKMDesign.sidebarIconWidth)
                    .accessibilityHidden(true)

                Text(title)
                    .font(.body)
                    .bold(isSelected)
                    .foregroundStyle(isSelected ? .primary : .secondary)
                    .lineLimit(1)

                Spacer(minLength: 8)

                if let count {
                    Text(count, format: .number)
                        .font(.callout.monospacedDigit())
                        .foregroundStyle(isSelected ? .secondary : .tertiary)
                        .frame(minWidth: 24, alignment: .trailing)
                }
            }
            .padding(.horizontal, SKMDesign.sidebarRowHorizontalPadding)
            .frame(maxWidth: .infinity, minHeight: SKMDesign.sidebarRowHeight, alignment: .leading)
            .contentShape(Rectangle())
            .background {
                RoundedRectangle(cornerRadius: SKMDesign.controlRadius)
                    .fill(backgroundFill)
                    .overlay {
                        RoundedRectangle(cornerRadius: SKMDesign.controlRadius)
                            .strokeBorder(borderColor, lineWidth: 0.5)
                    }
            }
        }
        .buttonStyle(.plain)
        .focused($isFocused)
        .onHover { isHovered = $0 }
        // Native source lists select on mouse-down; SwiftUI buttons otherwise wait for mouse-up.
        .simultaneousGesture(
            DragGesture(minimumDistance: 0)
                .onChanged { _ in
                    guard !hasTriggeredPointerDown else { return }
                    hasTriggeredPointerDown = true
                    action()
                }
                .onEnded { _ in
                    hasTriggeredPointerDown = false
                }
        )
        .accessibilityLabel(title)
        .accessibilityValue(count.map(String.init) ?? "")
        .accessibilityAddTraits(isSelected ? .isSelected : [])
    }

    private var backgroundFill: Color {
        if isSelected {
            return Color.primary.opacity(colorScheme == .dark ? 0.15 : 0.085)
        }
        if isHovered {
            return Color.primary.opacity(colorScheme == .dark ? 0.075 : 0.045)
        }
        if isFocused {
            return Color.primary.opacity(colorScheme == .dark ? 0.06 : 0.035)
        }
        return .clear
    }

    private var borderColor: Color {
        if isSelected {
            return Color.primary.opacity(contrast == .increased ? 0.32 : (colorScheme == .dark ? 0.16 : 0.09))
        }
        return isFocused ? Color.primary.opacity(contrast == .increased ? 0.28 : 0.16) : .clear
    }
}

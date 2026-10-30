import SwiftUI

/// A precisely aligned, keyboard-focusable row shared by the library and Settings sidebars.
struct SidebarNavigationButton: View {
    let title: LocalizedStringKey
    let systemImage: String
    let isSelected: Bool
    var count: Int?
    var disclosureExpanded: Bool?
    let action: () -> Void

    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.colorSchemeContrast) private var contrast
    @State private var isHovered = false
    @State private var hasTriggeredPointerDown = false
    @FocusState private var isFocused: Bool

    var body: some View {
        Button(action: action) {
            HStack(spacing: 9) {
                Image(systemName: systemImage)
                    .symbolRenderingMode(.hierarchical)
                    .font(.body)
                    .foregroundStyle(isSelected ? .primary : .secondary)
                    .frame(width: SKMDesign.sidebarIconWidth, height: SKMDesign.sidebarIconWidth)
                    .accessibilityHidden(true)

                Text(title)
                    .font(.body)
                    .fontWeight(isSelected ? .semibold : .medium)
                    .foregroundStyle(isSelected ? .primary : .secondary)
                    .lineLimit(1)

                Spacer(minLength: 8)

                if let count {
                    Text(count, format: .number)
                        .font(.system(size: 12).monospacedDigit())
                        .foregroundStyle(.secondary)
                        .frame(minWidth: SKMDesign.sidebarCountWidth, alignment: .trailing)
                }
            }
            .padding(.horizontal, SKMDesign.sidebarRowHorizontalPadding)
            .frame(maxWidth: .infinity, minHeight: SKMDesign.sidebarRowHeight, alignment: .leading)
            .contentShape(Rectangle())
            .background {
                RoundedRectangle(cornerRadius: 6)
                    .fill(backgroundFill)
                    .overlay {
                        RoundedRectangle(cornerRadius: 6)
                            .strokeBorder(borderColor, lineWidth: 0.5)
                    }
            }
        }
        .buttonStyle(.plain)
        .focused($isFocused)
        .onHover { isHovered = $0 }
        // Native source lists select on mouse-down. Disclosure uses only the Button action
        // so a single click cannot toggle once on mouse-down and again on mouse-up.
        .simultaneousGesture(
            disclosureExpanded == nil ? DragGesture(minimumDistance: 0)
                .onChanged { _ in
                    guard !hasTriggeredPointerDown else { return }
                    hasTriggeredPointerDown = true
                    action()
                }
                .onEnded { _ in
                    hasTriggeredPointerDown = false
                } : nil
        )
        .accessibilityLabel(title)
        .accessibilityValue(disclosureExpanded.map {
            AppLocalization.string($0 ? "已展开" : "已收起")
        } ?? count.map(String.init) ?? "")
        .accessibilityAddTraits(isSelected ? .isSelected : [])
    }

    private var backgroundFill: Color {
        if isSelected {
            return Color.primary.opacity(colorScheme == .dark ? 0.15 : 0.12)
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
            return Color.primary.opacity(contrast == .increased ? 0.32 : 0)
        }
        return isFocused ? Color.primary.opacity(contrast == .increased ? 0.28 : 0.16) : .clear
    }
}

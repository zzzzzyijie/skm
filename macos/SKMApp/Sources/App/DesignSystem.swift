import SwiftUI

/// Shared proportions for collection chrome, reading surfaces and sheets.
enum SKMDesign {
    static let pagePadding: CGFloat = 26
    static let cardRadius: CGFloat = 12
    static let compactCardRadius: CGFloat = 9
    static let controlRadius: CGFloat = 8
    static let sectionSpacing: CGFloat = 24
    static let sidebarRowHorizontalPadding: CGFloat = 14
    static let sidebarRowHeight: CGFloat = 34
    static let sidebarIconWidth: CGFloat = 20
    static let settingsSidebarWidth: CGFloat = 210
    static let agentCardMinimumWidth: CGFloat = 210
    static let toolbarActionSize: CGFloat = 28
    static let toolbarActionPadding: CGFloat = 4
    static let readingWidth: CGFloat = 840
    static let canvas = Color(nsColor: .windowBackgroundColor)
    static let detailCanvas = Color(nsColor: .textBackgroundColor)
    static let surface = Color(nsColor: .controlBackgroundColor)
    static let tagTint = Color.orange
    static let successTint = Color.green

    // Main library window proportions, measured from the macOS reference screens.
    static let librarySidebarWidth: CGFloat = 218
    static let libraryListWidth: CGFloat = 372
    static let libraryToolbarHeight: CGFloat = 52
    static let librarySectionSpacing: CGFloat = 30
    static let libraryGreen = Color(red: 0.16, green: 0.68, blue: 0.39)
    static let librarySelection = adaptiveColor(light: 0xF3F3F5, dark: 0x303033)
    static let librarySidebar = adaptiveColor(light: 0xF5F5F5, dark: 0x242426)
    static let libraryBorder = adaptiveColor(light: 0xDEDFE5, dark: 0x424247)

    private static func adaptiveColor(light: UInt32, dark: UInt32) -> Color {
        Color(nsColor: NSColor(name: nil) { appearance in
            let rgb = appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua ? dark : light
            return NSColor(srgbRed: CGFloat((rgb >> 16) & 255) / 255,
                           green: CGFloat((rgb >> 8) & 255) / 255,
                           blue: CGFloat(rgb & 255) / 255, alpha: 1)
        })
    }
}

struct CollectionHeader<Actions: View>: View {
    let title: LocalizedStringKey
    @ViewBuilder var actions: Actions

    var body: some View {
        HStack(spacing: 8) {
            Text(title).font(.system(size: 18, weight: .semibold))
            Spacer(minLength: 8)
            actions
        }
        .padding(.leading, 20)
        .padding(.trailing, 16)
        .frame(height: SKMDesign.libraryToolbarHeight)
        .background(SKMDesign.detailCanvas)
    }
}

struct DetailToolbar<Actions: View>: View {
    var health = "available"
    var isLoading = false
    @ViewBuilder var actions: Actions

    var body: some View {
        HStack(spacing: 10) {
            actions
            Spacer(minLength: 12)
            if isLoading {
                ProgressView().controlSize(.small).frame(width: 28)
            } else {
                HealthBadge(health: health, size: 16)
                    .frame(width: 28)
            }
        }
        .padding(.horizontal, 18)
        .frame(height: SKMDesign.libraryToolbarHeight)
        .background(SKMDesign.detailCanvas)
        .overlay(alignment: .bottom) { Rectangle().fill(SKMDesign.libraryBorder).frame(height: 1) }
    }
}

struct LibraryActionButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled
    var prominent = false

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 13, weight: .semibold))
            .padding(.horizontal, prominent ? 16 : 12)
            .frame(height: prominent ? 36 : 32)
            .foregroundStyle(prominent ? Color.white : Color.primary)
            .background(prominent ? Color(red: 0.28, green: 0.69, blue: 0.30) : SKMDesign.librarySelection,
                        in: RoundedRectangle(cornerRadius: prominent ? 18 : 6))
            .opacity(!isEnabled ? 0.45 : configuration.isPressed ? 0.7 : 1)
    }
}

extension View {
    func libraryListRow(isSelected: Bool) -> some View {
        padding(.horizontal, 10)
            .foregroundStyle(.primary)
            .listRowInsets(EdgeInsets())
            .listRowSeparator(.hidden)
            .listRowBackground(isSelected ? SKMDesign.librarySelection : Color.clear)
            .background(LibrarySelectionAppearance())
    }

    func libraryReadingLayout() -> some View {
        padding(.top, 34)
            .padding(.horizontal, 42)
            .padding(.trailing, 18)
            .padding(.bottom, 40)
            .frame(maxWidth: 1000, alignment: .leading)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    func libraryCard() -> some View {
        background(SKMDesign.detailCanvas, in: RoundedRectangle(cornerRadius: 8))
            .overlay {
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(SKMDesign.libraryBorder, lineWidth: 1)
                    .allowsHitTesting(false)
            }
    }

    func libraryMetadataPill() -> some View {
        font(.system(size: 12))
            .foregroundStyle(Color.orange)
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
            .background(Color.orange.opacity(0.11), in: Capsule())
    }
}

/// Keep native list selection and keyboard navigation, with the reference's neutral highlight.
private struct LibrarySelectionAppearance: NSViewRepresentable {
    func makeNSView(context: Context) -> SelectionView { SelectionView() }
    func updateNSView(_ nsView: SelectionView, context: Context) { nsView.updateSelectionAppearance() }

    final class SelectionView: NSView {
        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            updateSelectionAppearance()
        }

        func updateSelectionAppearance() {
            var ancestor = superview
            while let view = ancestor {
                if let table = view as? NSTableView {
                    table.selectionHighlightStyle = .none
                    return
                }
                ancestor = view.superview
            }
        }
    }
}

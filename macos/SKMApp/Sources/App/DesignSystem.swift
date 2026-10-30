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
    static let sidebarChildIndent: CGFloat = 18
    static let sidebarChildRowHeight: CGFloat = 32
    static let sidebarCountWidth: CGFloat = 24
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
    static let libraryListWidth: CGFloat = 320
    static let libraryListInset: CGFloat = 20
    // Native macOS List cells contribute another 8 pt outside the row content.
    static let libraryListRowPadding: CGFloat = libraryListInset - 8
    static let libraryToolbarHeight: CGFloat = 52
    static let libraryFooterHeight: CGFloat = 40
    static let libraryReadingInset: CGFloat = 32
    static let libraryReadingWidth: CGFloat = 864
    static let librarySectionSpacing: CGFloat = 28
    static let libraryTitleFont = Font.system(size: 28, weight: .semibold)
    static let libraryDescriptionFont = Font.system(size: 14)
    static let tagManagerWidth: CGFloat = 620
    static let tagManagerHeight: CGFloat = 560
    static let tagManagerRowHeight: CGFloat = 46
    static let tagManagerSkillTint = adaptiveColor(light: 0x3478F6, dark: 0x6EA8FF)
    static let tagManagerPromptTint = adaptiveColor(light: 0x8B5CF6, dark: 0xB59AFF)
    static let libraryGreen = Color(red: 0.16, green: 0.68, blue: 0.39)
    static let librarySelection = adaptiveColor(light: 0xF3F3F5, dark: 0x303033)
    static let libraryRowSelection = Color.accentColor.opacity(0.10)
    static let librarySidebar = adaptiveColor(light: 0xF5F5F5, dark: 0x242426)
    static let libraryBorder = adaptiveColor(light: 0xDEDFE5, dark: 0x424247)
    static let sheetHorizontalPadding: CGFloat = 22
    static let sheetVerticalPadding: CGFloat = 20

    private static func adaptiveColor(light: UInt32, dark: UInt32) -> Color {
        Color(nsColor: NSColor(name: nil) { appearance in
            let rgb = appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua ? dark : light
            return NSColor(srgbRed: CGFloat((rgb >> 16) & 255) / 255,
                           green: CGFloat((rgb >> 8) & 255) / 255,
                           blue: CGFloat(rgb & 255) / 255, alpha: 1)
        })
    }
}

/// A compact, native-feeling title area for modal workflows.
struct SheetHeader<Trailing: View>: View {
    let title: String
    let subtitle: String
    let symbol: String
    var tint: Color = .accentColor
    @ViewBuilder let trailing: Trailing

    init(
        title: String,
        subtitle: String,
        symbol: String,
        tint: Color = .accentColor,
        @ViewBuilder trailing: () -> Trailing
    ) {
        self.title = title
        self.subtitle = subtitle
        self.symbol = symbol
        self.tint = tint
        self.trailing = trailing()
    }

    var body: some View {
        HStack(spacing: 16) {
            Image(systemName: symbol)
                .font(.system(size: 22, weight: .medium))
                .symbolRenderingMode(.hierarchical)
                .foregroundStyle(.white)
                .frame(width: 48, height: 48)
                .background(tint.gradient, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                .overlay {
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .strokeBorder(.white.opacity(0.18), lineWidth: 1)
                }
                .shadow(color: tint.opacity(0.16), radius: 5, y: 3)
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 5) {
                Text(title)
                    .font(.system(size: 21, weight: .semibold))
                    .lineLimit(1)
                if !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            trailing
        }
        .padding(24)
    }
}

extension SheetHeader where Trailing == EmptyView {
    init(title: String, subtitle: String, symbol: String, tint: Color = .accentColor) {
        self.init(title: title, subtitle: subtitle, symbol: symbol, tint: tint) { EmptyView() }
    }
}

/// Quiet section labels keep the controls, rather than nested containers, in focus.
struct SheetSection<Content: View>: View {
    let title: LocalizedStringKey
    var subtitle: LocalizedStringKey? = nil
    let symbol: String
    @ViewBuilder let content: Content

    init(
        _ title: LocalizedStringKey,
        subtitle: LocalizedStringKey? = nil,
        symbol: String,
        @ViewBuilder content: () -> Content
    ) {
        self.title = title
        self.subtitle = subtitle
        self.symbol = symbol
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 11) {
            HStack(alignment: .firstTextBaseline, spacing: 7) {
                Text(title)
                    .font(.system(size: 13, weight: .semibold))
                Spacer(minLength: 8)
                if let subtitle {
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            content
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

/// Keeps primary and cancel actions stationary at the bottom of a sheet.
struct SheetActionBar<Leading: View, Actions: View>: View {
    @ViewBuilder let leading: Leading
    @ViewBuilder let actions: Actions

    init(@ViewBuilder leading: () -> Leading, @ViewBuilder actions: () -> Actions) {
        self.leading = leading()
        self.actions = actions()
    }

    var body: some View {
        HStack(spacing: 10) {
            leading
            Spacer(minLength: 16)
            actions
        }
        .padding(.horizontal, SKMDesign.sheetHorizontalPadding)
        .padding(.vertical, 16)
        .background(.bar)
        .overlay(alignment: .top) { Divider() }
    }
}

extension SheetActionBar where Leading == EmptyView {
    init(@ViewBuilder actions: () -> Actions) {
        self.init(leading: { EmptyView() }, actions: actions)
    }
}

struct CollectionHeader<Actions: View>: View {
    let title: LocalizedStringKey
    @ViewBuilder var actions: Actions

    var body: some View {
        HStack(spacing: 8) {
            Text(title).font(.headline).lineLimit(1)
            Spacer(minLength: 8)
            actions
        }
        .padding(.horizontal, SKMDesign.libraryListInset)
        .frame(height: SKMDesign.libraryToolbarHeight)
        .background(SKMDesign.detailCanvas)
        .overlay(alignment: .bottom) { Divider() }
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
        .padding(.horizontal, SKMDesign.libraryListInset)
        .frame(height: SKMDesign.libraryToolbarHeight)
        .background(SKMDesign.detailCanvas)
        .overlay(alignment: .bottom) { Divider() }
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
    func libraryToolbarIcon() -> some View {
        labelStyle(.iconOnly)
            .buttonStyle(.borderless)
            .font(.system(size: 14, weight: .regular))
            .foregroundStyle(.secondary)
            .controlSize(.regular)
            .frame(width: SKMDesign.toolbarActionSize, height: SKMDesign.toolbarActionSize)
            .contentShape(Rectangle())
    }

    func libraryFooterStyle() -> some View {
        font(.caption).foregroundStyle(.secondary)
            .padding(.horizontal, SKMDesign.libraryListInset)
            .frame(height: SKMDesign.libraryFooterHeight)
            .overlay(alignment: .top) { Divider() }
    }

    func sidebarSearchStyle() -> some View {
        textFieldStyle(.roundedBorder)
            .font(.callout).controlSize(.regular)
            .padding(.horizontal, SKMDesign.sidebarRowHorizontalPadding)
            .padding(.vertical, 4)
    }

    func libraryListRow(isSelected: Bool, horizontalPadding: CGFloat = 10) -> some View {
        padding(.horizontal, horizontalPadding)
            .foregroundStyle(.primary)
            .listRowInsets(EdgeInsets())
            .listRowSeparator(.hidden)
            .listRowBackground(isSelected ? SKMDesign.libraryRowSelection : Color.clear)
            .background(LibrarySelectionAppearance())
    }

    func libraryReadingLayout() -> some View {
        padding(.top, 32)
            .padding(.horizontal, SKMDesign.libraryReadingInset)
            .padding(.bottom, 40)
            .frame(maxWidth: SKMDesign.libraryReadingWidth, alignment: .leading)
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

    func librarySourceBadge() -> some View {
        font(.callout)
            .foregroundStyle(.secondary)
            .padding(.vertical, 3)
    }

    func libraryTagPill() -> some View {
        font(.system(size: 12))
            .foregroundStyle(.secondary)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(SKMDesign.librarySelection, in: RoundedRectangle(cornerRadius: 5))
    }
}

/// Compact metadata that wraps within the reading column.
struct LibraryMetadataStrip: View {
    let source: String
    let tags: [String]
    var sourceHelp: String? = nil

    private var isLocalSource: Bool {
        let trimmed = source.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty || trimmed.caseInsensitiveCompare("local") == .orderedSame
    }

    private var sourceName: String {
        isLocalSource
            ? AppLocalization.string("本地")
            : source.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var sourceSymbol: String {
        isLocalSource ? "internaldrive" : "arrow.triangle.branch"
    }

    var body: some View {
        TagWrapLayout(spacing: 8) {
            sourceBadge
            tagBadges
        }
    }

    @ViewBuilder
    private var sourceBadge: some View {
        let badge = Label(sourceName, systemImage: sourceSymbol)
            .librarySourceBadge()
            .lineLimit(1).truncationMode(.middle)
            .accessibilityLabel("\(AppLocalization.string("来源")) \(sourceName)")

        if let sourceHelp, !sourceHelp.isEmpty {
            badge.help(sourceHelp)
        } else {
            badge
        }
    }

    @ViewBuilder
    private var tagBadges: some View {
        ForEach(tags, id: \.self) { tag in
            Label(tag, systemImage: "tag.fill")
                .libraryTagPill()
                .lineLimit(1).truncationMode(.tail).help(tag)
                .accessibilityLabel("\(AppLocalization.string("标签")) \(tag)")
        }
    }
}

/// Keep native list selection and keyboard navigation while using the shared row highlight.
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

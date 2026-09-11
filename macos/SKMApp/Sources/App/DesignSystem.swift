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
}

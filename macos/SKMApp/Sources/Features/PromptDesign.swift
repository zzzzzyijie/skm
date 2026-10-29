import SwiftUI

enum PromptDesign {
    static let listWidth: CGFloat = 320
    static let listInset: CGFloat = 20
    // macOS List contributes an 8 pt cell margin outside the row content.
    static let listRowPadding: CGFloat = listInset - 8
    static let readingInset: CGFloat = 32
    static let readingWidth: CGFloat = 864
    static let sectionSpacing: CGFloat = 28
    static let editorWidth: CGFloat = 880
    static let editorHeight: CGFloat = 580
    static let propertiesWidth: CGFloat = 248
}

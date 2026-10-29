import AppKit

@MainActor
enum PromptActions {
    static func copy(_ body: String, announces: Bool = false) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(body, forType: .string)
        if announces {
            NSAccessibility.post(element: NSApp as Any, notification: .announcementRequested, userInfo: [
                .announcement: AppLocalization.string("Prompt 已复制"),
                .priority: NSAccessibilityPriorityLevel.high.rawValue,
            ])
        }
    }

    static func export(_ details: PromptDetails) throws {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "\(details.name).md"
        if panel.runModal() == .OK, let url = panel.url {
            try details.content.write(to: url, atomically: true, encoding: .utf8)
        }
    }
}

import SwiftUI

@main
struct SKMApp: App {
    @State private var model = AppModel()

    var body: some Scene {
        Window("SKM", id: "main") {
            RootView(model: model)
                .frame(minWidth: 980, minHeight: 640)
                .task { await model.start() }
                .onDisappear { Task { await model.stop() } }
        }
        .defaultSize(width: 1180, height: 760)
        .commands {
            CommandGroup(after: .sidebar) {
                Button("刷新") { Task { await model.refresh() } }
                    .keyboardShortcut("r", modifiers: .command)
            }
        }

        Settings {
            SettingsView(model: model)
                .frame(width: 520, height: 300)
        }
    }
}

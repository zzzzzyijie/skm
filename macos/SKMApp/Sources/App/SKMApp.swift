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
            CommandGroup(replacing: .newItem) {
                Button(newItemTitle) { model.request(.create) }
                    .keyboardShortcut("n", modifiers: .command)
                    .disabled(!model.canCreate)
                Button(importTitle) { model.request(.importItem) }
                    .keyboardShortcut("o", modifiers: .command)
                    .disabled(!model.canImport)
            }
            CommandGroup(after: .sidebar) {
                Button("刷新") { Task { await model.refresh() } }
                    .keyboardShortcut("r", modifiers: .command)
            }
            CommandGroup(after: .pasteboard) {
                Button("删除所选项目") { model.request(.deleteSelection) }
                    .keyboardShortcut(.delete, modifiers: .command)
                    .disabled(!model.canDeleteSelection)
            }
            CommandMenu("导航") {
                ForEach(Array(AppSection.allCases.enumerated()), id: \.element.id) { index, section in
                    Button(section.rawValue) { model.section = section }
                        .keyboardShortcut(KeyEquivalent(Character(String(index + 1))), modifiers: .command)
                }
            }
        }

        Settings {
            SettingsView(model: model)
                .frame(width: 520, height: 300)
        }
    }

    private var newItemTitle: String {
        switch model.section {
        case .skills: String(localized: "添加 Skill")
        case .prompts: String(localized: "新建 Prompt")
        case .projects: String(localized: "添加项目（Phase 2）")
        case .sources: String(localized: "添加 Git Source")
        case .workspace: String(localized: "配置个人工作区")
        case .agents: String(localized: "添加自定义 Agent")
        case .diagnostics: String(localized: "运行诊断")
        }
    }

    private var importTitle: String {
        switch model.section {
        case .prompts: String(localized: "导入 Prompt…")
        case .projects: String(localized: "添加项目…")
        default: String(localized: "导入 Skill…")
        }
    }
}

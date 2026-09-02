import SwiftUI

/// SKM macOS 原生应用程序入口
/// 组织了主窗口、系统菜单栏（快捷键绑定）、QuickLook 快速查看唤起以及 Settings 设置面板场景。
@main
struct SKMApp: App {
    /// 全局应用状态 ViewModel
    @State private var model = AppModel()

    var body: some Scene {
        // 主应用程序窗口
        Window("SKM", id: "main") {
            RootView(model: model)
                .frame(minWidth: 980, minHeight: 640)
                .task { await model.start() }
                .onDisappear { Task { await model.stop() } }
        }
        .defaultSize(width: 1180, height: 760)
        .commands {
            // 文件菜单（新建 / 导入快捷键：Cmd+N / Cmd+O）
            CommandGroup(replacing: .newItem) {
                Button(newItemTitle) { model.request(.create) }
                    .keyboardShortcut("n", modifiers: .command)
                    .disabled(!model.canCreate)
                Button(importTitle) { model.request(.importItem) }
                    .keyboardShortcut("o", modifiers: .command)
                    .disabled(!model.canImport)
            }
            // 视图与更新菜单（刷新 Cmd+R、检查更新）
            CommandGroup(after: .sidebar) {
                Button("刷新") { Task { await model.refresh() } }
                    .keyboardShortcut("r", modifiers: .command)
                Button("检查更新…") { Task { await model.checkForUpdates() } }
            }
            // 快捷预览与删除（空格预览）
            CommandGroup(after: .pasteboard) {
                Button("快速查看") { Task { await showQuickLook() } }
                    .keyboardShortcut(.space, modifiers: [])
                    .disabled(model.section != .skills && model.section != .prompts)
                Button("删除所选项目") { model.request(.deleteSelection) }
                    .disabled(!model.canDeleteSelection)
            }
            // 导航快捷键（Cmd+1/2/3 切换 Skills / Prompts / Projects）
            CommandMenu("导航") {
                ForEach(Array(AppSection.allCases.enumerated()), id: \.element.id) { index, section in
                    Button(section.rawValue) { model.section = section }
                        .keyboardShortcut(KeyEquivalent(Character(String(index + 1))), modifiers: .command)
                }
            }
        }

        // macOS 标准偏好设置窗口（Cmd+,）
        Settings {
            SettingsView(model: model)
                .frame(minWidth: 780, minHeight: 560)
        }
    }

    /// 根据当前所选业务分区动态返回“新建”按钮文案
    private var newItemTitle: String {
        switch model.section {
        case .skills: String(localized: "添加 Skill")
        case .prompts: String(localized: "新建 Prompt")
        case .projects: String(localized: "添加项目")
        }
    }

    /// 根据当前所选业务分区动态返回“导入”按钮文案
    private var importTitle: String {
        switch model.section {
        case .prompts: String(localized: "导入 Prompt…")
        case .projects: String(localized: "添加项目…")
        default: String(localized: "导入 Skill…")
        }
    }

    /// 获取当前选中项的文件路径并调用系统 QuickLook 进行预览
    private func showQuickLook() async {
        do {
            guard let url = try await model.quickLookURL() else { return }
            QuickLookPresenter.shared.show(url)
        } catch { model.errorMessage = error.localizedDescription }
    }
}

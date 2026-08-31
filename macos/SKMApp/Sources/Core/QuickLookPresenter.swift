import AppKit
import QuickLookUI

/// QuickLook 快速查看呈现器
/// 实现了 `QLPreviewPanelDataSource` 数据源协议，响应用户按下空格键时唤起系统原生的 QuickLook 预览面板，
/// 允许直接查看 SKILL.md 或 PROMPT.md 的 Markdown 格式渲染效果。
@MainActor
final class QuickLookPresenter: NSObject, @preconcurrency QLPreviewPanelDataSource {
    static let shared = QuickLookPresenter()
    private var previewURL: URL?

    /// 打开指定文件 URL 的 QuickLook 预览
    func show(_ url: URL) {
        guard FileManager.default.fileExists(atPath: url.path), let panel = QLPreviewPanel.shared() else {
            NSSound.beep()
            return
        }
        previewURL = url
        panel.dataSource = self
        panel.reloadData()
        panel.makeKeyAndOrderFront(nil)
    }

    func numberOfPreviewItems(in panel: QLPreviewPanel!) -> Int {
        previewURL == nil ? 0 : 1
    }

    func previewPanel(_ panel: QLPreviewPanel!, previewItemAt index: Int) -> QLPreviewItem! {
        previewURL as NSURL?
    }
}

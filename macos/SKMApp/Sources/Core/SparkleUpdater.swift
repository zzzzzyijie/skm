import Foundation
import Sparkle

/// Sparkle 自动更新管理器封装
/// 检查 Info.plist 中的 `SUFeedURL` 与 `SUPublicEDKey` 配置。
/// 若签名与 Appcast 配置齐全，则使用 Sparkle 原生 UI 进行自动更新检测；
/// 若未配置（如开发调试构建），则降级走 GitHub Releases API 检查。
@MainActor
final class SparkleUpdater {
    static let shared = SparkleUpdater()

    private let controller: SPUStandardUpdaterController?

    private init() {
        let info = Bundle.main.infoDictionary ?? [:]
        let feedURL = (info["SUFeedURL"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let publicKey = (info["SUPublicEDKey"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !feedURL.isEmpty, !publicKey.isEmpty else {
            controller = nil
            return
        }
        controller = SPUStandardUpdaterController(
            startingUpdater: true,
            updaterDelegate: nil,
            userDriverDelegate: nil
        )
    }

    /// 是否已完整配置 Sparkle 更新源与公钥
    var isConfigured: Bool { controller != nil }

    /// 触发检查更新弹窗
    func checkForUpdates() {
        controller?.checkForUpdates(nil)
    }
}

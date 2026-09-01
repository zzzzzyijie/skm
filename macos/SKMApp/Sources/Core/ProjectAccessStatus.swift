import SwiftUI

enum ProjectAccessStatus: Equatable, Sendable {
    case available
    case missing
    case permissionDenied
    case unavailable

    init(project: ProjectModel) {
        switch project.access {
        case "available":
            self = .available
        case nil where project.exists:
            self = .available
        case "missing", nil:
            self = .missing
        case "permission-denied":
            self = .permissionDenied
        default:
            self = .unavailable
        }
    }

    var canRead: Bool { self == .available }

    var title: String {
        switch self {
        case .available: String(localized: "读取正常")
        case .missing: String(localized: "路径不存在")
        case .permissionDenied: String(localized: "需要文件访问权限")
        case .unavailable: String(localized: "目录不可用")
        }
    }

    var detail: String {
        switch self {
        case .available:
            String(localized: "SKM 可以扫描此项目；写入仍只会在部署预览确认后进行。")
        case .missing:
            String(localized: "登记的项目目录已被移动、重命名或删除。")
        case .permissionDenied:
            String(localized: "macOS 或文件系统拒绝了目录读取，请重新选择该目录授权。")
        case .unavailable:
            String(localized: "目录当前无法读取；请检查磁盘、网络卷或 Finder 权限。")
        }
    }

    var symbol: String {
        switch self {
        case .available: "checkmark.circle.fill"
        case .missing: "folder.badge.questionmark"
        case .permissionDenied: "lock.fill"
        case .unavailable: "exclamationmark.triangle.fill"
        }
    }

    var color: Color {
        switch self {
        case .available: .green
        case .missing, .unavailable: .orange
        case .permissionDenied: .red
        }
    }
}

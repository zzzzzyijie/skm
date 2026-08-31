import Darwin
import Foundation

/// 本地目录/文件变化监听器
/// 通过 Darwin 的 `open(..., O_EVTONLY)` 和 GCD `DispatchSourceFileSystemObject` 监控指定目录（如 ~/.skm）的写入、重命名与删除事件。
/// 内置 250ms 防抖机制，当 CLI 或外部工具产生多连发文件改动时，合并触发一次 UI 刷新回调。
final class FileChangeMonitor: @unchecked Sendable {
    private let queue = DispatchQueue(label: "com.zzzzzyijie.skm.file-monitor", qos: .utility)
    private var sources: [DispatchSourceFileSystemObject] = []
    private var pending: DispatchWorkItem?

    /// 启动对指定路径列表的监听
    /// - Parameters:
    ///   - paths: 待监听的目录路径列表
    ///   - onChange: 发生变更并防抖沉降后的主线程回调
    func start(paths: [String], onChange: @escaping @MainActor @Sendable () -> Void) {
        stop()
        for path in paths {
            let descriptor = open(path, O_EVTONLY)
            guard descriptor >= 0 else { continue }
            let source = DispatchSource.makeFileSystemObjectSource(
                fileDescriptor: descriptor,
                eventMask: [.write, .extend, .attrib, .rename, .delete],
                queue: queue
            )
            source.setEventHandler { [weak self] in
                guard let self else { return }
                self.pending?.cancel()
                let work = DispatchWorkItem {
                    Task { @MainActor in onChange() }
                }
                self.pending = work
                self.queue.asyncAfter(deadline: .now() + .milliseconds(250), execute: work)
            }
            source.setCancelHandler { close(descriptor) }
            sources.append(source)
            source.resume()
        }
    }

    /// 停止所有监听并释放文件描述符
    func stop() {
        pending?.cancel()
        pending = nil
        for source in sources { source.cancel() }
        sources.removeAll()
    }

    deinit {
        stop()
    }
}

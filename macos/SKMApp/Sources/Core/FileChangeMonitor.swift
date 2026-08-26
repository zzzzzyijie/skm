import Darwin
import Foundation

final class FileChangeMonitor: @unchecked Sendable {
    private let queue = DispatchQueue(label: "com.zzzzzyijie.skm.file-monitor", qos: .utility)
    private var sources: [DispatchSourceFileSystemObject] = []
    private var pending: DispatchWorkItem?

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

import Foundation

/// SKM Core 客户端通信协议
/// 抽象了与底层 Go 核心服务（skm-core 进程）的交互接口
protocol CoreServing: Sendable {
    /// 与 Core 建立初始握手，协商协议版本与应用能力
    func handshake() async throws -> Handshake
    /// 调用 Core 暴露的 JSON-RPC 2.0 方法
    func call<Params: Encodable & Sendable, Result: Decodable & Sendable>(
        _ method: String,
        params: Params
    ) async throws -> Result
    /// 停止与 Core 进程的连接并回收资源
    func stop() async
}

/// Core 客户端错误分类
enum CoreClientError: LocalizedError, Sendable {
    /// 找不到 skm-core 可执行文件
    case executableMissing
    /// Core 进程异常退出或管道断开
    case processStopped(String)
    /// 接收到无法解析的非 JSON-RPC 响应数据
    case malformedResponse
    /// 请求在设定时间内未得到 Core 响应
    case requestTimedOut
    /// 协议版本不匹配（App 与 Core 协议代数不一致）
    case protocolMismatch(Int)
    /// App 与 Core 发布版本不一致
    case versionMismatch(app: String, core: String)
    /// Core 业务端返回的 RPC 错误（包含业务 code、错误类型及是否支持重试）
    case remote(code: Int, message: String, kind: String, retryable: Bool)

    var errorDescription: String? {
        switch self {
        case .executableMissing:
            String(localized: "App 中缺少 skm-core，请重新构建或安装 SKM。")
        case let .processStopped(message):
            message.isEmpty
                ? String(localized: "SKM Core 已停止。")
                : String(
                    format: String(localized: "SKM Core 已停止。\n%@"),
                    locale: .current,
                    message
                )
        case .malformedResponse:
            String(localized: "SKM Core 返回了无法识别的数据。")
        case .requestTimedOut:
            String(localized: "SKM Core 请求超时，已停止无响应的 Core。")
        case let .protocolMismatch(version):
            String(
                format: String(localized: "Core 协议版本 %lld 与 App 不兼容。"),
                locale: .current,
                version
            )
        case let .versionMismatch(app, core):
            String(
                format: String(localized: "App 版本 %1$@ 与 Core 版本 %2$@ 不一致，请重新安装 SKM。"),
                locale: .current,
                app,
                core
            )
        case let .remote(_, message, _, _):
            message
        }
    }

    var kind: String {
        switch self {
        case .executableMissing: "executable_missing"
        case .processStopped: "process_stopped"
        case .malformedResponse: "malformed_response"
        case .requestTimedOut: "timeout"
        case .protocolMismatch: "protocol_mismatch"
        case .versionMismatch: "version_mismatch"
        case let .remote(_, _, kind, _): kind
        }
    }
}

private struct RPCRequest<Params: Encodable>: Encodable {
    let jsonrpc = "2.0"
    let id: String
    let method: String
    let params: Params
}

private struct RPCErrorEnvelope: Decodable {
    struct Details: Decodable {
        let kind: String
        let retryable: Bool
    }

    let code: Int
    let message: String
    let data: Details
}

private struct RPCEnvelope<Result: Decodable>: Decodable {
    let jsonrpc: String?
    let id: String?
    let result: Result?
    let error: RPCErrorEnvelope?
}

/// CoreClient Actor
/// 负责管理 `skm-core --stdio` 进程生命周期并通过标准输入输出（stdin/stdout）进行 JSON-RPC 2.0 通信。
/// 通过 CoreCallGate 确保请求的单飞互斥，并为幂等只读方法提供进程崩溃自动拉起与重试机制。
actor CoreClient: CoreServing {
    private let executableOverride: URL?
    private let requestTimeout: Duration
    private let callGate = CoreCallGate()
    private var process: Process?
    private var input: FileHandle?
    private var output: FileHandle?
    private var errorOutput: FileHandle?
    private var responseBuffer = Data()
    private var requestNumber = 0

    init(executableURL: URL? = nil, requestTimeout: Duration = .seconds(30)) {
        executableOverride = executableURL
        self.requestTimeout = requestTimeout
    }

    /// 启动 Go Core 子进程（若尚未启动）
    /// 优先级顺序查找可执行文件：初始化指定 > 环境变量 SKM_CORE_EXECUTABLE > App Bundle 内置 skm-core
    func start() throws {
        guard process == nil else { return }
        let executable: URL
        if let executableOverride {
            executable = executableOverride
        } else if let override = ProcessInfo.processInfo.environment["SKM_CORE_EXECUTABLE"] {
            executable = URL(fileURLWithPath: override)
        } else if let bundled = Bundle.main.url(forResource: "skm-core", withExtension: nil) {
            executable = bundled
        } else {
            throw CoreClientError.executableMissing
        }

        let standardInput = Pipe()
        let standardOutput = Pipe()
        let standardError = Pipe()
        let child = Process()
        child.executableURL = executable
        child.arguments = coreArguments()
        child.standardInput = standardInput
        child.standardOutput = standardOutput
        child.standardError = standardError
        try child.run()
        process = child
        input = standardInput.fileHandleForWriting
        output = standardOutput.fileHandleForReading
        errorOutput = standardError.fileHandleForReading
    }

    /// 执行握手检测：校验协议版本是否匹配，并在正式版本下比对 App 与 Core 版本号
    func handshake() async throws -> Handshake {
        let appVersion = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
        let value: Handshake = try await call(
            "system.handshake",
            params: HandshakeParams(protocolVersion: 1, appVersion: appVersion)
        )
        guard value.protocolVersion == 1 else {
            throw CoreClientError.protocolMismatch(value.protocolVersion)
        }
        if appVersion != "dev", value.coreVersion != "dev", appVersion != value.coreVersion {
            throw CoreClientError.versionMismatch(app: appVersion, core: value.coreVersion)
        }
        return value
    }

    /// 发起 RPC 调用
    /// 包含请求网关排队调度与只读幂等调用的自动重启重试保护
    func call<Params: Encodable & Sendable, Result: Decodable & Sendable>(
        _ method: String,
        params: Params
    ) async throws -> Result {
		await callGate.acquire()
        do {
			let value: Result
			do {
				value = try await send(method, params: params)
			} catch let error as CoreClientError where Self.isSafeToRetry(method) && error.isProcessStopped {
				// 若 Core 意外停止且该方法为幂等只读，则重启进程后重试一次
				stopProcess()
				value = try await send(method, params: params)
			}
			await callGate.release()
			return value
		} catch {
			await callGate.release()
			throw error
        }
    }

    /// 判断指定 RPC 方法是否属于无副作用的只读/查询操作，可在进程崩溃时安全重试
    static func isSafeToRetry(_ method: String) -> Bool {
        switch method {
        case "system.handshake", "system.doctor",
             "skills.list", "skills.get", "agents.list", "activations.status",
             "prompts.list", "prompts.get", "sources.list", "projects.list", "projects.get",
             "prompts.render", "history.list", "history.get", "history.diff",
             "workspace.get", "workspace.preview":
            true
        default:
            false
        }
    }

    /// 底层发送 JSON-RPC 请求，通过 stdin 写入并等待 stdout 的单行响应
    private func send<Params: Encodable, Result: Decodable>(_ method: String, params: Params) async throws -> Result {
        try start()
        requestNumber += 1
        let request = RPCRequest(id: String(requestNumber), method: method, params: params)
        var data = try JSONEncoder().encode(request)
        data.append(0x0A)
        guard let input else { throw CoreClientError.processStopped("") }
        do {
            try input.write(contentsOf: data)
        } catch {
            stopProcess()
            throw CoreClientError.processStopped(error.localizedDescription)
        }

        let responseData: Data
        do {
            responseData = try await readLine()
        } catch {
            stopProcess()
            throw error
        }
        let envelope: RPCEnvelope<Result>
        do {
            envelope = try JSONDecoder().decode(RPCEnvelope<Result>.self, from: responseData)
        } catch {
            stopProcess()
            throw CoreClientError.malformedResponse
        }
        guard envelope.jsonrpc == "2.0", envelope.id == request.id else {
            stopProcess()
            throw CoreClientError.malformedResponse
        }
        if let error = envelope.error {
            throw CoreClientError.remote(
                code: error.code,
                message: error.message,
                kind: error.data.kind,
                retryable: error.data.retryable
            )
        }
        guard let result = envelope.result else {
            stopProcess()
            throw CoreClientError.malformedResponse
        }
        return result
    }

    func stop() async {
        stopProcess()
    }

    private func stopProcess() {
        try? input?.close()
        if process?.isRunning == true { process?.terminate() }
        process = nil
        input = nil
        output?.readabilityHandler = nil
        output = nil
        errorOutput = nil
        responseBuffer.removeAll(keepingCapacity: false)
    }

    private func readLine() async throws -> Data {
        while true {
            if let newline = responseBuffer.firstIndex(of: 0x0A) {
                let line = responseBuffer[..<newline]
                responseBuffer.removeSubrange(...newline)
                return Data(line)
            }
            guard let output else { throw CoreClientError.processStopped("") }
            let chunk = try await AsyncFileRead.read(from: output, timeout: requestTimeout)
            if chunk.isEmpty {
                let diagnostic = errorOutput.flatMap { try? $0.readToEnd() }.flatMap { String(data: $0, encoding: .utf8) } ?? ""
                stopProcess()
                throw CoreClientError.processStopped(diagnostic.trimmingCharacters(in: .whitespacesAndNewlines))
            }
            responseBuffer.append(chunk)
        }
    }

    private func coreArguments() -> [String] {
        var values = ["core", "--stdio"]
        let environment = ProcessInfo.processInfo.environment
        if let home = environment["SKM_HOME"], !home.isEmpty {
            values += ["--home", home]
        }
        if let userHome = environment["SKM_USER_HOME"], !userHome.isEmpty {
            values += ["--user-home", userHome]
        }
        if let project = environment["SKM_PROJECT"], !project.isEmpty {
            values += ["--project", project]
        }
        return values
    }
}

/// 异步请求网关锁
/// 保证所有发送到 stdin/stdout 的 JSON-RPC 请求按序排队执行，防止多线程并发读写破坏单行协议帧
private actor CoreCallGate {
    private var locked = false
    private var waiters: [CheckedContinuation<Void, Never>] = []

    func acquire() async {
        if !locked {
            locked = true
            return
        }
        await withCheckedContinuation { continuation in
            waiters.append(continuation)
        }
    }

    func release() {
        if waiters.isEmpty {
            locked = false
            return
        }
        waiters.removeFirst().resume()
    }
}

private extension CoreClientError {
    var isProcessStopped: Bool {
        switch self {
        case .processStopped, .malformedResponse, .requestTimedOut: true
        default: false
        }
    }
}

/// 异步文件句柄读取器
/// 包装 FileHandle.readabilityHandler，提供支持超时的异步单次数据块读取
private final class AsyncFileRead: @unchecked Sendable {
    private let lock = NSLock()
    private var continuation: CheckedContinuation<Data, any Error>?
    private var timeoutTask: Task<Void, Never>?
    private weak var handle: FileHandle?

    static func read(from handle: FileHandle, timeout: Duration) async throws -> Data {
        let reader = AsyncFileRead()
        return try await reader.start(handle: handle, timeout: timeout)
    }

    private func start(handle: FileHandle, timeout: Duration) async throws -> Data {
        try await withCheckedThrowingContinuation { continuation in
            lock.withLock {
                self.continuation = continuation
                self.handle = handle
            }
            handle.readabilityHandler = { [weak self] readable in
                self?.finish(.success(readable.availableData))
            }
            let task = Task { [weak self] in
                try? await Task.sleep(for: timeout)
                guard !Task.isCancelled else { return }
                self?.finish(.failure(CoreClientError.requestTimedOut))
            }
            lock.withLock {
                if self.continuation == nil { task.cancel() }
                else { timeoutTask = task }
            }
        }
    }

    private func finish(_ result: Result<Data, any Error>) {
        let value: CheckedContinuation<Data, any Error>? = lock.withLock {
            guard let continuation else { return nil }
            self.continuation = nil
            timeoutTask?.cancel()
            timeoutTask = nil
            return continuation
        }
        handle?.readabilityHandler = nil
        value?.resume(with: result)
    }
}

private struct HandshakeParams: Encodable {
    let protocolVersion: Int
    let appVersion: String
}

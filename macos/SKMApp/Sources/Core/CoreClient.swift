import Foundation

protocol CoreServing: Sendable {
    func handshake() async throws -> Handshake
    func call<Params: Encodable & Sendable, Result: Decodable & Sendable>(
        _ method: String,
        params: Params
    ) async throws -> Result
    func stop() async
}

enum CoreClientError: LocalizedError, Sendable {
    case executableMissing
    case processStopped(String)
    case malformedResponse
    case requestTimedOut
    case protocolMismatch(Int)
    case versionMismatch(app: String, core: String)
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

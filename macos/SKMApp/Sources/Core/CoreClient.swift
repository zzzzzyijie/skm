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
    let result: Result?
    let error: RPCErrorEnvelope?
}

actor CoreClient: CoreServing {
    private var process: Process?
    private var input: FileHandle?
    private var output: FileHandle?
    private var errorOutput: FileHandle?
    private var responseBuffer = Data()
    private var requestNumber = 0

    func start() throws {
        guard process == nil else { return }
        let executable: URL
        if let override = ProcessInfo.processInfo.environment["SKM_CORE_EXECUTABLE"] {
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
        do {
            return try send(method, params: params)
        } catch let error as CoreClientError where Self.isSafeToRetry(method) && error.isProcessStopped {
            stopProcess()
            return try send(method, params: params)
        }
    }

    static func isSafeToRetry(_ method: String) -> Bool {
        switch method {
        case "system.handshake", "system.doctor",
             "skills.list", "skills.get", "agents.list", "activations.status",
             "prompts.list", "prompts.get", "sources.list", "projects.list", "workspace.get":
            true
        default:
            false
        }
    }

    private func send<Params: Encodable, Result: Decodable>(_ method: String, params: Params) throws -> Result {
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

        let responseData = try readLine()
        let envelope = try JSONDecoder().decode(RPCEnvelope<Result>.self, from: responseData)
        if let error = envelope.error {
            throw CoreClientError.remote(
                code: error.code,
                message: error.message,
                kind: error.data.kind,
                retryable: error.data.retryable
            )
        }
        guard let result = envelope.result else { throw CoreClientError.malformedResponse }
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
        output = nil
        errorOutput = nil
        responseBuffer.removeAll(keepingCapacity: false)
    }

    private func readLine() throws -> Data {
        while true {
            if let newline = responseBuffer.firstIndex(of: 0x0A) {
                let line = responseBuffer[..<newline]
                responseBuffer.removeSubrange(...newline)
                return Data(line)
            }
            guard let output else { throw CoreClientError.processStopped("") }
            let chunk = output.availableData
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

private extension CoreClientError {
    var isProcessStopped: Bool {
        if case .processStopped = self { return true }
        return false
    }
}

private struct HandshakeParams: Encodable {
    let protocolVersion: Int
    let appVersion: String
}

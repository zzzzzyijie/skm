import Foundation

enum CoreClientError: LocalizedError, Sendable {
    case executableMissing
    case processStopped(String)
    case malformedResponse
    case protocolMismatch(Int)
    case remote(code: Int, message: String, kind: String, retryable: Bool)

    var errorDescription: String? {
        switch self {
        case .executableMissing:
            "App 中缺少 skm-core，请重新构建或安装 SKM。"
        case let .processStopped(message):
            "SKM Core 已停止。\(message.isEmpty ? "" : "\n\(message)")"
        case .malformedResponse:
            "SKM Core 返回了无法识别的数据。"
        case let .protocolMismatch(version):
            "Core 协议版本 \(version) 与 App 不兼容。"
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

actor CoreClient {
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

    func handshake() throws -> Handshake {
        let value: Handshake = try call(
            "system.handshake",
            params: HandshakeParams(protocolVersion: 1, appVersion: "0.6.0")
        )
        guard value.protocolVersion == 1 else {
            throw CoreClientError.protocolMismatch(value.protocolVersion)
        }
        return value
    }

    func call<Params: Encodable, Result: Decodable>(_ method: String, params: Params) throws -> Result {
        try start()
        requestNumber += 1
        let request = RPCRequest(id: String(requestNumber), method: method, params: params)
        var data = try JSONEncoder().encode(request)
        data.append(0x0A)
        guard let input else { throw CoreClientError.processStopped("") }
        try input.write(contentsOf: data)

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

    func stop() {
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
                stop()
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

private struct HandshakeParams: Encodable {
    let protocolVersion: Int
    let appVersion: String
}

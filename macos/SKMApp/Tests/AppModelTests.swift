import Foundation
import XCTest
@testable import SKM

@MainActor
final class AppModelTests: XCTestCase {
    func testFirstLaunchRecognizesEmptyLibrary() async {
        let preferences = isolatedPreferences()
        let model = AppModel(
            core: StubCore(),
            preferences: preferences,
            monitorsFiles: false
        )

        await model.start()

        XCTAssertNotNil(model.handshake)
        XCTAssertNil(model.startupErrorMessage)
        XCTAssertFalse(model.hasExistingData)
        XCTAssertTrue(model.showsWelcome)
    }

    func testFirstLaunchRecognizesExistingLibrary() async {
        let preferences = isolatedPreferences()
        let core = StubCore(responses: [
            "skills.list": #"[{"id":"local/sample","name":"sample","description":"Sample","tags":[],"source":"local","location":"library","hash":"abc","path":"/tmp/sample","health":"available","effectivePath":"/tmp/sample","editable":true}]"#,
        ])
        let model = AppModel(
            core: core,
            preferences: preferences,
            monitorsFiles: false
        )

        await model.start()

        XCTAssertTrue(model.hasExistingData)
        XCTAssertEqual(model.skills.map(\.id), ["local/sample"])
        XCTAssertTrue(model.showsWelcome)
    }

    func testStartupFailureUsesBlockingState() async {
        let model = AppModel(
            core: StubCore(handshakeError: .executableMissing),
            preferences: isolatedPreferences(),
            monitorsFiles: false
        )

        await model.start()

        XCTAssertNil(model.handshake)
        XCTAssertEqual(model.startupErrorMessage, CoreClientError.executableMissing.localizedDescription)
        XCTAssertFalse(model.isLoading)
    }

    func testCompletingWelcomePersistsChoiceWithoutConfiguringAgent() async {
        let preferences = isolatedPreferences()
        let model = AppModel(
            core: StubCore(),
            preferences: preferences,
            monitorsFiles: false
        )
        await model.start()

        model.completeWelcome()

        XCTAssertFalse(model.showsWelcome)
        XCTAssertTrue(preferences.bool(forKey: "SKMHasCompletedWelcome"))
        XCTAssertTrue(model.agents.allSatisfy { !$0.configured })
    }

    func testReadOnlyMethodsAreTheOnlyAutomaticallyRetryableCalls() {
        XCTAssertTrue(CoreClient.isSafeToRetry("skills.list"))
        XCTAssertTrue(CoreClient.isSafeToRetry("system.doctor"))
        XCTAssertFalse(CoreClient.isSafeToRetry("skills.update"))
        XCTAssertFalse(CoreClient.isSafeToRetry("activations.enable"))
    }

    private func isolatedPreferences() -> UserDefaults {
        let suiteName = "SKMTests.\(UUID().uuidString)"
        let preferences = UserDefaults(suiteName: suiteName)!
        addTeardownBlock { preferences.removePersistentDomain(forName: suiteName) }
        return preferences
    }
}

private actor StubCore: CoreServing {
    private let handshakeError: CoreClientError?
    private let responses: [String: Data]

    init(handshakeError: CoreClientError? = nil, responses: [String: String] = [:]) {
        self.handshakeError = handshakeError
        var defaults: [String: String] = [
            "skills.list": "[]",
            "prompts.list": "[]",
            "agents.list": "[]",
            "activations.status": #"{"digest":"","operations":[]}"#,
            "sources.list": "[]",
            "projects.list": "[]",
            "workspace.get": #"{"configured":false}"#,
        ]
        defaults.merge(responses) { _, replacement in replacement }
        self.responses = defaults.mapValues { Data($0.utf8) }
    }

    func handshake() async throws -> Handshake {
        if let handshakeError { throw handshakeError }
        return Handshake(
            protocolVersion: 1,
            coreVersion: "0.5.1",
            schemaVersion: 2,
            promptSchemaVersion: 1,
            workspaceSchemaVersion: 1,
            capabilities: ["skills.read"]
        )
    }

    func call<Params: Encodable & Sendable, Result: Decodable & Sendable>(
        _ method: String,
        params: Params
    ) async throws -> Result {
        guard let data = responses[method] else {
            throw CoreClientError.remote(code: -32601, message: method, kind: "method_not_found", retryable: false)
        }
        return try JSONDecoder().decode(Result.self, from: data)
    }

    func stop() async {}
}

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

    func testPrimaryNavigationAndSettingsSectionsStaySeparated() {
        XCTAssertEqual(AppSection.allCases, [.skills, .prompts, .projects])
        XCTAssertEqual(
            SettingsSection.allCases,
            [.general, .fileAccess, .agents, .sources, .gitSync, .updates]
        )
    }

    func testReadOnlyMethodsAreTheOnlyAutomaticallyRetryableCalls() {
        XCTAssertTrue(CoreClient.isSafeToRetry("skills.list"))
        XCTAssertTrue(CoreClient.isSafeToRetry("system.doctor"))
        XCTAssertTrue(CoreClient.isSafeToRetry("history.diff"))
        XCTAssertTrue(CoreClient.isSafeToRetry("prompts.render"))
        XCTAssertFalse(CoreClient.isSafeToRetry("skills.update"))
        XCTAssertFalse(CoreClient.isSafeToRetry("history.rollback"))
        XCTAssertFalse(CoreClient.isSafeToRetry("projects.apply"))
        XCTAssertFalse(CoreClient.isSafeToRetry("activations.enable"))
    }

    func testSemanticVersionComparison() {
        XCTAssertTrue(AppModel.isVersion("v0.6.0", newerThan: "0.5.1"))
        XCTAssertFalse(AppModel.isVersion("0.5.1", newerThan: "0.5.1"))
        XCTAssertFalse(AppModel.isVersion("0.5.0", newerThan: "0.5.1"))
    }

    private func isolatedPreferences() -> UserDefaults {
        let suiteName = "SKMTests.\(UUID().uuidString)"
        let preferences = UserDefaults(suiteName: suiteName)!
        addTeardownBlock { preferences.removePersistentDomain(forName: suiteName) }
        return preferences
    }
}

final class CoreClientResilienceTests: XCTestCase {
    func testConcurrentReadRequestsAreSerializedAndMatchedByID() async throws {
        let script = try executableScript("""
        #!/bin/sh
        request_id=1
        while IFS= read -r request; do
          printf '{"jsonrpc":"2.0","id":"%s","result":[]}\n' "$request_id"
          request_id=$((request_id + 1))
        done
        """)
        let client = CoreClient(executableURL: script, requestTimeout: .seconds(2))
        async let first: [MutationSkill] = client.call("skills.list", params: EmptyParams())
        async let second: [MutationSkill] = client.call("prompts.list", params: EmptyParams())
        let values = try await (first, second)
        XCTAssertTrue(values.0.isEmpty)
        XCTAssertTrue(values.1.isEmpty)
        await client.stop()
    }

    func testMalformedResponseRestartsSafeReadAndIgnoresStderrNoise() async throws {
        let marker = FileManager.default.temporaryDirectory.appendingPathComponent("SKMCoreClientMarker-\(UUID().uuidString)")
        addTeardownBlock { try? FileManager.default.removeItem(at: marker) }
        let script = try executableScript("""
        #!/bin/sh
        IFS= read -r request || exit 0
        if [ ! -f '\(marker.path)' ]; then
          /usr/bin/touch '\(marker.path)'
          printf 'not-json\n'
          exit 0
        fi
        printf 'diagnostic noise on stderr\n' >&2
        printf '{"jsonrpc":"2.0","id":"2","result":[]}\n'
        """)
        let client = CoreClient(executableURL: script, requestTimeout: .seconds(2))
        let result: [MutationSkill] = try await client.call("skills.list", params: EmptyParams())
        XCTAssertTrue(result.isEmpty)
        await client.stop()
    }

    func testCoreCrashRestartsOnlySafeRead() async throws {
        let marker = FileManager.default.temporaryDirectory.appendingPathComponent("SKMCoreCrashMarker-\(UUID().uuidString)")
        addTeardownBlock { try? FileManager.default.removeItem(at: marker) }
        let script = try executableScript("""
        #!/bin/sh
        IFS= read -r request || exit 0
        if [ ! -f '\(marker.path)' ]; then
          /usr/bin/touch '\(marker.path)'
          printf 'simulated crash\n' >&2
          exit 1
        fi
        printf '{"jsonrpc":"2.0","id":"2","result":[]}\n'
        """)
        let client = CoreClient(executableURL: script, requestTimeout: .seconds(2))
        let result: [MutationSkill] = try await client.call("skills.list", params: EmptyParams())
        XCTAssertTrue(result.isEmpty)
        await client.stop()
    }

    func testWriteTimeoutStopsCoreWithoutAutomaticReplay() async throws {
        let script = try executableScript("""
        #!/bin/sh
        IFS= read -r request || exit 0
        while :; do :; done
        """)
        let client = CoreClient(executableURL: script, requestTimeout: .milliseconds(150))
        do {
            let _: MutationSkill = try await client.call("skills.update", params: EmptyParams())
            XCTFail("expected timeout")
        } catch let error as CoreClientError {
            XCTAssertEqual(error.kind, "timeout")
        }
        await client.stop()
    }

    private func executableScript(_ contents: String) throws -> URL {
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("skm-core-stub-\(UUID().uuidString).sh")
        try contents.write(to: url, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o700], ofItemAtPath: url.path)
        addTeardownBlock { try? FileManager.default.removeItem(at: url) }
        return url
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
            "system.doctor": "[]",
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

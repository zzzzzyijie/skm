import Foundation
import XCTest
@testable import SKM

@MainActor
final class AppModelTests: XCTestCase {
    func testPromptFiltersIntersectSearchAndKeepSelectionVisible() {
        let model = AppModel(core: StubCore(), preferences: isolatedPreferences(), monitorsFiles: false)
        let first = prompt("review-2", tags: ["Dev", "review"])
        let second = prompt("review-10", tags: ["Dev"])
        let untagged = prompt("notes", tags: [])
        model.prompts = [second, untagged, first]
        model.selectedPromptID = untagged.id
        model.promptBrowser.filter = .tag("dev")
        model.reconcilePromptSelection()
        XCTAssertEqual(model.visiblePrompts.map(\.name), ["review-2", "review-10"])
        XCTAssertEqual(model.selectedPromptID, first.id)
        model.promptBrowser.query = "review-10"
        model.reconcilePromptSelection()
        XCTAssertEqual(model.selectedPromptID, second.id)
        model.promptBrowser.query = "missing"
        model.reconcilePromptSelection()
        XCTAssertNil(model.selectedPromptID)
        model.promptBrowser.query = ""
        model.promptBrowser.filter = .untagged
        model.reconcilePromptSelection()
        XCTAssertEqual(model.visiblePrompts.map(\.id), [untagged.id])
        model.promptBrowser.filter = .tag("deleted")
        model.reconcilePromptSelection()
        XCTAssertEqual(model.promptBrowser.filter, .all)
    }

    func testPromptCustomEmptyCategorySurvivesReconciliation() {
        let model = AppModel(core: StubCore(), preferences: isolatedPreferences(), monitorsFiles: false)
        model.registerCustomTag("Empty", for: .prompts)
        model.promptBrowser.filter = .tag("Empty")
        model.reconcilePromptSelection()
        XCTAssertEqual(model.promptBrowser.filter, .tag("Empty"))
        XCTAssertNil(model.selectedPromptID)
    }

    func testCreatingAndImportingPromptRevealReturnedIdentityAfterReload() async throws {
        let old = prompt("old", tags: ["existing"])
        let created = prompt("new", tags: [])
        let encode = JSONEncoder()
        let core = StubCore(responses: [
            "prompts.list": String(decoding: try encode.encode([old, created]), as: UTF8.self),
            "prompts.create": String(decoding: try encode.encode(created), as: UTF8.self),
        ])
        let model = AppModel(core: core, preferences: isolatedPreferences(), monitorsFiles: false)
        model.prompts = [old]
        model.selectedPromptID = old.id
        model.promptBrowser.filter = .tag("existing")
        model.promptBrowser.query = "old"
        let saved = await model.savePrompt(id: nil, name: "new", description: "New", body: "  Raw\n", tags: [], variables: [], baseHash: nil)
        XCTAssertTrue(saved)
        XCTAssertEqual(model.selectedPromptID, created.id)
        XCTAssertEqual(model.promptBrowser.filter, .all)
        XCTAssertEqual(model.promptBrowser.query, "")
        model.selectedPromptID = old.id
        model.promptBrowser.query = "old"
        await model.importPrompt(content: "fixture")
        XCTAssertEqual(model.selectedPromptID, created.id)
        XCTAssertEqual(model.visiblePrompts.count, 2)
    }

    func testFailedPromptSavePreservesSearchAndSelection() async {
        let model = AppModel(core: StubCore(), preferences: isolatedPreferences(), monitorsFiles: false)
        let item = prompt("old", tags: ["dev"])
        model.prompts = [item]
        model.selectedPromptID = item.id
        model.promptBrowser.query = "old"
        let saved = await model.savePrompt(id: nil, name: "new", description: "New", body: "Body", tags: [], variables: [], baseHash: nil)
        XCTAssertFalse(saved)
        XCTAssertEqual(model.selectedPromptID, item.id)
        XCTAssertEqual(model.promptBrowser.query, "old")
    }

    func testPromptDraftValidationAndCopyNameBoundaries() {
        var draft = PromptDraft(details: nil, initialTags: ["dev"])
        XCTAssertEqual(draft.tags, ["dev"])
        draft.description = "Description"
        draft.body = " Body\n"
        for name in ["x", "code-review", String(repeating: "a", count: 64)] {
            draft.name = name
            XCTAssertTrue(draft.isValid, name)
        }
        for name in ["", "-a", "a-", "A", "a_b", "a\n", String(repeating: "a", count: 65)] {
            draft.name = name
            XCTAssertFalse(draft.isValid, name)
        }
        draft.name = String(repeating: "a", count: 64)
        let firstCopy = draft.copyName(existing: [])
        let secondCopy = draft.copyName(existing: [prompt(firstCopy, tags: [])])
        XCTAssertNotEqual(firstCopy, secondCopy)
        XCTAssertLessThanOrEqual(secondCopy.count, 64)
        draft.description = String(repeating: "中", count: 342)
        XCTAssertFalse(draft.validDescription)
        XCTAssertTrue(PromptDraft.validTagName("开发-ＵＩ"))
        for tag in ["", "two words", "a-", String(repeating: "中", count: 33)] {
            XCTAssertFalse(PromptDraft.validTagName(tag), tag)
        }
    }

    func testMarkdownPreservesCodeAndRendersListsAndQuotes() {
        let blocks = MarkdownBlock.parse("## Review\n\n- First\n  2. Nested\n> Keep context\n> Second line\n\n````swift\n- literal\n```\n````\nAfter")
        XCTAssertEqual(blocks, [
            .heading(2, "Review"),
            .listItem(marker: "•", text: "First", depth: 0),
            .listItem(marker: "2.", text: "Nested", depth: 1),
            .quote("Keep context\nSecond line"),
            .code("- literal\n```"),
            .paragraph("After"),
        ])
    }

    private func prompt(_ name: String, tags: [String]) -> PromptSummary {
        PromptSummary(id: "local/\(name)", name: name, description: "Description", tags: tags, source: "local", hash: name, path: "/tmp/\(name)", variables: [])
    }

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

    func testAppPreferencesPersistLanguageAndAppearance() {
        defer { AppLocalization.configure(language: .system) }
        let defaults = isolatedPreferences()
        let preferences = AppPreferences(defaults: defaults)

        XCTAssertEqual(preferences.language, .system)
        XCTAssertEqual(preferences.appearance, .system)

        preferences.language = .english
        preferences.appearance = .dark

        let restored = AppPreferences(defaults: defaults)
        XCTAssertEqual(restored.language, .english)
        XCTAssertEqual(restored.appearance, .dark)
    }

    func testAppLocalizationSwitchesLanguagesUsingCachedContext() {
        defer { AppLocalization.configure(language: .system) }

        AppLocalization.configure(language: .english)
        XCTAssertEqual(AppLocalization.string("通用"), "General")

        AppLocalization.configure(language: .simplifiedChinese)
        XCTAssertEqual(AppLocalization.string("通用"), "通用")
    }

    func testRegisterCustomTagsKeepsSkillAndPromptPoolsIndependent() {
        let preferences = isolatedPreferences()
        let model = AppModel(core: StubCore(), preferences: preferences, monitorsFiles: false)

        model.registerCustomTag("  AI  ", for: .skills)
        model.registerCustomTag("ai", for: .skills)
        model.registerCustomTag("iOS", for: .prompts)

        XCTAssertEqual(model.customTags(for: .skills), ["AI"])
        XCTAssertEqual(model.customTags(for: .prompts), ["iOS"])
        XCTAssertEqual(preferences.stringArray(forKey: "skm.custom.tags.skills"), ["AI"])
        XCTAssertEqual(preferences.stringArray(forKey: "skm.custom.tags.prompts"), ["iOS"])
    }

    func testUnregisterCustomTagOnlyRemovesItFromRequestedPool() {
        let preferences = isolatedPreferences()
        let model = AppModel(core: StubCore(), preferences: preferences, monitorsFiles: false)
        model.registerCustomTag("shared", for: .skills)
        model.registerCustomTag("shared", for: .prompts)

        model.unregisterCustomTag("shared", from: .prompts)

        XCTAssertEqual(model.customTags(for: .skills), ["shared"])
        XCTAssertTrue(model.customTags(for: .prompts).isEmpty)
        XCTAssertEqual(preferences.stringArray(forKey: "skm.custom.tags.skills"), ["shared"])
        XCTAssertEqual(preferences.stringArray(forKey: "skm.custom.tags.prompts"), [])
    }

    func testLegacyCustomTagsMigrateIntoIndependentPools() async {
        let preferences = isolatedPreferences()
        preferences.set(["skill-only", "prompt-only", "shared", "unused"], forKey: "skm.custom.tags")
        let core = StubCore(responses: [
            "skills.list": #"[{"id":"local/sample","name":"sample","description":"Sample","tags":["skill-only","shared"],"source":"local","location":"library","hash":"abc","path":"/tmp/sample","health":"available","effectivePath":"/tmp/sample","editable":true}]"#,
            "prompts.list": #"[{"id":"local/prompt","name":"prompt","description":"Prompt","tags":["prompt-only","shared"],"source":"local","hash":"def","path":"/tmp/prompt","variables":[]}]"#,
        ])
        let model = AppModel(core: core, preferences: preferences, monitorsFiles: false)

        await model.start()

        XCTAssertEqual(model.customTags(for: .skills), ["shared", "skill-only", "unused"])
        XCTAssertEqual(model.customTags(for: .prompts), ["prompt-only", "shared", "unused"])
        XCTAssertNil(preferences.stringArray(forKey: "skm.custom.tags"))
    }

    func testRenameSkillTagUsesTagOnlyEndpointForGitManagedSkill() async {
        let core = StubCore(responses: [
            "skills.list": #"[{"id":"team/sample","name":"sample","description":"Sample","tags":["ai"],"source":"team","location":"library","hash":"abc","path":"/tmp/sample","health":"available","effectivePath":"/tmp/sample","editable":false,"editReason":"this Skill is managed by a Git source"}]"#,
            "skills.tags.replace": #"{"id":"team/sample","tags":["AI"]}"#,
        ])
        let model = AppModel(core: core, preferences: isolatedPreferences(), monitorsFiles: false)
        await model.start()

        await model.renameSkillTag(from: "ai", to: "AI")

        let methods = await core.calledMethods()
        XCTAssertTrue(methods.contains("skills.tags.replace"))
        XCTAssertFalse(methods.contains("skills.update"))
        XCTAssertNil(model.errorMessage)
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

    func testProjectDetailsOnlyPublishForTheSelectedProject() async {
        let core = StubCore(responses: [
            "projects.get": #"{"project":{"id":"first","path":"/tmp/first"},"exists":true,"activations":[],"manifest":{"version":1,"skills":[]},"scan":{"scannedAt":"","skillCount":0,"agentCounts":{},"agents":[],"skills":[]},"plan":{"digest":"","operations":[]}}"#,
        ])
        let model = AppModel(core: core, preferences: isolatedPreferences(), monitorsFiles: false)
        model.selectedProjectID = "second"
        await model.loadProjectDetails("first")
        XCTAssertNil(model.projectDetails)

        model.selectedProjectID = "first"
        await model.loadProjectDetails("first")
        XCTAssertEqual(model.projectDetails?.project.id, "first")
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
    private var methods: [String] = []

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
        methods.append(method)
        guard let data = responses[method] else {
            throw CoreClientError.remote(code: -32601, message: method, kind: "method_not_found", retryable: false)
        }
        return try JSONDecoder().decode(Result.self, from: data)
    }

    func calledMethods() -> [String] { methods }

    func stop() async {}
}

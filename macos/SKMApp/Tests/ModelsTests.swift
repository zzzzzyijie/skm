import XCTest
@testable import SKM

final class ModelsTests: XCTestCase {
    func testHandshakeDecoding() throws {
        let data = Data(#"{"protocolVersion":1,"coreVersion":"0.5.1","schemaVersion":2,"promptSchemaVersion":1,"workspaceSchemaVersion":1,"capabilities":["skills.read"]}"#.utf8)
        let value = try JSONDecoder().decode(Handshake.self, from: data)
        XCTAssertEqual(value.protocolVersion, 1)
        XCTAssertEqual(value.coreVersion, "0.5.1")
        XCTAssertEqual(value.schemaVersion, 2)
    }

    func testSkillSummaryDecoding() throws {
        let data = Data(#"{"id":"sample@local","name":"sample","description":"Sample","tags":["general"],"source":"local","location":"library","hash":"abc","path":"/tmp/sample","health":"available","effectivePath":"/tmp/sample","editable":true}"#.utf8)
        let value = try JSONDecoder().decode(SkillSummary.self, from: data)
        XCTAssertEqual(value.name, "sample")
        XCTAssertTrue(value.editable)
        XCTAssertEqual(value.tags, ["general"])
    }

    func testTagParsingNormalizesInput() {
        XCTAssertEqual(parseTags("general, mac, general"), ["general", "mac"])
    }

    func testTagFilterBuildsUniqueOptionsAndMatchesSelection() {
        XCTAssertEqual(
            availableFilterTags(from: [["review", "general"], ["general", "swift"]]),
            ["general", "review", "swift"]
        )
        XCTAssertTrue(matchesSelectedTag(["general", "swift"], selectedTag: nil))
        XCTAssertTrue(matchesSelectedTag(["general", "swift"], selectedTag: "swift"))
        XCTAssertFalse(matchesSelectedTag(["general"], selectedTag: "swift"))
    }

    func testProjectAccessStatusDecodesCoreStateAndLegacyFallback() throws {
        let denied = try JSONDecoder().decode(
            ProjectModel.self,
            from: Data(#"{"id":"demo","path":"/tmp/demo","exists":true,"access":"permission-denied","accessMessage":"permission denied","activationCount":0,"skillCount":0,"agentCounts":{}}"#.utf8)
        )
        XCTAssertEqual(ProjectAccessStatus(project: denied), .permissionDenied)

        let legacyMissing = try JSONDecoder().decode(
            ProjectModel.self,
            from: Data(#"{"id":"old","path":"/tmp/old","exists":false,"activationCount":0,"skillCount":0,"agentCounts":{}}"#.utf8)
        )
        XCTAssertEqual(ProjectAccessStatus(project: legacyMissing), .missing)
    }

    func testPhaseThreeResponsesDecode() throws {
        let render = try JSONDecoder().decode(
            PromptRenderResponse.self,
            from: Data(#"{"content":"Hello Codex","missingVariables":[]}"#.utf8)
        )
        XCTAssertEqual(render.content, "Hello Codex")

        let history = try JSONDecoder().decode(
            HistoryEntryModel.self,
            from: Data(#"{"id":"current","itemId":"sample","kind":"skill","hash":"abc","createdAt":"2026-08-28T00:00:00Z","reason":"current","current":true}"#.utf8)
        )
        XCTAssertTrue(history.current == true)

        let project = try JSONDecoder().decode(
            ProjectAdvancedResponse.self,
            from: Data(#"{"project":{"id":"demo","path":"/tmp/demo"},"manifest":{"version":1,"skills":[],"dependencies":[]},"plan":{"digest":"","operations":[]},"satisfiedByUser":[],"applied":true}"#.utf8)
        )
        XCTAssertTrue(project.applied)
    }
}

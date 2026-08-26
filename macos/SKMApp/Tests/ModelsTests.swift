import XCTest
@testable import SKM

final class ModelsTests: XCTestCase {
    func testHandshakeDecoding() throws {
        let data = Data(#"{"protocolVersion":1,"coreVersion":"0.6.0","schemaVersion":2,"promptSchemaVersion":1,"workspaceSchemaVersion":1,"capabilities":["skills.read"]}"#.utf8)
        let value = try JSONDecoder().decode(Handshake.self, from: data)
        XCTAssertEqual(value.protocolVersion, 1)
        XCTAssertEqual(value.coreVersion, "0.6.0")
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
}

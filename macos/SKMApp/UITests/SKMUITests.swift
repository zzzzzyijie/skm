import XCTest

final class SKMUITests: XCTestCase {
    private var temporaryRoot: URL!

    override func setUpWithError() throws {
        continueAfterFailure = false
        temporaryRoot = FileManager.default.temporaryDirectory
            .appendingPathComponent("SKMUITests-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: temporaryRoot, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(
            at: temporaryRoot.appendingPathComponent("user", isDirectory: true),
            withIntermediateDirectories: true
        )
        try FileManager.default.createDirectory(
            at: temporaryRoot.appendingPathComponent("project", isDirectory: true),
            withIntermediateDirectories: true
        )
    }

    override func tearDownWithError() throws {
        if let temporaryRoot {
            try? FileManager.default.removeItem(at: temporaryRoot)
        }
    }

    @MainActor
    func testChineseEmptyLibraryAndNewItemKeyboardCommand() {
        let app = application(language: "zh-Hans")
        app.launch()
        defer { app.terminate() }

        XCTAssertTrue(app.staticTexts["还没有 Skill"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.buttons["添加 Skill"].exists)

        app.typeKey("n", modifierFlags: .command)

        XCTAssertTrue(app.staticTexts["添加 Skill"].waitForExistence(timeout: 3))
        XCTAssertTrue(app.buttons["导入"].exists)
        XCTAssertTrue(app.buttons["取消"].exists)
    }

    @MainActor
    func testEnglishEmptyLibraryAndNewItemKeyboardCommand() {
        let app = application(language: "en")
        app.launch()
        defer { app.terminate() }

        XCTAssertTrue(app.staticTexts["No Skills Yet"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.buttons["Add Skill"].exists)

        app.typeKey("n", modifierFlags: .command)

        XCTAssertTrue(app.staticTexts["Add Skill"].waitForExistence(timeout: 3))
        XCTAssertTrue(app.buttons["Import"].exists)
        XCTAssertTrue(app.buttons["Cancel"].exists)
    }

    @MainActor
    private func application(language: String) -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments = ["-AppleLanguages", "(\(language))"]
        app.launchEnvironment = [
            "SKM_HOME": temporaryRoot.appendingPathComponent("state").path,
            "SKM_USER_HOME": temporaryRoot.appendingPathComponent("user").path,
            "SKM_PROJECT": temporaryRoot.appendingPathComponent("project").path,
            "SKM_SKIP_WELCOME": "1",
        ]
        return app
    }
}

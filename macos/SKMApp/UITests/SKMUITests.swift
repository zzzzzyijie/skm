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
    func testChineseSkillAgentAndPromptWriteFlow() throws {
        let skillDirectory = temporaryRoot.appendingPathComponent("fixture-skill", isDirectory: true)
        try FileManager.default.createDirectory(at: skillDirectory, withIntermediateDirectories: true)
        try """
        ---
        name: ui-smoke-skill
        description: UI smoke Skill
        ---

        Use this fixture only in the isolated UI test.
        """.write(to: skillDirectory.appendingPathComponent("SKILL.md"), atomically: true, encoding: .utf8)

        let app = application(language: "zh-Hans")
        app.launch()
        defer { app.terminate() }

        XCTAssertTrue(app.staticTexts["还没有 Skill"].waitForExistence(timeout: 10))
        app.typeKey("n", modifierFlags: .command)
        let pathField = app.textFields["Skill 目录或 ZIP"]
        XCTAssertTrue(pathField.waitForExistence(timeout: 3))
        pathField.click()
        pathField.typeText(skillDirectory.path)
        app.buttons["导入"].click()
        XCTAssertTrue(app.staticTexts["ui-smoke-skill"].waitForExistence(timeout: 8))

        app.descendants(matching: .any)["navigation-agents"].click()
        let managementToggle = app.checkBoxes["允许 SKM 向此 Agent 部署 Skill"]
        XCTAssertTrue(managementToggle.waitForExistence(timeout: 5))
        if managementToggle.value as? Int == 0 { managementToggle.click() }

        app.descendants(matching: .any)["navigation-prompts"].click()
        app.typeKey("n", modifierFlags: .command)
        XCTAssertTrue(app.staticTexts["新建 Prompt"].waitForExistence(timeout: 3))
        let nameField = app.textFields["prompt-name-field"]
        XCTAssertTrue(nameField.waitForExistence(timeout: 3))
        nameField.click()
        nameField.typeText("ui-smoke-prompt")
        let descriptionField = app.textFields["prompt-description-field"]
        descriptionField.click()
        descriptionField.typeText("UI smoke Prompt")
        let editor = app.textViews["prompt-body-editor"]
        editor.click()
        editor.typeText("Review this isolated fixture.")
        app.buttons["保存"].click()
        XCTAssertTrue(app.staticTexts["ui-smoke-prompt"].waitForExistence(timeout: 8))
    }

    @MainActor
    func testChineseProjectRegistrationAndWorkspacePreview() throws {
        let registeredProject = temporaryRoot.appendingPathComponent("registered-project", isDirectory: true)
        let workspaceRemote = temporaryRoot.appendingPathComponent("workspace.git", isDirectory: true)
        try FileManager.default.createDirectory(at: registeredProject, withIntermediateDirectories: true)
        try run(
            "/Applications/Xcode.app/Contents/Developer/usr/bin/git",
            arguments: ["init", "--bare", workspaceRemote.path]
        )

        let app = application(language: "zh-Hans")
        app.launch()
        defer { app.terminate() }
        XCTAssertTrue(app.staticTexts["还没有 Skill"].waitForExistence(timeout: 10))

        let projectsNavigation = app.descendants(matching: .any)["navigation-projects"]
        XCTAssertTrue(projectsNavigation.waitForExistence(timeout: 5))
        projectsNavigation.click()
        app.typeKey("o", modifierFlags: .command)
        let projectField = app.textFields["project-path-field"]
        XCTAssertTrue(projectField.waitForExistence(timeout: 5))
        projectField.click()
        projectField.typeText(registeredProject.path)
        app.buttons["添加"].click()
        XCTAssertTrue(app.staticTexts["registered-project"].waitForExistence(timeout: 8))

        let workspaceNavigation = app.descendants(matching: .any)["navigation-workspace"]
        XCTAssertTrue(workspaceNavigation.waitForExistence(timeout: 5))
        workspaceNavigation.click()
        let urlField = app.textFields["workspace-url-field"]
        XCTAssertTrue(urlField.waitForExistence(timeout: 5))
        urlField.click()
        urlField.typeText(workspaceRemote.path)
        app.buttons["workspace-configure-button"].click()
        XCTAssertTrue(app.buttons["workspace-preview-button"].waitForExistence(timeout: 8))
        app.buttons["workspace-preview-button"].click()
        XCTAssertTrue(app.staticTexts["同步预览"].waitForExistence(timeout: 8))
        XCTAssertTrue(app.staticTexts["没有需要同步的更改"].exists)
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

    private func run(_ executable: String, arguments: [String]) throws {
        let process = Process()
        let output = Pipe()
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = arguments
        process.standardOutput = output
        process.standardError = output
        try process.run()
        process.waitUntilExit()
        let diagnostics = String(
            data: output.fileHandleForReading.readDataToEndOfFile(),
            encoding: .utf8
        ) ?? ""
        XCTAssertEqual(process.terminationStatus, 0, diagnostics)
    }
}

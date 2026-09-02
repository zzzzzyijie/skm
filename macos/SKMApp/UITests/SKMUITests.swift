import AppKit
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
        app.buttons["Cancel"].click()

        app.descendants(matching: .any)["open-settings"].click()
        XCTAssertTrue(app.staticTexts["Agent Management"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Skill Sources"].exists)
        XCTAssertTrue(app.staticTexts["Git Sync"].exists)
        XCTAssertTrue(app.staticTexts["Software Updates"].exists)
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
        paste(skillDirectory.path, into: pathField)
        app.buttons["导入"].click()
        XCTAssertTrue(app.staticTexts["ui-smoke-skill"].waitForExistence(timeout: 8))

        app.descendants(matching: .any)["open-settings"].click()
        let agentsSettings = app.descendants(matching: .any)["settings-agents"]
        XCTAssertTrue(agentsSettings.waitForExistence(timeout: 5))
        agentsSettings.click()
        let managementToggle = app.checkBoxes["agent-management-codex"]
        XCTAssertTrue(managementToggle.waitForExistence(timeout: 5))
        if managementToggle.value as? Int == 0 { managementToggle.click() }
        app.typeKey("w", modifierFlags: .command)

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
        let renderButton = app.buttons["填写变量"]
        XCTAssertTrue(renderButton.waitForExistence(timeout: 5))
        renderButton.click()
        XCTAssertTrue(app.staticTexts["没有变量"].waitForExistence(timeout: 5))
        app.buttons["关闭"].click()
        XCTAssertTrue(app.buttons["历史"].waitForExistence(timeout: 5))
    }

    @MainActor
    func testChineseTagGroupsRenderAsSeparateListRows() throws {
        let skillDirectory = temporaryRoot.appendingPathComponent("tag-layout-skill", isDirectory: true)
        try FileManager.default.createDirectory(at: skillDirectory, withIntermediateDirectories: true)
        try """
        ---
        name: tag-layout-skill
        description: A deliberately long description used to verify that group headers and Skill rows keep independent vertical space in a narrow list.
        ---

        Verify the grouped list layout.
        """.write(to: skillDirectory.appendingPathComponent("SKILL.md"), atomically: true, encoding: .utf8)

        let app = application(language: "zh-Hans")
        app.launch()
        defer { app.terminate() }

        XCTAssertTrue(app.staticTexts["还没有 Skill"].waitForExistence(timeout: 10))
        app.typeKey("n", modifierFlags: .command)

        let pathField = app.textFields["Skill 目录或 ZIP"]
        XCTAssertTrue(pathField.waitForExistence(timeout: 3))
        pathField.click()
        paste(skillDirectory.path, into: pathField)

        let tagsField = app.textFields["标签，以逗号分隔（可选）"]
        tagsField.click()
        paste("开发, 这是一个用于验证窄窗口截断表现的超长标签名称", into: tagsField)
        app.buttons["导入"].click()

        let allHeader = app.descendants(matching: .any)["skills-group-all"]
        let skillRow = app.descendants(matching: .any)["skill-row-local/tag-layout-skill"].firstMatch
        let longTagHeader = app.descendants(matching: .any)["skills-group-这是一个用于验证窄窗口截断表现的超长标签名称"]

        XCTAssertTrue(allHeader.waitForExistence(timeout: 8))
        XCTAssertTrue(skillRow.waitForExistence(timeout: 8))
        XCTAssertTrue(longTagHeader.waitForExistence(timeout: 8))
        XCTAssertLessThanOrEqual(allHeader.frame.maxY, skillRow.frame.minY)
        XCTAssertLessThanOrEqual(skillRow.frame.maxY, longTagHeader.frame.minY)

        allHeader.click()
        XCTAssertFalse(skillRow.waitForExistence(timeout: 1))

        longTagHeader.click()
        XCTAssertTrue(skillRow.waitForExistence(timeout: 3))
        XCTAssertLessThanOrEqual(longTagHeader.frame.maxY, skillRow.frame.minY)
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
        paste(registeredProject.path, into: projectField)
        app.buttons["添加"].click()
        XCTAssertTrue(app.staticTexts["registered-project"].waitForExistence(timeout: 8))

        let settingsEntry = app.descendants(matching: .any)["open-settings"]
        XCTAssertTrue(settingsEntry.waitForExistence(timeout: 5))
        settingsEntry.click()
        let gitSyncSettings = app.descendants(matching: .any)["settings-gitSync"]
        XCTAssertTrue(gitSyncSettings.waitForExistence(timeout: 5))
        gitSyncSettings.click()
        let urlField = app.textFields["workspace-url-field"]
        XCTAssertTrue(urlField.waitForExistence(timeout: 5))
        urlField.click()
        paste(workspaceRemote.path, into: urlField)
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

    @MainActor
    private func paste(_ value: String, into element: XCUIElement) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(value, forType: .string)
        element.typeKey("v", modifierFlags: .command)
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

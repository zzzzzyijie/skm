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
            let suite = "SKMUITests.\(temporaryRoot.lastPathComponent)"
            UserDefaults(suiteName: suite)?.removePersistentDomain(forName: suite)
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
    func testSettingsSidebarUpdatesImmediatelyWhenLanguageChanges() {
        let app = application(language: "zh-Hans")
        app.launch()
        defer { app.terminate() }

        XCTAssertTrue(app.descendants(matching: .any)["open-settings"].waitForExistence(timeout: 10))
        app.descendants(matching: .any)["open-settings"].click()

        let languagePicker = app.popUpButtons["settings-language"]
        XCTAssertTrue(languagePicker.waitForExistence(timeout: 5))
        languagePicker.click()
        app.menuItems["English"].click()

        assertSettingsSidebar(
            app,
            labels: ["General", "Permission Access", "Agent Management", "Skill Sources", "Git Sync", "Software Updates"]
        )
        XCTAssertTrue(app.staticTexts["Your skills, prompts, and projects, together."].waitForExistence(timeout: 3))
        XCTAssertTrue(app.windows["General"].exists)

        app.popUpButtons["settings-language"].click()
        app.menuItems["Simplified Chinese"].click()

        assertSettingsSidebar(
            app,
            labels: ["通用", "权限访问", "Agent 管理", "技能来源", "Git 同步", "软件更新"]
        )
        XCTAssertTrue(app.staticTexts["你的 Skills、Prompts 与项目，一处管理。"].waitForExistence(timeout: 3))
        XCTAssertTrue(app.windows["通用"].exists)
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

        // Search is keyboard-first and Escape restores the collection without losing data.
        app.typeKey("f", modifierFlags: .command)
        app.typeText("no-matching-skill")
        let clearSearch = app.buttons["清除搜索"]
        XCTAssertTrue(clearSearch.waitForExistence(timeout: 3))
        XCTAssertFalse(app.descendants(matching: .any)["skill-row-local/ui-smoke-skill"].exists)
        app.typeKey(.escape, modifierFlags: [])
        XCTAssertTrue(app.descendants(matching: .any)["skill-row-local/ui-smoke-skill"].waitForExistence(timeout: 3))
        XCTAssertFalse(clearSearch.exists)

        app.descendants(matching: .any)["skill-row-local/ui-smoke-skill"].click()
        let collectionScreenshot = XCTAttachment(screenshot: app.screenshot())
        collectionScreenshot.name = "Skills — native layout"
        collectionScreenshot.lifetime = .keepAlways
        add(collectionScreenshot)

        app.descendants(matching: .any)["open-settings"].click()
        let agentsSettings = app.descendants(matching: .any)["settings-agents"]
        XCTAssertTrue(agentsSettings.waitForExistence(timeout: 5))
        agentsSettings.click()
        XCTAssertTrue(app.staticTexts["Agents"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["其他 Agent"].exists)
        let managementToggle = app.descendants(matching: .any)["agent-management-codex"]
        XCTAssertTrue(managementToggle.waitForExistence(timeout: 5))
        let toggleValue = String(describing: managementToggle.value ?? "").lowercased()
        let toggleIsOn = toggleValue == "1" || toggleValue == "on" || toggleValue == "true"
        if !toggleIsOn { managementToggle.click() }
        app.typeKey("w", modifierFlags: .command)

        app.descendants(matching: .any)["navigation-prompts"].click()
        app.typeKey("n", modifierFlags: .command)
        XCTAssertTrue(app.staticTexts["新建 Prompt"].waitForExistence(timeout: 3))
        let nameField = app.textFields["prompt-name-field"]
        XCTAssertTrue(nameField.waitForExistence(timeout: 3))
        nameField.click()
        paste("ui-smoke-prompt", into: nameField)
        let descriptionField = app.textFields["prompt-description-field"]
        descriptionField.click()
        paste("UI smoke Prompt", into: descriptionField)
        let editor = app.textViews["prompt-body-editor"]
        editor.click()
        paste("Review this isolated fixture.", into: editor)
        app.buttons["保存"].click()
        XCTAssertTrue(app.staticTexts["ui-smoke-prompt"].waitForExistence(timeout: 8))
        XCTAssertTrue(app.buttons["快速查看"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["编辑"].exists)
        XCTAssertTrue(app.buttons["导出"].exists)
    }

    @MainActor
    func testChineseTagGroupsRenderAsSeparateListRows() throws {
        let tagSuffix = String(UUID().uuidString.prefix(8)).lowercased()
        let shortTag = "开发-\(tagSuffix)"
        let longTag = "这是一个用于验证窄窗口截断表现的超长标签名称-\(tagSuffix)"
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

        let tagsField = app.textFields["add-skill-tags-new-field"]
        tagsField.click()
        paste(shortTag, into: tagsField)
        app.buttons["add-skill-tags-add-button"].click()
        tagsField.click()
        paste(longTag, into: tagsField)
        app.buttons["add-skill-tags-add-button"].click()
        app.buttons["导入"].click()

        let allHeader = app.descendants(matching: .any)["skills-group-all"]
        let skillRow = app.descendants(matching: .any)["skill-row-local/tag-layout-skill"].firstMatch
        let longTagHeader = app.descendants(matching: .any)["skills-group-\(longTag)"]

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
        let applications = try FileManager.default.contentsOfDirectory(
            at: URL(fileURLWithPath: "/Applications", isDirectory: true),
            includingPropertiesForKeys: nil
        )
        let gitExecutable = try XCTUnwrap(applications.lazy
            .filter { $0.lastPathComponent.hasPrefix("Xcode") && $0.pathExtension == "app" }
            .map { $0.appendingPathComponent("Contents/Developer/usr/bin/git").path }
            .first(where: FileManager.default.fileExists(atPath:)))
        try run(
            gitExecutable,
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
        let importButton = app.buttons["从我的 Skill 里导入"].firstMatch
        XCTAssertTrue(importButton.waitForExistence(timeout: 8))
        XCTAssertFalse(app.staticTexts["项目清单"].exists)
        importButton.click()
        XCTAssertTrue(app.staticTexts["我的 Skill 为空"].waitForExistence(timeout: 5))
        app.buttons["取消"].click()

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
    func testPromptDraftProtectionAndVariablePreview() {
        let app = application(language: "zh-Hans")
        app.launch()
        defer { app.terminate() }
        XCTAssertTrue(app.staticTexts["还没有 Skill"].waitForExistence(timeout: 10))
        app.typeKey("2", modifierFlags: .command)
        app.typeKey("n", modifierFlags: .command)
        let name = app.textFields["prompt-name-field"]
        XCTAssertTrue(name.waitForExistence(timeout: 3))
        name.click()
        paste("draft-protection", into: name)
        let description = app.textFields["prompt-description-field"]
        description.click()
        paste("Reusable review instructions", into: description)
        let editor = app.textViews["prompt-body-editor"]
        editor.click()
        paste("# Review\n\nA reusable review prompt.", into: editor)
        capture(app, name: "prompt-editor")

        app.buttons["取消"].click()
        XCTAssertTrue(app.buttons["继续编辑"].waitForExistence(timeout: 3))
        capture(app, name: "discard-confirmation")
        app.windows.buttons["继续编辑"].firstMatch.click()
        XCTAssertEqual(name.value as? String, "draft-protection")
        app.typeKey("s", modifierFlags: .command)
        XCTAssertTrue(app.staticTexts["draft-protection"].waitForExistence(timeout: 8))
        let render = app.buttons["填写变量"]
        XCTAssertTrue(render.waitForExistence(timeout: 5))
        capture(app, name: "prompt-detail")
        render.click()
        XCTAssertTrue(app.staticTexts["没有变量"].waitForExistence(timeout: 5))
        let copy = app.sheets.buttons["复制"]
        XCTAssertTrue(copy.waitForExistence(timeout: 5))
        XCTAssertTrue(copy.isEnabled)
        capture(app, name: "prompt-render")
        app.typeKey(.escape, modifierFlags: [])
        XCTAssertTrue(render.waitForExistence(timeout: 3))
    }

    @MainActor
    func testSettingsAndSheetsInBothAppearances() {
        for appearance in ["Aqua", "DarkAqua"] {
            let app = application(language: appearance == "Aqua" ? "zh-Hans" : "en")
            app.launchEnvironment["SKM_TEST_APPEARANCE"] = appearance
            app.launch()
            XCTAssertTrue(app.descendants(matching: .any)["navigation-skills"].waitForExistence(timeout: 10))
            capture(app, name: "\(appearance)-library")
            app.typeKey("n", modifierFlags: .command)
            XCTAssertTrue(app.sheets.firstMatch.waitForExistence(timeout: 3))
            capture(app, name: "\(appearance)-add-skill")
            app.typeKey(.escape, modifierFlags: [])
            app.descendants(matching: .any)["skills-manage-tags-button"].click()
            XCTAssertTrue(app.sheets.firstMatch.waitForExistence(timeout: 3))
            capture(app, name: "\(appearance)-tags")
            app.windows.buttons[appearance == "Aqua" ? "完成" : "Done"].firstMatch.click()
            app.descendants(matching: .any)["open-settings"].click()
            for section in ["general", "fileAccess", "agents", "sources", "gitSync", "updates"] {
                let row = app.descendants(matching: .any)["settings-\(section)"]
                XCTAssertTrue(row.waitForExistence(timeout: 5))
                row.click()
                capture(app, name: "\(appearance)-settings-\(section)")
            }
            app.terminate()
        }
    }

    @MainActor
    private func capture(_ app: XCUIApplication, name: String) {
        let attachment = XCTAttachment(screenshot: app.windows.firstMatch.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }

    @MainActor
    private func assertSettingsSidebar(_ app: XCUIApplication, labels: [String]) {
        for (section, label) in zip(["general", "fileAccess", "agents", "sources", "gitSync", "updates"], labels) {
            let row = app.descendants(matching: .any)["settings-\(section)"]
            XCTAssertTrue(row.waitForExistence(timeout: 3))
            XCTAssertEqual(row.label, label)
        }
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
            "SKM_PREFERENCES_SUITE": "SKMUITests.\(temporaryRoot.lastPathComponent)",
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

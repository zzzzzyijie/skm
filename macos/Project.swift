import ProjectDescription

let project = Project(
    name: "SKM",
    options: .options(
        defaultKnownRegions: ["zh-Hans", "en"],
        developmentRegion: "zh-Hans"
    ),
    settings: .settings(base: [
        "MACOSX_DEPLOYMENT_TARGET": "14.0",
        "SWIFT_VERSION": "6.0",
        "CURRENT_PROJECT_VERSION": "2",
        "MARKETING_VERSION": "0.5.2",
        "ENABLE_USER_SCRIPT_SANDBOXING": "NO",
        "ENABLE_APP_SANDBOX": "NO",
        "ENABLE_HARDENED_RUNTIME": "YES",
        "ASSETCATALOG_COMPILER_APPICON_NAME": "AppIcon",
        "CODE_SIGN_STYLE": "Automatic",
    ]),
    targets: [
        .target(
            name: "SKM",
            destinations: .macOS,
            product: .app,
            bundleId: "com.zzzzzyijie.skm",
            deploymentTargets: .macOS("14.0"),
            infoPlist: .extendingDefault(with: [
                "CFBundleDisplayName": "SKM",
                "CFBundleShortVersionString": "$(MARKETING_VERSION)",
                "CFBundleVersion": "$(CURRENT_PROJECT_VERSION)",
                "LSApplicationCategoryType": "public.app-category.developer-tools",
                "NSHumanReadableCopyright": "Copyright © 2026 SKM Contributors",
                "NSPrincipalClass": "NSApplication",
            ]),
            sources: ["SKMApp/Sources/**"],
            resources: ["SKMApp/Resources/**"],
            scripts: [
                .pre(
                    script: #""$PROJECT_DIR/Scripts/build-go-core.sh""#,
                    name: "Build bundled Go Core",
                    basedOnDependencyAnalysis: false
                ),
            ]
        ),
        .target(
            name: "SKMTests",
            destinations: .macOS,
            product: .unitTests,
            bundleId: "com.zzzzzyijie.skm.tests",
            deploymentTargets: .macOS("14.0"),
            sources: ["SKMApp/Tests/**"],
            dependencies: [.target(name: "SKM")]
        ),
        .target(
            name: "SKMUITests",
            destinations: .macOS,
            product: .uiTests,
            bundleId: "com.zzzzzyijie.skm.ui-tests",
            deploymentTargets: .macOS("14.0"),
            sources: ["SKMApp/UITests/**"],
            dependencies: [.target(name: "SKM")],
            settings: .settings(base: ["ENABLE_HARDENED_RUNTIME": "NO"])
        ),
    ]
)

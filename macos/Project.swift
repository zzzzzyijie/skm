import ProjectDescription

let project = Project(
    name: "SKM",
    options: .options(
        defaultKnownRegions: ["zh-Hans", "en"],
        developmentRegion: "zh-Hans"
    ),
    packages: [
        .remote(url: "https://github.com/sparkle-project/Sparkle", requirement: .exact("2.9.6")),
    ],
    settings: .settings(base: [
        "MACOSX_DEPLOYMENT_TARGET": "14.0",
        "SWIFT_VERSION": "6.0",
        "CURRENT_PROJECT_VERSION": "3",
        "MARKETING_VERSION": "0.5.3",
        "SKM_SPARKLE_FEED_URL": "https://github.com/zzzzzyijie/skm/releases/latest/download/appcast.xml",
        "SKM_SPARKLE_PUBLIC_KEY": "",
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
                "SUEnableAutomaticChecks": true,
                "SUFeedURL": "$(SKM_SPARKLE_FEED_URL)",
                "SUPublicEDKey": "$(SKM_SPARKLE_PUBLIC_KEY)",
            ]),
            sources: ["SKMApp/Sources/**"],
            resources: ["SKMApp/Resources/**"],
            scripts: [
                .pre(
                    script: #""$PROJECT_DIR/Scripts/build-go-core.sh""#,
                    name: "Build bundled Go Core",
                    basedOnDependencyAnalysis: false
                ),
            ],
            dependencies: [.package(product: "Sparkle")]
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

import Foundation
import Observation

@MainActor
@Observable
final class AppPreferences {
    nonisolated static let languageKey = "SKMAppLanguage"
    nonisolated static let appearanceKey = "SKMAppAppearance"

    var language: AppLanguage {
        didSet {
            defaults.set(language.rawValue, forKey: Self.languageKey)
            AppLocalization.configure(language: language)
        }
    }

    var appearance: AppAppearance {
        didSet { defaults.set(appearance.rawValue, forKey: Self.appearanceKey) }
    }

    @ObservationIgnored private let defaults: UserDefaults

    init(defaults: UserDefaults = .skmPreferences) {
        self.defaults = defaults
        language = defaults.string(forKey: Self.languageKey)
            .flatMap(AppLanguage.init(rawValue:)) ?? .system
        appearance = defaults.string(forKey: Self.appearanceKey)
            .flatMap(AppAppearance.init(rawValue:)) ?? .system
        AppLocalization.configure(language: language)
    }
}

extension UserDefaults {
    static var skmPreferences: UserDefaults {
        let environment = ProcessInfo.processInfo.environment
        if let suite = environment["SKM_PREFERENCES_SUITE"], suite.hasPrefix("SKMUITests.") {
            return UserDefaults(suiteName: suite) ?? .standard
        }
        return .standard
    }
}

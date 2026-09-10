import Foundation

/// Keeps localization lookup cheap and safe from non-main-actor error paths.
/// Bundles are loaded once; language changes only replace the small locked snapshot.
final class AppLocalizationStore: @unchecked Sendable {
    static let shared = AppLocalizationStore()

    private let lock = NSLock()
    private let bundles: [AppLanguage: Bundle]
    private var language: AppLanguage

    private init() {
        let defaults = UserDefaults.skmPreferences
        language = defaults.string(forKey: AppPreferences.languageKey)
            .flatMap(AppLanguage.init(rawValue:)) ?? .system
        bundles = Dictionary(uniqueKeysWithValues: AppLanguage.allCases.compactMap { language in
            guard language != .system,
                  let path = Bundle.main.path(forResource: language.rawValue, ofType: "lproj"),
                  let bundle = Bundle(path: path) else {
                return nil
            }
            return (language, bundle)
        })
    }

    func update(language: AppLanguage) {
        lock.lock()
        defer { lock.unlock() }
        self.language = language
    }

    func context() -> (locale: Locale, bundle: Bundle) {
        lock.lock()
        defer { lock.unlock() }
        return context(for: language)
    }

    func context(for language: AppLanguage) -> (locale: Locale, bundle: Bundle) {
        (language.locale, bundles[language] ?? .main)
    }
}

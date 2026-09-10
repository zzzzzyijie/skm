import Foundation

enum AppLocalization {
    static var locale: Locale {
        AppLocalizationStore.shared.context().locale
    }

    static func configure(language: AppLanguage) {
        AppLocalizationStore.shared.update(language: language)
    }

    static func string(_ key: String.LocalizationValue) -> String {
        let context = AppLocalizationStore.shared.context()
        return String(localized: key, bundle: context.bundle, locale: context.locale)
    }

    static func string(_ key: String.LocalizationValue, language: AppLanguage) -> String {
        let context = AppLocalizationStore.shared.context(for: language)
        return String(localized: key, bundle: context.bundle, locale: context.locale)
    }
}

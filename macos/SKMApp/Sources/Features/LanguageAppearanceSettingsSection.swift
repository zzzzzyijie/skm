import SwiftUI

struct LanguageAppearanceSettingsSection: View {
    @Bindable var preferences: AppPreferences

    var body: some View {
        Section {
            Picker("语言", selection: $preferences.language) {
                ForEach(AppLanguage.allCases) { language in
                    Text(language.title)
                        .tag(language)
                }
            }
            .accessibilityIdentifier("settings-language")

            Picker("外观", selection: $preferences.appearance) {
                ForEach(AppAppearance.allCases) { appearance in
                    Label(appearance.title, systemImage: appearance.symbol)
                        .tag(appearance)
                }
            }
            .accessibilityIdentifier("settings-appearance")
        } header: {
            Text("语言与外观")
        } footer: {
            Text("更改会立即应用到所有 SKM 窗口。")
        }
    }
}

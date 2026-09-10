import SwiftUI

/// Feedback stays in the active sheet, where the user can read and recover from errors.
struct SheetChrome: ViewModifier {
    @Bindable var model: AppModel

    func body(content: Content) -> some View {
        content
            .textFieldStyle(.roundedBorder)
            .controlSize(.regular)
            .groupBoxStyle(InspectorGroupBoxStyle())
            .background(SKMDesign.canvas)
            .interactiveDismissDisabled(model.isLoading)
            .safeAreaInset(edge: .bottom, spacing: 0) {
                if let error = model.errorMessage {
                    HStack(alignment: .top, spacing: 10) {
                        Image(systemName: "exclamationmark.triangle.fill").foregroundStyle(.orange)
                        Text(error).font(.callout).textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                        Button("关闭", systemImage: "xmark") { model.errorMessage = nil }
                            .labelStyle(.iconOnly)
                            .buttonStyle(.borderless)
                    }
                    .padding(16)
                    .background(.bar)
                } else if model.isLoading {
                    HStack(spacing: 10) {
                        ProgressView().controlSize(.small)
                        Text(model.statusMessage ?? AppLocalization.string("正在应用更改…"))
                            .font(.callout).foregroundStyle(.secondary)
                        Spacer()
                    }
                    .padding(16)
                    .background(.bar)
                }
            }
    }
}

extension View {
    func sheetChrome(model: AppModel) -> some View { modifier(SheetChrome(model: model)) }
}

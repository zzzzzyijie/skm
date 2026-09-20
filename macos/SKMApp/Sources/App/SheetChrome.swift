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
                        ScrollView {
                            Text(error)
                                .font(.callout)
                                .textSelection(.enabled)
                                .fixedSize(horizontal: false, vertical: true)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        .frame(maxHeight: 88)
                        Button("关闭", systemImage: "xmark") { model.errorMessage = nil }
                            .labelStyle(.iconOnly)
                            .buttonStyle(QuietIconButtonStyle())
                            .foregroundStyle(.secondary)
                    }
                    .padding(.horizontal, SKMDesign.sheetHorizontalPadding)
                    .padding(.vertical, 12)
                    .background(.bar)
                    .overlay(alignment: .top) { Divider() }
                } else if model.isLoading {
                    HStack(spacing: 10) {
                        ProgressView().controlSize(.small)
                        Text(model.statusMessage ?? AppLocalization.string("正在应用更改…"))
                            .font(.callout).foregroundStyle(.secondary)
                            .lineLimit(2)
                            .fixedSize(horizontal: false, vertical: true)
                        Spacer()
                    }
                    .padding(.horizontal, SKMDesign.sheetHorizontalPadding)
                    .padding(.vertical, 12)
                    .background(.bar)
                    .overlay(alignment: .top) { Divider() }
                }
            }
    }
}

extension View {
    func sheetChrome(model: AppModel) -> some View { modifier(SheetChrome(model: model)) }
}

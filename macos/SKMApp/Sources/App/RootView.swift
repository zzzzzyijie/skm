import SwiftUI

struct RootView: View {
    @Bindable var model: AppModel

    var body: some View {
        NavigationSplitView {
            sidebar
                .navigationSplitViewColumnWidth(min: 180, ideal: 210, max: 260)
        } content: {
            content
                .navigationSplitViewColumnWidth(min: 260, ideal: 320, max: 420)
        } detail: {
            detail
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("刷新", systemImage: "arrow.clockwise") {
                    Task { await model.refresh() }
                }
                .disabled(model.isLoading)
                .help("从 ~/.skm 重新载入")
            }
        }
        .overlay(alignment: .bottom) {
            if let status = model.statusMessage {
                StatusPill(text: status, symbol: model.isLoading ? "arrow.triangle.2.circlepath" : "checkmark.circle.fill")
                    .padding(.bottom, 16)
            }
        }
        .alert("无法完成操作", isPresented: Binding(
            get: { model.errorMessage != nil },
            set: { if !$0 { model.errorMessage = nil } }
        )) {
            Button("好", role: .cancel) { model.errorMessage = nil }
        } message: {
            Text(model.errorMessage ?? "未知错误")
        }
    }

    private var sidebar: some View {
        List(AppSection.allCases, selection: $model.section) { section in
            Label(section.rawValue, systemImage: section.symbol)
                .tag(section)
        }
        .safeAreaInset(edge: .bottom) {
            VStack(alignment: .leading, spacing: 6) {
                Divider()
                Label(
                    model.workspace?.configured == true ? "个人工作区已配置" : "个人工作区未配置",
                    systemImage: model.workspace?.configured == true ? "checkmark.icloud" : "icloud.slash"
                )
                .font(.caption)
                .foregroundStyle(.secondary)
                Text("Core \(model.handshake?.coreVersion ?? "—") · Schema \(model.handshake?.schemaVersion.description ?? "—")")
                    .font(.caption2.monospacedDigit())
                    .foregroundStyle(.tertiary)
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 10)
        }
        .navigationTitle("SKM")
    }

    @ViewBuilder
    private var content: some View {
        switch model.section {
        case .skills: SkillsListView(model: model)
        case .prompts: PromptsListView(model: model)
        case .projects: ProjectsListView(model: model)
        case .agents: AgentsListView(model: model)
        }
    }

    @ViewBuilder
    private var detail: some View {
        switch model.section {
        case .skills: SkillDetailView(model: model)
        case .prompts: PromptDetailView(model: model)
        case .projects: ProjectDetailView(model: model)
        case .agents: AgentDetailView(model: model)
        }
    }
}

struct StatusPill: View {
    let text: String
    let symbol: String

    var body: some View {
        Label(text, systemImage: symbol)
            .font(.callout)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .background(.regularMaterial, in: Capsule())
            .shadow(color: .black.opacity(0.12), radius: 8, y: 3)
            .accessibilityAddTraits(.isStaticText)
    }
}

struct SettingsView: View {
    let model: AppModel

    var body: some View {
        Form {
            Section("版本") {
                LabeledContent("App", value: Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev")
                LabeledContent("Go Core", value: model.handshake?.coreVersion ?? "未连接")
                LabeledContent("数据 Schema", value: model.handshake?.schemaVersion.description ?? "—")
            }
            Section("存储") {
                LabeledContent("个人数据", value: "~/.skm")
                Text("Skill、Prompt、部署状态继续与命令行共享；App 不直接修改 YAML。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

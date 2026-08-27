import SwiftUI

struct RootView: View {
    @Bindable var model: AppModel

    var body: some View {
        Group {
            if model.handshake == nil {
                startupContent
            } else {
                mainContent
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
            Text(model.errorMessage ?? String(localized: "未知错误"))
        }
        .sheet(isPresented: $model.showsWelcome) {
            WelcomeView(model: model)
        }
    }

    private var mainContent: some View {
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
    }

    @ViewBuilder
    private var startupContent: some View {
        if model.isLoading || model.startupErrorMessage == nil {
            VStack(spacing: 14) {
                ProgressView()
                Text("正在连接 SKM Core…")
                    .foregroundStyle(.secondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .accessibilityElement(children: .combine)
        } else {
            ContentUnavailableView {
                Label("无法启动 SKM", systemImage: "exclamationmark.triangle")
            } description: {
                Text(model.startupErrorMessage ?? String(localized: "Core 启动失败"))
            } actions: {
                HStack {
                    Button("复制诊断信息") { model.copyDiagnostics() }
                    Button("重试") { Task { await model.retryStart() } }
                        .buttonStyle(.borderedProminent)
                        .keyboardShortcut(.defaultAction)
                }
            }
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
                    model.workspace?.configured == true
                        ? String(localized: "个人工作区已配置")
                        : String(localized: "个人工作区未配置"),
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

struct WelcomeView: View {
    @Bindable var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 24) {
            Image(systemName: model.hasExistingData ? "square.stack.3d.up.fill" : "sparkles")
                .font(.system(size: 42, weight: .medium))
                .foregroundStyle(Color.accentColor)
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 8) {
                Text(model.hasExistingData ? String(localized: "欢迎回来") : String(localized: "欢迎使用 SKM"))
                    .font(.largeTitle.bold())
                Text(model.hasExistingData
                     ? String(localized: "已检测到现有的 ~/.skm 资料库，可以直接继续使用，无需导入或迁移。")
                     : String(localized: "管理本机的 Skills、Prompts 和 Agent 部署。SKM 不会自动启用任何 Agent。"))
                    .font(.title3)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }

            if model.hasExistingData {
                HStack(spacing: 12) {
                    WelcomeMetric(title: "Skills", value: model.skills.count)
                    WelcomeMetric(title: "Prompts", value: model.prompts.count)
                    WelcomeMetric(title: "Projects", value: model.projects.count)
                }
            } else {
                Label("先添加 Skill，再到 Agents 中选择由 SKM 管理的工具。", systemImage: "1.circle")
                    .foregroundStyle(.secondary)
            }

            Spacer()

            HStack {
                Button("管理 Agents") { model.completeWelcome(openAgents: true) }
                Spacer()
                Button(model.hasExistingData ? String(localized: "继续使用现有资料库") : String(localized: "开始使用")) {
                    model.completeWelcome()
                }
                .buttonStyle(.borderedProminent)
                .keyboardShortcut(.defaultAction)
            }
        }
        .padding(32)
        .frame(width: 620, height: 430)
        .interactiveDismissDisabled()
    }
}

private struct WelcomeMetric: View {
    let title: String
    let value: Int

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(value.description)
                .font(.title.bold())
                .monospacedDigit()
            Text(title)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(.quaternary.opacity(0.5), in: RoundedRectangle(cornerRadius: 12))
        .accessibilityElement(children: .combine)
    }
}

struct SettingsView: View {
    let model: AppModel

    var body: some View {
        Form {
            Section("版本") {
                LabeledContent("App", value: Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev")
                LabeledContent("Go Core", value: model.handshake?.coreVersion ?? String(localized: "未连接"))
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

import SwiftUI

/// RootView - macOS App 根视图
/// 负责根据 Core 握手状态在“启动/重试加载屏”与“三栏主界面（NavigationSplitView）”之间切换，
/// 并承载底部全局状态气泡（StatusPill）、全局错误弹窗以及新用户欢迎向导（WelcomeView）。
struct RootView: View {
    @Bindable var model: AppModel

    var body: some View {
        Group {
            if model.handshake == nil {
                // 尚未完成 Core 握手时展示启动中或错误重试屏
                startupContent
            } else {
                // 握手成功后展示三栏主界面
                mainContent
            }
        }
        .overlay(alignment: .bottom) {
            // 底部居中悬浮状态胶囊
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

    /// 三栏主界面布局：侧边栏（Sidebar） -> 内容列表栏（Content） -> 详情面板（Detail）
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
    }

    /// Core 启动连接状态视图：加载中旋转菊花 或 启动失败诊断重试视图
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
                .accessibilityIdentifier("navigation-\(section.rawValue.lowercased())")
                .tag(section)
        }
        .safeAreaInset(edge: .bottom) {
            VStack(alignment: .leading, spacing: 0) {
                Divider()
                SettingsLink {
                    Label("设置", systemImage: "gearshape")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityIdentifier("open-settings")
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
        }
        .navigationTitle("SKM")
    }

    @ViewBuilder
    private var content: some View {
        switch model.section {
        case .skills: SkillsListView(model: model)
        case .prompts: PromptsListView(model: model)
        case .projects: ProjectsListView(model: model)
        }
    }

    @ViewBuilder
    private var detail: some View {
        switch model.section {
        case .skills: SkillDetailView(model: model)
        case .prompts: PromptDetailView(model: model)
        case .projects: ProjectDetailView(model: model)
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
    @Environment(\.openSettings) private var openSettings
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
                Button("管理 Agents") {
                    model.completeWelcome()
                    model.settingsSection = .agents
                    openSettings()
                }
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
    @Bindable var model: AppModel

    var body: some View {
        NavigationSplitView {
            List(SettingsSection.allCases, selection: $model.settingsSection) { section in
                Label(section.title, systemImage: section.symbol)
                    .tag(section)
                    .accessibilityIdentifier("settings-\(section.rawValue)")
            }
            .navigationTitle("设置")
            .navigationSplitViewColumnWidth(min: 170, ideal: 190, max: 230)
        } detail: {
            detail
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
    }

    @ViewBuilder
    private var detail: some View {
        switch model.settingsSection {
        case .general: GeneralSettingsView(model: model)
        case .fileAccess: ProjectAccessSettingsView(model: model)
        case .agents: AgentsSettingsView(model: model)
        case .sources: SourcesSettingsView(model: model)
        case .gitSync: WorkspaceDetailView(model: model)
        case .updates: UpdatesSettingsView(model: model)
        }
    }
}

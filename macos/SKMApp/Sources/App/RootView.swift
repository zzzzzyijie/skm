import SwiftUI

/// RootView - macOS App 根视图
/// 负责根据 Core 握手状态在“启动/重试加载屏”与“三栏主界面（NavigationSplitView）”之间切换，
/// 并承载底部全局状态气泡（StatusPill）、全局错误弹窗以及新用户欢迎向导（WelcomeView）。
struct RootView: View {
    @Bindable var model: AppModel
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var sidebarSelection: AppSection

    init(model: AppModel) {
        self.model = model
        _sidebarSelection = State(initialValue: model.section)
    }

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
                StatusPill(text: status, isLoading: model.isLoading)
                    .padding(.bottom, 16)
                    .transition(.opacity)
            }
        }
        .animation(reduceMotion ? nil : .easeOut(duration: 0.15), value: model.statusMessage)
        .groupBoxStyle(InspectorGroupBoxStyle())
        .alert("无法完成操作", isPresented: Binding(
            get: { model.errorMessage != nil },
            set: { if !$0 { model.errorMessage = nil } }
        )) {
            Button("好", role: .cancel) { model.errorMessage = nil }
        } message: {
            Text(model.errorMessage ?? AppLocalization.string("未知错误"))
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
                .navigationSplitViewColumnWidth(min: 280, ideal: 340, max: 440)
        } detail: {
            detail
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(SKMDesign.canvas)
        }
        .navigationSplitViewStyle(.balanced)
        .toolbar(removing: .sidebarToggle)
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
                Text(model.startupErrorMessage ?? AppLocalization.string("Core 启动失败"))
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
        List {
            ForEach(AppSection.allCases) { section in
                SidebarNavigationButton(
                    title: section.title,
                    systemImage: section.symbol,
                    isSelected: sidebarSelection == section,
                    count: collectionCount(section),
                    action: { selectSidebarSection(section) }
                )
                .listRowInsets(EdgeInsets(
                    top: 1,
                    leading: 0,
                    bottom: 1,
                    trailing: 0
                ))
                .listRowBackground(Color.clear)
                .accessibilityIdentifier("navigation-\(section.rawValue.lowercased())")
            }
        }
        .listStyle(.sidebar)
        .onChange(of: model.section) { _, section in
            guard sidebarSelection != section else { return }
            sidebarSelection = section
        }
        .safeAreaInset(edge: .bottom) {
            VStack(spacing: 0) {
                Divider()
                SidebarBottomToolbar(model: model)
            }
        }
        .navigationTitle(AppLocalization.string("SKM"))
    }

    private func selectSidebarSection(_ section: AppSection) {
        guard sidebarSelection != section else { return }

        // Paint the lightweight sidebar selection before rebuilding both content columns.
        sidebarSelection = section
        Task { @MainActor in
            await Task.yield()
            guard sidebarSelection == section, model.section != section else { return }
            model.section = section
        }
    }

    private func collectionCount(_ section: AppSection) -> Int {
        switch section {
        case .skills: model.skills.count
        case .prompts: model.prompts.count
        case .projects: model.projects.count
        }
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

/// Shared search chrome keeps keyboard and accessibility behavior consistent across collections.
struct CollectionSearchField: View {
    let title: LocalizedStringKey
    @Binding var text: String
    @FocusState private var isFocused: Bool

    var body: some View {
        HStack(spacing: 7) {
            Button { isFocused = true } label: {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
            }
            .buttonStyle(.plain)
            .keyboardShortcut("f", modifiers: .command)
            .accessibilityLabel(title)
            .help("⌘F")

            TextField(title, text: $text)
                .textFieldStyle(.plain)
                .focused($isFocused)
                .onExitCommand {
                    if text.isEmpty { isFocused = false } else { text = "" }
                }

            if !text.isEmpty {
                Button { text = ""; isFocused = true } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("清除搜索")
            } else if !isFocused {
                Text("⌘F")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .accessibilityHidden(true)
            }
        }
        .font(.callout)
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(isFocused ? Color.accentColor : Color.primary.opacity(0.1), lineWidth: isFocused ? 2 : 0.5)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(.bar)
    }
}

struct StatusPill: View {
    let text: String
    let isLoading: Bool

    var body: some View {
        HStack(spacing: 8) {
            if isLoading {
                ProgressView().controlSize(.small)
            } else {
                Image(systemName: "checkmark.circle.fill").foregroundStyle(.green)
            }
            Text(text).lineLimit(3)
        }
            .font(.callout)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .background(.regularMaterial, in: Capsule())
            .shadow(color: .black.opacity(0.12), radius: 8, y: 3)
            .padding(.horizontal, 24)
            .allowsHitTesting(false)
            .accessibilityAddTraits(.isStaticText)
    }
}

/// 主窗口顶部工具栏按钮使用统一的正方形槽位，确保单按钮为圆形、按钮组为等高胶囊。
private struct TopToolbarActionModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .labelStyle(.iconOnly)
            .buttonStyle(.borderless)
            .controlSize(.regular)
            .frame(
                width: SKMDesign.toolbarActionSize,
                height: SKMDesign.toolbarActionSize
            )
            .contentShape(Rectangle())
    }
}

extension View {
    func topToolbarActionStyle() -> some View {
        modifier(TopToolbarActionModifier())
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
                Text(model.hasExistingData ? AppLocalization.string("欢迎回来") : AppLocalization.string("欢迎使用 SKM"))
                    .font(.largeTitle.bold())
                Text(model.hasExistingData
                     ? AppLocalization.string("已检测到现有的 ~/.skm 资料库，可以直接继续使用，无需导入或迁移。")
                     : AppLocalization.string("管理本机的 Skills、Prompts 和 Agent 部署。SKM 不会自动启用任何 Agent。"))
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
                Button(model.hasExistingData ? AppLocalization.string("继续使用现有资料库") : AppLocalization.string("开始使用")) {
                    model.completeWelcome()
                }
                .buttonStyle(.borderedProminent)
                .keyboardShortcut(.defaultAction)
            }
        }
        .padding(32)
        .frame(width: 620, height: 430)
        .background(SKMDesign.canvas)
        .groupBoxStyle(InspectorGroupBoxStyle())
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
        .skmSurface()
        .accessibilityElement(children: .combine)
    }
}

struct SettingsView: View {
    @Bindable var model: AppModel
    @Bindable var preferences: AppPreferences

    var body: some View {
        let language = preferences.language

        NavigationSplitView {
            List {
                ForEach(SettingsSection.allCases) { section in
                    SidebarNavigationButton(
                        title: section.title,
                        systemImage: section.symbol,
                        isSelected: model.settingsSection == section,
                        action: {
                            guard model.settingsSection != section else { return }
                            model.settingsSection = section
                        }
                    )
                    .listRowInsets(EdgeInsets(
                        top: 1,
                        leading: 0,
                        bottom: 1,
                        trailing: 0
                    ))
                    .listRowBackground(Color.clear)
                    .accessibilityIdentifier("settings-\(section.rawValue)")
                }
            }
            .listStyle(.sidebar)
            .navigationTitle(AppLocalization.string("设置", language: language))
            .navigationSplitViewColumnWidth(min: 180, ideal: 210, max: 250)
        } detail: {
            // Keep the detail view's identity and state intact. Passing language as a value
            // input invalidates only its rendered content; `.id(language)` would rebuild
            // native controls and rerun lifecycle tasks on every switch.
            detail(language: language)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(SKMDesign.canvas)
        }
        .groupBoxStyle(InspectorGroupBoxStyle())
        .textFieldStyle(.roundedBorder)
        .toolbar(removing: .sidebarToggle)
        .overlay(alignment: .bottom) {
            if let status = model.statusMessage {
                StatusPill(text: status, isLoading: model.isLoading)
                    .padding(.bottom, 16)
            }
        }
        .alert("无法完成操作", isPresented: Binding(
            get: { model.errorMessage != nil },
            set: { if !$0 { model.errorMessage = nil } }
        )) { } message: {
            Text(model.errorMessage ?? "")
        }
    }

    @ViewBuilder
    private func detail(language: AppLanguage) -> some View {
        switch model.settingsSection {
        case .general: GeneralSettingsView(model: model, preferences: preferences, language: language)
        case .fileAccess: ProjectAccessSettingsView(model: model, language: language)
        case .agents: AgentsSettingsView(model: model, language: language)
        case .sources: SourcesSettingsView(model: model, language: language)
        case .gitSync: WorkspaceDetailView(model: model, language: language)
        case .updates: UpdatesSettingsView(model: model, language: language)
        }
    }
}

/// 主侧边栏左下角常驻操作栏（设置与一键同步纯图标按钮）
private struct SidebarBottomToolbar: View {
    @Environment(\.openSettings) private var openSettings
    @Bindable var model: AppModel

    var body: some View {
        HStack(spacing: 12) {
            // 一键同步纯图标按钮
            Button {
                if model.workspace?.configured == true {
                    Task { await model.oneClickSync() }
                } else {
                    model.settingsSection = .gitSync
                    openSettings()
                }
            } label: {
                Group {
                    if model.isLoading {
                        ProgressView()
                            .controlSize(.mini)
                            .frame(width: 16, height: 16)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(.secondary)
                            .frame(width: 16, height: 16)
                    }
                }
                .frame(width: 30, height: 30)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .disabled(model.isLoading)
            .accessibilityLabel(syncHelpText)
            .accessibilityIdentifier("sidebar-sync-button")
            .help(syncHelpText)

            // 设置纯图标按钮
            SettingsLink {
                Image(systemName: "gearshape")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(.secondary)
                    .frame(width: 30, height: 30)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel(AppLocalization.string("偏好设置 (⌘,)"))
            .accessibilityIdentifier("open-settings")
            .help(AppLocalization.string("偏好设置 (⌘,)"))

            Spacer()
            Text("SKM")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.tertiary)
                .accessibilityHidden(true)
        }
        .padding(.leading, 21)
        .padding(.trailing, 26)
        .padding(.vertical, 7)
    }

    private var syncHelpText: String {
        if model.isLoading {
            return AppLocalization.string("正在同步…")
        }
        if model.workspace?.configured == true {
            if let revision = model.workspace?.state?.revision, !revision.isEmpty {
                return String(format: AppLocalization.string("一键同步 (Rev: %@)"), String(revision.prefix(7)))
            }
            return AppLocalization.string("一键同步个人工作区")
        }
        return AppLocalization.string("未配置 Git 同步，点击前往设置")
    }
}

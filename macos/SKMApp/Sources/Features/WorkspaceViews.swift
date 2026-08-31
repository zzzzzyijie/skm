import SwiftUI

/// 个人 Git 工作区侧栏概览
struct WorkspaceSidebarView: View {
    let model: AppModel

    var body: some View {
        List {
            Label("个人 Git 同步", systemImage: model.workspace?.configured == true ? "checkmark.icloud" : "icloud.slash")
            Label("同步预览", systemImage: "list.bullet.rectangle")
                .foregroundStyle(model.workspacePreview == nil ? .secondary : .primary)
        }
        .navigationTitle("Git 同步")
    }
}

/// WorkspaceDetailView - 个人 Git 工作区配置与双向同步视图
/// 允许用户绑定私有 Git 仓库，同步本机的 Skills、Prompts 与配置。
/// 同步流程严格遵循“两阶段确认”：
/// 1. 生成同步预览：计算上传、下载、删除与双端冲突列表；
/// 2. 交互式决议：用户针对每个冲突项选择“保留本地”或“使用远端”；
/// 3. 执行同步：提交变更并拉取合并。
struct WorkspaceDetailView: View {
    @Bindable var model: AppModel
    @State private var url = ""
    @State private var ref = "main"
    @State private var root = ""

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                Text("个人 Git 同步").font(.largeTitle.bold())
                Text("通过私人 Git 仓库在多台设备间同步 Skills、Prompts 和 Source 配置。认证信息不会写入 SKM。")
                    .foregroundStyle(.secondary)
                configuration
                if model.workspace?.configured == true {
                    syncControls
                }
                if let preview = model.workspacePreview {
                    previewView(preview)
                }
            }
            .padding(26)
            .frame(maxWidth: 860, alignment: .leading)
        }
        .task { loadConfiguration() }
        .navigationTitle("Git 同步")
    }

    private var configuration: some View {
        GroupBox("连接") {
            VStack(alignment: .leading, spacing: 12) {
                TextField("Git URL", text: $url)
                    .accessibilityIdentifier("workspace-url-field")
                HStack {
                    TextField("分支", text: $ref)
                    TextField("仓库内目录（可选）", text: $root)
                }
                HStack {
                    if let revision = model.workspace?.state?.revision, !revision.isEmpty {
                        Text("当前 Revision：\(revision.prefix(12))")
                            .font(.caption.monospaced()).foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button(model.workspace?.configured == true ? String(localized: "保存并测试连接") : String(localized: "配置并测试连接")) {
                        Task { await model.configureWorkspace(url: url, ref: ref, root: root) }
                    }
                    .accessibilityIdentifier("workspace-configure-button")
                    .buttonStyle(.borderedProminent)
                    .disabled(url.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || ref.isEmpty)
                }
            }
            .padding(8)
        }
    }

    private var syncControls: some View {
        HStack {
            Label("同步执行前会列出上传、下载、删除与冲突。", systemImage: "info.circle")
                .foregroundStyle(.secondary)
            Spacer()
            Button("生成同步预览", systemImage: "arrow.left.arrow.right") { Task { await model.previewWorkspace() } }
                .accessibilityIdentifier("workspace-preview-button")
                .disabled(model.isLoading)
        }
    }

    private func previewView(_ preview: WorkspacePreview) -> some View {
        let unresolved = preview.changes.filter { $0.action == "conflict" && model.workspaceResolutions[$0.id] == nil }
        return GroupBox("同步预览") {
            VStack(alignment: .leading, spacing: 12) {
                HStack(spacing: 16) {
                    Label("上传 \(preview.uploads)", systemImage: "arrow.up.circle")
                    Label("下载 \(preview.downloads)", systemImage: "arrow.down.circle")
                    Label("删除 \(preview.deletes)", systemImage: "trash")
                    Label("冲突 \(preview.conflicts)", systemImage: "exclamationmark.triangle")
                        .foregroundStyle(preview.conflicts > 0 ? .orange : .secondary)
                }
                Divider()
                if preview.changes.isEmpty {
                    ContentUnavailableView("没有需要同步的更改", systemImage: "checkmark.icloud")
                        .frame(minHeight: 130)
                } else {
                    ForEach(preview.changes) { change in
                        WorkspaceChangeRow(change: change, resolution: Binding(
                            get: { model.workspaceResolutions[change.id] },
                            set: { model.workspaceResolutions[change.id] = $0 }
                        ))
                        if change.id != preview.changes.last?.id { Divider() }
                    }
                }
                HStack {
                    if !unresolved.isEmpty {
                        Label("还有 \(unresolved.count) 个冲突未选择", systemImage: "hand.raised")
                            .foregroundStyle(.orange)
                    }
                    Spacer()
                    Button("取消预览") { model.workspacePreview = nil }
                    Button("开始同步") { Task { await model.syncWorkspace() } }
                        .buttonStyle(.borderedProminent)
                        .disabled(!unresolved.isEmpty || model.isLoading)
                }
            }
            .padding(8)
        }
    }

    private func loadConfiguration() {
        guard let config = model.workspace?.config else { return }
        url = config.url
        ref = config.ref
        root = config.root ?? ""
    }
}

private struct WorkspaceChangeRow: View {
    let change: WorkspaceChange
    @Binding var resolution: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label(change.name, systemImage: symbol)
                    .fontWeight(.medium)
                Text(change.kind).font(.caption).foregroundStyle(.secondary)
                Spacer()
                Text(actionLabel).font(.caption).foregroundStyle(change.action == "conflict" ? .orange : .secondary)
            }
            if let detail = change.detail, !detail.isEmpty {
                Text(detail).font(.caption).foregroundStyle(.secondary)
            }
            if change.action == "conflict" {
                Picker("冲突选择", selection: Binding(
                    get: { resolution ?? "" },
                    set: { resolution = $0.isEmpty ? nil : $0 }
                )) {
                    Text("请选择…").tag("")
                    Text("保留本地").tag("local")
                    Text("使用远端").tag("remote")
                }
                .pickerStyle(.segmented)
            }
        }
        .padding(.vertical, 6)
    }

    private var symbol: String {
        switch change.action {
        case "upload": "arrow.up.circle"
        case "download": "arrow.down.circle"
        case "delete-local", "delete-remote": "trash"
        default: "exclamationmark.triangle"
        }
    }

    private var actionLabel: String {
        switch change.action {
        case "upload": String(localized: "上传")
        case "download": String(localized: "下载")
        case "delete-local", "delete-remote": String(localized: "删除")
        default: String(localized: "冲突")
        }
    }
}

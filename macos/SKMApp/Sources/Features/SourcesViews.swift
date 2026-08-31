import SwiftUI

/// SourcesSettingsView - 技能来源管理设置页面
/// 支持查看已添加的 Git 源仓库列表、分支与提交 Revision，
/// 提供单源单独更新、批量“更新所有来源”以及安全移除功能。
struct SourcesSettingsView: View {
    @Bindable var model: AppModel
    @State private var showsAddSource = false
    @State private var sourceToRemove: SourceModel?
    @State private var confirmsRemoval = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                HStack(alignment: .firstTextBaseline) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("技能来源").font(.largeTitle.bold())
                        Text("通过团队或社区 Git 仓库获取并更新 Skills。认证继续由系统 Git 管理。")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("更新所有来源", systemImage: "arrow.triangle.2.circlepath") {
                        Task { await model.syncSources() }
                    }
                    .disabled(model.sources.isEmpty || model.isLoading)
                    .help("重新拉取所有 Git Source 并刷新导入的 Skills")
                    Button("添加 Source", systemImage: "plus") { showsAddSource = true }
                        .buttonStyle(.borderedProminent)
                }

                if model.sources.isEmpty {
                    ContentUnavailableView {
                        Label("暂无技能来源", systemImage: "arrow.triangle.branch")
                    } description: {
                        Text("添加 Git 仓库后，可以统一更新其中的 Skills。")
                    } actions: {
                        Button("添加 Source") { showsAddSource = true }
                            .buttonStyle(.borderedProminent)
                    }
                    .frame(minHeight: 260)
                } else {
                    LazyVStack(spacing: 10) {
                        ForEach(model.sources) { source in
                            SourceSettingsRow(
                                source: source,
                                onUpdate: { Task { await model.updateSource(name: source.name) } },
                                onDelete: {
                                    sourceToRemove = source
                                    confirmsRemoval = true
                                }
                            )
                        }
                    }
                }

                if let result = model.sourceSyncResult {
                    GroupBox("最近一次同步") {
                        VStack(alignment: .leading, spacing: 10) {
                            Text(String(
                                format: String(localized: "成功 %lld · 失败 %lld · 更新 %lld 个 Skills"),
                                locale: .current,
                                result.updated,
                                result.failed,
                                result.skillCount
                            ))
                            ForEach(result.results) { item in
                                SourceSyncRow(item: item)
                            }
                            if let error = result.deploymentError {
                                Text(error).foregroundStyle(.orange).textSelection(.enabled)
                            }
                        }
                        .padding(8)
                    }
                }
            }
            .padding(26)
            .frame(maxWidth: 860, alignment: .leading)
        }
        .navigationTitle("技能来源")
        .sheet(isPresented: $showsAddSource) { AddSourceSheet(model: model) }
        .confirmationDialog(
            String(format: String(localized: "移除 %@？"), locale: .current, sourceToRemove?.name ?? ""),
            isPresented: $confirmsRemoval
        ) {
            if let sourceToRemove {
                Button("移除", role: .destructive) {
                    Task { await model.removeSource(name: sourceToRemove.name) }
                }
            }
        } message: {
            Text("移除来源绑定和本机 checkout；已经导入 Library 的不可变快照会保留。")
        }
    }
}

private struct SourceSettingsRow: View {
    let source: SourceModel
    let onUpdate: () -> Void
    let onDelete: () -> Void

    var body: some View {
        HStack(spacing: 14) {
            Image(systemName: "arrow.triangle.branch")
                .font(.title3)
                .foregroundStyle(Color.accentColor)
                .frame(width: 38, height: 38)
                .background(.quaternary, in: RoundedRectangle(cornerRadius: 9))

            VStack(alignment: .leading, spacing: 4) {
                Text(source.name).fontWeight(.semibold)
                Text(source.url)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .textSelection(.enabled)
                Text(status)
                    .font(.caption2.monospaced())
                    .foregroundStyle(.tertiary)
            }

            Spacer(minLength: 12)

            Button("更新", systemImage: "arrow.clockwise", action: onUpdate)
                .buttonStyle(.borderless)

            Menu("更多", systemImage: "ellipsis.circle") {
                Button("移除 Source", systemImage: "trash", role: .destructive, action: onDelete)
            }
            .menuStyle(.borderlessButton)
            .fixedSize()
        }
        .padding(14)
        .background(.quaternary.opacity(0.45), in: RoundedRectangle(cornerRadius: 12))
    }

    private var status: String {
        let ref = source.ref ?? "HEAD"
        let revision = source.revision.map { String($0.prefix(10)) } ?? String(localized: "尚未同步")
        return "\(ref) · \(revision)"
    }
}

struct SourcesListView: View {
    @Bindable var model: AppModel
    @State private var showsAddSource = false

    var body: some View {
        content
        .navigationTitle("Sources")
        .toolbar {
            Button("同步全部", systemImage: "arrow.triangle.2.circlepath") { Task { await model.syncSources() } }
                .disabled(model.sources.isEmpty || model.isLoading)
            Button("添加 Source", systemImage: "plus") { showsAddSource = true }
        }
        .sheet(isPresented: $showsAddSource) { AddSourceSheet(model: model) }
    }

    @ViewBuilder
    private var content: some View {
        if model.sources.isEmpty {
            ContentUnavailableView {
                Label("没有 Git Source", systemImage: "arrow.triangle.branch")
            } description: {
                Text("添加团队或社区 Skill 仓库；凭据继续由系统 Git 管理。")
            } actions: {
                Button("添加 Source") { showsAddSource = true }
                    .buttonStyle(.borderedProminent)
            }
        } else {
            List(model.sources, selection: $model.selectedSourceID) { source in
                SourceRow(source: source)
                    .tag(source.id)
            }
        }
    }
}

private struct SourceRow: View {
    let source: SourceModel

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(source.name).fontWeight(.medium)
            Text(source.url).font(.caption).foregroundStyle(.secondary).lineLimit(1)
            Text(revision).font(.caption2.monospaced()).foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
    }

    private var revision: String {
        source.revision.map { String($0.prefix(10)) } ?? String(localized: "尚未同步")
    }
}

struct SourceDetailView: View {
    @Bindable var model: AppModel
    @State private var confirmsRemoval = false

    var body: some View {
        if let id = model.selectedSourceID,
           let source = model.sources.first(where: { $0.id == id }) {
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(source.name).font(.largeTitle.bold())
                        Text(source.url).foregroundStyle(.secondary).textSelection(.enabled)
                    }
                    GroupBox("Source 配置") {
                        VStack(alignment: .leading, spacing: 10) {
                            LabeledContent("分支 / Ref", value: source.ref ?? "HEAD")
                            LabeledContent("Revision", value: source.revision ?? String(localized: "尚未同步"))
                            LabeledContent("路径", value: source.paths?.joined(separator: ", ") ?? String(localized: "自动发现"))
                            LabeledContent("标签", value: source.tags.joined(separator: ", "))
                        }
                        .padding(8)
                    }
                    HStack {
                        Button("移除 Source", role: .destructive) { confirmsRemoval = true }
                        Spacer()
                        Button("更新此 Source", systemImage: "arrow.clockwise") { Task { await model.updateSource(name: source.name) } }
                            .buttonStyle(.borderedProminent)
                    }
                    if let result = model.sourceSyncResult {
                        GroupBox("最近一次统一同步") {
                            VStack(alignment: .leading, spacing: 8) {
                                Text(String(format: String(localized: "成功 %lld · 失败 %lld · 更新 %lld 个 Skills"), locale: .current, result.updated, result.failed, result.skillCount))
                                ForEach(result.results) { item in
                                    SourceSyncRow(item: item)
                                }
                                if let error = result.deploymentError {
                                    Text(error).foregroundStyle(.orange).textSelection(.enabled)
                                }
                            }
                            .padding(8)
                        }
                    }
                }
                .padding(26)
                .frame(maxWidth: 820, alignment: .leading)
            }
            .confirmationDialog("移除 \(source.name)？", isPresented: $confirmsRemoval) {
                Button("移除", role: .destructive) { Task { await model.removeSource(name: source.name) } }
            } message: {
                Text("移除来源绑定和本机 checkout；已经导入 Library 的不可变快照会保留。")
            }
        } else {
            ContentUnavailableView("选择一个 Git Source", systemImage: "arrow.triangle.branch")
        }
    }
}

struct SourceSyncRow: View {
    let item: SourceSyncItem

    var body: some View {
        Label(item.error ?? item.name, systemImage: symbol)
            .foregroundStyle(color)
    }

    private var symbol: String {
        item.status == "updated" ? "checkmark.circle" : "exclamationmark.triangle"
    }

    private var color: Color {
        item.status == "updated" ? .secondary : .orange
    }
}

struct AddSourceSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    @State private var input = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("添加 Git Source").font(.title2.bold())
            TextField("Git URL、owner/repo 或 npx skills add …", text: $input)
            Text("SKM 不保存密码或 Token；SSH Agent 与 Git Credential Helper 的行为和终端一致。")
                .font(.caption).foregroundStyle(.secondary)
            Spacer()
            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button("添加并导入") {
                    Task {
                        await model.addSource(input: input)
                        if model.errorMessage == nil { dismiss() }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24)
        .frame(width: 560, height: 260)
    }
}

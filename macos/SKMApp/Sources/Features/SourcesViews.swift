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
    @State private var sourceName = ""
    @State private var tags = ""

    // 两步向导状态
    @State private var wizardStep = 0
    @State private var isScanning = false
    @State private var previewResult: SourcePreviewResult?
    @State private var selectedPaths: Set<String> = []

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            headerView

            if wizardStep == 0 {
                inputStepView
            } else {
                previewStepView
            }
        }
        .padding(24)
        .frame(
            minWidth: wizardStep == 1 ? 660 : 560,
            minHeight: wizardStep == 1 ? 480 : 280
        )
    }

    private var headerView: some View {
        HStack {
            Text(wizardStep == 0 ? String(localized: "添加 Skill Source") : String(localized: "选择要导入的 Skill"))
                .font(.title2.bold())
            Spacer()
            Text(wizardStep == 0 ? "1/2 步：输入仓库" : "2/2 步：勾选技能")
                .font(.caption)
                .foregroundStyle(.secondary)
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background(.quaternary, in: Capsule())
        }
    }

    private var inputStepView: some View {
        VStack(alignment: .leading, spacing: 14) {
            TextField("Git URL、owner/repo 或 npx skills add …", text: $input)
            TextField("来源名称（可选，留空自动提取）", text: $sourceName)
            TextField("标签，以逗号分隔（可选）", text: $tags)
            Text("SKM 不保存密码或 Token；SSH Agent 与 Git Credential Helper 的行为和终端一致。")
                .font(.caption).foregroundStyle(.secondary)

            Spacer()

            HStack {
                Spacer()
                Button("取消", role: .cancel) { dismiss() }
                Button {
                    Task { await startPreview() }
                } label: {
                    HStack(spacing: 6) {
                        if isScanning {
                            ProgressView().controlSize(.small)
                            Text("正在扫描…")
                        } else {
                            Text("下一步：扫描技能")
                            Image(systemName: "arrow.right")
                        }
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || isScanning)
            }
        }
    }

    private var previewStepView: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let preview = previewResult {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(preview.source.name).fontWeight(.semibold)
                        Text(preview.source.url)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                    Spacer()
                    if let rev = preview.source.revision {
                        Text(String(rev.prefix(8)))
                            .font(.caption.monospaced())
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(.quaternary, in: RoundedRectangle(cornerRadius: 4))
                    }
                }
                .padding(10)
                .background(.quaternary.opacity(0.4), in: RoundedRectangle(cornerRadius: 8))

                HStack {
                    let validCandidates = preview.skills.filter(\.valid)
                    Text(String(format: String(localized: "已选择 %lld / %lld 个可用技能"), locale: .current, selectedPaths.count, validCandidates.count))
                        .font(.callout)
                        .foregroundStyle(.secondary)
                    Spacer()
                    Button("全选") {
                        selectedPaths = Set(validCandidates.map(\.path))
                    }
                    .buttonStyle(.link)
                    .disabled(selectedPaths.count == validCandidates.count)

                    Text("·").foregroundStyle(.secondary)

                    Button("取消全选") {
                        selectedPaths.removeAll()
                    }
                    .buttonStyle(.link)
                    .disabled(selectedPaths.isEmpty)
                }

                ScrollView {
                    LazyVStack(spacing: 8) {
                        ForEach(preview.skills) { candidate in
                            SourceSkillCandidateRow(
                                candidate: candidate,
                                isSelected: selectedPaths.contains(candidate.path),
                                onToggle: {
                                    if selectedPaths.contains(candidate.path) {
                                        selectedPaths.remove(candidate.path)
                                    } else {
                                        selectedPaths.insert(candidate.path)
                                    }
                                }
                            )
                        }
                    }
                    .padding(.vertical, 2)
                }
                .frame(maxHeight: 260)
            }

            Spacer()

            HStack {
                Button("上一步", systemImage: "arrow.left") {
                    wizardStep = 0
                }
                .disabled(model.isLoading)

                Spacer()

                Button("取消", role: .cancel) { dismiss() }
                Button(String(format: String(localized: "添加并导入 (%lld)"), locale: .current, selectedPaths.count)) {
                    Task { await confirmImport() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(selectedPaths.isEmpty || model.isLoading)
            }
        }
    }

    private func startPreview() async {
        isScanning = true
        defer { isScanning = false }
        do {
            let result = try await model.previewSource(
                input: input,
                name: sourceName.trimmingCharacters(in: .whitespaces).isEmpty ? nil : sourceName
            )
            previewResult = result
            let valid = result.skills.filter(\.valid)
            if let requested = result.requestedSkills, !requested.isEmpty {
                let requestedSet = Set(requested)
                selectedPaths = Set(valid.filter { requestedSet.contains($0.name) || requestedSet.contains($0.path) }.map(\.path))
                if selectedPaths.isEmpty {
                    selectedPaths = Set(valid.map(\.path))
                }
            } else {
                selectedPaths = Set(valid.map(\.path))
            }
            wizardStep = 1
        } catch {
            model.errorMessage = error.localizedDescription
        }
    }

    private func confirmImport() async {
        guard let preview = previewResult else { return }
        let success = await model.addSource(
            input: preview.source.url,
            name: preview.source.name,
            ref: preview.source.ref,
            paths: Array(selectedPaths),
            tags: parseTags(tags)
        )
        if success {
            dismiss()
        }
    }
}

/// SourceSkillCandidateRow - Source 中的候选技能行
private struct SourceSkillCandidateRow: View {
    let candidate: SkillCandidate
    let isSelected: Bool
    let onToggle: () -> Void

    var body: some View {
        Button(action: {
            if candidate.valid { onToggle() }
        }) {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: candidate.valid ? (isSelected ? "checkmark.square.fill" : "square") : "xmark.square")
                    .font(.title3)
                    .foregroundStyle(candidate.valid ? (isSelected ? Color.accentColor : Color.secondary) : Color.secondary.opacity(0.4))
                    .frame(width: 20)

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 8) {
                        Text(candidate.name)
                            .fontWeight(.semibold)
                            .foregroundStyle(candidate.valid ? Color.primary : Color.secondary)
                        if !candidate.path.isEmpty && candidate.path != candidate.name {
                            Text(candidate.path)
                                .font(.caption.monospaced())
                                .foregroundStyle(Color.secondary)
                        }
                    }

                    if let desc = candidate.description, !desc.isEmpty {
                        Text(desc)
                            .font(.caption)
                            .foregroundStyle(Color.secondary)
                            .lineLimit(2)
                    }

                    if !candidate.valid, let error = candidate.error {
                        Label(error, systemImage: "exclamationmark.triangle.fill")
                            .font(.caption2)
                            .foregroundStyle(Color.orange)
                    }
                }

                Spacer()
            }
            .padding(10)
            .background(
                isSelected ? Color.accentColor.opacity(0.08) : Color.primary.opacity(0.04),
                in: RoundedRectangle(cornerRadius: 8)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(isSelected ? Color.accentColor.opacity(0.35) : Color.clear, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .disabled(!candidate.valid)
    }
}

import AppKit
import SwiftUI

struct ProjectAccessSettingsView: View {
    @Bindable var model: AppModel

    private var readableProjectCount: Int {
        model.projects.count { ProjectAccessStatus(project: $0).canRead }
    }

    private var hasPermissionIssue: Bool {
        model.projects.contains { ProjectAccessStatus(project: $0) == .permissionDenied }
    }

    private var allProjectsReadable: Bool {
        !model.projects.isEmpty && readableProjectCount == model.projects.count
    }

    private var summaryTitle: String {
        if model.projects.isEmpty { return String(localized: "尚无项目需要检查") }
        return allProjectsReadable ? String(localized: "所有项目均可读取") : String(localized: "部分项目需要处理")
    }

    private var summarySymbol: String {
        if model.projects.isEmpty { return "folder" }
        return allProjectsReadable ? "checkmark.shield" : "exclamationmark.shield"
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                HStack(alignment: .firstTextBaseline) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("文件访问").font(.largeTitle.bold())
                        Text("SKM 只保存项目路径。这里实时检查读取状态；写入权限会在部署确认后由 Core 实际校验。")
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("重新检查", systemImage: "arrow.clockwise", action: refreshAccess)
                        .disabled(model.isLoading)
                }

                GroupBox {
                    VStack(alignment: .leading, spacing: 12) {
                        LabeledContent {
                            Text("\(readableProjectCount)/\(model.projects.count)")
                                .monospacedDigit()
                        } label: {
                            Label(summaryTitle, systemImage: summarySymbol)
                        }

                        if hasPermissionIssue {
                            Divider()
                            HStack {
                                Text("如果重新选择目录后仍被拒绝，请在 macOS“文件与文件夹”中允许 SKM 访问。")
                                    .font(.callout)
                                    .foregroundStyle(.secondary)
                                Spacer()
                                Button("打开系统设置…", systemImage: "gearshape", action: openFilePrivacySettings)
                            }
                        }
                    }
                    .padding(8)
                }

                if model.projects.isEmpty {
                    ContentUnavailableView {
                        Label("没有已注册项目", systemImage: "folder")
                    } description: {
                        Text("请先在主窗口的 Projects 中添加项目。")
                    }
                    .frame(minHeight: 260)
                } else {
                    LazyVStack(spacing: 10) {
                        ForEach(model.projects) { project in
                            let status = ProjectAccessStatus(project: project)
                            HStack(spacing: 14) {
                                Image(systemName: status.symbol)
                                    .font(.title3)
                                    .foregroundStyle(status.color)
                                    .frame(width: 38, height: 38)
                                    .background(.quaternary, in: RoundedRectangle(cornerRadius: 9))
                                    .accessibilityHidden(true)

                                VStack(alignment: .leading, spacing: 4) {
                                    Text(project.id).bold()
                                    Text(project.path)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                        .lineLimit(1)
                                        .textSelection(.enabled)
                                    Label(status.title, systemImage: status.symbol)
                                        .font(.callout)
                                        .foregroundStyle(status.color)
                                    Text(status.detail)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }

                                Spacer(minLength: 12)

                                if status == .permissionDenied || status == .unavailable {
                                    Button("重新授权…", systemImage: "folder.badge.plus") {
                                        reauthorize(project)
                                    }
                                    .disabled(model.isLoading)
                                }
                            }
                            .padding(14)
                            .background(.quaternary.opacity(0.45), in: RoundedRectangle(cornerRadius: 12))
                            .accessibilityElement(children: .contain)
                        }
                    }
                }
            }
            .padding(26)
            .frame(maxWidth: 860, alignment: .leading)
        }
        .navigationTitle("文件访问")
    }

    private func refreshAccess() {
        Task { await model.refresh() }
    }

    private func reauthorize(_ project: ProjectModel) {
        let panel = NSOpenPanel()
        panel.title = String(localized: "重新授权项目目录")
        panel.message = String(
            format: String(localized: "请选择已登记的项目目录“%@”。"),
            locale: .current,
            project.id
        )
        panel.prompt = String(localized: "授权")
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.canCreateDirectories = false
        panel.allowsMultipleSelection = false
        panel.directoryURL = URL(fileURLWithPath: project.path).deletingLastPathComponent()
        guard panel.runModal() == .OK, let selectedURL = panel.url else { return }

        let selectedPath = selectedURL.resolvingSymlinksInPath().standardizedFileURL.path
        let registeredPath = URL(fileURLWithPath: project.path).resolvingSymlinksInPath().standardizedFileURL.path
        guard selectedPath == registeredPath else {
            model.errorMessage = String(localized: "请选择当前已登记的同一个项目目录；如项目已移动，请在 Projects 中注销后重新添加。")
            return
        }

        Task {
            await model.refresh()
            guard let refreshed = model.projects.first(where: { $0.id == project.id }),
                  ProjectAccessStatus(project: refreshed).canRead else {
                model.errorMessage = String(localized: "仍无法读取该目录。请检查 macOS 文件与文件夹设置，或 Finder 中的共享与权限。")
                return
            }
            model.announce(String(localized: "项目文件访问已恢复"))
        }
    }

    private func openFilePrivacySettings() {
        guard let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_FilesAndFolders") else { return }
        NSWorkspace.shared.open(url)
    }
}

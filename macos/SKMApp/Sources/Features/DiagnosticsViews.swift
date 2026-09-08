import SwiftUI

/// GeneralSettingsView - 通用设置与系统诊断视图
/// 包含四大模块：
/// 1. 版本与存储：App 版本与个人数据存储路径说明；
/// 2. 同步状态：技能来源、Git 同步、已管理 Agent 等同步状态概览；
/// 3. Doctor 健康检查：展示存储目录、Git 环境、Go Core 状态及各 Agent 安装状态，支持一键重新运行；
/// 4. 诊断报告：生成自动脱敏的系统诊断报告并支持一键复制与导出。
struct GeneralSettingsView: View {
    @Bindable var model: AppModel

    var body: some View {
        Form {
            Section {
                PanelHeader(title: "SKM", subtitle: String(localized: "你的 Skills、Prompts 与项目，一处管理。"), symbol: "square.stack.3d.up")
                    .padding(.vertical, 8)
            }
            Section("版本") {
                LabeledContent("App", value: Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev")
            }

            Section("存储") {
                LabeledContent("个人数据", value: "~/.skm")
                Text("Skill、Prompt 和部署状态与命令行共享。")
                    .font(.caption)
                    .disabled(model.isLoading)
                    .foregroundStyle(.secondary)
            }

            Section("同步状态") {
                LabeledContent("技能来源", value: String(model.sources.count))
                LabeledContent(
                    "个人 Git 同步",
                    value: model.workspace?.configured == true ? String(localized: "已配置") : String(localized: "未配置")
                )
                LabeledContent(
                    "已管理 Agent",
                    value: String(model.agents.filter(\.configured).count)
                )
            }

            Section {
                ForEach(model.doctorChecks) { check in
                    HStack(alignment: .top, spacing: 10) {
                        Image(systemName: check.status == "ok" ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                            .foregroundStyle(check.status == "ok" ? .green : .orange)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(check.name)
                                .fontWeight(.medium)
                            Text(check.message)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .textSelection(.enabled)
                        }
                    }
                    .padding(.vertical, 2)
                }
            } header: {
                HStack {
                    Text("Doctor 健康检查")
                    Spacer()
                    Button("重新运行", systemImage: "arrow.clockwise") {
                        Task { await model.runDoctor() }
                    }
                    .buttonStyle(.borderless)
                    .font(.caption)
                }
            }

            Section("诊断报告") {
                Text("导出内容包含 App/Core 版本、系统架构和脱敏后的 Doctor 结果，不包含公证密钥或 Git 凭据。")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                HStack(spacing: 12) {
                    Button("复制诊断信息", systemImage: "doc.on.doc") {
                        model.copyDiagnostics()
                    }
                    Button("导出到文件…", systemImage: "square.and.arrow.down") {
                        model.exportDiagnostics()
                    }
                }
            }
        }
        .formStyle(.grouped)
        .navigationTitle("通用")
    }
}

/// UpdatesSettingsView - 软件更新设置视图
/// 展示当前版本、Sparkle 2 签名升级状态与手动检查更新。
struct UpdatesSettingsView: View {
    @Bindable var model: AppModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                PanelHeader(title: String(localized: "软件更新"), subtitle: String(localized: "保持最新，获得体验改进与问题修复。"), symbol: "arrow.down.circle", tint: .indigo)
                GroupBox("版本与更新") {
                    HStack {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(model.updateStatus ?? String(localized: "检查是否有新版本可供下载。"))
                            Text("当前版本：\(Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev")")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                        Spacer()
                        Button("检查更新") { Task { await model.checkForUpdates() } }
                            .buttonStyle(.borderedProminent)
                            .disabled(model.isLoading)
                    }
                    .padding(8)
                }
            }
            .readingLayout()
        }
        .navigationTitle("软件更新")
    }
}

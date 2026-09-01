import SwiftUI

/// 系统诊断侧栏列表
struct DiagnosticsSidebarView: View {
    let model: AppModel

    var body: some View {
        List(model.doctorChecks) { check in
            DoctorSidebarRow(check: check)
        }
        .navigationTitle("Diagnostics")
    }
}

private struct DoctorSidebarRow: View {
    let check: DoctorCheck

    var body: some View {
        Label(check.name, systemImage: symbol)
            .foregroundStyle(color)
    }

    private var symbol: String {
        check.status == "ok" ? "checkmark.circle" : "exclamationmark.triangle"
    }

    private var color: Color {
        check.status == "ok" ? .primary : .orange
    }
}

/// DiagnosticsDetailView - 系统诊断详情视图
/// 包含两大模块：
/// 1. Doctor 健康检查：展示存储目录、Git 环境、Go Core 状态及各 Agent 安装状态；
/// 2. 诊断报告导出：生成自动脱敏（隐藏家目录用户名及 URL Token）的系统报告并支持一键复制与导出。
struct DiagnosticsDetailView: View {
    @Bindable var model: AppModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                HStack {
                    Text("系统诊断").font(.largeTitle.bold())
                    Spacer()
                    Button("重新运行", systemImage: "arrow.clockwise") { Task { await model.runDoctor() } }
                }
                GroupBox("Doctor") {
                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(model.doctorChecks) { check in
                            HStack(alignment: .top, spacing: 12) {
                                Image(systemName: check.status == "ok" ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                                    .foregroundStyle(check.status == "ok" ? .green : .orange)
                                VStack(alignment: .leading, spacing: 3) {
                                    Text(check.name).fontWeight(.medium)
                                    Text(check.message).font(.caption).foregroundStyle(.secondary).textSelection(.enabled)
                                }
                            }
                            .padding(.vertical, 10)
                            if check.id != model.doctorChecks.last?.id { Divider() }
                        }
                    }
                    .padding(8)
                }
                GroupBox("诊断报告") {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("导出内容包含 App/Core 版本、系统架构和脱敏后的 Doctor 结果，不包含公证密钥或 Git 凭据。")
                            .foregroundStyle(.secondary)
                        HStack {
                            Button("复制诊断信息", systemImage: "doc.on.doc") { model.copyDiagnostics() }
                            Button("导出到文件…", systemImage: "square.and.arrow.down") { model.exportDiagnostics() }
                        }
                    }
                    .padding(8)
                }
            }
            .padding(26)
            .frame(maxWidth: 820, alignment: .leading)
        }
        .navigationTitle("系统诊断")
    }
}

/// UpdatesSettingsView - 软件更新设置视图
/// 展示当前版本、Sparkle 2 签名升级状态与手动检查更新。
struct UpdatesSettingsView: View {
    @Bindable var model: AppModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                Text("软件更新").font(.largeTitle.bold())
                GroupBox("版本与更新") {
                    HStack {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(model.updateStatus ?? String(localized: "Sparkle 2 可验证并安装签名更新；未配置发布公钥时回退为 GitHub Releases 版本检查。"))
                            Text("当前版本：\(Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev")")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                        Spacer()
                        Button("检查更新") { Task { await model.checkForUpdates() } }
                            .buttonStyle(.borderedProminent)
                    }
                    .padding(8)
                }
            }
            .padding(26)
            .frame(maxWidth: 820, alignment: .leading)
        }
        .navigationTitle("软件更新")
    }
}

import SwiftUI

struct ProjectsListView: View {
    @Bindable var model: AppModel

    var body: some View {
        Group {
            if model.projects.isEmpty {
                ContentUnavailableView("没有已注册项目", systemImage: "folder", description: Text("首版提供项目状态查看；注册与部署写操作将在下一阶段开放。"))
            } else {
                List(model.projects, selection: $model.selectedProjectID) { project in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(project.id).fontWeight(.medium)
                            Spacer()
                            Image(systemName: project.exists ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                                .foregroundStyle(project.exists ? .green : .orange)
                        }
                        Text(project.path).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                    }
                    .padding(.vertical, 4)
                    .tag(project.id)
                }
            }
        }
        .navigationTitle("Projects")
    }
}

struct ProjectDetailView: View {
    let model: AppModel

    var body: some View {
        if let id = model.selectedProjectID, let project = model.projects.first(where: { $0.id == id }) {
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text(project.id).font(.largeTitle.bold())
                        Text(project.path).foregroundStyle(.secondary).textSelection(.enabled)
                    }
                    HStack(spacing: 12) {
                        MetricCard(title: "Skills", value: project.skillCount.description, symbol: "square.stack.3d.up")
                        MetricCard(title: "Activations", value: project.activationCount.description, symbol: "bolt")
                        MetricCard(title: "Agents", value: project.agentCounts.count.description, symbol: "cpu")
                    }
                    GroupBox {
                        Label("第一版仅展示项目扫描与部署摘要。项目注册、迁移、Link/Copy/Vendor 和冲突处理将在 Phase 2 接入同一 Core Bridge。", systemImage: "info.circle")
                            .foregroundStyle(.secondary)
                            .padding(6)
                    }
                }
                .padding(26)
                .frame(maxWidth: 820, alignment: .leading)
            }
        } else {
            ContentUnavailableView("选择一个项目", systemImage: "folder")
        }
    }
}

struct MetricCard: View {
    let title: String
    let value: String
    let symbol: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(title, systemImage: symbol).foregroundStyle(.secondary)
            Text(value).font(.title.bold()).monospacedDigit()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(.quaternary.opacity(0.5), in: RoundedRectangle(cornerRadius: 12))
    }
}

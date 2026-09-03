import SwiftUI

/// 从个人 Skill 库选择内容并部署到当前项目。
struct ProjectSkillImportSheet: View {
    @Environment(\.dismiss) private var dismiss
    let model: AppModel
    let project: RegisteredProject
    @State private var selectedSkillID: String
    @State private var selectedAgentIDs: Set<String>
    @State private var deploymentMode = "symlink"

    private var supportedAgents: [AgentModel] {
        model.agents.filter(\.supported)
    }

    private var selectedSkill: SkillSummary? {
        model.skills.first(where: { $0.id == selectedSkillID })
    }

    init(model: AppModel, project: RegisteredProject) {
        self.model = model
        self.project = project
        _selectedSkillID = State(initialValue: model.skills.first?.id ?? "")

        let configuredAgents = model.agents.filter { $0.supported && $0.configured }.map(\.id)
        let detectedAgents = model.agents.filter { $0.supported && $0.detected }.map(\.id)
        _selectedAgentIDs = State(initialValue: Set(configuredAgents.isEmpty ? detectedAgents : configuredAgents))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            VStack(alignment: .leading, spacing: 6) {
                Label("从我的 Skill 里导入", systemImage: "square.and.arrow.down")
                    .font(.title2.bold())
                Text("选择一个 Skill 和要使用它的 Agent。提交前会先展示文件变更预览。")
                    .foregroundStyle(.secondary)
            }

            if model.skills.isEmpty {
                ContentUnavailableView(
                    "我的 Skill 为空",
                    systemImage: "square.stack.3d.up.slash",
                    description: Text("请先在 Skills 中添加内容，再返回项目导入。")
                )
                .frame(maxWidth: .infinity, minHeight: 220)
            } else {
                Form {
                    Section("Skill") {
                        Picker("选择 Skill", selection: $selectedSkillID) {
                            ForEach(model.skills) { skill in
                                Text(skill.name).tag(skill.id)
                            }
                        }

                        if let selectedSkill {
                            VStack(alignment: .leading, spacing: 5) {
                                Text(selectedSkill.name).font(.headline)
                                if !selectedSkill.description.isEmpty {
                                    Text(selectedSkill.description)
                                        .font(.callout)
                                        .foregroundStyle(.secondary)
                                }
                                if !selectedSkill.tags.isEmpty {
                                    Text(selectedSkill.tags.joined(separator: " · "))
                                        .font(.caption)
                                        .foregroundStyle(.tertiary)
                                }
                            }
                            .padding(.vertical, 4)
                        }
                    }

                    Section("导入方式") {
                        Picker("导入方式", selection: $deploymentMode) {
                            Text("软链接").tag("symlink")
                            Text("复制").tag("copy")
                        }
                        .pickerStyle(.segmented)

                        Label(deploymentHelp, systemImage: deploymentMode == "symlink" ? "link" : "doc.on.doc")
                            .font(.callout)
                            .foregroundStyle(.secondary)
                    }

                    Section("使用此 Skill 的 Agent") {
                        if supportedAgents.isEmpty {
                            Label("没有可用的 Agent，请先在设置中配置。", systemImage: "exclamationmark.triangle")
                                .foregroundStyle(.secondary)
                        } else {
                            LazyVGrid(columns: [GridItem(.adaptive(minimum: 180), alignment: .leading)], spacing: 10) {
                                ForEach(supportedAgents) { agent in
                                    AgentSelectionButton(
                                        agent: agent,
                                        isSelected: selectedAgentIDs.contains(agent.id),
                                        action: { toggleAgent(agent.id) }
                                    )
                                }
                            }
                        }
                    }
                }
                .formStyle(.grouped)
            }

            Spacer(minLength: 0)

            HStack {
                Text(String(format: String(localized: "将导入到 %@"), locale: .current, project.id))
                    .font(.callout)
                    .foregroundStyle(.secondary)
                Spacer()
                Button("取消", role: .cancel, action: dismiss.callAsFunction)
                Button("预览导入", action: previewImport)
                    .buttonStyle(.borderedProminent)
                    .keyboardShortcut(.defaultAction)
                    .disabled(selectedSkillID.isEmpty || selectedAgentIDs.isEmpty || model.isLoading)
            }
        }
        .padding(24)
        .frame(minWidth: 600, idealWidth: 640, minHeight: 620, idealHeight: 640)
    }

    private var deploymentHelp: String {
        if deploymentMode == "symlink" {
            return String(localized: "软链接：跟随个人 Library 保持同步，库中改动会自动反映到项目中。")
        }
        return String(localized: "复制：创建独立副本，项目可脱离 SKM 单独使用，后续不随 Library 自动更新。")
    }

    private func toggleAgent(_ id: String) {
        if selectedAgentIDs.contains(id) {
            selectedAgentIDs.remove(id)
        } else {
            selectedAgentIDs.insert(id)
        }
    }

    private func previewImport() {
        Task {
            await model.deployProject(
                project: project.id,
                skill: selectedSkillID,
                agents: selectedAgentIDs.sorted(),
                mode: deploymentMode,
                dryRun: true
            )
            if model.projectDeploymentPreview != nil {
                dismiss()
            }
        }
    }
}

private struct AgentSelectionButton: View {
    let agent: AgentModel
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 10) {
                Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                    .foregroundStyle(isSelected ? Color.accentColor : .secondary)
                    .font(.title3)
                AgentIconView(agentId: agent.id, isCustom: agent.custom, size: 24)
                VStack(alignment: .leading, spacing: 2) {
                    Text(agent.name)
                        .foregroundStyle(.primary)
                    Text(agent.detected ? String(localized: "已检测") : String(localized: "未检测"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer(minLength: 0)
            }
            .padding(10)
            .contentShape(Rectangle())
            .background(
                isSelected ? Color.accentColor.opacity(0.1) : Color.primary.opacity(0.035),
                in: RoundedRectangle(cornerRadius: 10)
            )
            .overlay {
                RoundedRectangle(cornerRadius: 10)
                    .stroke(isSelected ? Color.accentColor.opacity(0.45) : Color.primary.opacity(0.08))
            }
        }
        .buttonStyle(.plain)
        .accessibilityLabel(agent.name)
        .accessibilityValue(isSelected ? String(localized: "已选择") : String(localized: "未选择"))
    }
}

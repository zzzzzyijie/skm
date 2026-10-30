import AppKit
import SwiftUI

/// Registered projects are the children of the Projects navigation row.
struct ProjectSidebar: View {
    @Bindable var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            ForEach(model.projects) { project in
                projectRow(project)
            }
            if model.projects.isEmpty && !model.isLoading {
                Text("暂无项目")
                    .font(.callout).foregroundStyle(.secondary)
                    .padding(.leading, SKMDesign.sidebarRowHorizontalPadding + SKMDesign.sidebarIconWidth + 9)
                    .padding(.trailing, SKMDesign.sidebarRowHorizontalPadding)
                    .frame(minHeight: SKMDesign.sidebarChildRowHeight, alignment: .leading)
            }
        }
        .onChange(of: model.projects, initial: true) { _, _ in model.reconcileProjectSelection() }
    }

    private func projectRow(_ project: ProjectModel) -> some View {
        let isSelected = model.section == .projects && model.selectedProjectID == project.id
        return Button {
            model.section = .projects
            model.selectedProjectID = project.id
        } label: {
            HStack(spacing: 9) {
                Image(systemName: "folder").frame(width: SKMDesign.sidebarIconWidth).foregroundStyle(.secondary).accessibilityHidden(true)
                Text(project.id).lineLimit(1).truncationMode(.tail)
                Spacer(minLength: 4)
            }
            .font(.callout)
            .padding(.horizontal, SKMDesign.sidebarRowHorizontalPadding)
            .frame(minHeight: SKMDesign.sidebarChildRowHeight)
            .frame(maxWidth: .infinity, alignment: .leading)
            .contentShape(Rectangle())
        }
        .background(isSelected ? SKMDesign.libraryRowSelection : .clear, in: RoundedRectangle(cornerRadius: 6))
        .fontWeight(isSelected ? .semibold : .regular)
        .buttonStyle(SelectionCardButtonStyle(isStatic: true))
        .help(project.path)
        .accessibilityAddTraits(isSelected ? .isSelected : [])
        .accessibilityIdentifier("project-row-\(project.id)")
        .contextMenu {
            Button("在 Finder 中显示") { NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: project.path) }
            Divider()
            Button("注销项目", role: .destructive) {
                model.section = .projects
                model.selectedProjectID = project.id
                model.request(.deleteSelection)
            }
        }
    }
}

/* global App */

// ===== i18n =====
var translations = {
    en: {
        'nav.library': 'My Skills', 'nav.prompts': 'Prompts', 'nav.projects': 'Projects', 'nav.settings': 'Settings', 'nav.openSettings': 'Open settings', 'nav.openMenu': 'Open navigation',
        'view.label': 'Display mode', 'view.grid': 'Grid', 'view.list': 'List',
        'settings.title': 'Settings', 'settings.intro': 'Personalize how SKM looks and reads on this device.', 'settings.language': 'Language', 'settings.languageDesc': 'Choose the language used throughout the interface.', 'settings.english': 'English', 'settings.chinese': '简体中文', 'settings.appearance': 'Appearance', 'settings.darkMode': 'Dark mode', 'settings.darkModeDesc': 'Use the darker color theme for comfortable low-light viewing.',
        'settings.general': 'General', 'settings.gitSync': 'Git sync', 'settings.gitIntro': 'Manage your personal workspace and existing external Skill sources.', 'settings.gitSources': 'Configured sources', 'settings.gitNone': 'No Git sources configured', 'settings.gitNoneDesc': 'Add one from My Skills → Add Skill → Repository.', 'settings.gitRemove': 'Remove source', 'settings.gitRemoveTitle': 'Remove Git source', 'settings.gitRemoveConfirm': 'Remove the “{0}” Git source binding?', 'settings.gitRemoveNote': 'The cached checkout will be removed. Imported Library Skills and snapshots are retained.', 'settings.gitRemoved': 'Git source removed', 'settings.gitUpdatedAt': 'Updated {0}', 'settings.gitDefaultRef': 'Default branch', 'settings.gitSkillCount': '{0} Skill(s)',
        'workspace.title': 'Personal workspace', 'workspace.desc': 'Synchronize your local Skills and Prompts through one private Git repository on every computer.', 'workspace.url': 'Workspace repository', 'workspace.ref': 'Branch', 'workspace.root': 'Repository subdirectory (optional)', 'workspace.rootHint': 'Only independent local Skills and local Prompts are published. Agent deployments, device paths, caches, and credentials stay local.', 'workspace.connect': 'Connect and verify', 'workspace.connecting': 'Verifying workspace…', 'workspace.connected': 'Workspace connected', 'workspace.notConfigured': 'No personal workspace configured', 'workspace.lastSync': 'Last sync {0}', 'workspace.neverSynced': 'Not synced yet', 'workspace.externalTitle': 'External Skill sources', 'workspace.externalDesc': 'Read-only repositories that supply additional Skills. Their contents are not duplicated into your personal workspace.',
        'sync.action': 'Sync', 'sync.open': 'Sync Git sources', 'sync.running': 'Syncing Git sources…', 'sync.configureFirst': 'Configure a Git source before syncing.', 'sync.success': 'Synced {0} source(s) and refreshed {1} Skill(s).', 'sync.partial': 'Sync completed with {0} failed source(s).', 'sync.failed': 'Git sync failed', 'sync.results': 'Sync results', 'sync.sourceUpdated': '{0} Skill(s) refreshed', 'sync.deploymentFailed': 'Sources were updated, but managed deployments could not be refreshed: {0}',
        'sync.workspaceOpen': 'Sync Skills and Prompts', 'sync.workspaceRunning': 'Synchronizing workspace…', 'sync.workspaceConfigure': 'Configure a personal workspace before syncing.', 'sync.workspacePreview': 'Workspace sync preview', 'sync.workspaceSummary': '{0} upload(s), {1} download(s), {2} deletion(s)', 'sync.workspaceEmpty': 'This computer and the remote workspace are already in sync.', 'sync.workspaceConfirm': 'Sync now', 'sync.workspaceConflict': '{0} conflict(s) must be resolved before syncing.', 'sync.workspaceUpload': 'Publish local version', 'sync.workspaceDownload': 'Use remote version', 'sync.workspaceUseLocal': 'Keep local state', 'sync.workspaceUseRemote': 'Use remote state', 'sync.workspaceDeleteLocal': 'Remove from this computer', 'sync.workspaceDeleteRemote': 'Publish deletion', 'sync.workspaceDone': 'Skills and Prompts synchronized.', 'sync.workspaceRevision': 'Revision {0}', 'sync.kindSkill': 'Skill', 'sync.kindPrompt': 'Prompt', 'sync.kindSource': 'Source', 'sync.reason.created-both': 'Created differently on both computers.', 'sync.reason.changed-both': 'Changed differently locally and remotely.', 'sync.reason.deleted-local-changed-remote': 'Deleted locally but changed remotely.', 'sync.reason.changed-local-deleted-remote': 'Changed locally but deleted remotely.', 'sync.reason.enabled-skill-delete': 'This Skill is enabled on this computer. Disable it before accepting the remote deletion.', 'sync.reason.enabled-source-delete': 'Skills from this source are enabled on this computer. Disable them before accepting the remote deletion.',
        'prompt.title': 'Prompts', 'prompt.count': '{0} Prompt(s)', 'prompt.new': 'New Prompt', 'prompt.import': 'Import PROMPT.md', 'prompt.search': 'Search Prompts', 'prompt.searchPlaceholder': 'Search by name, ID, description, or source', 'prompt.all': 'All', 'prompt.noPrompts': 'No Prompts yet', 'prompt.noPromptsDesc': 'Create a reusable Prompt template or import a PROMPT.md file.', 'prompt.edit': 'Edit', 'prompt.export': 'Export', 'prompt.remove': 'Remove', 'prompt.newTitle': 'Create Prompt', 'prompt.editorTitle': 'Edit Prompt', 'prompt.nameLabel': 'Name', 'prompt.namePlaceholder': 'e.g. code-review', 'prompt.nameHint': 'Use lowercase letters, numbers, and hyphens.', 'prompt.nameLocked': 'The name cannot be changed after creation.', 'prompt.descriptionLabel': 'Description', 'prompt.descriptionPlaceholder': 'Describe when and why to use this Prompt', 'prompt.tagsLabel': 'Tags', 'prompt.tagsPlaceholder': 'e.g. review, coding', 'prompt.contentLabel': 'Prompt content', 'prompt.contentPlaceholder': 'Enter the complete Prompt you want to reuse and copy…', 'prompt.editorHint': 'Fill in the details and Prompt content. SKM will generate the PROMPT.md metadata for you.', 'prompt.validate': 'Validate', 'prompt.valid': 'Prompt is valid', 'prompt.notValidated': 'Not validated', 'prompt.unsaved': 'Unsaved changes', 'prompt.lines': '{0} lines', 'prompt.characters': '{0} characters', 'prompt.shortcuts': '⌘/Ctrl+S to save · ⌘/Ctrl+Enter to validate · Tab inserts two spaces', 'prompt.save': 'Save Prompt', 'prompt.created': 'Prompt created', 'prompt.updated': 'Prompt updated', 'prompt.removed': 'Prompt removed', 'prompt.imported': 'PROMPT.md loaded', 'prompt.copy': 'Copy Prompt', 'prompt.copied': 'Prompt copied to the clipboard', 'prompt.copiedShort': 'Copied', 'prompt.noVariables': 'This Prompt has no variables.', 'prompt.confirmRemoveTitle': 'Remove Prompt', 'prompt.confirmRemove': 'Remove “{0}” from the Prompt Library?', 'prompt.confirmRemoveNote': 'The managed Prompt snapshot will be deleted. This cannot be undone.', 'prompt.loadFailed': 'Failed to load Prompts', 'prompt.invalidFile': 'Choose a PROMPT.md or Markdown file.',
        'prompt.required': 'Please enter {0}',
        'lib.title': 'My Skills', 'lib.addSkill': 'Add Skill', 'lib.manageAgents': 'Agent management', 'lib.addAgentTitle': 'Agent management', 'lib.saveAgents': 'Save', 'lib.all': 'All', 'lib.search': 'Search Skills',
        'lib.summaryLabel': 'Library overview', 'lib.totalSkills': 'Total Skills', 'lib.gitSources': 'Git Sources', 'lib.tags': 'Tags', 'lib.added': 'Added', 'lib.hash': 'Hash',
        'lib.searchPlaceholder': 'Search by name, ID, description, or source', 'lib.manageTags': 'Manage tags',
        'lib.selectTags': 'Select tags', 'lib.saveTags': 'Save tags', 'lib.savingTags': 'Saving…', 'lib.tagsUpdated': 'Tags updated', 'lib.tagsSaved': 'Saved', 'lib.tagsUnsaved': 'Unsaved changes', 'lib.tagSelectionCount': '{0} selected', 'lib.newTag': 'New tag', 'lib.createTag': 'Create tag', 'lib.createAndSelectTag': 'Create & select', 'lib.quickAddTag': 'Create a tag without leaving this Skill', 'lib.tagCreated': 'Tag created and selected', 'lib.tagDeleted': 'Tag deleted', 'lib.tagNamePlaceholder': 'e.g. development', 'lib.defaultTag': 'Default', 'lib.tagUsage': '{0} item(s)', 'lib.defaultTagLocked': 'The default tag cannot be deleted.', 'lib.tagInUse': 'Remove this tag from every Skill and Prompt before deleting it.', 'lib.noManagedTags': 'Create a tag before selecting one.',
        'lib.noSkills': 'No Skills found', 'lib.noSkillsDesc': 'Add a Skill to your personal Library with the button above.',
        'lib.skillPath': 'Local source', 'lib.tagsComma': 'Tags (comma-separated)',
        'lib.chooseSkillPath': 'Choose a Skill folder or .zip archive', 'lib.choosePath': 'Choose folder', 'lib.chooseZIP': 'Choose .zip', 'lib.localImportHint': 'Import one Skill folder, or a .zip containing exactly one Skill.', 'lib.skillTagsPlaceholder': 'Skill tags',
        'lib.importLocal': 'Local import', 'lib.importGit': 'Repository', 'lib.importCommand': 'Install command', 'lib.gitSourceName': 'Source name (optional)',
        'lib.gitInput': 'Git repository', 'lib.commandInput': 'Install command', 'lib.commandHint': 'Paste an npx skills add command. SKM parses the command and imports its Git source.', 'lib.gitNamePlaceholder': 'Generated automatically',
        'lib.gitUrl': 'Git repository URL', 'lib.gitRef': 'Ref (branch, tag, or commit)', 'lib.import': 'Import',
        'lib.importedSource': 'Imported {0} Skill(s) from {1}', 'lib.updateSource': 'Update source',
        'lib.updatingSource': 'Updating source...', 'lib.gitRequired': 'Git repository is required', 'lib.commandRequired': 'Install command is required',
        'lib.agentClaude': 'Claude Code', 'lib.agentCodex': 'Codex', 'lib.agentsUpdated': 'Agents updated', 'lib.agentScanUpdated': 'Device scan refreshed', 'lib.scanAgents': 'Scan device', 'lib.detectedAgents': 'Detected on this device', 'lib.otherAgents': 'Other supported Agents', 'lib.customAgents': 'Custom Agents', 'lib.detectedAgent': 'Detected', 'lib.notDetectedAgent': 'Not detected', 'lib.noDetectedAgents': 'No known Agent directory was detected. You can still select a supported Agent or add a custom one.', 'lib.noManagedAgents': 'No managed Agents yet', 'lib.customAgent': 'Custom Agent', 'lib.addCustomAgent': 'Add custom Agent', 'lib.agentId': 'Agent ID', 'lib.agentName': 'Display name', 'lib.agentPath': 'User Skill directory', 'lib.agentIcon': 'Icon (optional)', 'lib.chooseIcon': 'Choose icon', 'lib.deleteAgent': 'Delete Agent',
        'lib.cancel': 'Cancel', 'lib.remove': 'Remove',
        'lib.confirmRemoveTitle': 'Remove Skill', 'lib.confirmRemove': 'Are you sure you want to remove',
        'lib.confirmRemoveNote': 'The Skill must be disabled first.',
        'lib.addedSuccess': 'Skill added successfully', 'lib.pathRequired': 'Path is required', 'lib.loadFailed': 'Failed to load Library',
        'lib.addSkillTitle': 'Add Skill to Library', 'lib.enabled': 'Enabled', 'lib.disabled': 'Disabled', 'lib.removed': 'Removed',
        'lib.details': 'Skill details', 'lib.description': 'Description', 'lib.source': 'Source', 'lib.local': 'Local', 'lib.location': 'Location',
        'lib.healthAvailable': 'Available', 'lib.healthChanged': 'Source changed', 'lib.healthMissing': 'Missing on this branch', 'lib.healthUnreachable': 'Project unavailable', 'lib.healthInvalid': 'Invalid source', 'lib.usingFallback': 'Using last known snapshot', 'lib.followingProject': 'Following project', 'lib.detach': 'Make independent', 'lib.detachConfirm': 'Create an independent Library copy from the current source or last known snapshot?', 'lib.detached': 'Skill is now independent', 'lib.effectivePath': 'Active path',
        'lib.path': 'Stored path', 'lib.revision': 'Revision', 'lib.addTag': 'Add tag', 'lib.tagName': 'Tag name',
        'lib.tagAdded': 'Tag added', 'lib.tagRemoved': 'Tag removed', 'lib.renameTag': 'Rename tag',
        'lib.newTagName': 'New tag name', 'lib.renamedTag': 'Tag renamed', 'lib.noTags': 'No tags yet',
        'lib.viewDetails': 'View details', 'lib.close': 'Close', 'lib.skillCount': '{0} Skill(s)', 'lib.content': 'Skill content', 'lib.noContent': 'No Skill content', 'lib.skillOverview': 'Overview', 'lib.metadata': 'Metadata',
		'lib.edit': 'Edit', 'lib.editorTitle': 'Edit Skill', 'lib.editorContent': 'Complete SKILL.md', 'lib.editorHint': 'The Skill name cannot change. Saving creates a new immutable snapshot; scripts, references, and assets remain unchanged.', 'lib.validate': 'Validate', 'lib.valid': 'Skill is valid', 'lib.notValidated': 'Not validated', 'lib.unsaved': 'Unsaved changes', 'lib.shortcuts': '⌘/Ctrl+S to save · ⌘/Ctrl+Enter to validate · Tab inserts two spaces', 'lib.saveSkill': 'Save Skill', 'lib.updated': 'Skill updated', 'lib.deploymentWarning': 'The Skill was saved, but user deployments could not be refreshed: {0}', 'lib.editConflict': 'This Skill changed after the editor opened. Reload it before saving again.',
        'tag.general': 'General',
        'act.title': 'Activation Status', 'act.applyPlan': 'Apply Plan', 'act.quickEnable': 'Quick Enable',
        'act.planDigest': 'Plan digest', 'act.noActivations': 'No activations',
        'act.noActivationsDesc': 'Enable Skills from the Library to see them here.',
        'act.status': 'Status', 'act.agent': 'Agent', 'act.placement': 'Placement', 'act.skill': 'Skill',
        'act.target': 'Target', 'act.mode': 'Mode', 'act.actions': 'Actions', 'act.disable': 'Disable',
        'act.planApplied': 'Plan applied successfully', 'act.selectSkills': 'Select Skills', 'act.agents': 'Agents',
        'act.selectOne': 'Select at least one Skill', 'act.selectAgent': 'Select at least one Agent', 'act.loadFailed': 'Failed to load status',
        'act.disabled': 'Disabled', 'act.for': 'for', 'act.enabledN': 'Enabled {0} Skill(s)',
        'act.linkMode': 'Link mode', 'act.modeAuto': 'Auto', 'act.modeSymlink': 'Symlink', 'act.modeCopy': 'Copy',
        'act.summary': '{0} activation operation(s)', 'act.noChanges': 'Everything is up to date',
        'loading': 'Loading...', 'loadingSKM': 'Loading SKM...',
        'proj.title': 'Projects', 'proj.add': 'Add project', 'proj.empty': 'No projects registered',
        'proj.emptyDesc': 'Register a local project to deploy Library Skills into its Agent directories.',
        'proj.path': 'Project path', 'proj.chooseProjectPath': 'Choose a project folder in Finder',
        'proj.register': 'Register project', 'proj.registered': 'Project registered', 'proj.list': 'Registered projects',
        'proj.ready': 'Ready', 'proj.missing': 'Missing', 'proj.activations': 'Activations', 'proj.skills': 'Skills', 'proj.skill': 'Skill',
        'proj.agents': 'Agents', 'proj.claudeCode': 'Claude Code', 'proj.codex': 'Codex', 'proj.allAgents': 'All agents', 'proj.mode': 'Mode', 'proj.actions': 'Actions', 'proj.addSkill': 'Add Skill',
        'proj.scan': 'Scan project', 'proj.scanTitle': 'Project Skills', 'proj.scanDesc': 'Skills found in the project Agent directories, grouped by Skill.', 'proj.lastScan': 'Last scan',
        'proj.scanOk': 'Valid', 'proj.scanWarning': 'Check', 'proj.scanError': 'Unavailable', 'proj.managed': 'Managed', 'proj.external': 'Detected',
        'proj.noScannedSkills': 'No Skills found in the project Agent directories.', 'proj.noFilteredSkills': 'No Skills found for this Agent.', 'proj.noDescription': 'No description',
        'proj.link': 'Link', 'proj.copy': 'Copy', 'proj.unlink': 'Unlink',
        'proj.confirmUnlinkTitle': 'Unlink Skill', 'proj.confirmUnlink': 'Unlink', 'proj.confirmUnlinkDesc': 'Unlink "{0}" from this project? The managed deployment will be removed.',
        'proj.forceUnlinkTitle': 'Remove modified Skill', 'proj.forceUnlink': 'Remove anyway', 'proj.forceUnlinkDesc': '"{0}" has been modified in this project. Removing it will permanently delete the project copy.',
        'proj.unregister': 'Remove', 'proj.noSkills': 'No Skills deployed to this project.',
        'proj.noSkillsDesc': 'Choose a Library Skill below to link or copy it into the project.',
        'proj.selectSkill': 'Select a Library Skill', 'proj.chooseAgent': 'Select at least one Agent',
        'proj.chooseSkill': 'Select a Skill', 'proj.unregistered': 'Project removed',
        'proj.deployed': 'Skill deployed', 'proj.deployedWithSkipped': 'Skill added; already present in {0}, so those Agents were skipped.', 'proj.alreadyExists': '“{0}” already exists for the selected Agents. Nothing was changed.', 'proj.modeAlreadyExists': '“{0}” already uses {1} mode. Unlink it before switching modes.', 'proj.unlinked': 'Skill unlinked', 'proj.confirmUnregister': 'Remove this project? Managed Skills must be unlinked first.',
        'proj.noProjects': 'No projects', 'proj.pathRequired': 'Project path is required', 'proj.viewDetails': 'View details', 'proj.skillDetails': 'Project Skill details', 'proj.skillPath': 'Skill path', 'proj.metadata': 'Metadata', 'proj.content': 'Skill content', 'proj.noContent': 'No Skill content',
        'proj.migrate': 'Add to My Skills', 'proj.migrateTitle': 'Add project Skill to My Skills', 'proj.migrateSource': 'Source Agent', 'proj.migrateLink': 'Follow project', 'proj.migrateCopy': 'Copy', 'proj.migrateLinkDesc': 'Use the project directory as the live source. Switching to a branch without this Skill will use the last known snapshot until the source returns.', 'proj.migrateCopyDesc': 'Recommended. Create an independent snapshot in My Skills that is unaffected by project or branch changes.', 'proj.removeAfterCopy': 'Remove the project originals after copying (move)', 'proj.removeAfterCopyNote': 'All Agent copies must be identical and unmanaged. This cannot be undone.', 'proj.migrateConfirm': 'Add to My Skills', 'proj.migrated': 'Skill added to My Skills',
        'proj.removeSkill': 'Remove', 'proj.confirmRemoveSkillTitle': 'Remove project Skill', 'proj.confirmRemoveSkillDesc': 'Remove "{0}" from this project?', 'proj.confirmRemoveSkillNote': 'Every copy in this project’s Agent directories will be permanently deleted. This cannot be undone.', 'proj.projectSkillRemoved': 'Project Skill removed',
    },
    zh: {
        'nav.library': '技能', 'nav.prompts': '提示词', 'nav.projects': '项目', 'nav.settings': '设置', 'nav.openSettings': '打开设置', 'nav.openMenu': '打开导航',
        'view.label': '显示方式', 'view.grid': '网格', 'view.list': '列表',
        'settings.title': '设置', 'settings.intro': '自定义这台设备上的 SKM 显示与语言偏好。', 'settings.language': '语言', 'settings.languageDesc': '选择界面中使用的语言。', 'settings.english': 'English', 'settings.chinese': '简体中文', 'settings.appearance': '外观', 'settings.darkMode': '暗黑模式', 'settings.darkModeDesc': '在弱光环境中使用更舒适的深色主题。',
        'settings.general': '通用', 'settings.gitSync': 'Git 同步', 'settings.gitIntro': '管理个人工作区与已有的外部 Skill 来源。', 'settings.gitSources': '已配置来源', 'settings.gitNone': '尚未配置 Git 来源', 'settings.gitNoneDesc': '请从“Skill → 添加 Skill → 仓库来源”添加。', 'settings.gitRemove': '移除来源', 'settings.gitRemoveTitle': '移除 Git 来源', 'settings.gitRemoveConfirm': '确定移除 Git 来源“{0}”吗？', 'settings.gitRemoveNote': '缓存的仓库副本会被删除；已导入的 Library Skill 和快照会保留。', 'settings.gitRemoved': 'Git 来源已移除', 'settings.gitUpdatedAt': '更新于 {0}', 'settings.gitDefaultRef': '默认分支', 'settings.gitSkillCount': '{0} 个 Skill',
        'workspace.title': '个人工作区', 'workspace.desc': '通过一个私有 Git 仓库，在多台电脑之间同步本地 Skill 和 Prompt。', 'workspace.url': '工作区仓库', 'workspace.ref': '分支', 'workspace.root': '仓库子目录（可选）', 'workspace.rootHint': '只发布独立的本地 Skill 和本地 Prompt；Agent 部署、本机路径、缓存和凭据始终留在当前电脑。', 'workspace.connect': '连接并校验', 'workspace.connecting': '正在校验工作区…', 'workspace.connected': '个人工作区已连接', 'workspace.notConfigured': '尚未配置个人工作区', 'workspace.lastSync': '上次同步 {0}', 'workspace.neverSynced': '尚未同步', 'workspace.externalTitle': '外部 Skill 来源', 'workspace.externalDesc': '以只读方式提供额外 Skill 的仓库，其内容不会重复写入个人工作区。',
        'sync.action': '同步', 'sync.open': '同步 Git 来源', 'sync.running': '正在同步 Git 来源…', 'sync.configureFirst': '请先配置 Git 来源再同步。', 'sync.success': '已同步 {0} 个来源，并刷新 {1} 个 Skill。', 'sync.partial': '同步完成，但有 {0} 个来源失败。', 'sync.failed': 'Git 同步失败', 'sync.results': '同步结果', 'sync.sourceUpdated': '已刷新 {0} 个 Skill', 'sync.deploymentFailed': '来源已更新，但托管部署刷新失败：{0}',
        'sync.workspaceOpen': '同步 Skill 和 Prompt', 'sync.workspaceRunning': '正在同步个人工作区…', 'sync.workspaceConfigure': '请先配置个人工作区再同步。', 'sync.workspacePreview': '工作区同步预览', 'sync.workspaceSummary': '上传 {0} 项，下载 {1} 项，删除 {2} 项', 'sync.workspaceEmpty': '当前电脑与远程工作区已经一致。', 'sync.workspaceConfirm': '立即同步', 'sync.workspaceConflict': '存在 {0} 个冲突，处理后才能同步。', 'sync.workspaceUpload': '发布本地版本', 'sync.workspaceDownload': '使用远程版本', 'sync.workspaceUseLocal': '保留本地状态', 'sync.workspaceUseRemote': '使用远程状态', 'sync.workspaceDeleteLocal': '从当前电脑移除', 'sync.workspaceDeleteRemote': '发布删除', 'sync.workspaceDone': 'Skill 和 Prompt 已完成同步。', 'sync.workspaceRevision': '版本 {0}', 'sync.kindSkill': 'Skill', 'sync.kindPrompt': 'Prompt', 'sync.kindSource': 'Git 来源', 'sync.reason.created-both': '两台电脑分别创建了不同内容。', 'sync.reason.changed-both': '本地和远程都修改了此内容。', 'sync.reason.deleted-local-changed-remote': '本地已删除，但远程又有修改。', 'sync.reason.changed-local-deleted-remote': '本地已有修改，但远程已删除。', 'sync.reason.enabled-skill-delete': '此 Skill 正在当前电脑启用；请先禁用，再接受远程删除。', 'sync.reason.enabled-source-delete': '此来源的 Skill 正在当前电脑启用；请先禁用，再接受远程删除。',
        'prompt.title': 'Prompt', 'prompt.count': '{0} 个 Prompt', 'prompt.new': '新建 Prompt', 'prompt.import': '导入 PROMPT.md', 'prompt.search': '搜索 Prompt', 'prompt.searchPlaceholder': '按名称、ID、描述或来源搜索', 'prompt.all': '全部', 'prompt.noPrompts': '暂无 Prompt', 'prompt.noPromptsDesc': '创建可复用的 Prompt 模板，或导入 PROMPT.md 文件。', 'prompt.edit': '编辑', 'prompt.export': '导出', 'prompt.remove': '删除', 'prompt.newTitle': '新建 Prompt', 'prompt.editorTitle': '编辑 Prompt', 'prompt.nameLabel': '名称', 'prompt.namePlaceholder': '例如：code-review', 'prompt.nameHint': '使用小写字母、数字和连字符。', 'prompt.nameLocked': '名称创建后不可修改。', 'prompt.descriptionLabel': '描述', 'prompt.descriptionPlaceholder': '简单说明这个 Prompt 的用途和适用场景', 'prompt.tagsLabel': '标签', 'prompt.tagsPlaceholder': '例如：review, coding', 'prompt.contentLabel': 'Prompt 内容', 'prompt.contentPlaceholder': '输入需要复用并复制到剪贴板的完整 Prompt 内容…', 'prompt.editorHint': '填写基本信息和 Prompt 内容即可，SKM 会自动生成 PROMPT.md 元数据。', 'prompt.validate': '校验', 'prompt.valid': 'Prompt 格式有效', 'prompt.notValidated': '尚未校验', 'prompt.unsaved': '有未保存修改', 'prompt.lines': '{0} 行', 'prompt.characters': '{0} 个字符', 'prompt.shortcuts': '⌘/Ctrl+S 保存 · ⌘/Ctrl+Enter 校验 · Tab 插入两个空格', 'prompt.save': '保存 Prompt', 'prompt.created': 'Prompt 已创建', 'prompt.updated': 'Prompt 已更新', 'prompt.removed': 'Prompt 已删除', 'prompt.imported': '已载入 PROMPT.md', 'prompt.copy': '复制 Prompt', 'prompt.copied': 'Prompt 已复制到设备剪贴板', 'prompt.copiedShort': '已复制', 'prompt.noVariables': '这个 Prompt 不包含变量。', 'prompt.confirmRemoveTitle': '删除 Prompt', 'prompt.confirmRemove': '确定从 Prompt 库删除“{0}”吗？', 'prompt.confirmRemoveNote': '对应的托管快照将被删除，此操作不可撤销。', 'prompt.loadFailed': '加载 Prompt 失败', 'prompt.invalidFile': '请选择 PROMPT.md 或 Markdown 文件。',
        'prompt.required': '请填写{0}',
        'lib.title': 'Skill', 'lib.addSkill': '添加 Skill', 'lib.manageAgents': 'Agent 管理', 'lib.addAgentTitle': 'Agent 管理', 'lib.saveAgents': '保存', 'lib.all': '全部', 'lib.search': '搜索 Skill',
        'lib.summaryLabel': 'Skill 库概览', 'lib.totalSkills': 'Skill 总数', 'lib.gitSources': '来源数量', 'lib.tags': '标签', 'lib.added': '添加时间', 'lib.hash': '哈希',
        'lib.searchPlaceholder': '按名称、ID、描述或来源搜索', 'lib.manageTags': '管理标签',
        'lib.selectTags': '选择标签', 'lib.saveTags': '保存标签', 'lib.savingTags': '正在保存…', 'lib.tagsUpdated': '标签已更新', 'lib.tagsSaved': '已保存', 'lib.tagsUnsaved': '有未保存修改', 'lib.tagSelectionCount': '已选择 {0} 个', 'lib.newTag': '新建标签', 'lib.createTag': '创建标签', 'lib.createAndSelectTag': '新建并选择', 'lib.quickAddTag': '无需离开当前 Skill，直接创建新标签', 'lib.tagCreated': '标签已创建并选中', 'lib.tagDeleted': '标签已删除', 'lib.tagNamePlaceholder': '例如：development', 'lib.defaultTag': '默认', 'lib.tagUsage': '{0} 个引用', 'lib.defaultTagLocked': '默认标签不能删除。', 'lib.tagInUse': '请先从所有 Skill 和 Prompt 中取消此标签。', 'lib.noManagedTags': '请先在标签管理中创建标签。',
        'lib.noSkills': '暂无 Skill', 'lib.noSkillsDesc': '点击上方按钮将 Skill 添加到个人库。',
        'lib.skillPath': '本地来源', 'lib.tagsComma': '标签（逗号分隔）',
        'lib.chooseSkillPath': '选择 Skill 文件夹或 .zip 压缩包', 'lib.choosePath': '选择文件夹', 'lib.chooseZIP': '选择 .zip', 'lib.localImportHint': '支持一个 Skill 文件夹，或仅包含一个 Skill 的 .zip 压缩包。', 'lib.skillTagsPlaceholder': 'skill所属标签',
        'lib.importLocal': '本地导入', 'lib.importGit': '仓库来源', 'lib.importCommand': '安装命令', 'lib.gitSourceName': '来源名称（可选）',
        'lib.gitInput': 'Git 仓库地址', 'lib.commandInput': '安装命令', 'lib.commandHint': '粘贴 npx skills add 命令；SKM 会解析命令并导入其中的 Git 来源。', 'lib.gitNamePlaceholder': '自动生成',
        'lib.gitUrl': 'Git 仓库地址', 'lib.gitRef': '引用（分支、Tag 或提交）', 'lib.import': '导入',
        'lib.importedSource': '已从 {1} 导入 {0} 个 Skill', 'lib.updateSource': '更新来源',
        'lib.updatingSource': '正在更新来源...', 'lib.gitRequired': '请填写 Git 仓库地址', 'lib.commandRequired': '请填写安装命令',
        'lib.agentClaude': 'Claude Code', 'lib.agentCodex': 'Codex', 'lib.agentsUpdated': 'Agent 已更新', 'lib.agentScanUpdated': '已重新扫描设备', 'lib.scanAgents': '扫描设备', 'lib.detectedAgents': '本机已检测到', 'lib.otherAgents': '其他支持的 Agent', 'lib.customAgents': '自定义 Agent', 'lib.detectedAgent': '已检测', 'lib.notDetectedAgent': '未检测到', 'lib.noDetectedAgents': '暂未检测到已知 Agent，仍可手动选择支持的 Agent，或添加自定义 Agent。', 'lib.noManagedAgents': '尚未管理任何 Agent', 'lib.customAgent': '自定义 Agent', 'lib.addCustomAgent': '添加自定义 Agent', 'lib.agentId': 'Agent ID', 'lib.agentName': '显示名称', 'lib.agentPath': '用户级 Skill 目录', 'lib.agentIcon': '图标（可选）', 'lib.chooseIcon': '选择图标', 'lib.deleteAgent': '删除 Agent',
        'lib.cancel': '取消', 'lib.remove': '移除',
        'lib.confirmRemoveTitle': '移除 Skill', 'lib.confirmRemove': '确定要移除',
        'lib.confirmRemoveNote': '必须先禁用该 Skill。',
        'lib.addedSuccess': 'Skill 添加成功', 'lib.pathRequired': '路径不能为空', 'lib.loadFailed': '加载 Skill 库失败',
        'lib.addSkillTitle': '添加 Skill 到库', 'lib.enabled': '已启用', 'lib.disabled': '已禁用', 'lib.removed': '已移除',
        'lib.details': 'Skill 详情', 'lib.description': '描述', 'lib.source': '来源', 'lib.local': '本地', 'lib.location': '位置',
        'lib.healthAvailable': '可用', 'lib.healthChanged': '来源已变更', 'lib.healthMissing': '当前skill源缺失', 'lib.healthUnreachable': '项目不可达', 'lib.healthInvalid': '来源无效', 'lib.usingFallback': '正在使用最后快照', 'lib.followingProject': '跟随项目', 'lib.detach': '转为独立副本', 'lib.detachConfirm': '确认从当前来源或最后快照创建独立的 Library 副本？', 'lib.detached': 'Skill 已转为独立副本', 'lib.effectivePath': '当前使用路径',
        'lib.path': '存储路径', 'lib.revision': '版本', 'lib.addTag': '添加标签', 'lib.tagName': '标签名称',
        'lib.tagAdded': '标签已添加', 'lib.tagRemoved': '标签已移除', 'lib.renameTag': '重命名标签',
        'lib.newTagName': '新标签名称', 'lib.renamedTag': '标签已重命名', 'lib.noTags': '暂无标签',
        'lib.viewDetails': '查看详情', 'lib.close': '关闭', 'lib.skillCount': '{0} 个 Skill', 'lib.content': 'Skill 内容', 'lib.noContent': '暂无 Skill 内容', 'lib.skillOverview': '概览', 'lib.metadata': '元数据',
		'lib.edit': '编辑', 'lib.editorTitle': '编辑 Skill', 'lib.editorContent': '完整 SKILL.md', 'lib.editorHint': 'Skill 名称不可修改。保存时会创建新的不可变快照；scripts、references 和 assets 会保持不变。', 'lib.validate': '校验', 'lib.valid': 'Skill 格式有效', 'lib.notValidated': '尚未校验', 'lib.unsaved': '有未保存修改', 'lib.shortcuts': '⌘/Ctrl+S 保存 · ⌘/Ctrl+Enter 校验 · Tab 插入两个空格', 'lib.saveSkill': '保存 Skill', 'lib.updated': 'Skill 已更新', 'lib.deploymentWarning': 'Skill 已保存，但用户级部署刷新失败：{0}', 'lib.editConflict': '打开编辑器后此 Skill 已发生变化，请重新加载后再保存。',
        'tag.general': '通用',
        'act.title': '激活状态', 'act.applyPlan': '应用计划', 'act.quickEnable': '快速启用',
        'act.planDigest': '计划摘要', 'act.noActivations': '暂无激活',
        'act.noActivationsDesc': '从 Skill 库启用后会显示在这里。',
        'act.status': '状态', 'act.agent': '代理', 'act.placement': '位置', 'act.skill': 'Skill',
        'act.target': '目标', 'act.mode': '模式', 'act.actions': '操作', 'act.disable': '禁用',
        'act.planApplied': '计划已成功应用', 'act.selectSkills': '选择 Skill', 'act.agents': '代理',
        'act.selectOne': '请至少选择一个 Skill', 'act.selectAgent': '请至少选择一个 Agent', 'act.loadFailed': '加载状态失败',
        'act.disabled': '已禁用', 'act.for': '', 'act.enabledN': '已启用 {0} 个 Skill',
        'act.linkMode': '链接模式', 'act.modeAuto': '自动', 'act.modeSymlink': '软链接', 'act.modeCopy': '复制',
        'act.summary': '{0} 个激活操作', 'act.noChanges': '当前状态已是最新',
        'loading': '加载中...', 'loadingSKM': '正在加载 SKM...',
        'proj.title': 'Project', 'proj.add': '添加项目', 'proj.empty': '暂无项目',
        'proj.emptyDesc': '登记本机项目后，可将 Skill 库中的 Skill 部署到项目 Agent 目录。',
        'proj.path': '项目路径', 'proj.chooseProjectPath': '在 Finder 中选择项目文件夹',
        'proj.register': '添加项目', 'proj.registered': '项目已添加', 'proj.list': '项目列表',
        'proj.ready': '可用', 'proj.missing': '不存在', 'proj.activations': '激活数', 'proj.skills': '个 Skill', 'proj.skill': 'Skill',
        'proj.agents': 'Agent', 'proj.claudeCode': 'Claude Code', 'proj.codex': 'Codex', 'proj.allAgents': '全部 Agent', 'proj.mode': '模式', 'proj.actions': '操作', 'proj.addSkill': '添加 Skill',
        'proj.scan': '扫描项目', 'proj.scanTitle': '项目中的 Skill', 'proj.scanDesc': '扫描项目 Agent 目录，并按 Skill 合并展示。', 'proj.lastScan': '上次扫描',
        'proj.scanOk': '正常', 'proj.scanWarning': '需检查', 'proj.scanError': '不可用', 'proj.managed': 'SKM 管理', 'proj.external': '已检测',
        'proj.noScannedSkills': '项目 Agent 目录中暂无 Skill。', 'proj.noFilteredSkills': '该 Agent 下暂无 Skill。', 'proj.noDescription': '暂无描述',
        'proj.link': '软链接', 'proj.copy': '复制', 'proj.unlink': '解绑',
        'proj.confirmUnlinkTitle': '确认解绑 Skill', 'proj.confirmUnlink': '确认解绑', 'proj.confirmUnlinkDesc': '确定要从当前项目解绑“{0}”吗？对应的托管部署将被移除。',
        'proj.forceUnlinkTitle': '移除已修改的 Skill', 'proj.forceUnlink': '仍要移除', 'proj.forceUnlinkDesc': '“{0}”已在项目中被修改。继续操作将永久删除该项目副本。',
        'proj.unregister': '移除', 'proj.noSkills': '该项目暂无已部署 Skill。',
        'proj.noSkillsDesc': '从下方选择 Library Skill，将它软链或复制到项目中。',
        'proj.selectSkill': '选择 Library Skill', 'proj.chooseAgent': '请至少选择一个 Agent',
        'proj.chooseSkill': '请选择 Skill', 'proj.unregistered': '项目已移除',
        'proj.deployed': 'Skill 已部署', 'proj.deployedWithSkipped': 'Skill 已添加；{0} 中已存在，因此已自动跳过。', 'proj.alreadyExists': '“{0}”已存在于所选 Agent，无需重复添加。', 'proj.modeAlreadyExists': '“{0}”已使用{1}模式；如需切换，请先解绑。', 'proj.unlinked': 'Skill 已解绑', 'proj.confirmUnregister': '确认移除该项目？必须先解绑已管理的 Skill。',
        'proj.noProjects': '暂无项目', 'proj.pathRequired': '项目路径不能为空', 'proj.viewDetails': '查看详情', 'proj.skillDetails': '项目 Skill 详情', 'proj.skillPath': 'Skill 路径', 'proj.metadata': '元数据', 'proj.content': 'Skill 内容', 'proj.noContent': '暂无 Skill 内容',
        'proj.migrate': '迁移到 Skill', 'proj.migrateTitle': '迁移项目 Skill', 'proj.migrateSource': '来源 Agent', 'proj.migrateLink': '跟随项目', 'proj.migrateCopy': '复制', 'proj.migrateLinkDesc': '以项目目录为实时来源；切换到不含此 Skill 的分支时，将临时使用最后快照，直到来源恢复。', 'proj.migrateCopyDesc': '推荐。在Skill中创建独立快照，不受项目修改或分支切换影响。', 'proj.removeAfterCopy': '复制成功后移除项目原件（移动）', 'proj.removeAfterCopyNote': '仅当所有 Agent 副本内容一致且未由 SKM 托管时可用，此操作不可撤销。', 'proj.migrateConfirm': '添加到 Skill', 'proj.migrated': '已添加到 Skill',
        'proj.removeSkill': '移除', 'proj.confirmRemoveSkillTitle': '移除项目 Skill', 'proj.confirmRemoveSkillDesc': '确定要从当前项目移除“{0}”吗？', 'proj.confirmRemoveSkillNote': '该 Skill 在项目所有 Agent 目录中的副本都会被永久删除，此操作不可撤销。', 'proj.projectSkillRemoved': '项目 Skill 已移除',
    }
};

var currentLang = (function () {
    var saved = localStorage.getItem('skm-lang');
    if (saved && translations[saved]) return saved;
    var nav = (navigator.language || 'en').toLowerCase();
    if (nav.startsWith('zh')) return 'zh';
    return 'en';
})();

var currentTheme = document.documentElement.dataset.theme === 'light' ? 'light' : 'dark';
var settingsState = { section: 'general', sources: [], workspace: null };
var gitSyncBusy = false;
var workspaceConflictChoices = {};

function t(key) {
    return (translations[currentLang] && translations[currentLang][key]) || translations.en[key] || key;
}

function setLang(lang) {
    if (!translations[lang] || lang === currentLang) return;
    suppressTransitionsForSwap(function () {
        currentLang = lang;
        localStorage.setItem('skm-lang', lang);
        document.documentElement.lang = lang;
        document.documentElement.dataset.language = lang;
        updateStaticLabels();
        repaintCurrentPageLanguage();
        refreshOpenSettingsLanguage();
    });
}

function suppressTransitionsForSwap(update) {
    var style = document.createElement('style');
    style.textContent = '*,*::before,*::after{transition:none !important}';
    document.head.appendChild(style);
    update();
    void document.body.offsetHeight;
    requestAnimationFrame(function () {
        requestAnimationFrame(function () { style.remove(); });
    });
}

function repaintCurrentPageLanguage() {
    var container = document.getElementById('main-content');
    var loading = container && container.querySelector('.loading-full');
    if (loading) {
        var label = loading.querySelector('p');
        if (label) label.textContent = t('loading');
        return;
    }
    if (App.currentPage === 'library' && typeof paintLibrary === 'function') paintLibrary();
    if (App.currentPage === 'prompts' && typeof paintPrompts === 'function') paintPrompts();
    if (App.currentPage === 'projects' && typeof paintProjects === 'function') paintProjects();
}

function refreshOpenSettingsLanguage() {
    var modal = document.querySelector('.modal.settings-modal');
    if (!modal) return;
    modal.querySelector('.modal-title').textContent = t('settings.title');
    var close = modal.querySelector('.modal-close');
    close.setAttribute('aria-label', t('lib.close'));
    var action = modal.querySelector('.modal-actions [data-close-modal]');
    if (action) action.textContent = t('lib.close');
    var nav = modal.querySelector('.settings-nav');
    nav.setAttribute('aria-label', t('settings.title'));
    var labels = { general: 'settings.general', git: 'settings.gitSync' };
    nav.querySelectorAll('[data-settings-section]').forEach(function (button) {
        button.querySelector('span').textContent = t(labels[button.dataset.settingsSection]);
    });
    renderSettingsSection();
    requestAnimationFrame(function () {
        var selected = modal.querySelector('[data-settings-lang="' + currentLang + '"]');
        if (selected) selected.focus({ preventScroll: true });
    });
}

function updateStaticLabels() {
    var map = { library: 'nav.library', prompts: 'nav.prompts', projects: 'nav.projects' };
    document.querySelectorAll('.nav-item').forEach(function (item) {
        var key = map[item.dataset.page];
        if (key) item.querySelector('.nav-label').textContent = t(key);
    });
    var settingsLabel = document.querySelector('#settings-toggle .nav-label');
    if (settingsLabel) settingsLabel.textContent = t('nav.settings');
    document.querySelectorAll('#settings-toggle, #mobile-settings-toggle').forEach(function (button) {
        button.setAttribute('aria-label', t('nav.openSettings'));
        button.title = t('nav.settings');
    });
    document.querySelectorAll('#sync-toggle, #mobile-sync-toggle').forEach(function (button) {
        button.setAttribute('aria-label', t('sync.workspaceOpen'));
        button.title = t('sync.action');
    });
    var menuToggle = document.getElementById('menu-toggle');
    if (menuToggle) menuToggle.setAttribute('aria-label', t('nav.openMenu'));
}

function setTheme(theme) {
    theme = theme === 'light' ? 'light' : 'dark';
    if (theme === currentTheme) return;
    suppressTransitionsForSwap(function () {
        currentTheme = theme;
        localStorage.setItem('skm-theme', theme);
        document.documentElement.dataset.theme = theme;
        document.documentElement.style.colorScheme = theme;
    });
}

function displayTag(tag) {
    return tag === 'general' ? t('tag.general') : tag;
}

function displaySource(source) {
    return source === 'local' ? t('lib.local') : source;
}

function collectionViewMode(scope) {
    return localStorage.getItem('skm-' + scope + '-view') === 'list' ? 'list' : 'grid';
}

function collectionViewSwitcherMarkup(scope, activeMode) {
    var modes = [
        { id: 'grid', icon: 'grid', label: t('view.grid') },
        { id: 'list', icon: 'list', label: t('view.list') },
    ];
    return '<div class="collection-view-switcher" data-collection-view-scope="' + escapeHtml(scope) + '" role="group" aria-label="' +
        escapeHtml(t('view.label')) + '">' + modes.map(function (mode) {
            var active = activeMode === mode.id;
            return '<button class="collection-view-option' + (active ? ' active' : '') + '" type="button" data-collection-view="' + mode.id +
                '" aria-pressed="' + active + '" title="' + escapeHtml(mode.label) + '">' + uiIcon(mode.icon) + '<span>' + escapeHtml(mode.label) + '</span></button>';
        }).join('') + '</div>';
}

function bindCollectionViewSwitcher(scope, state, repaint) {
    var switcher = document.querySelector('[data-collection-view-scope="' + scope + '"]');
    if (!switcher) return;
    switcher.querySelectorAll('[data-collection-view]').forEach(function (button) {
        button.addEventListener('click', function () {
            var mode = button.dataset.collectionView === 'list' ? 'list' : 'grid';
            if (state.viewMode === mode) return;
            state.viewMode = mode;
            localStorage.setItem('skm-' + scope + '-view', mode);
            repaint();
            requestAnimationFrame(function () {
                var next = document.querySelector('[data-collection-view-scope="' + scope + '"] [data-collection-view="' + mode + '"]');
                if (next) next.focus({ preventScroll: true });
            });
        });
    });
}

function uiIcon(name) {
    var paths = {
        plus: '<path d="M12 5v14M5 12h14"/>',
        tags: '<path d="M3.5 6.5v5.2c0 .7.3 1.3.8 1.8l6.2 6.2a2.5 2.5 0 0 0 3.5 0l5.7-5.7a2.5 2.5 0 0 0 0-3.5L13.5 4.3a2.5 2.5 0 0 0-1.8-.8H6.5a3 3 0 0 0-3 3Z"/><circle cx="8" cy="8" r="1.25"/>',
        settings: '<path d="M4 7h10M18 7h2M4 17h2M10 17h10M14 4v6M6 14v6"/>',
        globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/>',
        moon: '<path d="M20 15.4A8.5 8.5 0 0 1 8.6 4a8.5 8.5 0 1 0 11.4 11.4Z"/>',
        search: '<circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/>',
        eye: '<path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"/><circle cx="12" cy="12" r="2.5"/>',
        trash: '<path d="M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5"/>',
        refresh: '<path d="M20 7v5h-5M4 17v-5h5"/><path d="M6.1 8.5A7 7 0 0 1 18.8 7L20 12M4 12l1.2 5A7 7 0 0 0 17.9 15.5"/>',
        folder: '<path d="M3.5 6.5A2.5 2.5 0 0 1 6 4h4l2 2h6A2.5 2.5 0 0 1 20.5 8.5v8A2.5 2.5 0 0 1 18 19H6a2.5 2.5 0 0 1-2.5-2.5v-10Z"/>',
        archive: '<path d="M5 4h14v4H5zM6 8h12v12H6zM10 12h4"/>',
        library: '<path d="M4 6.5A2.5 2.5 0 0 1 6.5 4H20v15.5H6.5A2.5 2.5 0 0 1 4 17V6.5Z"/><path d="M4 17a2.5 2.5 0 0 1 2.5-2.5H20M8 8h8"/>',
        check: '<path d="m5 12 4.5 4.5L19 7"/>',
        alert: '<path d="M10.3 4.2 2.8 17.3A1.8 1.8 0 0 0 4.4 20h15.2a1.8 1.8 0 0 0 1.6-2.7L13.7 4.2a2 2 0 0 0-3.4 0ZM12 9v4M12 17h.01"/>',
        link: '<path d="M9.5 14.5 14.5 9.5M7.5 16.5l-1 1a3.5 3.5 0 0 1-5-5l4-4a3.5 3.5 0 0 1 5 0M16.5 7.5l1-1a3.5 3.5 0 0 1 5 5l-4 4a3.5 3.5 0 0 1-5 0"/>',
        lock: '<rect x="5" y="10" width="14" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>',
        copy: '<rect x="8" y="8" width="11" height="11" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>',
        grid: '<rect x="4" y="4" width="6" height="6" rx="1"/><rect x="14" y="4" width="6" height="6" rx="1"/><rect x="4" y="14" width="6" height="6" rx="1"/><rect x="14" y="14" width="6" height="6" rx="1"/>',
        list: '<path d="M9 6h11M9 12h11M9 18h11"/><rect x="4" y="5" width="2" height="2" rx=".5"/><rect x="4" y="11" width="2" height="2" rx=".5"/><rect x="4" y="17" width="2" height="2" rx=".5"/>',
        sparkles: '<path d="m12 3 1.1 3.2a4.2 4.2 0 0 0 2.7 2.7L19 10l-3.2 1.1a4.2 4.2 0 0 0-2.7 2.7L12 17l-1.1-3.2a4.2 4.2 0 0 0-2.7-2.7L5 10l3.2-1.1a4.2 4.2 0 0 0 2.7-2.7L12 3ZM19 16l.5 1.5L21 18l-1.5.5L19 20l-.5-1.5L17 18l1.5-.5L19 16Z"/>'
    };
    return '<svg class="ui-icon" viewBox="0 0 24 24" aria-hidden="true">' + (paths[name] || paths.sparkles) + '</svg>';
}

// ===== Version =====
function truncateVersion(v) {
    if (!v) return 'dev';
    v = v.replace(/^v/, '');
    var m = v.match(/^(\d+\.\d+\.\d+)/);
    return m ? m[1] : v;
}

// ===== API Client =====
var api = {
    async request(method, url, data) {
        var options = { method: method, headers: { 'Accept': 'application/json' } };
        if (data !== undefined) {
            options.headers['Content-Type'] = 'application/json';
            options.body = JSON.stringify(data);
        }
        var res = await fetch(url, options);
        if (!res.ok) {
            var body = await res.json().catch(function () { return {}; });
			var error = new Error(body.error || 'Request failed: ' + res.status);
			error.status = res.status;
			throw error;
        }
        if (res.status === 204) return null;
        return res.json();
    },
    get: function (url) { return this.request('GET', url); },
    post: function (url, data) { return this.request('POST', url, data); },
    put: function (url, data) { return this.request('PUT', url, data); },
    del: function (url) { return this.request('DELETE', url); },
};

// ===== Toast =====
function showToast(message, type) {
    type = type || 'success';
    var container = document.getElementById('toast-container');
    var toast = document.createElement('div');
    toast.className = 'toast toast-' + type;
    toast.setAttribute('role', type === 'error' ? 'alert' : 'status');
    toast.innerHTML = '<span class="toast-icon" aria-hidden="true">' + uiIcon(type === 'error' ? 'alert' : (type === 'info' ? 'sparkles' : 'check')) + '</span><span>' + escapeHtml(message) + '</span>';
    container.appendChild(toast);
    setTimeout(function () {
        toast.classList.add('removing');
        setTimeout(function () { toast.remove(); }, 300);
    }, 3000);
}

// ===== Utilities =====
function formatDate(dateStr) {
    if (!dateStr) return '—';
    var d = new Date(dateStr);
    if (isNaN(d.getTime())) return '—';
    if (currentLang === 'zh') {
        return d.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' });
    }
    return d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
}

function shortHash(hash) {
    if (!hash) return '—';
    return hash.substring(0, 12);
}

function shortRevision(rev) {
    if (!rev) return '—';
    if (rev.length > 12) return rev.substring(0, 12);
    return rev;
}

function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function statusBadgeClass(status) {
    var map = {
        'ok': 'badge-ok', 'create': 'badge-create', 'unchanged': 'badge-unchanged',
        'replace-managed': 'badge-replace-managed', 'broken': 'badge-broken',
        'conflict-unmanaged': 'badge-conflict-unmanaged', 'error': 'badge-error',
        'optional': 'badge-optional', 'not-created': 'badge-not-created', 'warning': 'badge-warning',
    };
    return map[status] || 'badge-muted';
}

function confirmationMarkup(message, note, tone) {
    tone = tone || 'danger';
    var icon = tone === 'danger' ? 'alert' : 'sparkles';
    return '<div class="confirm-dialog confirm-dialog-' + tone + '"><div class="confirm-dialog-icon" aria-hidden="true">' + uiIcon(icon) +
        '</div><div class="confirm-dialog-copy"><p class="confirm-dialog-message">' + escapeHtml(message) + '</p>' +
        (note ? '<p class="confirm-dialog-note">' + escapeHtml(note) + '</p>' : '') + '</div></div>';
}

// ===== Modal =====
var modalReturnFocus = null;

function showModal(title, contentHtml, actions) {
    var existing = document.querySelector('.modal-overlay');
    if (existing) existing.remove();
    if (!modalReturnFocus || !modalReturnFocus.isConnected) modalReturnFocus = document.activeElement;
    var overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML =
        '<div class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">' +
            '<div class="modal-header"><div class="modal-title" id="modal-title">' + escapeHtml(title) + '</div>' +
            '<button class="icon-btn modal-close" type="button" data-close-modal aria-label="' + escapeHtml(t('lib.close')) + '"><svg class="ui-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg></button></div>' +
            '<div class="modal-body">' + contentHtml + '</div>' +
            '<div class="modal-actions">' + (actions || '') + '</div>' +
        '</div>';
    overlay.addEventListener('click', function (e) {
        if (e.target === overlay) closeModal();
    });
    document.body.appendChild(overlay);
    document.body.classList.add('modal-open');
    overlay.querySelectorAll('[data-close-modal]').forEach(function (button) {
        button.addEventListener('click', closeModal);
    });
    document.addEventListener('keydown', onModalEsc);
    var focusTarget = overlay.querySelector('input, select, button');
    if (focusTarget) focusTarget.focus();
}

function closeModal() {
    var overlay = document.querySelector('.modal-overlay');
    if (overlay && !overlay.classList.contains('is-closing')) {
        overlay.classList.add('is-closing');
        setTimeout(function () {
            overlay.remove();
            document.body.classList.remove('modal-open');
            if (modalReturnFocus && modalReturnFocus.isConnected) modalReturnFocus.focus();
            modalReturnFocus = null;
        }, 140);
    }
    document.removeEventListener('keydown', onModalEsc);
}

function onModalEsc(e) {
    if (e.key === 'Escape') closeModal();
}

// ===== Settings =====
function openSettings(section) {
    settingsState.section = section === 'git' ? 'git' : 'general';
    var content = '<div class="settings-layout"><nav class="settings-nav" aria-label="' + escapeHtml(t('settings.title')) + '">' +
        '<button type="button" data-settings-section="general">' + uiIcon('settings') + '<span>' + escapeHtml(t('settings.general')) + '</span></button>' +
        '<button type="button" data-settings-section="git">' + uiIcon('refresh') + '<span>' + escapeHtml(t('settings.gitSync')) + '</span></button></nav>' +
        '<div class="settings-content" id="settings-content"></div></div>';
    showModal(t('settings.title'), content, '<button class="btn btn-primary" type="button" data-close-modal>' + t('lib.close') + '</button>');
    document.querySelector('.modal').classList.add('settings-modal');
    document.querySelectorAll('[data-settings-section]').forEach(function (button) {
        button.addEventListener('click', function () {
            settingsState.section = button.dataset.settingsSection;
            renderSettingsSection();
        });
    });
    renderSettingsSection();
}

function renderSettingsSection() {
    var content = document.getElementById('settings-content');
    if (!content) return;
    document.querySelectorAll('[data-settings-section]').forEach(function (button) {
        var selected = button.dataset.settingsSection === settingsState.section;
        button.classList.toggle('active', selected);
        button.setAttribute('aria-current', selected ? 'page' : 'false');
    });
    if (settingsState.section === 'git') {
        content.innerHTML = '<div class="settings-loading"><span class="spinner spinner-sm" aria-hidden="true"></span>' + escapeHtml(t('loading')) + '</div>';
        loadGitSettings();
        return;
    }
    content.innerHTML = settingsGeneralMarkup();
    bindGeneralSettings();
}

function settingsGeneralMarkup() {
    return '<div class="settings-panel"><p class="settings-intro">' + escapeHtml(t('settings.intro')) + '</p>' +
        '<section class="settings-section"><div class="settings-section-heading"><span class="settings-section-icon" aria-hidden="true">' + uiIcon('globe') +
        '</span><div><h3>' + escapeHtml(t('settings.language')) + '</h3><p>' + escapeHtml(t('settings.languageDesc')) + '</p></div></div>' +
        '<div class="settings-segmented" role="group" aria-label="' + escapeHtml(t('settings.language')) + '">' +
        '<button type="button" data-settings-lang="en" aria-pressed="' + String(currentLang === 'en') + '">' + escapeHtml(t('settings.english')) + '</button>' +
        '<button type="button" data-settings-lang="zh" aria-pressed="' + String(currentLang === 'zh') + '">' + escapeHtml(t('settings.chinese')) + '</button></div></section>' +
        '<section class="settings-section"><label class="settings-theme-row" for="settings-dark-mode"><span class="settings-section-heading"><span class="settings-section-icon" aria-hidden="true">' + uiIcon('moon') +
        '</span><span><strong>' + escapeHtml(t('settings.darkMode')) + '</strong><small>' + escapeHtml(t('settings.darkModeDesc')) + '</small></span></span>' +
        '<span class="settings-switch"><input id="settings-dark-mode" type="checkbox"' + (currentTheme === 'dark' ? ' checked' : '') +
        ' aria-label="' + escapeHtml(t('settings.darkMode')) + '"><span class="settings-switch-track" aria-hidden="true"><span class="settings-switch-thumb"></span></span></span></label></section></div>';
}

function bindGeneralSettings() {
    document.querySelectorAll('[data-settings-lang]').forEach(function (button) {
        button.addEventListener('click', function () {
            var lang = button.dataset.settingsLang;
            if (lang === currentLang) return;
            setLang(lang);
        });
    });
    document.getElementById('settings-dark-mode').addEventListener('change', function (event) {
        setTheme(event.target.checked ? 'dark' : 'light');
    });
}

async function loadGitSettings() {
    try {
        var values = await Promise.all([api.get('/api/workspace'), api.get('/api/sources')]);
        settingsState.workspace = values[0] || { configured: false };
        settingsState.sources = values[1] || [];
        if (settingsState.section !== 'git') return;
        var content = document.getElementById('settings-content');
        if (!content) return;
        content.innerHTML = gitSettingsMarkup(settingsState.workspace, settingsState.sources);
        bindGitSettings();
    } catch (err) {
        var content = document.getElementById('settings-content');
        if (content) content.innerHTML = '<div class="inline-empty">' + escapeHtml(err.message) + '</div>';
    }
}

function gitSettingsMarkup(workspace, sources) {
    workspace = workspace || { configured: false };
    var config = workspace.config || {};
    var state = workspace.state || {};
    var workspaceStatus = workspace.configured
        ? '<span class="workspace-status is-connected">' + uiIcon('check') + escapeHtml(t('workspace.connected')) + '</span>'
        : '<span class="workspace-status">' + escapeHtml(t('workspace.notConfigured')) + '</span>';
    var lastSync = state.lastSyncedAt ? t('workspace.lastSync').replace('{0}', formatDate(state.lastSyncedAt)) : t('workspace.neverSynced');
    var rows = sources.map(function (source) {
        var revision = source.revision ? shortRevision(source.revision) : t('settings.gitDefaultRef');
        var updated = source.updatedAt ? t('settings.gitUpdatedAt').replace('{0}', formatDate(source.updatedAt)) : t('settings.gitDefaultRef');
        return '<article class="git-source-row"><div class="git-source-row-main"><div class="git-source-row-title"><strong>' + escapeHtml(source.name) +
            '</strong><span class="badge badge-source">' + escapeHtml(revision) + '</span></div><div class="git-source-url mono">' + escapeHtml(source.url) +
            '</div><small>' + escapeHtml(source.ref || t('settings.gitDefaultRef')) + ' · ' + escapeHtml(updated) + '</small></div>' +
            '<button class="btn btn-danger btn-sm" type="button" data-remove-git-source="' + escapeHtml(source.name) + '">' + uiIcon('trash') + t('settings.gitRemove') + '</button></article>';
    }).join('');
    var sourceList = rows || '<div class="git-source-empty"><strong>' + escapeHtml(t('settings.gitNone')) + '</strong><p>' + escapeHtml(t('settings.gitNoneDesc')) + '</p></div>';
    return '<div class="git-settings"><section class="workspace-card"><div class="git-settings-header"><div><h3>' + escapeHtml(t('workspace.title')) +
        '</h3><p>' + escapeHtml(t('workspace.desc')) + '</p></div>' + workspaceStatus + '</div><div class="workspace-form-grid"><label class="form-group workspace-url-field"><span class="form-label">' +
        escapeHtml(t('workspace.url')) + '</span><input class="input mono" id="settings-workspace-url" autocomplete="off" spellcheck="false" value="' + escapeHtml(config.url || '') +
        '" placeholder="git@github.com:owner/skm-workspace.git"></label><label class="form-group"><span class="form-label">' + escapeHtml(t('workspace.ref')) +
        '</span><input class="input mono" id="settings-workspace-ref" autocomplete="off" spellcheck="false" value="' + escapeHtml(config.ref || 'main') +
        '" placeholder="main"></label><label class="form-group"><span class="form-label">' + escapeHtml(t('workspace.root')) +
        '</span><input class="input mono" id="settings-workspace-root" autocomplete="off" spellcheck="false" value="' + escapeHtml(config.root || '') +
        '" placeholder="skm"></label></div><p class="form-hint">' + escapeHtml(t('workspace.rootHint')) + '</p><div class="workspace-card-footer"><small>' +
        escapeHtml(lastSync) + (state.revision ? ' · ' + escapeHtml(shortRevision(state.revision)) : '') + '</small><button class="btn btn-primary" type="button" id="btn-save-workspace">' +
        uiIcon('link') + t('workspace.connect') + '</button></div></section><section class="git-source-section"><div class="git-settings-header"><div><h3>' +
        escapeHtml(t('workspace.externalTitle')) + '</h3><p>' + escapeHtml(t('workspace.externalDesc')) + '</p></div><span class="git-source-count">' + sources.length +
        '</span></div><span class="form-label">' +
        escapeHtml(t('settings.gitSources')) + '</span><div class="git-source-list">' + sourceList + '</div></section></div>';
}

function bindGitSettings() {
    document.getElementById('btn-save-workspace').addEventListener('click', saveWorkspace);
    document.querySelectorAll('[data-remove-git-source]').forEach(function (button) {
        button.addEventListener('click', function () { confirmRemoveGitSource(button.dataset.removeGitSource); });
    });
}

async function saveWorkspace() {
    var input = document.getElementById('settings-workspace-url');
    var url = input.value.trim();
    if (!url) {
        input.setAttribute('aria-invalid', 'true');
        input.focus();
        showToast(t('workspace.notConfigured'), 'error');
        return;
    }
    var button = document.getElementById('btn-save-workspace');
    button.disabled = true;
    button.innerHTML = '<span class="spinner spinner-sm" aria-hidden="true"></span>' + t('workspace.connecting');
    try {
        await api.put('/api/workspace', {
            url: url,
            ref: document.getElementById('settings-workspace-ref').value.trim() || 'main',
            root: document.getElementById('settings-workspace-root').value.trim(),
        });
        showToast(t('workspace.connected'));
        await loadGitSettings();
    } catch (err) {
        button.disabled = false;
        button.innerHTML = uiIcon('link') + t('workspace.connect');
        showToast(err.message, 'error');
    }
}

function confirmRemoveGitSource(name) {
    var actions = '<button class="btn btn-ghost" type="button" id="btn-cancel-remove-source">' + t('lib.cancel') + '</button>' +
        '<button class="btn btn-danger" type="button" id="btn-confirm-remove-source">' + t('settings.gitRemove') + '</button>';
    showModal(t('settings.gitRemoveTitle'), confirmationMarkup(t('settings.gitRemoveConfirm').replace('{0}', name), t('settings.gitRemoveNote'), 'danger'), actions);
    document.getElementById('btn-cancel-remove-source').addEventListener('click', function () { openSettings('git'); });
    document.getElementById('btn-confirm-remove-source').addEventListener('click', async function () {
        var button = document.getElementById('btn-confirm-remove-source');
        button.disabled = true;
        try {
            await api.del('/api/sources/' + encodeURIComponent(name));
            showToast(t('settings.gitRemoved'));
            openSettings('git');
        } catch (err) {
            button.disabled = false;
            showToast(err.message, 'error');
        }
    });
}

function setGitSyncBusy(busy) {
    gitSyncBusy = busy;
    document.querySelectorAll('#sync-toggle, #mobile-sync-toggle').forEach(function (button) {
        button.disabled = busy;
        button.classList.toggle('is-syncing', busy);
        button.setAttribute('aria-busy', String(busy));
        button.title = busy ? t('sync.workspaceRunning') : t('sync.action');
    });
}

async function runGitSync() {
    if (gitSyncBusy) return;
    setGitSyncBusy(true);
    try {
        var workspace = await api.get('/api/workspace');
        if (!workspace.configured) {
            setGitSyncBusy(false);
            showToast(t('sync.workspaceConfigure'), 'info');
            openSettings('git');
            return;
        }
        var preview = await api.get('/api/workspace/preview');
        showWorkspacePreview(preview);
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        setGitSyncBusy(false);
    }
}

function showWorkspacePreview(preview) {
    workspaceConflictChoices = {};
    var changeRows = (preview.changes || []).map(function (change) {
        var actionKey = {
            upload: 'sync.workspaceUpload', download: 'sync.workspaceDownload',
            'delete-local': 'sync.workspaceDeleteLocal', 'delete-remote': 'sync.workspaceDeleteRemote',
        }[change.action];
        var isConflict = change.action === 'conflict';
        var remoteBlocked = change.reason === 'enabled-skill-delete';
        var conflictActions = isConflict ? '<div class="workspace-conflict-actions"><button class="btn btn-sm btn-secondary" type="button" data-workspace-resolution="local" data-workspace-conflict="' +
            escapeHtml(change.kind + ':' + change.id) + '">' + t('sync.workspaceUseLocal') + '</button><button class="btn btn-sm btn-secondary" type="button" data-workspace-resolution="remote" data-workspace-conflict="' +
            escapeHtml(change.kind + ':' + change.id) + '"' + (remoteBlocked ? ' disabled' : '') + '>' + t('sync.workspaceUseRemote') + '</button></div>' : '';
        var reasonKey = 'sync.reason.' + (change.reason || '');
        var action = isConflict ? (I18N[App.language][reasonKey] ? t(reasonKey) : change.detail) : t(actionKey);
        return '<div class="workspace-change-row ' + (isConflict ? 'is-conflict' : '') + '"><span class="workspace-change-kind">' +
            escapeHtml(t(change.kind === 'skill' ? 'sync.kindSkill' : (change.kind === 'source' ? 'sync.kindSource' : 'sync.kindPrompt'))) + '</span><div><strong>' + escapeHtml(change.name) +
            '</strong><p>' + escapeHtml(action) + '</p>' + conflictActions + '</div></div>';
    }).join('');
    if (!changeRows) changeRows = '<div class="git-source-empty"><strong>' + escapeHtml(t('sync.workspaceEmpty')) + '</strong></div>';
    var summary = '<div class="workspace-sync-summary"><strong>' + escapeHtml(t('sync.workspaceSummary').replace('{0}', preview.uploads || 0).replace('{1}', preview.downloads || 0).replace('{2}', preview.deletes || 0)) +
        '</strong>' + (preview.remoteRevision ? '<span class="mono">' + escapeHtml(t('sync.workspaceRevision').replace('{0}', shortRevision(preview.remoteRevision))) + '</span>' : '') + '</div>';
    if (preview.conflicts) {
        summary += '<div class="workspace-conflict-note">' + uiIcon('alert') + '<span>' + escapeHtml(t('sync.workspaceConflict').replace('{0}', preview.conflicts)) + '</span></div>';
    }
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>';
    actions += '<button class="btn btn-primary" type="button" id="btn-apply-workspace-sync"' + (preview.conflicts ? ' disabled' : '') + '>' + uiIcon('refresh') + t('sync.workspaceConfirm') + '</button>';
    showModal(t('sync.workspacePreview'), summary + '<div class="workspace-change-list">' + changeRows + '</div>', actions);
    document.getElementById('btn-apply-workspace-sync').addEventListener('click', applyWorkspaceSync);
    document.querySelectorAll('[data-workspace-resolution]').forEach(function (button) {
        button.addEventListener('click', function () {
            workspaceConflictChoices[button.dataset.workspaceConflict] = button.dataset.workspaceResolution;
            document.querySelectorAll('[data-workspace-conflict]').forEach(function (candidate) {
                if (candidate.dataset.workspaceConflict !== button.dataset.workspaceConflict) return;
                candidate.classList.toggle('is-selected', candidate === button);
                candidate.setAttribute('aria-pressed', String(candidate === button));
            });
            var resolved = Object.keys(workspaceConflictChoices).length;
            document.getElementById('btn-apply-workspace-sync').disabled = resolved < (preview.conflicts || 0);
        });
    });
}

async function applyWorkspaceSync() {
    var button = document.getElementById('btn-apply-workspace-sync');
    if (button) {
        button.disabled = true;
        button.innerHTML = '<span class="spinner spinner-sm" aria-hidden="true"></span>' + t('sync.workspaceRunning');
    }
    setGitSyncBusy(true);
    try {
        var workspaceResult = await api.post('/api/workspace/sync', { resolutions: workspaceConflictChoices });
        var sources = await api.get('/api/sources') || [];
        var sourceResult = null;
        if (sources.length) sourceResult = await api.post('/api/sync', {});
        if (App.currentPage === 'library' && typeof renderLibrary === 'function') await renderLibrary();
        if (App.currentPage === 'prompts' && typeof renderPrompts === 'function') await renderPrompts();
        closeModal();
        var detail = t('sync.workspaceDone') + (workspaceResult.revision ? ' ' + t('sync.workspaceRevision').replace('{0}', shortRevision(workspaceResult.revision)) : '');
        var sourceWarnings = workspaceResult.sourceWarnings || [];
        if (sourceWarnings.length) detail += ' ' + sourceWarnings.join(' ');
        if (workspaceResult.deploymentError || (sourceResult && sourceResult.failed)) {
            detail += ' ' + (workspaceResult.deploymentError || t('sync.partial').replace('{0}', sourceResult.failed));
            showToast(detail, 'info');
        } else {
            showToast(detail);
        }
    } catch (err) {
        if (button) {
            button.disabled = false;
            button.innerHTML = uiIcon('refresh') + t('sync.workspaceConfirm');
        }
        showToast(err.message, 'error');
    } finally {
        setGitSyncBusy(false);
    }
}

function showGitSyncResult(result) {
    var rows = (result.results || []).map(function (item) {
        var ok = item.status === 'updated';
        var detail = ok ? t('sync.sourceUpdated').replace('{0}', item.skillCount || 0) : item.error;
        return '<div class="sync-result-row ' + (ok ? 'is-success' : 'is-error') + '"><span class="sync-result-icon" aria-hidden="true">' +
            uiIcon(ok ? 'check' : 'alert') + '</span><div><strong>' + escapeHtml(item.name) + '</strong><p>' + escapeHtml(detail) + '</p></div></div>';
    }).join('');
    if (result.deploymentError) {
        rows += '<div class="sync-result-row is-error"><span class="sync-result-icon" aria-hidden="true">' + uiIcon('alert') +
            '</span><div><strong>' + escapeHtml(t('act.title')) + '</strong><p>' + escapeHtml(t('sync.deploymentFailed').replace('{0}', result.deploymentError)) + '</p></div></div>';
    }
    showModal(t('sync.results'), '<div class="sync-result-list">' + rows + '</div>', '<button class="btn btn-primary" type="button" data-close-modal>' + t('lib.close') + '</button>');
}

// ===== Main App =====
var App = {
    currentPage: 'library',

    init: function () {
        document.documentElement.lang = currentLang;
        document.documentElement.dataset.language = currentLang;
        this.setupNav();
        this.setupSettings();
        this.setupGitSync();
        this.setupMobileNav();
        this.loadVersion();
        updateStaticLabels();
        var initialPage = window.location.hash.replace(/^#\/?/, '');
        if (!['library', 'prompts', 'projects'].includes(initialPage)) {
            initialPage = 'library';
            window.history.replaceState(null, '', '#/library');
        }
        this.navigate(initialPage, false);
        window.addEventListener('hashchange', function () {
            var page = window.location.hash.replace(/^#\/?/, '');
            if (!['library', 'prompts', 'projects'].includes(page)) {
                window.history.replaceState(null, '', '#/library');
                page = 'library';
            }
            if (page !== App.currentPage) App.navigate(page, false);
        });
    },

    setupNav: function () {
        var self = this;
        document.querySelectorAll('.nav-item').forEach(function (item) {
            item.addEventListener('click', function () {
                self.navigate(item.dataset.page, true);
            });
        });
    },

    setupSettings: function () {
        document.querySelectorAll('#settings-toggle, #mobile-settings-toggle').forEach(function (button) {
            button.addEventListener('click', function () { openSettings('general'); });
        });
    },

    setupGitSync: function () {
        document.querySelectorAll('#sync-toggle, #mobile-sync-toggle').forEach(function (button) {
            button.addEventListener('click', runGitSync);
        });
    },

    setupMobileNav: function () {
        var toggle = document.getElementById('menu-toggle');
        var backdrop = document.getElementById('sidebar-backdrop');
        function close() {
            document.body.classList.remove('nav-open');
            toggle.setAttribute('aria-expanded', 'false');
        }
        toggle.addEventListener('click', function () {
            var open = document.body.classList.toggle('nav-open');
            toggle.setAttribute('aria-expanded', String(open));
        });
        backdrop.addEventListener('click', close);
        document.querySelectorAll('.nav-item').forEach(function (item) { item.addEventListener('click', close); });
    },

    navigate: function (page, updateHash) {
        this.currentPage = page;
        if (updateHash !== false && window.location.hash !== '#/' + page) window.location.hash = '/' + page;
        document.querySelectorAll('.nav-item').forEach(function (item) {
            var active = item.dataset.page === page;
            item.classList.toggle('active', active);
            if (active) item.setAttribute('aria-current', 'page');
            else item.removeAttribute('aria-current');
        });
        var container = document.getElementById('main-content');
        container.dataset.page = page;
        container.innerHTML = '<div class="loading-full"><div class="spinner"></div><p>' + t('loading') + '</p></div>';
        switch (page) {
            case 'library':    renderLibrary(); break;
            case 'prompts':    renderPrompts(); break;
            case 'projects':   renderProjects(); break;
        }
    },

    loadVersion: function () {
        api.get('/api/version').then(function (data) {
            document.getElementById('version-label').textContent = 'v' + truncateVersion(data.version);
        }).catch(function () {});
    },
};

function isCurrentPage(page) {
    var container = document.getElementById('main-content');
    return container && container.dataset.page === page;
}

// ===== Export =====
window.api = api;
window.App = App;
window.t = t;
window.showToast = showToast;
window.showModal = showModal;
window.closeModal = closeModal;
window.formatDate = formatDate;
window.shortHash = shortHash;
window.shortRevision = shortRevision;
window.escapeHtml = escapeHtml;
window.statusBadgeClass = statusBadgeClass;
window.isCurrentPage = isCurrentPage;
window.uiIcon = uiIcon;
window.confirmationMarkup = confirmationMarkup;
window.collectionViewMode = collectionViewMode;
window.collectionViewSwitcherMarkup = collectionViewSwitcherMarkup;
window.bindCollectionViewSwitcher = bindCollectionViewSwitcher;

document.addEventListener('DOMContentLoaded', function () { App.init(); });

/* global App */

// ===== i18n =====
var translations = {
    en: {
        'nav.dashboard': 'Dashboard', 'nav.library': 'Library', 'nav.projects': 'Projects',
        'dash.title': 'Dashboard', 'dash.totalSkills': 'Total Skills', 'dash.enabled': 'Enabled', 'dash.gitSources': 'Git Sources',
        'dash.recentlyAdded': 'Recently Added', 'dash.id': 'ID', 'dash.tags': 'Tags', 'dash.added': 'Added', 'dash.hash': 'Hash',
        'dash.loadFailed': 'Failed to load dashboard',
        'dash.noRecent': 'No Skills have been added yet.',
        'lib.title': 'Skill Library', 'lib.addSkill': 'Add Skill', 'lib.all': 'All', 'lib.search': 'Search Skills',
        'lib.searchPlaceholder': 'Search by name, ID, description, or source', 'lib.manageTags': 'Manage tags',
        'lib.noSkills': 'No Skills found', 'lib.noSkillsDesc': 'Add a Skill to your personal Library with the button above.',
        'lib.skillPath': 'Skill Path', 'lib.tagsComma': 'Tags (comma-separated)',
        'lib.chooseSkillPath': 'Choose a Skill directory in Finder', 'lib.choosePath': 'Choose directory', 'lib.skillTagsPlaceholder': 'Skill tags',
        'lib.importLocal': 'Local import', 'lib.importGit': 'Repository', 'lib.gitSourceName': 'Source name (optional)',
        'lib.gitInput': 'Repository or install command', 'lib.gitNamePlaceholder': 'Generated automatically',
        'lib.gitUrl': 'Git repository URL', 'lib.gitRef': 'Ref (branch, tag, or commit)', 'lib.import': 'Import',
        'lib.importedSource': 'Imported {0} Skill(s) from {1}', 'lib.updateSource': 'Update source',
        'lib.updatingSource': 'Updating source...', 'lib.gitRequired': 'Repository or install command is required',
        'lib.agentClaude': 'Claude', 'lib.agentCodex': 'Codex',
        'lib.cancel': 'Cancel', 'lib.remove': 'Remove',
        'lib.confirmRemoveTitle': 'Remove Skill', 'lib.confirmRemove': 'Are you sure you want to remove',
        'lib.confirmRemoveNote': 'The Skill must be disabled first.',
        'lib.addedSuccess': 'Skill added successfully', 'lib.pathRequired': 'Path is required', 'lib.loadFailed': 'Failed to load Library',
        'lib.addSkillTitle': 'Add Skill to Library', 'lib.enabled': 'Enabled', 'lib.disabled': 'Disabled', 'lib.removed': 'Removed',
        'lib.details': 'Skill details', 'lib.description': 'Description', 'lib.source': 'Source', 'lib.local': 'Local', 'lib.location': 'Location',
        'lib.path': 'Stored path', 'lib.revision': 'Revision', 'lib.addTag': 'Add tag', 'lib.tagName': 'Tag name',
        'lib.tagAdded': 'Tag added', 'lib.tagRemoved': 'Tag removed', 'lib.renameTag': 'Rename tag',
        'lib.newTagName': 'New tag name', 'lib.renamedTag': 'Tag renamed', 'lib.noTags': 'No tags yet',
        'lib.viewDetails': 'View details', 'lib.close': 'Close', 'lib.skillCount': '{0} Skill(s)', 'lib.content': 'Skill content', 'lib.noContent': 'No Skill content',
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
        'proj.link': 'Link', 'proj.copy': 'Copy', 'proj.symlink': 'Symlink', 'proj.copied': 'Copy', 'proj.unlink': 'Unlink', 'proj.status': 'Refresh status',
        'proj.confirmUnlinkTitle': 'Unlink Skill', 'proj.confirmUnlink': 'Unlink', 'proj.confirmUnlinkDesc': 'Unlink "{0}" from this project? The managed deployment will be removed.',
        'proj.forceUnlinkTitle': 'Remove modified Skill', 'proj.forceUnlink': 'Remove anyway', 'proj.forceUnlinkDesc': '"{0}" has been modified in this project. Removing it will permanently delete the project copy.',
        'proj.unregister': 'Remove', 'proj.noSkills': 'No Skills deployed to this project.',
        'proj.noSkillsDesc': 'Choose a Library Skill below to link or copy it into the project.',
        'proj.selectSkill': 'Select a Library Skill', 'proj.chooseAgent': 'Select at least one Agent',
        'proj.chooseSkill': 'Select a Skill', 'proj.unregistered': 'Project removed',
        'proj.deployed': 'Skill deployed', 'proj.unlinked': 'Skill unlinked', 'proj.confirmUnregister': 'Remove this project? Managed Skills must be unlinked first.',
        'proj.noProjects': 'No projects', 'proj.pathRequired': 'Project path is required', 'proj.statusTitle': 'Deployment status', 'proj.statusDesc': 'Managed deployment operations for this project.',
        'proj.statusEmpty': 'No deployment operations', 'proj.statusCreate': 'Pending creation', 'proj.statusUnchanged': 'Deployed', 'proj.statusReplaceManaged': 'Update required', 'proj.statusConflictUnmanaged': 'Conflict', 'proj.statusBroken': 'Broken', 'proj.target': 'Target', 'proj.source': 'Source', 'proj.viewDetails': 'View details', 'proj.skillDetails': 'Project Skill details', 'proj.skillPath': 'Skill path', 'proj.metadata': 'Metadata', 'proj.content': 'Skill content', 'proj.noContent': 'No Skill content',
    },
    zh: {
        'nav.dashboard': '仪表盘', 'nav.library': 'Skill 库', 'nav.projects': '项目',
        'dash.title': '仪表盘', 'dash.totalSkills': 'Skill 总数', 'dash.enabled': '已启用', 'dash.gitSources': 'Git 来源',
        'dash.recentlyAdded': '最近添加', 'dash.id': 'ID', 'dash.tags': '标签', 'dash.added': '添加时间', 'dash.hash': '哈希',
        'dash.loadFailed': '加载仪表盘失败',
        'dash.noRecent': '尚未添加任何 Skill。',
        'lib.title': 'Skill 库', 'lib.addSkill': '添加 Skill', 'lib.all': '全部', 'lib.search': '搜索 Skill',
        'lib.searchPlaceholder': '按名称、ID、描述或来源搜索', 'lib.manageTags': '管理标签',
        'lib.noSkills': '暂无 Skill', 'lib.noSkillsDesc': '点击上方按钮将 Skill 添加到个人库。',
        'lib.skillPath': 'Skill 路径', 'lib.tagsComma': '标签（逗号分隔）',
        'lib.chooseSkillPath': '在 Finder 中选择 Skill 文件夹', 'lib.choosePath': '选择文件夹', 'lib.skillTagsPlaceholder': 'skill所属标签',
        'lib.importLocal': '本地导入', 'lib.importGit': '仓库来源', 'lib.gitSourceName': '来源名称（可选）',
        'lib.gitInput': '仓库地址或安装命令', 'lib.gitNamePlaceholder': '自动生成',
        'lib.gitUrl': 'Git 仓库地址', 'lib.gitRef': '引用（分支、Tag 或提交）', 'lib.import': '导入',
        'lib.importedSource': '已从 {1} 导入 {0} 个 Skill', 'lib.updateSource': '更新来源',
        'lib.updatingSource': '正在更新来源...', 'lib.gitRequired': '请填写仓库地址或安装命令',
        'lib.agentClaude': 'Claude', 'lib.agentCodex': 'Codex',
        'lib.cancel': '取消', 'lib.remove': '移除',
        'lib.confirmRemoveTitle': '移除 Skill', 'lib.confirmRemove': '确定要移除',
        'lib.confirmRemoveNote': '必须先禁用该 Skill。',
        'lib.addedSuccess': 'Skill 添加成功', 'lib.pathRequired': '路径不能为空', 'lib.loadFailed': '加载 Skill 库失败',
        'lib.addSkillTitle': '添加 Skill 到库', 'lib.enabled': '已启用', 'lib.disabled': '已禁用', 'lib.removed': '已移除',
        'lib.details': 'Skill 详情', 'lib.description': '描述', 'lib.source': '来源', 'lib.local': '本地', 'lib.location': '位置',
        'lib.path': '存储路径', 'lib.revision': '版本', 'lib.addTag': '添加标签', 'lib.tagName': '标签名称',
        'lib.tagAdded': '标签已添加', 'lib.tagRemoved': '标签已移除', 'lib.renameTag': '重命名标签',
        'lib.newTagName': '新标签名称', 'lib.renamedTag': '标签已重命名', 'lib.noTags': '暂无标签',
        'lib.viewDetails': '查看详情', 'lib.close': '关闭', 'lib.skillCount': '{0} 个 Skill', 'lib.content': 'Skill 内容', 'lib.noContent': '暂无 Skill 内容',
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
        'proj.title': '项目', 'proj.add': '添加项目', 'proj.empty': '暂无项目',
        'proj.emptyDesc': '登记本机项目后，可将 Skill 库中的 Skill 部署到项目 Agent 目录。',
        'proj.path': '项目路径', 'proj.chooseProjectPath': '在 Finder 中选择项目文件夹',
        'proj.register': '添加项目', 'proj.registered': '项目已添加', 'proj.list': '项目列表',
        'proj.ready': '可用', 'proj.missing': '不存在', 'proj.activations': '激活数', 'proj.skills': '个 Skill', 'proj.skill': 'Skill',
        'proj.agents': 'Agent', 'proj.claudeCode': 'Claude Code', 'proj.codex': 'Codex', 'proj.allAgents': '全部 Agent', 'proj.mode': '模式', 'proj.actions': '操作', 'proj.addSkill': '添加 Skill',
        'proj.scan': '扫描项目', 'proj.scanTitle': '项目中的 Skill', 'proj.scanDesc': '扫描项目 Agent 目录，并按 Skill 合并展示。', 'proj.lastScan': '上次扫描',
        'proj.scanOk': '正常', 'proj.scanWarning': '需检查', 'proj.scanError': '不可用', 'proj.managed': 'SKM 管理', 'proj.external': '已检测',
        'proj.noScannedSkills': '项目 Agent 目录中暂无 Skill。', 'proj.noFilteredSkills': '该 Agent 下暂无 Skill。', 'proj.noDescription': '暂无描述',
        'proj.link': '软链接', 'proj.copy': '复制', 'proj.symlink': '软链', 'proj.copied': '复制', 'proj.unlink': '解绑', 'proj.status': '刷新状态',
        'proj.confirmUnlinkTitle': '确认解绑 Skill', 'proj.confirmUnlink': '确认解绑', 'proj.confirmUnlinkDesc': '确定要从当前项目解绑“{0}”吗？对应的托管部署将被移除。',
        'proj.forceUnlinkTitle': '移除已修改的 Skill', 'proj.forceUnlink': '仍要移除', 'proj.forceUnlinkDesc': '“{0}”已在项目中被修改。继续操作将永久删除该项目副本。',
        'proj.unregister': '移除', 'proj.noSkills': '该项目暂无已部署 Skill。',
        'proj.noSkillsDesc': '从下方选择 Library Skill，将它软链或复制到项目中。',
        'proj.selectSkill': '选择 Library Skill', 'proj.chooseAgent': '请至少选择一个 Agent',
        'proj.chooseSkill': '请选择 Skill', 'proj.unregistered': '项目已移除',
        'proj.deployed': 'Skill 已部署', 'proj.unlinked': 'Skill 已解绑', 'proj.confirmUnregister': '确认移除该项目？必须先解绑已管理的 Skill。',
        'proj.noProjects': '暂无项目', 'proj.pathRequired': '项目路径不能为空', 'proj.statusTitle': '部署状态', 'proj.statusDesc': 'SKM 管理的项目部署操作。',
        'proj.statusEmpty': '没有部署操作', 'proj.statusCreate': '待创建', 'proj.statusUnchanged': '已部署', 'proj.statusReplaceManaged': '待更新', 'proj.statusConflictUnmanaged': '存在冲突', 'proj.statusBroken': '已损坏', 'proj.target': '目标', 'proj.source': '来源', 'proj.viewDetails': '查看详情', 'proj.skillDetails': '项目 Skill 详情', 'proj.skillPath': 'Skill 路径', 'proj.metadata': '元数据', 'proj.content': 'Skill 内容', 'proj.noContent': '暂无 Skill 内容',
    }
};

var currentLang = (function () {
    var saved = localStorage.getItem('skm-lang');
    if (saved && translations[saved]) return saved;
    var nav = (navigator.language || 'en').toLowerCase();
    if (nav.startsWith('zh')) return 'zh';
    return 'en';
})();

function t(key) {
    return (translations[currentLang] && translations[currentLang][key]) || translations.en[key] || key;
}

function setLang(lang) {
    currentLang = lang;
    localStorage.setItem('skm-lang', lang);
    updateNavLabels();
    App.navigate(App.currentPage, false);
    document.querySelectorAll('#lang-toggle, #mobile-lang-toggle').forEach(function (toggle) {
        toggle.textContent = lang === 'zh' ? 'EN' : '中文';
    });
}

function updateNavLabels() {
    var map = { dashboard: 'nav.dashboard', library: 'nav.library', projects: 'nav.projects' };
    document.querySelectorAll('.nav-item').forEach(function (item) {
        var key = map[item.dataset.page];
        if (key) item.querySelector('.nav-label').textContent = t(key);
    });
}

function displayTag(tag) {
    return tag === 'general' ? t('tag.general') : tag;
}

function displaySource(source) {
    return source === 'local' ? t('lib.local') : source;
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
            throw new Error(body.error || 'Request failed: ' + res.status);
        }
        if (res.status === 204) return null;
        return res.json();
    },
    get: function (url) { return this.request('GET', url); },
    post: function (url, data) { return this.request('POST', url, data); },
    del: function (url) { return this.request('DELETE', url); },
};

// ===== Toast =====
function showToast(message, type) {
    type = type || 'success';
    var container = document.getElementById('toast-container');
    var toast = document.createElement('div');
    toast.className = 'toast toast-' + type;
    toast.textContent = message;
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

// ===== Modal =====
function showModal(title, contentHtml, actions) {
    var existing = document.querySelector('.modal-overlay');
    if (existing) existing.remove();
    var overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML =
        '<div class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title">' +
            '<div class="modal-header"><div class="modal-title" id="modal-title">' + escapeHtml(title) + '</div>' +
            '<button class="icon-btn modal-close" type="button" data-close-modal aria-label="Close">&times;</button></div>' +
            '<div class="modal-body">' + contentHtml + '</div>' +
            '<div class="modal-actions">' + (actions || '') + '</div>' +
        '</div>';
    overlay.addEventListener('click', function (e) {
        if (e.target === overlay) closeModal();
    });
    document.body.appendChild(overlay);
    overlay.querySelectorAll('[data-close-modal]').forEach(function (button) {
        button.addEventListener('click', closeModal);
    });
    document.addEventListener('keydown', onModalEsc);
    var focusTarget = overlay.querySelector('input, select, button');
    if (focusTarget) focusTarget.focus();
}

function closeModal() {
    var overlay = document.querySelector('.modal-overlay');
    if (overlay) overlay.remove();
    document.removeEventListener('keydown', onModalEsc);
}

function onModalEsc(e) {
    if (e.key === 'Escape') closeModal();
}

// ===== Main App =====
var App = {
    currentPage: 'dashboard',

    init: function () {
        this.setupNav();
        this.setupLangToggle();
        this.setupMobileNav();
        this.loadVersion();
        updateNavLabels();
        var initialPage = window.location.hash.replace(/^#\/?/, '');
        if (!['dashboard', 'library', 'projects'].includes(initialPage)) initialPage = 'dashboard';
        this.navigate(initialPage, false);
        window.addEventListener('hashchange', function () {
            var page = window.location.hash.replace(/^#\/?/, '');
            if (['dashboard', 'library', 'projects'].includes(page) && page !== App.currentPage) {
                App.navigate(page, false);
            }
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

    setupLangToggle: function () {
        document.querySelectorAll('#lang-toggle, #mobile-lang-toggle').forEach(function (btn) {
            btn.textContent = currentLang === 'zh' ? 'EN' : '中文';
            btn.addEventListener('click', function () {
                setLang(currentLang === 'zh' ? 'en' : 'zh');
            });
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
            item.classList.toggle('active', item.dataset.page === page);
        });
        var container = document.getElementById('main-content');
        container.dataset.page = page;
        container.innerHTML = '<div class="loading-full"><div class="spinner"></div><p>' + t('loading') + '</p></div>';
        switch (page) {
            case 'dashboard':  renderDashboard(); break;
            case 'library':    renderLibrary(); break;
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

document.addEventListener('DOMContentLoaded', function () { App.init(); });

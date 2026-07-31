/* global App */

// ===== i18n =====
var translations = {
    en: {
        'nav.dashboard': 'Dashboard', 'nav.library': 'Library',
        'dash.title': 'Dashboard', 'dash.totalSkills': 'Total Skills', 'dash.enabled': 'Enabled', 'dash.gitSources': 'Git Sources',
        'dash.recentlyAdded': 'Recently Added', 'dash.id': 'ID', 'dash.tags': 'Tags', 'dash.added': 'Added', 'dash.hash': 'Hash',
        'dash.loadFailed': 'Failed to load dashboard',
        'dash.noRecent': 'No Skills have been added yet.',
        'lib.title': 'Skill Library', 'lib.addSkill': 'Add Skill', 'lib.all': 'All', 'lib.search': 'Search Skills',
        'lib.searchPlaceholder': 'Search by name, ID, description, or source', 'lib.manageTags': 'Manage tags',
        'lib.noSkills': 'No Skills found', 'lib.noSkillsDesc': 'Add a Skill to your personal Library with the button above.',
        'lib.skillPath': 'Skill Path', 'lib.tagsComma': 'Tags (comma-separated)',
        'lib.chooseSkillPath': 'Choose a Skill directory in Finder', 'lib.choosePath': 'Choose directory', 'lib.skillTagsPlaceholder': 'Skill tags',
        'lib.importLocal': 'Local import', 'lib.importGit': 'Git source', 'lib.gitSourceName': 'Source name',
        'lib.gitUrl': 'Git repository URL', 'lib.gitRef': 'Ref (branch, tag, or commit)', 'lib.import': 'Import',
        'lib.importedSource': 'Imported {0} Skill(s) from {1}', 'lib.updateSource': 'Update source',
        'lib.updatingSource': 'Updating source...', 'lib.gitRequired': 'Source name and Git URL are required',
        'lib.agentClaude': 'Claude', 'lib.agentCodex': 'Codex',
        'lib.cancel': 'Cancel', 'lib.remove': 'Remove',
        'lib.confirmRemoveTitle': 'Remove Skill', 'lib.confirmRemove': 'Are you sure you want to remove',
        'lib.confirmRemoveNote': 'The Skill must be disabled first.',
        'lib.addedSuccess': 'Skill added successfully', 'lib.pathRequired': 'Path is required', 'lib.loadFailed': 'Failed to load Library',
        'lib.addSkillTitle': 'Add Skill to Library', 'lib.enabled': 'Enabled', 'lib.disabled': 'Disabled', 'lib.removed': 'Removed',
        'lib.details': 'Skill details', 'lib.description': 'Description', 'lib.source': 'Source', 'lib.location': 'Location',
        'lib.path': 'Stored path', 'lib.revision': 'Revision', 'lib.addTag': 'Add tag', 'lib.tagName': 'Tag name',
        'lib.tagAdded': 'Tag added', 'lib.tagRemoved': 'Tag removed', 'lib.renameTag': 'Rename tag',
        'lib.newTagName': 'New tag name', 'lib.renamedTag': 'Tag renamed', 'lib.noTags': 'No tags yet',
        'lib.viewDetails': 'View details', 'lib.close': 'Close', 'lib.skillCount': '{0} Skill(s)',
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
    },
    zh: {
        'nav.dashboard': '仪表盘', 'nav.library': 'Skill 库',
        'dash.title': '仪表盘', 'dash.totalSkills': 'Skill 总数', 'dash.enabled': '已启用', 'dash.gitSources': 'Git 来源',
        'dash.recentlyAdded': '最近添加', 'dash.id': 'ID', 'dash.tags': '标签', 'dash.added': '添加时间', 'dash.hash': '哈希',
        'dash.loadFailed': '加载仪表盘失败',
        'dash.noRecent': '尚未添加任何 Skill。',
        'lib.title': 'Skill 库', 'lib.addSkill': '添加 Skill', 'lib.all': '全部', 'lib.search': '搜索 Skill',
        'lib.searchPlaceholder': '按名称、ID、描述或来源搜索', 'lib.manageTags': '管理标签',
        'lib.noSkills': '暂无 Skill', 'lib.noSkillsDesc': '点击上方按钮将 Skill 添加到个人库。',
        'lib.skillPath': 'Skill 路径', 'lib.tagsComma': '标签（逗号分隔）',
        'lib.chooseSkillPath': '在 Finder 中选择 Skill 文件夹', 'lib.choosePath': '选择文件夹', 'lib.skillTagsPlaceholder': 'skill所属标签',
        'lib.importLocal': '本地导入', 'lib.importGit': 'Git 来源', 'lib.gitSourceName': '来源名称',
        'lib.gitUrl': 'Git 仓库地址', 'lib.gitRef': '引用（分支、Tag 或提交）', 'lib.import': '导入',
        'lib.importedSource': '已从 {1} 导入 {0} 个 Skill', 'lib.updateSource': '更新来源',
        'lib.updatingSource': '正在更新来源...', 'lib.gitRequired': '请填写来源名称和 Git 仓库地址',
        'lib.agentClaude': 'Claude', 'lib.agentCodex': 'Codex',
        'lib.cancel': '取消', 'lib.remove': '移除',
        'lib.confirmRemoveTitle': '移除 Skill', 'lib.confirmRemove': '确定要移除',
        'lib.confirmRemoveNote': '必须先禁用该 Skill。',
        'lib.addedSuccess': 'Skill 添加成功', 'lib.pathRequired': '路径不能为空', 'lib.loadFailed': '加载 Skill 库失败',
        'lib.addSkillTitle': '添加 Skill 到库', 'lib.enabled': '已启用', 'lib.disabled': '已禁用', 'lib.removed': '已移除',
        'lib.details': 'Skill 详情', 'lib.description': '描述', 'lib.source': '来源', 'lib.location': '位置',
        'lib.path': '存储路径', 'lib.revision': '版本', 'lib.addTag': '添加标签', 'lib.tagName': '标签名称',
        'lib.tagAdded': '标签已添加', 'lib.tagRemoved': '标签已移除', 'lib.renameTag': '重命名标签',
        'lib.newTagName': '新标签名称', 'lib.renamedTag': '标签已重命名', 'lib.noTags': '暂无标签',
        'lib.viewDetails': '查看详情', 'lib.close': '关闭', 'lib.skillCount': '{0} 个 Skill',
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
    var map = { dashboard: 'nav.dashboard', library: 'nav.library' };
    document.querySelectorAll('.nav-item').forEach(function (item) {
        var key = map[item.dataset.page];
        if (key) item.querySelector('.nav-label').textContent = t(key);
    });
}

function displayTag(tag) {
    return tag === 'general' ? t('tag.general') : tag;
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
        'optional': 'badge-optional', 'not-created': 'badge-not-created',
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
        if (!['dashboard', 'library'].includes(initialPage)) initialPage = 'dashboard';
        this.navigate(initialPage, false);
        window.addEventListener('hashchange', function () {
            var page = window.location.hash.replace(/^#\/?/, '');
            if (['dashboard', 'library'].includes(page) && page !== App.currentPage) {
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

/* global api, showToast, showModal, closeModal, displayTag, formatDate, shortHash, shortRevision, escapeHtml, isCurrentPage, t */

var libraryState = { skills: [], tags: [], activeTag: '', query: '', enabled: {}, gitSources: {} };

async function renderLibrary() {
    var container = document.getElementById('main-content');
    try {
        var url = '/api/skills' + (libraryState.activeTag ? '?tag=' + encodeURIComponent(libraryState.activeTag) : '');
        var results = await Promise.all([api.get(url), api.get('/api/tags'), api.get('/api/status'), api.get('/api/sources')]);
        if (!isCurrentPage('library')) return;
        libraryState.skills = results[0] || [];
        libraryState.tags = results[1] || [];
        libraryState.enabled = {};
        libraryState.gitSources = {};
        (results[2].operations || []).forEach(function (operation) {
            if (operation.placement !== 'user') return;
            if (!libraryState.enabled[operation.skillId]) libraryState.enabled[operation.skillId] = {};
            libraryState.enabled[operation.skillId][operation.agent] = true;
        });
        (results[3] || []).forEach(function (source) { libraryState.gitSources[source.name] = source; });
        paintLibrary();
    } catch (err) {
        if (!isCurrentPage('library')) return;
        container.innerHTML = libraryError(t('lib.loadFailed'), err.message);
        showToast(err.message, 'error');
    }
}

function paintLibrary() {
    if (!isCurrentPage('library')) return;
    var container = document.getElementById('main-content');
    var query = libraryState.query.toLowerCase();
    var visible = libraryState.skills.filter(function (skill) {
        return !query || [skill.name, skill.id, skill.description, skill.source].join(' ').toLowerCase().includes(query);
    });
    var html = '<div class="page animate-in"><div class="page-header"><div><h1 class="page-title">' + t('lib.title') + '</h1>' +
        '<p class="page-subtitle">' + t('lib.skillCount').replace('{0}', visible.length) + '</p></div><div class="header-actions">' +
        '<button class="btn btn-secondary" type="button" id="btn-manage-tags">' + t('lib.manageTags') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-add-skill">+ ' + t('lib.addSkill') + '</button></div></div>';

    html += '<div class="library-tools"><label class="search-box"><span class="sr-only">' + t('lib.search') + '</span>' +
        '<span class="search-mark" aria-hidden="true">/</span><input class="input" id="skill-search" value="' + escapeHtml(libraryState.query) +
        '" placeholder="' + t('lib.searchPlaceholder') + '"></label>';
    html += '<div class="filter-bar"><span class="filter-label">' + t('dash.tags') + '</span><div class="tag-filter-list">' +
        '<button class="tag clickable' + (!libraryState.activeTag ? ' active' : '') + '" type="button" data-tag="">' + t('lib.all') + '</button>';
    libraryState.tags.forEach(function (tag) {
        html += '<button class="tag clickable' + (libraryState.activeTag === tag.name ? ' active' : '') + '" type="button" data-tag="' +
            escapeHtml(tag.name) + '">' + escapeHtml(displayTag(tag.name)) + ' <small>' + tag.count + '</small></button>';
    });
    html += '</div></div></div>';

    if (!visible.length) {
        html += '<div class="empty-state"><div class="empty-state-mark">L</div><div class="empty-state-title">' + t('lib.noSkills') +
            '</div><div class="empty-state-desc">' + t('lib.noSkillsDesc') + '</div></div>';
    } else {
        html += '<div class="card-grid">';
        visible.forEach(function (skill) { html += skillCard(skill); });
        html += '</div>';
    }
    html += '</div>';
    container.innerHTML = html;

    document.getElementById('btn-add-skill').addEventListener('click', showAddSkillModal);
    document.getElementById('btn-manage-tags').addEventListener('click', showManageTagsModal);
    document.getElementById('skill-search').addEventListener('input', function (event) {
        libraryState.query = event.target.value;
        paintLibrary();
        var next = document.getElementById('skill-search');
        next.focus();
        next.setSelectionRange(libraryState.query.length, libraryState.query.length);
    });
    container.querySelectorAll('[data-tag]').forEach(function (button) {
        button.addEventListener('click', function () { libraryState.activeTag = button.dataset.tag; renderLibrary(); });
    });
    container.querySelectorAll('.btn-details-skill').forEach(function (button) {
        button.addEventListener('click', function () { openSkillDetails(button.dataset.id); });
    });
    container.querySelectorAll('.agent-toggle').forEach(function (button) {
        button.addEventListener('click', function () { toggleSkillAgent(button.dataset.id, button.dataset.agent, button); });
    });
    container.querySelectorAll('.btn-remove-skill').forEach(function (button) {
        button.addEventListener('click', function () { confirmRemoveSkill(button.dataset.id); });
    });
}

function isGitSkill(skill) {
    return Boolean(skill && libraryState.gitSources[skill.source]);
}

function skillCard(skill) {
    var tags = (skill.tags || []).map(function (tag) { return '<span class="tag">' + escapeHtml(displayTag(tag)) + '</span>'; }).join('');
    var agents = libraryState.enabled[skill.id] || {};
    var hasActivation = Boolean(agents.claude || agents.codex);
    return '<article class="card skill-card"><div class="skill-header"><div><div class="skill-name">' + escapeHtml(skill.name) +
        '</div><div class="skill-id mono">' + escapeHtml(skill.id) + '</div></div><span class="badge badge-source">' + escapeHtml(displaySource(skill.source || 'local')) +
        '</span></div><p class="skill-desc">' + escapeHtml(skill.description || 'No description') + '</p><div class="tag-list">' + tags +
        '</div><div class="skill-meta"><span>' + shortHash(skill.hash) + '</span><span>' + formatDate(skill.addedAt) + '</span></div>' +
        agentControls(skill.id, agents) + '<div class="skill-actions"><button class="btn btn-ghost btn-sm btn-details-skill" type="button" data-id="' +
        escapeHtml(skill.id) + '">' + t('lib.viewDetails') + '</button><div class="action-spacer"></div><button class="btn btn-danger btn-sm btn-remove-skill" type="button" data-id="' +
        escapeHtml(skill.id) + '"' + (hasActivation ? ' disabled' : '') + '>' + t('lib.remove') + '</button></div></article>';
}

function agentControls(skillID, agents) {
    return '<div class="agent-controls" aria-label="Agent activation"><button class="agent-toggle' + (agents.codex ? ' active' : '') +
        '" type="button" data-id="' + escapeHtml(skillID) + '" data-agent="codex" aria-pressed="' + Boolean(agents.codex) + '">' +
        '<img class="agent-logo" src="/assets/codex.svg" alt=""><span>' + t('lib.agentCodex') + '</span></button><button class="agent-toggle' +
        (agents.claude ? ' active' : '') + '" type="button" data-id="' + escapeHtml(skillID) + '" data-agent="claude" aria-pressed="' +
        Boolean(agents.claude) + '"><img class="agent-logo" src="/assets/claude.svg" alt=""><span>' + t('lib.agentClaude') + '</span></button></div>';
}

async function toggleSkillAgent(skillID, agent, button) {
    var isEnabled = button.classList.contains('active');
    button.classList.toggle('active', !isEnabled);
    button.setAttribute('aria-pressed', String(!isEnabled));
    button.disabled = true;
    try {
        if (isEnabled) {
            await api.post('/api/disable', { skills: [skillID], agents: [agent] });
        } else {
            await api.post('/api/enable', { skills: [skillID], agents: [agent], mode: 'auto' });
        }
        renderLibrary();
    } catch (err) {
        button.classList.toggle('active', isEnabled);
        button.setAttribute('aria-pressed', String(isEnabled));
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

function showAddSkillModal() {
    var content = '<div class="import-mode" role="tablist"><button class="import-mode-option active" type="button" data-import-mode="local">' +
        t('lib.importLocal') + '</button><button class="import-mode-option" type="button" data-import-mode="git">' + t('lib.importGit') + '</button></div>' +
        '<div class="import-pane" data-import-pane="local"><div class="form-group"><label class="form-label" for="add-skill-path">' + t('lib.skillPath') +
        '</label><div class="path-picker"><input class="input" id="add-skill-path" readonly placeholder="' + t('lib.chooseSkillPath') + '"><button class="btn btn-secondary" type="button" id="btn-choose-skill-path">' + t('lib.choosePath') + '</button></div></div><div class="form-group"><label class="form-label" for="add-skill-tags">' +
        t('lib.tagsComma') + '</label><input class="input" id="add-skill-tags" placeholder="' + t('lib.skillTagsPlaceholder') + '"></div></div>' +
        '<div class="import-pane is-hidden" data-import-pane="git"><div class="form-group"><label class="form-label" for="git-source-name">' +
        t('lib.gitSourceName') + '</label><input class="input" id="git-source-name" placeholder="team-skills"></div><div class="form-group"><label class="form-label" for="git-source-url">' +
        t('lib.gitUrl') + '</label><input class="input" id="git-source-url" placeholder="git@github.com:org/skills.git"></div><div class="form-group"><label class="form-label" for="git-source-tags">' +
        t('lib.tagsComma') + '</label><input class="input" id="git-source-tags" placeholder="team, review"></div></div>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-confirm-add">' + t('lib.addSkill') + '</button>';
    showModal(t('lib.addSkillTitle'), content, actions);
    var mode = 'local';
    document.getElementById('btn-choose-skill-path').addEventListener('click', chooseSkillDirectory);
    document.querySelectorAll('[data-import-mode]').forEach(function (button) {
        button.addEventListener('click', function () {
            mode = button.dataset.importMode;
            document.querySelectorAll('[data-import-mode]').forEach(function (item) { item.classList.toggle('active', item === button); });
            document.querySelectorAll('[data-import-pane]').forEach(function (pane) { pane.classList.toggle('is-hidden', pane.dataset.importPane !== mode); });
            document.getElementById('btn-confirm-add').textContent = mode === 'git' ? t('lib.import') : t('lib.addSkill');
            var input = document.querySelector('[data-import-pane="' + mode + '"] input');
            if (input) input.focus();
        });
    });
    document.getElementById('btn-confirm-add').addEventListener('click', function () {
        if (mode === 'git') doImportGitSource(); else doAddSkill();
    });
}

async function chooseSkillDirectory() {
    var button = document.getElementById('btn-choose-skill-path');
    button.disabled = true;
    try {
        var result = await api.post('/api/dialogs/skill-directory', {});
        document.getElementById('add-skill-path').value = result.path;
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        button.disabled = false;
    }
}

function splitList(value) {
    return value ? value.split(',').map(function (item) { return item.trim(); }).filter(Boolean) : [];
}

async function doAddSkill() {
    var path = document.getElementById('add-skill-path').value.trim();
    if (!path) { showToast(t('lib.pathRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-add');
    button.disabled = true;
    try {
        await api.post('/api/skills', { path: path, tags: splitList(document.getElementById('add-skill-tags').value.trim()), source: '' });
        closeModal();
        showToast(t('lib.addedSuccess'));
        renderLibrary();
    } catch (err) { button.disabled = false; showToast(err.message, 'error'); }
}

async function doImportGitSource() {
    var name = document.getElementById('git-source-name').value.trim();
    var url = document.getElementById('git-source-url').value.trim();
    if (!name || !url) { showToast(t('lib.gitRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-add');
    button.disabled = true;
    try {
        var result = await api.post('/api/sources', {
            name: name, url: url, ref: '', paths: [], tags: splitList(document.getElementById('git-source-tags').value.trim()),
        });
        closeModal();
        showToast(t('lib.importedSource').replace('{0}', (result.skills || []).length).replace('{1}', result.source.name));
        renderLibrary();
    } catch (err) { button.disabled = false; showToast(err.message, 'error'); }
}

async function openSkillDetails(id) {
    try {
        var skill = await api.get('/api/skills/' + encodeURIComponent(id));
        showSkillDetails(skill);
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function showSkillDetails(skill) {
    if (!skill) return;
    var source = libraryState.gitSources[skill.source];
    var skillTags = skill.tags || [];
    var tags = skillTags.map(function (tag) {
        var remove = '<button type="button" class="tag-remove" data-remove-tag="' + escapeHtml(tag) + '" aria-label="' + t('lib.remove') + ' ' + escapeHtml(displayTag(tag)) + '">&times;</button>';
        return '<span class="tag removable">' + escapeHtml(displayTag(tag)) + remove + '</span>';
    }).join('') || '<span class="muted">' + t('lib.noTags') + '</span>';
    var sourceDetails = source ? '<div><dt>' + t('lib.gitUrl') + '</dt><dd class="mono detail-path">' + escapeHtml(source.url) + '</dd></div><div><dt>' +
        t('lib.gitRef') + '</dt><dd>' + escapeHtml(source.ref || 'default') + '</dd></div>' : '';
    var content = '<div class="detail-hero"><div><div class="detail-name">' + escapeHtml(skill.name) + '</div><div class="mono muted">' + escapeHtml(skill.id) +
        '</div></div><span class="badge badge-source">' + escapeHtml(displaySource(skill.source || 'local')) + '</span></div><dl class="detail-list"><div><dt>' +
        t('lib.description') + '</dt><dd>' + escapeHtml(skill.description || '-') + '</dd></div><div><dt>' + t('lib.path') + '</dt><dd class="mono detail-path">' +
        escapeHtml(skill.path) + '</dd></div><div><dt>' + t('dash.hash') + '</dt><dd class="mono">' + escapeHtml(skill.hash) + '</dd></div><div><dt>' +
        t('lib.revision') + '</dt><dd class="mono">' + shortRevision(skill.revision) + '</dd></div>' + sourceDetails + '<div><dt>' + t('dash.added') +
        '</dt><dd>' + formatDate(skill.addedAt) + '</dd></div></dl><section class="skill-detail-section"><div class="form-group"><label class="form-label">' + t('dash.tags') +
        '</label><div class="tag-list detail-tags">' + tags + '</div></div><div class="inline-form"><input class="input" id="detail-new-tag" placeholder="' + t('lib.tagName') + '"><button class="btn btn-secondary" type="button" id="btn-add-tag">' + t('lib.addTag') + '</button></div></section>' +
        '<section class="skill-detail-section skill-content-section"><div class="skill-detail-section-heading"><span class="form-label">' + t('lib.content') + '</span><span class="mono muted">SKILL.md</span></div>' +
        (skill.body ? '<pre class="skill-detail-content">' + escapeHtml(skill.body) + '</pre>' : '<div class="inline-empty">' + t('lib.noContent') + '</div>') + '</section>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.close') + '</button>' +
        (source ? '<button class="btn btn-primary" type="button" id="btn-update-source">' + t('lib.updateSource') + '</button>' : '');
    showModal(t('lib.details'), content, actions);
    document.querySelector('.modal').classList.add('skill-detail-modal');
    document.getElementById('btn-add-tag').addEventListener('click', function () { addSkillTag(skill.id); });
    document.querySelectorAll('[data-remove-tag]').forEach(function (button) {
        button.addEventListener('click', function () { removeSkillTag(skill.id, button.dataset.removeTag); });
    });
    if (source) document.getElementById('btn-update-source').addEventListener('click', function () { updateGitSource(source.name); });
}

async function updateGitSource(name) {
    var button = document.getElementById('btn-update-source');
    button.disabled = true;
    button.textContent = t('lib.updatingSource');
    try {
        var result = await api.post('/api/sources/' + encodeURIComponent(name) + '/update', {});
        closeModal();
        showToast(t('lib.importedSource').replace('{0}', (result.skills || []).length).replace('{1}', name));
        renderLibrary();
    } catch (err) { button.disabled = false; button.textContent = t('lib.updateSource'); showToast(err.message, 'error'); }
}

async function addSkillTag(id) {
    var input = document.getElementById('detail-new-tag');
    var tag = input.value.trim();
    if (!tag) return;
    try {
        await api.post('/api/skill-tags/add', { skill: id, tags: [tag] });
        await refreshLibraryAfterTagChange();
        showToast(t('lib.tagAdded'));
        await openSkillDetails(id);
    } catch (err) { showToast(err.message, 'error'); }
}

async function removeSkillTag(id, tag) {
    try {
        await api.post('/api/skill-tags/remove', { skill: id, tag: tag });
        await refreshLibraryAfterTagChange();
        showToast(t('lib.tagRemoved'));
        await openSkillDetails(id);
    } catch (err) { showToast(err.message, 'error'); }
}

async function refreshLibraryAfterTagChange() {
    var url = '/api/skills' + (libraryState.activeTag ? '?tag=' + encodeURIComponent(libraryState.activeTag) : '');
    var results = await Promise.all([api.get(url), api.get('/api/tags')]);
    libraryState.skills = results[0] || [];
    libraryState.tags = results[1] || [];
    paintLibrary();
}

function showManageTagsModal() {
    var content = libraryState.tags.length ? '<div class="tag-manager">' : '<div class="inline-empty">' + t('lib.noTags') + '</div>';
    libraryState.tags.forEach(function (tag) {
        content += '<div class="tag-manager-row"><span class="tag">' + escapeHtml(displayTag(tag.name)) + '</span><span class="muted">' + tag.count + '</span>' +
            '<button class="btn btn-ghost btn-sm" type="button" data-rename-tag="' + escapeHtml(tag.name) + '">' + t('lib.renameTag') + '</button></div>';
    });
    if (libraryState.tags.length) content += '</div>';
    showModal(t('lib.manageTags'), content, '<button class="btn btn-primary" type="button" data-close-modal>' + t('lib.close') + '</button>');
    document.querySelectorAll('[data-rename-tag]').forEach(function (button) {
        button.addEventListener('click', function () { showRenameTagModal(button.dataset.renameTag); });
    });
}

function showRenameTagModal(oldName) {
    var content = '<div class="form-group"><label class="form-label" for="rename-tag-input">' + t('lib.newTagName') + '</label>' +
        '<input class="input" id="rename-tag-input" value="' + escapeHtml(displayTag(oldName)) + '"></div>';
    showModal(t('lib.renameTag') + ': ' + displayTag(oldName), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') +
        '</button><button class="btn btn-primary" type="button" id="btn-do-rename">' + t('lib.renameTag') + '</button>');
    document.getElementById('btn-do-rename').addEventListener('click', async function () {
        var next = document.getElementById('rename-tag-input').value.trim();
        if (!next || next === oldName || (oldName === 'general' && next === displayTag(oldName))) return;
        try {
            await api.post('/api/tags/rename', { old: oldName, new: next });
            closeModal();
            if (libraryState.activeTag === oldName) libraryState.activeTag = next;
            showToast(t('lib.renamedTag'));
            renderLibrary();
        } catch (err) { showToast(err.message, 'error'); }
    });
}

function confirmRemoveSkill(id) {
    var content = '<p class="confirm-copy">' + t('lib.confirmRemove') + ' <strong>' + escapeHtml(id) + '</strong>?</p>';
    showModal(t('lib.confirmRemoveTitle'), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') +
        '</button><button class="btn btn-danger" type="button" id="btn-confirm-remove">' + t('lib.remove') + '</button>');
    document.getElementById('btn-confirm-remove').addEventListener('click', function () { doRemoveSkill(id); });
}

async function doRemoveSkill(id) {
    var button = document.getElementById('btn-confirm-remove');
    button.disabled = true;
    try {
        await api.del('/api/skills/' + encodeURIComponent(id));
        closeModal();
        showToast(t('lib.removed') + ' ' + id);
        renderLibrary();
    } catch (err) { button.disabled = false; showToast(err.message, 'error'); }
}

function libraryError(title, message) {
    return '<div class="empty-state"><div class="empty-state-mark">!</div><div class="empty-state-title">' + title +
        '</div><div class="empty-state-desc">' + escapeHtml(message) + '</div></div>';
}

window.renderLibrary = renderLibrary;

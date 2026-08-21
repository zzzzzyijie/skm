/* global api, showToast, showModal, closeModal, displayTag, formatDate, shortHash, shortRevision, escapeHtml, isCurrentPage, t, uiIcon, confirmationMarkup */

var libraryState = { skills: [], tags: [], agents: [], activeTag: '', query: '', enabled: {}, gitSources: {}, summary: {} };
var skillDetailState = { skill: null, savedTags: [] };
var agentIcons = {
    claude: 'claude.svg', codex: 'codex.svg', cursor: 'cursor.svg', copilot: 'copilot.svg', gemini: 'gemini.svg',
    windsurf: 'windsurf.svg', kiro: 'kiro.svg', cline: 'cline.svg', opencode: 'opencode.svg', trae: 'trae.svg',
    hermes: 'hermes.svg', openclaw: 'openclaw.svg'
};

function agentIconSource(agent) {
    return agent.icon || ('/assets/' + (agentIcons[agent.id] || 'codex.svg'));
}

async function renderLibrary() {
    var container = document.getElementById('main-content');
    try {
        var url = '/api/skills' + (libraryState.activeTag ? '?tag=' + encodeURIComponent(libraryState.activeTag) : '');
        var results = await Promise.all([api.get(url), api.get('/api/tags'), api.get('/api/agents'), api.get('/api/sources'), api.get('/api/dashboard')]);
        var status = null;
        try { status = await api.get('/api/status'); } catch (statusErr) { console.warn('status unavailable:', statusErr.message); }
        if (!isCurrentPage('library')) return;
        libraryState.skills = results[0] || [];
        libraryState.tags = results[1] || [];
        libraryState.agents = results[2] || [];
        libraryState.summary = results[4] || {};
        libraryState.enabled = {};
        libraryState.gitSources = {};
        ((status && status.operations) || []).forEach(function (operation) {
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
    var html = '<div class="page"><div class="page-header"><div><h1 class="page-title">' + t('lib.title') + '</h1>' +
        '<p class="page-subtitle" id="library-skill-count">' + t('lib.skillCount').replace('{0}', visible.length) + '</p></div><div class="header-actions">' +
        '<button class="btn btn-secondary" type="button" id="btn-manage-tags">' + uiIcon('tags') + t('lib.manageTags') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-add-skill">' + uiIcon('plus') + t('lib.addSkill') + '</button></div></div>';

    html += librarySummaryMarkup();
    html += agentManagementBand();

    html += '<div class="library-tools"><label class="search-box"><span class="sr-only">' + t('lib.search') + '</span>' +
        '<span class="search-mark" aria-hidden="true">' + uiIcon('search') + '</span><input class="input" id="skill-search" value="' + escapeHtml(libraryState.query) +
        '" placeholder="' + t('lib.searchPlaceholder') + '"></label>';
    html += '<div class="filter-bar"><span class="filter-label">' + uiIcon('tags') + t('lib.tags') + '</span><div class="tag-filter-list" id="skill-tag-filters">' +
        libraryTagFiltersMarkup() + '</div></div></div>';

    if (!visible.length) {
        html += '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('library') + '</div><div class="empty-state-title">' + t('lib.noSkills') +
            '</div><div class="empty-state-desc">' + t('lib.noSkillsDesc') + '</div></div>';
    } else {
        html += '<div class="card-grid" id="library-skill-grid">';
        visible.forEach(function (skill) { html += skillCard(skill); });
        html += '</div>';
    }
    html += '</div>';
    container.innerHTML = html;

    document.getElementById('btn-add-skill').addEventListener('click', showAddSkillModal);
    document.getElementById('btn-manage-tags').addEventListener('click', showManageTagsModal);
    document.getElementById('btn-manage-agents').addEventListener('click', showAgentManager);
    document.getElementById('skill-search').addEventListener('input', function (event) {
        libraryState.query = event.target.value;
        paintLibrary();
        var next = document.getElementById('skill-search');
        next.focus();
        next.setSelectionRange(libraryState.query.length, libraryState.query.length);
    });
    bindLibraryTagFilters();
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

function libraryTagFiltersMarkup() {
    var html = '<button class="tag clickable' + (!libraryState.activeTag ? ' active' : '') + '" type="button" data-tag="">' + t('lib.all') + '</button>';
    libraryState.tags.filter(function (tag) { return tag.skillCount > 0 || libraryState.activeTag === tag.name; }).forEach(function (tag) {
        html += '<button class="tag clickable' + (libraryState.activeTag === tag.name ? ' active' : '') + '" type="button" data-tag="' +
            escapeHtml(tag.name) + '">' + escapeHtml(displayTag(tag.name)) + ' <small>' + Number(tag.skillCount || 0) + '</small></button>';
    });
    return html;
}

function bindLibraryTagFilters() {
    var filters = document.getElementById('skill-tag-filters');
    if (!filters) return;
    filters.querySelectorAll('[data-tag]').forEach(function (button) {
        button.addEventListener('click', function () { libraryState.activeTag = button.dataset.tag; renderLibrary(); });
    });
}

function librarySummaryMarkup() {
    var skillCount = Number(libraryState.summary.skillCount || 0);
    var sourceCount = Number(libraryState.summary.sourceCount || 0);
    return '<section class="library-summary" aria-label="' + escapeHtml(t('lib.summaryLabel')) + '">' +
        librarySummaryItem(skillCount, t('lib.totalSkills'), 'library', 'accent') +
        librarySummaryItem(sourceCount, t('lib.gitSources'), 'link', 'info') + '</section>';
}

function librarySummaryItem(value, label, icon, tone) {
    return '<div class="library-summary-item library-summary-' + tone + '"><span class="library-summary-icon" aria-hidden="true">' + uiIcon(icon) +
        '</span><span class="library-summary-copy"><strong>' + value + '</strong><span>' + escapeHtml(label) + '</span></span></div>';
}

function isGitSkill(skill) {
    return Boolean(skill && libraryState.gitSources[skill.source]);
}

function skillCard(skill) {
    var tags = (skill.tags || []).map(function (tag) { return '<span class="tag">' + escapeHtml(displayTag(tag)) + '</span>'; }).join('');
    var agents = libraryState.enabled[skill.id] || {};
    var hasActivation = Object.keys(agents).some(function (agent) { return agents[agent]; });
    var health = skillHealthMarkup(skill);
    return '<article class="card skill-card" data-skill-card-id="' + escapeHtml(skill.id) + '"><div class="skill-header"><div><div class="skill-name">' + escapeHtml(skill.name) +
        '</div><div class="skill-id mono">' + escapeHtml(skill.id) + '</div></div><span class="badge badge-source">' + escapeHtml(displaySource(skill.source || 'local')) +
        '</span></div>' + health + '<p class="skill-desc">' + escapeHtml(skill.description || 'No description') + '</p><div class="tag-list">' + tags +
        '</div><div class="skill-meta"><span>' + shortHash(skill.hash) + '</span><span>' + formatDate(skill.addedAt) + '</span></div>' +
        agentControls(skill.id, agents) + '<div class="skill-actions"><button class="btn btn-ghost btn-sm btn-details-skill" type="button" data-id="' +
        escapeHtml(skill.id) + '">' + uiIcon('eye') + t('lib.viewDetails') + '</button><div class="action-spacer"></div><button class="btn btn-danger btn-sm btn-remove-skill" type="button" data-id="' +
        escapeHtml(skill.id) + '"' + (hasActivation ? ' disabled' : '') + '>' + uiIcon('trash') + t('lib.remove') + '</button></div></article>';
}

function skillHealthMarkup(skill) {
    var health = skill.health || 'available';
    var linked = skill.mode === 'symlink' && skill.projectRoot;
    if (!linked && health === 'available') return '';
    var labels = { available: 'lib.healthAvailable', changed: 'lib.healthChanged', missing: 'lib.healthMissing', unreachable: 'lib.healthUnreachable', invalid: 'lib.healthInvalid' };
    var badgeClass = health === 'available' ? 'badge-ok' : (health === 'changed' ? 'badge-warning' : 'badge-error');
    return '<div class="skill-health"><span class="badge ' + badgeClass + '">' + escapeHtml(t(labels[health] || labels.invalid)) + '</span>' +
        (linked ? '<span class="badge badge-muted">' + escapeHtml(t('lib.followingProject')) + '</span>' : '') +
        (skill.usingFallback ? '<span class="skill-fallback">' + escapeHtml(t('lib.usingFallback')) + '</span>' : '') + '</div>';
}

function agentControls(skillID, agents) {
    var visible = libraryState.agents.filter(function (agent) { return agent.configured && agent.supported; });
    var controls = visible.map(function (agent) {
        return '<button class="agent-toggle' + (agents[agent.id] ? ' active' : '') + '" type="button" data-id="' + escapeHtml(skillID) +
            '" data-agent="' + agent.id + '" aria-pressed="' + Boolean(agents[agent.id]) + '"><img class="agent-logo" src="' +
            agentIconSource(agent) + '" alt=""><span>' + escapeHtml(agent.name) + '</span></button>';
    }).join('');
    return controls ? '<div class="agent-controls" aria-label="Agent activation">' + controls + '</div>' : '';
}

function agentManagementBand() {
    var configured = libraryState.agents.filter(function (agent) { return agent.configured; });
    var managed = configured.length ? configured.map(function (agent) {
        return '<span class="managed-agent"><img class="agent-logo" src="' + agentIconSource(agent) + '" alt=""><span>' +
            escapeHtml(agent.name) + '</span></span>';
    }).join('') : '<span class="agent-manager-empty">' + t('lib.noManagedAgents') + '</span>';
    return '<section class="agent-manager"><div class="agent-manager-main"><span class="agent-manager-label">' + t('lib.manageAgents') +
        '</span><div class="managed-agent-list">' + managed + '</div></div><button class="btn btn-secondary" type="button" id="btn-manage-agents">' + uiIcon('settings') + t('lib.manageAgents') + '</button></section>';
}

function showAgentManager() {
    var detected = libraryState.agents.filter(function (agent) { return !agent.custom && agent.detected; });
    var other = libraryState.agents.filter(function (agent) { return !agent.custom && !agent.detected; });
    var custom = libraryState.agents.filter(function (agent) { return agent.custom; });
    var content = '<div class="agent-manager-actions"><button class="btn btn-secondary" type="button" id="btn-scan-agents">' + uiIcon('refresh') + t('lib.scanAgents') + '</button>' +
        '<button class="btn btn-secondary" type="button" id="btn-new-custom-agent">' + uiIcon('plus') + t('lib.addCustomAgent') + '</button></div><div class="agent-picker-groups">' +
        agentManagerGroup(t('lib.detectedAgents'), detected, t('lib.noDetectedAgents')) +
        agentManagerGroup(t('lib.otherAgents'), other, '') + agentManagerGroup(t('lib.customAgents'), custom, '') + '</div>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-save-agents">' + t('lib.saveAgents') + '</button>';
    showModal(t('lib.addAgentTitle'), content, actions);
    document.getElementById('btn-save-agents').addEventListener('click', saveManagedAgents);
    document.getElementById('btn-scan-agents').addEventListener('click', refreshAgentScan);
    document.getElementById('btn-new-custom-agent').addEventListener('click', function () { showCustomAgentEditor(null); });
    document.querySelectorAll('[data-edit-custom-agent]').forEach(function (button) {
        button.addEventListener('click', function (event) {
            event.preventDefault();
            event.stopPropagation();
            showCustomAgentEditor(libraryState.agents.find(function (agent) { return agent.id === button.dataset.editCustomAgent; }));
        });
    });
}

function agentManagerGroup(title, agents, emptyMessage) {
    if (!agents.length && !emptyMessage) return '';
    return '<section class="agent-picker-section"><div class="agent-picker-heading"><span>' + escapeHtml(title) + '</span><span class="badge badge-muted">' + agents.length + '</span></div>' +
        (agents.length ? '<div class="agent-picker">' + agents.map(agentPickerOption).join('') + '</div>' : '<div class="agent-picker-empty">' + escapeHtml(emptyMessage) + '</div>') + '</section>';
}

function agentPickerOption(agent) {
    var meta = agent.path + ' · ' + agent.format;
    var detectedLabel = agent.detected ? t('lib.detectedAgent') : t('lib.notDetectedAgent');
    return '<div class="agent-picker-option' + (agent.detected ? ' is-detected' : ' is-not-detected') + '"><label class="agent-picker-select"><input type="checkbox" name="managed-agent" value="' + escapeHtml(agent.id) + '"' +
        (agent.configured ? ' checked' : '') + (!agent.supported ? ' disabled' : '') + '><img class="agent-logo" src="' + agentIconSource(agent) +
        '" alt=""><span><span class="agent-picker-name"><strong>' + escapeHtml(agent.name) + '</strong><small class="agent-detection-state">' + escapeHtml(detectedLabel) + '</small></span><small class="mono">' +
        escapeHtml(meta) + '</small></span></label>' + (agent.custom ? '<button class="agent-edit" type="button" data-edit-custom-agent="' + escapeHtml(agent.id) + '" aria-label="' + t('lib.customAgent') + '">•••</button>' : '') + '</div>';
}

async function refreshAgentScan() {
    var button = document.getElementById('btn-scan-agents');
    var selected = {};
    document.querySelectorAll('input[name="managed-agent"]:checked').forEach(function (input) { selected[input.value] = true; });
    button.disabled = true;
    try {
        libraryState.agents = await api.get('/api/agents') || [];
        libraryState.agents.forEach(function (agent) { agent.configured = Boolean(selected[agent.id]); });
        showAgentManager();
        showToast(t('lib.agentScanUpdated'), 'info');
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function saveManagedAgents() {
    var button = document.getElementById('btn-save-agents');
    var selected = [];
    document.querySelectorAll('input[name="managed-agent"]:checked').forEach(function (input) { selected.push(input.value); });
    button.disabled = true;
    try {
        await api.put('/api/agents', { agents: selected });
        closeModal();
        showToast(t('lib.agentsUpdated'), 'success');
        renderLibrary();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

function showCustomAgentEditor(agent) {
    agent = agent || {};
    var content = '<div class="form-group"><label class="form-label" for="custom-agent-id">' + t('lib.agentId') + '</label><input class="input mono" id="custom-agent-id" value="' + escapeHtml(agent.id || '') + '" placeholder="my-agent"' + (agent.id ? ' readonly' : '') + '></div>' +
        '<div class="form-group"><label class="form-label" for="custom-agent-name">' + t('lib.agentName') + '</label><input class="input" id="custom-agent-name" value="' + escapeHtml(agent.name || '') + '" placeholder="My Agent"></div>' +
        '<div class="form-group"><label class="form-label" for="custom-agent-path">' + t('lib.agentPath') + '</label><input class="input mono" id="custom-agent-path" value="' + escapeHtml((agent.path || '').replace(/\/<skill-name>$/, '')) + '" placeholder="~/.my-agent/skills"></div>' +
        '<div class="form-group"><span class="form-label">' + t('lib.agentIcon') + '</span><div class="custom-icon-row"><img class="custom-icon-preview" id="custom-agent-icon-preview" src="' + agentIconSource(agent) + '" alt=""><input class="sr-only" type="file" id="custom-agent-icon" accept="image/png,image/jpeg,image/webp,image/svg+xml"><button class="btn btn-secondary" type="button" id="btn-choose-agent-icon">' + t('lib.chooseIcon') + '</button></div></div>';
    var actions = (agent.id ? '<button class="btn btn-danger" type="button" id="btn-delete-custom-agent">' + t('lib.deleteAgent') + '</button><div class="action-spacer"></div>' : '') +
        '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-primary" type="button" id="btn-save-custom-agent">' + t('lib.saveAgents') + '</button>';
    showModal(t('lib.customAgent'), content, actions);
    var iconValue = agent.icon || '';
    var fileInput = document.getElementById('custom-agent-icon');
    document.getElementById('btn-choose-agent-icon').addEventListener('click', function () { fileInput.click(); });
    fileInput.addEventListener('change', function () {
        var file = fileInput.files[0];
        if (!file) return;
        var reader = new FileReader();
        reader.addEventListener('load', function () {
            iconValue = String(reader.result || '');
            document.getElementById('custom-agent-icon-preview').src = iconValue;
        });
        reader.readAsDataURL(file);
    });
    document.getElementById('btn-save-custom-agent').addEventListener('click', function () { saveCustomAgent(iconValue); });
    if (agent.id) document.getElementById('btn-delete-custom-agent').addEventListener('click', function () { deleteCustomAgent(agent.id); });
}

async function saveCustomAgent(icon) {
    var button = document.getElementById('btn-save-custom-agent');
    button.disabled = true;
    try {
        await api.post('/api/agents/custom', {
            id: document.getElementById('custom-agent-id').value.trim(),
            name: document.getElementById('custom-agent-name').value.trim(),
            skillsPath: document.getElementById('custom-agent-path').value.trim(),
            icon: icon
        });
        closeModal();
        showToast(t('lib.agentsUpdated'));
        renderLibrary();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function deleteCustomAgent(id) {
    var button = document.getElementById('btn-delete-custom-agent');
    button.disabled = true;
    try {
        await api.del('/api/agents/' + encodeURIComponent(id));
        closeModal();
        showToast(t('lib.agentsUpdated'));
        renderLibrary();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
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
    var localTags = tagPickerMarkup('add-skill-tags', [], libraryState.tags, true);
    var sourceTags = tagPickerMarkup('git-source-tags', [], libraryState.tags, true);
    var commandTags = tagPickerMarkup('command-source-tags', [], libraryState.tags, true);
    var content = '<div class="import-mode" role="tablist"><button class="import-mode-option active" role="tab" aria-selected="true" type="button" data-import-mode="local">' +
        t('lib.importLocal') + '</button><button class="import-mode-option" role="tab" aria-selected="false" type="button" data-import-mode="git">' + t('lib.importGit') +
        '</button><button class="import-mode-option" role="tab" aria-selected="false" type="button" data-import-mode="command">' + t('lib.importCommand') + '</button></div>' +
        '<div class="import-pane" data-import-pane="local"><div class="form-group"><label class="form-label" for="add-skill-path">' + t('lib.skillPath') +
        '</label><div class="path-picker"><input class="input" id="add-skill-path" readonly placeholder="' + t('lib.chooseSkillPath') + '"><div class="path-picker-actions"><button class="btn btn-secondary" type="button" id="btn-choose-skill-path">' +
        uiIcon('folder') + t('lib.choosePath') + '</button><button class="btn btn-secondary" type="button" id="btn-choose-skill-zip">' + uiIcon('archive') + t('lib.chooseZIP') + '</button></div></div><p class="form-hint">' +
        t('lib.localImportHint') + '</p></div><div class="form-group"><label class="form-label" for="add-skill-tags">' +
        t('lib.selectTags') + '</label>' + localTags + '</div></div>' +
        '<div class="import-pane is-hidden" data-import-pane="git"><div class="form-group"><label class="form-label" for="git-source-input">' +
        t('lib.gitInput') + '</label><input class="input mono" id="git-source-input" placeholder="https://github.com/owner/skills.git"></div><div class="form-group"><label class="form-label" for="git-source-name">' +
        t('lib.gitSourceName') + '</label><input class="input" id="git-source-name" placeholder="' + t('lib.gitNamePlaceholder') + '"></div><div class="form-group"><label class="form-label" for="git-source-tags">' +
        t('lib.selectTags') + '</label>' + sourceTags + '</div></div>' +
        '<div class="import-pane is-hidden" data-import-pane="command"><div class="form-group"><label class="form-label" for="command-source-input">' +
        t('lib.commandInput') + '</label><input class="input mono" id="command-source-input" placeholder="npx skills add owner/skills"><p class="form-hint">' + t('lib.commandHint') +
        '</p></div><div class="form-group"><label class="form-label" for="command-source-name">' + t('lib.gitSourceName') + '</label><input class="input" id="command-source-name" placeholder="' +
        t('lib.gitNamePlaceholder') + '"></div><div class="form-group"><label class="form-label" for="command-source-tags">' + t('lib.selectTags') + '</label>' + commandTags + '</div></div>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-confirm-add">' + t('lib.addSkill') + '</button>';
    showModal(t('lib.addSkillTitle'), content, actions);
    var mode = 'local';
    document.getElementById('btn-choose-skill-path').addEventListener('click', function () { chooseSkillSource('directory'); });
    document.getElementById('btn-choose-skill-zip').addEventListener('click', function () { chooseSkillSource('archive'); });
    document.querySelectorAll('[data-import-mode]').forEach(function (button) {
        button.addEventListener('click', function () {
            mode = button.dataset.importMode;
            document.querySelectorAll('[data-import-mode]').forEach(function (item) { item.classList.toggle('active', item === button); });
            document.querySelectorAll('[data-import-mode]').forEach(function (item) { item.setAttribute('aria-selected', String(item === button)); });
            document.querySelectorAll('[data-import-pane]').forEach(function (pane) { pane.classList.toggle('is-hidden', pane.dataset.importPane !== mode); });
            document.getElementById('btn-confirm-add').textContent = mode === 'local' ? t('lib.addSkill') : t('lib.import');
            var input = document.querySelector('[data-import-pane="' + mode + '"] input');
            if (input) input.focus();
        });
    });
    document.getElementById('btn-confirm-add').addEventListener('click', function () {
        if (mode === 'local') doAddSkill(); else doImportGitSource(mode);
    });
}

async function chooseSkillSource(kind) {
    var button = document.getElementById(kind === 'archive' ? 'btn-choose-skill-zip' : 'btn-choose-skill-path');
    button.disabled = true;
    try {
        var result = await api.post('/api/dialogs/skill-' + kind, {});
        document.getElementById('add-skill-path').value = result.path;
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        button.disabled = false;
    }
}

async function doAddSkill() {
    var path = document.getElementById('add-skill-path').value.trim();
    if (!path) { showToast(t('lib.pathRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-add');
    button.disabled = true;
    try {
        await api.post('/api/skills', { path: path, tags: selectedTagValues('add-skill-tags'), source: '' });
        closeModal();
        showToast(t('lib.addedSuccess'));
        renderLibrary();
    } catch (err) { button.disabled = false; showToast(err.message, 'error'); }
}

async function doImportGitSource(mode) {
    var prefix = mode === 'command' ? 'command' : 'git';
    var name = document.getElementById(prefix + '-source-name').value.trim();
    var input = document.getElementById(prefix + '-source-input').value.trim();
    if (!input) { showToast(t(mode === 'command' ? 'lib.commandRequired' : 'lib.gitRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-add');
    button.disabled = true;
    try {
        var result = await api.post('/api/sources', {
            input: input, name: name, ref: '', paths: [], tags: selectedTagValues(prefix + '-source-tags'),
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
    skillDetailState.skill = skill;
    skillDetailState.savedTags = (skill.tags || []).slice().sort();
    var source = libraryState.gitSources[skill.source];
    var skillTags = skill.tags || [];
    var tagPicker = tagPickerMarkup('detail-skill-tags', skillTags, libraryState.tags, false);
    var sourceDetails = source ? '<div><dt>' + t('lib.gitUrl') + '</dt><dd class="mono detail-path">' + escapeHtml(source.url) + '</dd></div><div><dt>' +
        t('lib.gitRef') + '</dt><dd>' + escapeHtml(source.ref || 'default') + '</dd></div>' : '';
    var linked = skill.mode === 'symlink' && skill.projectRoot;
    var health = skillHealthMarkup(skill);
    var originDetails = linked ? '<div><dt>' + t('lib.source') + '</dt><dd class="mono detail-path">' + escapeHtml(skill.sourcePath || skill.path) + '</dd></div>' +
        '<div><dt>' + t('lib.effectivePath') + '</dt><dd class="mono detail-path">' + escapeHtml(skill.effectivePath || skill.path) + '</dd></div>' : '';
    var metadata = '<dl class="detail-list"><div><dt>' + t('lib.path') + '</dt><dd class="mono detail-path">' + escapeHtml(skill.path) + '</dd></div><div><dt>' +
        t('lib.hash') + '</dt><dd class="mono detail-hash">' + escapeHtml(skill.hash) + '</dd></div><div><dt>' + t('lib.revision') + '</dt><dd class="mono">' +
        shortRevision(skill.revision) + '</dd></div>' + originDetails + sourceDetails + '<div><dt>' + t('lib.added') + '</dt><dd>' + formatDate(skill.addedAt) + '</dd></div></dl>';
    var tagEditor = '<section class="skill-detail-panel skill-detail-tags"><div class="skill-detail-section-heading"><div><span class="form-label">' + t('lib.tags') +
        '</span><span class="skill-tag-state" id="skill-tag-state">' + escapeHtml(t('lib.tagsSaved')) + '</span></div><span class="muted" id="skill-tag-count">' +
        escapeHtml(t('lib.tagSelectionCount').replace('{0}', skillTags.length)) + '</span></div><div class="skill-tag-create"><input class="input" id="detail-new-tag" autocomplete="off" placeholder="' +
        escapeHtml(t('lib.tagNamePlaceholder')) + '" aria-label="' + escapeHtml(t('lib.newTag')) + '"><button class="btn btn-secondary btn-sm" type="button" id="btn-create-detail-tag">' +
        uiIcon('plus') + t('lib.createAndSelectTag') + '</button></div><p class="skill-tag-create-hint">' + escapeHtml(t('lib.quickAddTag')) + '</p>' + tagPicker +
        '<div class="skill-tag-actions"><button class="btn btn-primary" type="button" id="btn-save-skill-tags" disabled>' + uiIcon('check') + t('lib.saveTags') + '</button></div></section>';
    var content = '<div class="skill-detail-shell"><div class="detail-hero"><div class="skill-detail-identity"><span class="skill-detail-mark" aria-hidden="true">' + uiIcon('library') +
        '</span><div><div class="detail-name">' + escapeHtml(skill.name) + '</div><div class="mono muted">' + escapeHtml(skill.id) + '</div></div></div><div class="skill-detail-badges"><span class="badge badge-source">' +
        escapeHtml(displaySource(skill.source || 'local')) + '</span>' + health + '</div></div><div class="skill-detail-layout"><main class="skill-detail-main"><section class="skill-detail-panel skill-detail-overview"><div class="skill-detail-section-heading"><span class="form-label">' +
        t('lib.skillOverview') + '</span></div><p class="skill-detail-description">' + escapeHtml(skill.description || '-') + '</p></section><section class="skill-detail-panel skill-content-section"><div class="skill-detail-section-heading"><span class="form-label">' +
        t('lib.content') + '</span><span class="mono muted">SKILL.md</span></div>' + (skill.body ? '<pre class="skill-detail-content">' + escapeHtml(skill.body) + '</pre>' : '<div class="inline-empty">' +
        t('lib.noContent') + '</div>') + '</section></main><aside class="skill-detail-aside">' + tagEditor + '<section class="skill-detail-panel skill-detail-metadata"><div class="skill-detail-section-heading"><span class="form-label">' +
        t('lib.metadata') + '</span></div>' + metadata + '</section></aside></div></div>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.close') + '</button>' +
        (linked ? '<button class="btn btn-primary" type="button" id="btn-detach-skill">' + t('lib.detach') + '</button>' : '') +
        (source ? '<button class="btn btn-primary" type="button" id="btn-update-source">' + t('lib.updateSource') + '</button>' : '');
    showModal(t('lib.details'), content, actions);
    document.querySelector('.modal').classList.add('skill-detail-modal');
    document.getElementById('btn-save-skill-tags').addEventListener('click', function () { saveSkillTags(skill.id); });
    document.getElementById('btn-create-detail-tag').addEventListener('click', createSkillDetailTag);
    document.getElementById('detail-new-tag').addEventListener('keydown', function (event) {
        if (event.key === 'Enter') { event.preventDefault(); createSkillDetailTag(); }
    });
    bindSkillDetailTagChanges();
    if (source) document.getElementById('btn-update-source').addEventListener('click', function () { updateGitSource(source.name); });
    if (linked) document.getElementById('btn-detach-skill').addEventListener('click', function () { confirmDetachLibrarySkill(skill.id); });
}

function confirmDetachLibrarySkill(id) {
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-confirm-detach">' + t('lib.detach') + '</button>';
    showModal(t('lib.detach'), confirmationMarkup(t('lib.detachConfirm'), '', 'info'), actions);
    document.getElementById('btn-confirm-detach').addEventListener('click', function () { detachLibrarySkill(id); });
}

async function detachLibrarySkill(id) {
    var button = document.getElementById('btn-confirm-detach');
    if (button) button.disabled = true;
    try {
        await api.post('/api/skills/detach', { skill: id });
        closeModal();
        showToast(t('lib.detached'), 'success');
        await renderLibrary();
    } catch (err) {
        if (button) button.disabled = false;
        showToast(err.message, 'error');
    }
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

async function saveSkillTags(id) {
    var button = document.getElementById('btn-save-skill-tags');
    button.disabled = true;
    button.innerHTML = '<span class="spinner spinner-sm" aria-hidden="true"></span>' + t('lib.savingTags');
    try {
        var updated = await api.put('/api/skill-tags', { skill: id, tags: selectedTagValues('detail-skill-tags') });
        var managedTags = await api.get('/api/tags') || [];
        syncSkillTagState(updated, managedTags);
        showToast(t('lib.tagsUpdated'));
    } catch (err) {
        button.disabled = false;
        button.innerHTML = uiIcon('check') + t('lib.saveTags');
        showToast(err.message, 'error');
    }
}

function bindSkillDetailTagChanges() {
    var picker = document.getElementById('detail-skill-tags');
    if (!picker) return;
    picker.querySelectorAll('input[type="checkbox"]').forEach(function (input) {
        input.addEventListener('change', updateSkillDetailTagState);
    });
    updateSkillDetailTagState();
}

function updateSkillDetailTagState() {
    var values = selectedTagValues('detail-skill-tags').slice().sort();
    var dirty = values.join('\n') !== skillDetailState.savedTags.join('\n');
    var state = document.getElementById('skill-tag-state');
    var count = document.getElementById('skill-tag-count');
    var button = document.getElementById('btn-save-skill-tags');
    if (state) {
        state.textContent = t(dirty ? 'lib.tagsUnsaved' : 'lib.tagsSaved');
        state.classList.toggle('is-dirty', dirty);
    }
    if (count) count.textContent = t('lib.tagSelectionCount').replace('{0}', values.length);
    if (button) {
        button.disabled = !dirty;
        button.innerHTML = uiIcon('check') + t('lib.saveTags');
    }
}

async function createSkillDetailTag() {
    var input = document.getElementById('detail-new-tag');
    var button = document.getElementById('btn-create-detail-tag');
    var name = input.value.trim();
    if (!name || button.disabled) {
        if (!name) input.focus();
        return;
    }
    var selected = selectedTagValues('detail-skill-tags');
    button.disabled = true;
    input.removeAttribute('aria-invalid');
    try {
        var created = await api.post('/api/tags', { name: name });
        var managedTags = await api.get('/api/tags') || [];
        libraryState.tags = managedTags;
        if (typeof promptState !== 'undefined') promptState.tags = managedTags;
        if (!selected.includes(created.name)) selected.push(created.name);
        var picker = document.getElementById('detail-skill-tags');
        picker.outerHTML = tagPickerMarkup('detail-skill-tags', selected, managedTags, false);
        input.value = '';
        bindSkillDetailTagChanges();
        showToast(t('lib.tagCreated'));
        input.focus();
    } catch (err) {
        input.setAttribute('aria-invalid', 'true');
        showToast(err.message, 'error');
    } finally {
        button.disabled = false;
    }
}

function syncSkillTagState(updated, managedTags) {
    var detailSkill = Object.assign({}, skillDetailState.skill || {}, updated);
    skillDetailState.skill = detailSkill;
    skillDetailState.savedTags = (updated.tags || []).slice().sort();
    libraryState.tags = managedTags;
    if (typeof promptState !== 'undefined') promptState.tags = managedTags;
    var index = libraryState.skills.findIndex(function (skill) { return skill.id === updated.id; });
    var remainsVisible = !libraryState.activeTag || (updated.tags || []).includes(libraryState.activeTag);
    if (index >= 0 && remainsVisible) libraryState.skills[index] = Object.assign({}, libraryState.skills[index], updated);
    if (index >= 0 && !remainsVisible) libraryState.skills.splice(index, 1);

    var picker = document.getElementById('detail-skill-tags');
    if (picker) picker.outerHTML = tagPickerMarkup('detail-skill-tags', updated.tags || [], managedTags, false);
    bindSkillDetailTagChanges();
    refreshLibraryTagSurface(updated, remainsVisible);
}

function refreshLibraryTagSurface(updated, remainsVisible) {
    var cards = Array.prototype.slice.call(document.querySelectorAll('[data-skill-card-id]'));
    var card = cards.find(function (element) { return element.dataset.skillCardId === updated.id; });
    if (card && remainsVisible) {
        var list = card.querySelector('.tag-list');
        if (list) list.innerHTML = (updated.tags || []).map(function (tag) {
            return '<span class="tag">' + escapeHtml(displayTag(tag)) + '</span>';
        }).join('');
    } else if (card) {
        card.remove();
    }
    var filters = document.getElementById('skill-tag-filters');
    if (filters) {
        filters.innerHTML = libraryTagFiltersMarkup();
        bindLibraryTagFilters();
    }
    var visibleCount = libraryState.skills.filter(function (skill) {
        var query = libraryState.query.toLowerCase();
        return !query || [skill.name, skill.id, skill.description, skill.source].join(' ').toLowerCase().includes(query);
    }).length;
    var count = document.getElementById('library-skill-count');
    if (count) count.textContent = t('lib.skillCount').replace('{0}', visibleCount);
    var grid = document.getElementById('library-skill-grid');
    if (grid && !grid.querySelector('.skill-card')) {
        grid.outerHTML = '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('library') + '</div><div class="empty-state-title">' +
            t('lib.noSkills') + '</div><div class="empty-state-desc">' + t('lib.noSkillsDesc') + '</div></div>';
    }
}

async function refreshLibraryAfterTagChange() {
    var url = '/api/skills' + (libraryState.activeTag ? '?tag=' + encodeURIComponent(libraryState.activeTag) : '');
    var results = await Promise.all([api.get(url), api.get('/api/tags')]);
    libraryState.skills = results[0] || [];
    libraryState.tags = results[1] || [];
    paintLibrary();
}

function tagPickerMarkup(id, selectedValues, availableTags, useDefaults) {
    var selected = {};
    (selectedValues || []).forEach(function (name) { selected[name] = true; });
    var values = (availableTags || []).slice();
    (selectedValues || []).forEach(function (name) {
        if (!values.some(function (tag) { return tag.name === name; })) values.push({ name: name, count: 0 });
    });
    values.sort(function (left, right) { return left.name.localeCompare(right.name); });
    if (!(selectedValues || []).length && useDefaults) {
        values.forEach(function (tag) { if (tag.default) selected[tag.name] = true; });
    }
    if (!values.length) return '<div class="tag-picker-empty" id="' + escapeHtml(id) + '" role="group" aria-label="' +
        escapeHtml(t('lib.selectTags')) + '">' + escapeHtml(t('lib.noManagedTags')) + '</div>';
    return '<div class="tag-picker" id="' + escapeHtml(id) + '" role="group" aria-label="' + escapeHtml(t('lib.selectTags')) + '">' + values.map(function (tag) {
        return '<label class="tag-picker-option"><input type="checkbox" value="' + escapeHtml(tag.name) + '"' + (selected[tag.name] ? ' checked' : '') +
            '><span class="tag-picker-copy"><strong>' + escapeHtml(displayTag(tag.name)) + '</strong><span>' +
            (tag.default ? '<small class="tag-default">' + escapeHtml(t('lib.defaultTag')) + '</small>' : '') +
            '<small>' + escapeHtml(t('lib.tagUsage').replace('{0}', Number(tag.count || 0))) + '</small></span></span></label>';
    }).join('') + '</div>';
}

function selectedTagValues(id) {
    var values = [];
    document.querySelectorAll('#' + id + ' input[type="checkbox"]:checked').forEach(function (input) { values.push(input.value); });
    return values;
}

async function showManageTagsModal() {
    try {
        var managedTags = await api.get('/api/tags') || [];
        libraryState.tags = managedTags;
        if (typeof promptState !== 'undefined') promptState.tags = managedTags;
        var content = '<div class="tag-manager-create"><div><label class="form-label" for="new-managed-tag">' + t('lib.newTag') + '</label><input class="input" id="new-managed-tag" placeholder="' +
            escapeHtml(t('lib.tagNamePlaceholder')) + '" autocomplete="off"></div><button class="btn btn-primary" type="button" id="btn-create-tag">' + uiIcon('plus') + t('lib.createTag') + '</button></div>';
        content += managedTags.length ? '<div class="tag-manager">' : '<div class="inline-empty">' + t('lib.noTags') + '</div>';
        managedTags.forEach(function (tag) {
            var locked = tag.default || tag.count > 0;
            content += '<div class="tag-manager-row"><div class="tag-manager-main"><span class="tag">' + escapeHtml(displayTag(tag.name)) + '</span>' +
                (tag.default ? '<span class="badge badge-muted">' + escapeHtml(t('lib.defaultTag')) + '</span>' : '') + '<span class="muted">' +
                escapeHtml(t('lib.tagUsage').replace('{0}', Number(tag.count || 0))) + '</span></div><div class="tag-manager-actions"><button class="btn btn-ghost btn-sm" type="button" data-rename-tag="' +
                escapeHtml(tag.name) + '">' + t('lib.renameTag') + '</button><button class="btn btn-danger btn-sm" type="button" data-delete-tag="' + escapeHtml(tag.name) + '"' +
                (locked ? ' disabled title="' + escapeHtml(t(tag.default ? 'lib.defaultTagLocked' : 'lib.tagInUse')) + '"' : '') + '>' + t('lib.remove') + '</button></div></div>';
        });
        if (managedTags.length) content += '</div>';
        showModal(t('lib.manageTags'), content, '<button class="btn btn-primary" type="button" data-close-modal>' + t('lib.close') + '</button>');
        document.getElementById('btn-create-tag').addEventListener('click', createManagedTag);
        document.getElementById('new-managed-tag').addEventListener('keydown', function (event) { if (event.key === 'Enter') createManagedTag(); });
    } catch (err) {
        showToast(err.message, 'error');
        return;
    }
    document.querySelectorAll('[data-rename-tag]').forEach(function (button) {
        button.addEventListener('click', function () { showRenameTagModal(button.dataset.renameTag); });
    });
    document.querySelectorAll('[data-delete-tag]').forEach(function (button) {
        button.addEventListener('click', function () { deleteManagedTag(button.dataset.deleteTag, button); });
    });
}

async function createManagedTag() {
    var input = document.getElementById('new-managed-tag');
    var button = document.getElementById('btn-create-tag');
    var name = input.value.trim();
    if (!name) return;
    button.disabled = true;
    try {
        await api.post('/api/tags', { name: name });
        showToast(t('lib.tagCreated'));
        await refreshTagOwningPage();
        await showManageTagsModal();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function deleteManagedTag(name, button) {
    button.disabled = true;
    try {
        await api.del('/api/tags/' + encodeURIComponent(name));
        showToast(t('lib.tagDeleted'));
        await refreshTagOwningPage();
        await showManageTagsModal();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function refreshTagOwningPage() {
    if (isCurrentPage('prompts') && typeof renderPrompts === 'function') {
        await renderPrompts();
    } else {
        await renderLibrary();
    }
}

function showRenameTagModal(oldName) {
    var content = '<div class="form-group"><label class="form-label" for="rename-tag-input">' + t('lib.newTagName') + '</label>' +
        '<input class="input" id="rename-tag-input" value="' + escapeHtml(oldName) + '"></div>';
    showModal(t('lib.renameTag') + ': ' + displayTag(oldName), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') +
        '</button><button class="btn btn-primary" type="button" id="btn-do-rename">' + t('lib.renameTag') + '</button>');
    document.getElementById('btn-do-rename').addEventListener('click', async function () {
        var next = document.getElementById('rename-tag-input').value.trim();
        if (!next || next === oldName) return;
        try {
            await api.post('/api/tags/rename', { old: oldName, new: next });
            if (libraryState.activeTag === oldName) libraryState.activeTag = next;
            if (typeof promptState !== 'undefined' && promptState.activeTag === oldName) promptState.activeTag = next;
            showToast(t('lib.renamedTag'));
            await refreshTagOwningPage();
            await showManageTagsModal();
        } catch (err) { showToast(err.message, 'error'); }
    });
}

function confirmRemoveSkill(id) {
    var content = confirmationMarkup(t('lib.confirmRemove') + ' ' + id + '?', t('lib.confirmRemoveNote'), 'danger');
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
    return '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('sparkles') + '</div><div class="empty-state-title">' + title +
        '</div><div class="empty-state-desc">' + escapeHtml(message) + '</div></div>';
}

window.renderLibrary = renderLibrary;

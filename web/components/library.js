/* global api, showToast, showModal, closeModal, displayTag, formatDate, shortHash, shortRevision, escapeHtml, isCurrentPage, t, uiIcon, confirmationMarkup */

var libraryState = { skills: [], tags: [], agents: [], activeTag: '', query: '', enabled: {}, gitSources: {}, summary: {} };
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
        var results = await Promise.all([api.get(url), api.get('/api/tags'), api.get('/api/status'), api.get('/api/sources'), api.get('/api/agents'), api.get('/api/dashboard')]);
        if (!isCurrentPage('library')) return;
        libraryState.skills = results[0] || [];
        libraryState.tags = results[1] || [];
        libraryState.agents = results[4] || [];
        libraryState.summary = results[5] || {};
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
    var html = '<div class="page"><div class="page-header"><div><h1 class="page-title">' + t('lib.title') + '</h1>' +
        '<p class="page-subtitle">' + t('lib.skillCount').replace('{0}', visible.length) + '</p></div><div class="header-actions">' +
        '<button class="btn btn-secondary" type="button" id="btn-manage-tags">' + uiIcon('tags') + t('lib.manageTags') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-add-skill">' + uiIcon('plus') + t('lib.addSkill') + '</button></div></div>';

    html += librarySummaryMarkup();
    html += agentManagementBand();

    html += '<div class="library-tools"><label class="search-box"><span class="sr-only">' + t('lib.search') + '</span>' +
        '<span class="search-mark" aria-hidden="true">' + uiIcon('search') + '</span><input class="input" id="skill-search" value="' + escapeHtml(libraryState.query) +
        '" placeholder="' + t('lib.searchPlaceholder') + '"></label>';
    html += '<div class="filter-bar"><span class="filter-label">' + t('lib.tags') + '</span><div class="tag-filter-list">' +
        '<button class="tag clickable' + (!libraryState.activeTag ? ' active' : '') + '" type="button" data-tag="">' + t('lib.all') + '</button>';
    libraryState.tags.forEach(function (tag) {
        html += '<button class="tag clickable' + (libraryState.activeTag === tag.name ? ' active' : '') + '" type="button" data-tag="' +
            escapeHtml(tag.name) + '">' + escapeHtml(displayTag(tag.name)) + ' <small>' + tag.count + '</small></button>';
    });
    html += '</div></div></div>';

    if (!visible.length) {
        html += '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('library') + '</div><div class="empty-state-title">' + t('lib.noSkills') +
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
    document.getElementById('btn-manage-agents').addEventListener('click', showAgentManager);
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
    return '<article class="card skill-card"><div class="skill-header"><div><div class="skill-name">' + escapeHtml(skill.name) +
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
    var content = '<div class="import-mode" role="tablist"><button class="import-mode-option active" type="button" data-import-mode="local">' +
        t('lib.importLocal') + '</button><button class="import-mode-option" type="button" data-import-mode="git">' + t('lib.importGit') + '</button></div>' +
        '<div class="import-pane" data-import-pane="local"><div class="form-group"><label class="form-label" for="add-skill-path">' + t('lib.skillPath') +
        '</label><div class="path-picker"><input class="input" id="add-skill-path" readonly placeholder="' + t('lib.chooseSkillPath') + '"><button class="btn btn-secondary" type="button" id="btn-choose-skill-path">' + t('lib.choosePath') + '</button></div></div><div class="form-group"><label class="form-label" for="add-skill-tags">' +
        t('lib.tagsComma') + '</label><input class="input" id="add-skill-tags" placeholder="' + t('lib.skillTagsPlaceholder') + '"></div></div>' +
        '<div class="import-pane is-hidden" data-import-pane="git"><div class="form-group"><label class="form-label" for="git-source-input">' +
        t('lib.gitInput') + '</label><input class="input mono" id="git-source-input" placeholder="npx skills add jakubkrehel/skills"></div><div class="form-group"><label class="form-label" for="git-source-name">' +
        t('lib.gitSourceName') + '</label><input class="input" id="git-source-name" placeholder="' + t('lib.gitNamePlaceholder') + '"></div><div class="form-group"><label class="form-label" for="git-source-tags">' +
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
    var input = document.getElementById('git-source-input').value.trim();
    if (!input) { showToast(t('lib.gitRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-add');
    button.disabled = true;
    try {
        var result = await api.post('/api/sources', {
            input: input, name: name, ref: '', paths: [], tags: splitList(document.getElementById('git-source-tags').value.trim()),
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
    var linked = skill.mode === 'symlink' && skill.projectRoot;
    var health = skillHealthMarkup(skill);
    var originDetails = linked ? '<div><dt>' + t('lib.source') + '</dt><dd class="mono detail-path">' + escapeHtml(skill.sourcePath || skill.path) + '</dd></div>' +
        '<div><dt>' + t('lib.effectivePath') + '</dt><dd class="mono detail-path">' + escapeHtml(skill.effectivePath || skill.path) + '</dd></div>' : '';
    var content = '<div class="detail-hero"><div><div class="detail-name">' + escapeHtml(skill.name) + '</div><div class="mono muted">' + escapeHtml(skill.id) +
        '</div></div><span class="badge badge-source">' + escapeHtml(displaySource(skill.source || 'local')) + '</span></div><dl class="detail-list"><div><dt>' +
        t('lib.description') + '</dt><dd>' + escapeHtml(skill.description || '-') + '</dd></div><div><dt>' + t('lib.path') + '</dt><dd class="mono detail-path">' +
        escapeHtml(skill.path) + '</dd></div><div><dt>' + t('lib.hash') + '</dt><dd class="mono">' + escapeHtml(skill.hash) + '</dd></div><div><dt>' +
        t('lib.revision') + '</dt><dd class="mono">' + shortRevision(skill.revision) + '</dd></div>' + originDetails + sourceDetails + '<div><dt>' + t('lib.added') +
        '</dt><dd>' + formatDate(skill.addedAt) + '</dd></div></dl><section class="skill-detail-section"><div class="form-group"><label class="form-label">' + t('lib.tags') +
        '</label><div class="tag-list detail-tags">' + tags + '</div></div><div class="inline-form"><input class="input" id="detail-new-tag" placeholder="' + t('lib.tagName') + '"><button class="btn btn-secondary" type="button" id="btn-add-tag">' + t('lib.addTag') + '</button></div></section>' +
        health + '<section class="skill-detail-section skill-content-section"><div class="skill-detail-section-heading"><span class="form-label">' + t('lib.content') + '</span><span class="mono muted">SKILL.md</span></div>' +
        (skill.body ? '<pre class="skill-detail-content">' + escapeHtml(skill.body) + '</pre>' : '<div class="inline-empty">' + t('lib.noContent') + '</div>') + '</section>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.close') + '</button>' +
        (linked ? '<button class="btn btn-primary" type="button" id="btn-detach-skill">' + t('lib.detach') + '</button>' : '') +
        (source ? '<button class="btn btn-primary" type="button" id="btn-update-source">' + t('lib.updateSource') + '</button>' : '');
    showModal(t('lib.details'), content, actions);
    document.querySelector('.modal').classList.add('skill-detail-modal');
    document.getElementById('btn-add-tag').addEventListener('click', function () { addSkillTag(skill.id); });
    document.querySelectorAll('[data-remove-tag]').forEach(function (button) {
        button.addEventListener('click', function () { removeSkillTag(skill.id, button.dataset.removeTag); });
    });
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

/* global api, showToast, showModal, closeModal, escapeHtml, statusBadgeClass, formatDate, isCurrentPage, t */

var projectState = { projects: [], skills: [], agents: [], selectedID: '', detail: null, agentFilter: 'all' };

async function renderProjects() {
    var container = document.getElementById('main-content');
    try {
        var results = await Promise.all([api.get('/api/projects'), api.get('/api/skills'), api.get('/api/agents')]);
        if (!isCurrentPage('projects')) return;
        projectState.projects = results[0] || [];
        projectState.skills = results[1] || [];
        projectState.agents = results[2] || [];
        if (!projectState.projects.length) {
            projectState.selectedID = '';
            projectState.detail = null;
            paintProjects();
            return;
        }
        if (!projectState.projects.some(function (project) { return project.id === projectState.selectedID; })) {
            projectState.selectedID = projectState.projects[0].id;
        }
        await loadProjectDetail(projectState.selectedID);
    } catch (err) {
        if (!isCurrentPage('projects')) return;
        container.innerHTML = '<div class="empty-state"><div class="empty-state-mark">!</div><div class="empty-state-title">' +
            t('proj.title') + '</div><div class="empty-state-desc">' + escapeHtml(err.message) + '</div></div>';
        showToast(err.message, 'error');
    }
}

async function loadProjectDetail(id, repaint) {
    try {
        projectState.selectedID = id;
        var detail = await api.get('/api/projects/' + encodeURIComponent(id));
        if (projectState.selectedID !== id) return;
        projectState.detail = detail;
        if (repaint !== false) paintProjects();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function paintProjects() {
    if (!isCurrentPage('projects')) return;
    var container = document.getElementById('main-content');
    var headerActions = '<button class="btn btn-primary" type="button" id="btn-add-project">+ ' + t('proj.add') + '</button>';
    if (!projectState.projects.length) {
        headerActions += '<button class="btn btn-secondary" type="button" id="btn-configure-agents">' + t('proj.configureAgents') + '</button>';
    }
    var html = '<div class="page animate-in"><div class="page-header"><div><h1 class="page-title">' + t('proj.title') +
        '</h1><p class="page-subtitle">' + t('proj.list') + '</p></div><div class="header-actions">' + headerActions + '</div></div>';
    if (!projectState.projects.length) {
        html += '<div class="empty-state"><div class="empty-state-mark">P</div><div class="empty-state-title">' + t('proj.empty') +
            '</div><div class="empty-state-desc">' + t('proj.emptyDesc') + '</div></div></div>';
        container.innerHTML = html;
        document.getElementById('btn-add-project').addEventListener('click', showAddProjectModal);
        document.getElementById('btn-configure-agents').addEventListener('click', showAgentSettingsModal);
        return;
    }
    html += '<div class="project-layout"><section class="project-list" aria-label="' + t('proj.list') + '">';
    projectState.projects.forEach(function (project) { html += projectListItem(project); });
    html += '</section><section class="project-detail">' + projectDetailMarkup() + '</section></div></div>';
    container.innerHTML = html;
    document.getElementById('btn-add-project').addEventListener('click', showAddProjectModal);
    container.querySelectorAll('[data-project-id]').forEach(function (button) {
        button.addEventListener('click', function () { loadProjectDetail(button.dataset.projectId); });
    });
    container.querySelectorAll('[data-project-agent-filter]').forEach(function (button) {
        button.addEventListener('click', function () {
            projectState.agentFilter = button.dataset.projectAgentFilter;
            paintProjects();
        });
    });

    var detail = projectState.detail;
    if (!detail) return;
    var configureAgentsButton = document.getElementById('btn-configure-agents');
    if (configureAgentsButton) configureAgentsButton.addEventListener('click', showAgentSettingsModal);
    var addSkillButton = document.getElementById('btn-project-add-skill');
    if (addSkillButton) addSkillButton.addEventListener('click', showProjectSkillDeployModal);
    var refreshButton = document.getElementById('btn-project-refresh');
    if (refreshButton) refreshButton.addEventListener('click', function () { loadProjectDetail(projectState.selectedID); });
    var unregisterButton = document.getElementById('btn-project-unregister');
    if (unregisterButton) unregisterButton.addEventListener('click', unregisterProject);
    container.querySelectorAll('[data-unlink-skill]').forEach(function (button) {
        button.addEventListener('click', function () { unlinkProjectSkill(button.dataset.unlinkSkill); });
    });
    container.querySelectorAll('[data-project-skill-details]').forEach(function (button) {
        button.addEventListener('click', function () { showProjectSkillDetails(button.dataset.projectSkillDetails); });
    });
}

function projectListItem(project) {
    var selected = project.id === projectState.selectedID;
    var status = project.exists ? t('proj.ready') : t('proj.missing');
    var skillCount = typeof project.skillCount === 'number' ? project.skillCount : (project.activationCount || 0);
    var counts = project.agentCounts || {};
    var agentSummary = Object.keys(counts).sort().filter(function (agent) { return Number(counts[agent] || 0) > 0; }).map(function (agent) {
        return projectAgentName(agent) + ' ' + Number(counts[agent]);
    }).join(' / ');
    return '<button class="project-list-item' + (selected ? ' active' : '') + '" type="button" data-project-id="' + escapeHtml(project.id) + '">' +
        '<span class="project-list-name">' + escapeHtml(project.id) + '</span><span class="project-list-path mono">' + escapeHtml(project.path) +
        '</span><span class="project-list-meta"><span class="badge ' + (project.exists ? 'badge-ok' : 'badge-error') + '">' + status +
        '</span><span class="muted">' + skillCount + ' ' + t('proj.skills') + '</span></span>' +
        (agentSummary ? '<span class="project-list-agents">' + escapeHtml(agentSummary) + '</span>' : '') + '</button>';
}

function projectDetailMarkup() {
    var detail = projectState.detail;
    if (!detail) return '<div class="inline-empty">' + t('loading') + '</div>';
    var project = detail.project;
    var html = '<div class="project-detail-header"><div><h2 class="section-title project-detail-name">' + escapeHtml(project.id) +
        '</h2><div class="project-detail-status"><span class="badge ' + (detail.exists ? 'badge-ok' : 'badge-error') + '">' + (detail.exists ? t('proj.ready') : t('proj.missing')) +
        '</span><span class="mono project-path">' + escapeHtml(project.path) + '</span></div></div><div class="header-actions"><button class="btn btn-secondary btn-sm" type="button" id="btn-configure-agents">' +
        t('proj.configureAgents') + '</button><button class="btn btn-primary btn-sm" type="button" id="btn-project-add-skill">+ ' + t('proj.addSkill') + '</button><button class="btn btn-secondary btn-sm" type="button" id="btn-project-refresh">' +
        t('proj.scan') + '</button><button class="btn btn-danger btn-sm" type="button" id="btn-project-unregister">' + t('proj.unregister') + '</button></div></div>';
    html += projectScanMarkup(detail);
    html += projectPlanMarkup(detail.plan);
    return html;
}

function projectScanMarkup(detail) {
    var scan = detail.scan || { skillCount: 0, agentCounts: {}, agents: [], skills: [] };
    var skills = scan.skills || [];
    var agents = scan.agents || [];
    var filter = projectState.agentFilter || 'all';
    if (filter !== 'all' && !agents.some(function (agent) { return agent.id === filter; })) {
        filter = 'all';
        projectState.agentFilter = filter;
    }
    var visible = skills.filter(function (skill) {
        return filter === 'all' || (skill.agents || []).indexOf(filter) >= 0;
    });
    var html = '<section class="project-section project-scan-section"><div class="section-heading"><div><h3 class="section-title">' + t('proj.scanTitle') +
        '</h3><p class="section-caption">' + t('proj.scanDesc') + '</p></div><span class="project-scan-time">' + t('proj.lastScan') + ': ' + formatDate(scan.scannedAt) + '</span></div>';
    html += '<div class="project-scan-summary"><div class="project-scan-stat"><strong>' + Number(scan.skillCount || 0) + '</strong><span>' + t('proj.skills') + '</span></div>' +
        agents.map(projectScanAgentStat).join('') + '</div>';
    html += '<div class="project-scan-toolbar"><div class="project-agent-filter" role="tablist" aria-label="' + t('proj.agents') + '">' +
        projectAgentFilterButton('all', t('proj.allAgents'), filter) + agents.map(function (agent) { return projectAgentFilterButton(agent.id, agent.label || projectAgentName(agent.id), filter); }).join('') +
        '</div><span class="muted">' + visible.length + ' / ' + skills.length + ' ' + t('proj.skills') + '</span></div>';
    if (scan.errors && scan.errors.length) {
        html += '<div class="project-scan-warning">' + scan.errors.map(function (message) { return '<span>' + escapeHtml(message) + '</span>'; }).join('') + '</div>';
    }
    if (!visible.length) {
        html += '<div class="inline-empty">' + (skills.length ? t('proj.noFilteredSkills') : t('proj.noScannedSkills')) + '</div>';
    } else {
        html += '<div class="project-skill-list">' + visible.map(function (skill) { return projectScanSkillRow(skill, detail); }).join('') + '</div>';
    }
    return html + '</section>';
}

function projectScanAgentStat(agent) {
    return '<div class="project-scan-stat"><strong>' + Number(agent.skillCount || 0) + '</strong><span>' + escapeHtml(agent.label || projectAgentName(agent.id)) + '</span></div>';
}

function projectAgentFilterButton(agent, label, active) {
    return '<button class="project-agent-filter-btn' + (agent === active ? ' active' : '') + '" type="button" role="tab" aria-selected="' + (agent === active) +
        '" data-project-agent-filter="' + agent + '">' + escapeHtml(label) + '</button>';
}

function projectScanSkillRow(skill, detail) {
    var activation = projectActivationFor(skill, detail.activations || []);
    var agents = (skill.agents || []).map(function (agent) { return projectAgentBadge(agent); }).join('');
    var statusLabel = projectScanStatusLabel(skill.status);
    var managed = activation ? '<span class="badge badge-source">' + escapeHtml(t('proj.managed')) + '</span>' : '<span class="badge badge-muted">' + escapeHtml(t('proj.external')) + '</span>';
    var action = '<button class="btn btn-ghost btn-sm" type="button" data-project-skill-details="' + escapeHtml(skill.id) + '">' + t('proj.viewDetails') + '</button>' +
        (activation ? '<button class="btn btn-danger btn-sm" type="button" data-unlink-skill="' + escapeHtml(activation.skillId) + '">' + t('proj.unlink') + '</button>' : '');
    var issue = (skill.issues || []).map(function (message) { return '<div class="project-skill-issue">' + escapeHtml(message) + '</div>'; }).join('');
    return '<article class="project-skill-row"><div class="project-skill-main"><div class="project-skill-title"><strong>' + escapeHtml(skill.name || skill.id) +
        '</strong></div><p class="project-skill-description">' + escapeHtml(skill.description || t('proj.noDescription')) +
        '</p>' + issue + '</div><div class="project-skill-agents">' + agents + '</div><div class="project-skill-state"><div class="project-skill-badges"><span class="badge ' +
        statusBadgeClass(skill.status || 'ok') + '">' + escapeHtml(statusLabel) + '</span>' + managed + '</div>' +
        '</div><div class="project-skill-action">' + action + '</div></article>';
}

function showProjectSkillDeployModal() {
    var content = '<div class="form-group"><label class="form-label" for="project-skill-select">' + t('proj.skill') + '</label><select class="select" id="project-skill-select">' +
        '<option value="">' + t('proj.selectSkill') + '</option>';
    projectState.skills.forEach(function (skill) {
        content += '<option value="' + escapeHtml(skill.id) + '">' + escapeHtml(skill.id) + '</option>';
    });
    content += '</select></div><div class="form-group"><span class="form-label">' + t('proj.agents') + '</span><div class="choice-grid">' +
        projectState.agents.filter(function (agent) { return agent.enabled; }).map(function (agent) {
            return '<label class="check-option"><input type="checkbox" name="project-agent" value="' + escapeHtml(agent.id) + '" checked><span>' + escapeHtml(agent.name || projectAgentName(agent.id)) + '</span></label>';
        }).join('') + '</div></div>' +
        '<div class="form-group"><span class="form-label">' + t('proj.mode') + '</span><div class="import-mode project-mode"><button class="import-mode-option active" type="button" data-project-mode="link">' +
        t('proj.link') + '</button><button class="import-mode-option" type="button" data-project-mode="copy">' + t('proj.copy') + '</button></div></div>';
    showModal(t('proj.addSkill'), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-primary" type="button" id="btn-confirm-project-deploy">' + t('proj.addSkill') + '</button>');
    document.querySelectorAll('[data-project-mode]').forEach(function (button) {
        button.addEventListener('click', function () {
            document.querySelectorAll('[data-project-mode]').forEach(function (item) { item.classList.toggle('active', item === button); });
        });
    });
    document.getElementById('btn-confirm-project-deploy').addEventListener('click', deployProjectSkill);
}

async function showProjectSkillDetails(skillID) {
    try {
        var detail = await api.get('/api/projects/' + encodeURIComponent(projectState.selectedID) + '/skills/' + encodeURIComponent(skillID));
        var documents = detail.documents || [];
        var content = documents.map(projectSkillDocumentMarkup).join('');
        showModal(t('proj.skillDetails'), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.close') + '</button>');
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function projectSkillDocumentMarkup(document) {
    var metadata = document.metadata && Object.keys(document.metadata).length ? '<div class="form-group"><label class="form-label">' + t('proj.metadata') + '</label><pre class="project-skill-content">' + escapeHtml(JSON.stringify(document.metadata, null, 2)) + '</pre></div>' : '';
    var body = document.body ? '<pre class="project-skill-content">' + escapeHtml(document.body) + '</pre>' : '<p class="muted">' + t('proj.noContent') + '</p>';
    return '<section class="project-skill-document"><div class="detail-hero"><div><div class="detail-name">' + escapeHtml(document.name) + '</div><div class="mono muted">' + escapeHtml(projectAgentName(document.agent)) + '</div></div><span class="mono muted">' + escapeHtml((document.hash || '').substring(0, 12)) + '</span></div><dl class="detail-list"><div><dt>' +
        t('lib.description') + '</dt><dd>' + escapeHtml(document.description || '-') + '</dd></div><div><dt>' + t('proj.skillPath') + '</dt><dd class="mono detail-path">' + escapeHtml(document.path) + '</dd></div></dl>' + metadata + '<div class="form-group"><label class="form-label">' + t('proj.content') + '</label>' + body + '</div></section>';
}

function projectActivationFor(skill, activations) {
    return activations.find(function (activation) {
        return activation.skillId === skill.id || activation.name === skill.id || activation.name === skill.name;
    });
}

function projectAgentName(agent) {
    var configured = projectState.agents.find(function (item) { return item.id === agent; });
    if (configured && configured.name) return configured.name;
    if (agent === 'claude') return t('proj.claudeCode');
    if (agent === 'codex') return t('proj.codex');
    if (agent === 'cursor') return 'Cursor';
    if (agent === 'agent') return 'Agent';
    if (agent === 'agents') return 'Agents';
    return '.' + agent;
}

function projectAgentBadge(agent) {
    var logo = agent === 'claude' ? '/assets/claude.svg' : (agent === 'codex' ? '/assets/codex.svg' : '');
    var agentClass = agent === 'claude' || agent === 'codex' ? ' project-agent-' + agent : '';
    return '<span class="project-agent-badge' + agentClass + '">' + (logo ? '<img src="' + logo + '" alt="">' : '') + '<span>' + escapeHtml(projectAgentName(agent)) + '</span></span>';
}

function projectScanStatusLabel(status) {
    if (status === 'error') return t('proj.scanError');
    if (status === 'warning') return t('proj.scanWarning');
    return t('proj.scanOk');
}

function projectPlanMarkup(plan) {
    var operations = (plan && plan.operations) || [];
    var html = '<section class="project-section"><div class="section-heading"><div><h3 class="section-title">' + t('proj.statusTitle') + '</h3><p class="section-caption">' +
        t('proj.statusDesc') + '</p></div></div>';
    if (!operations.length) return html + '<div class="inline-empty">' + t('proj.statusEmpty') + '</div></section>';
    html += '<div class="table-wrap"><table><thead><tr><th>' + t('proj.skill') + '</th><th>' + t('proj.target') + '</th><th>' + t('proj.mode') + '</th><th>' + t('proj.statusTitle') + '</th></tr></thead><tbody>';
    operations.forEach(function (operation) {
        html += '<tr><td>' + escapeHtml(operation.skillId) + '</td><td class="mono cell-path" title="' + escapeHtml(operation.target) + '">' + escapeHtml(operation.target) +
            '</td><td>' + escapeHtml(operation.mode) + '</td><td><span class="badge ' + statusBadgeClass(operation.status) + '">' + escapeHtml(operation.status) + '</span></td></tr>';
    });
    return html + '</tbody></table></div></section>';
}

function showAddProjectModal() {
    var content = '<div class="form-group"><label class="form-label" for="project-path-input">' + t('proj.path') + '</label><div class="path-picker"><input class="input" id="project-path-input" readonly placeholder="' + t('proj.chooseProjectPath') + '"><button class="btn btn-secondary" type="button" id="btn-choose-project-path">' + t('lib.choosePath') + '</button></div></div>';
    showModal(t('proj.register'), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-primary" type="button" id="btn-confirm-project">' + t('proj.register') + '</button>');
    document.getElementById('btn-choose-project-path').addEventListener('click', chooseProjectDirectory);
    document.getElementById('btn-confirm-project').addEventListener('click', addProject);
}

function showAgentSettingsModal() {
    var agents = projectState.agents || [];
    var content = '<p class="form-hint" style="margin:0 0 14px">' + t('proj.agentFoldersDesc') + '</p><div class="selection-list">';
    content += agents.map(function (agent) {
        return '<label class="check-option"><input type="checkbox" data-agent-enabled="' + escapeHtml(agent.id) + '"' + (agent.enabled ? ' checked' : '') + '><span><strong>' + escapeHtml(agent.name || agent.id) + '</strong><small>' + escapeHtml(agent.userPath) + ' &middot; ' + escapeHtml(agent.projectPath) + '</small></span><button class="icon-btn btn-agent-edit" type="button" data-agent-edit="' + escapeHtml(agent.id) + '" aria-label="' + escapeHtml(t('proj.editAgent')) + '">&#9998;</button></label>';
    }).join('');
    content += '</div>';
    var actions = '<button class="btn btn-secondary" type="button" id="btn-add-agent">+ ' + t('proj.addAgent') + '</button><button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-primary" type="button" id="btn-save-agent-settings">' + t('proj.save') + '</button>';
    showModal(t('proj.agentFolders'), content, actions);
    document.getElementById('btn-add-agent').addEventListener('click', function () { showAgentEditor(null); });
    document.getElementById('btn-save-agent-settings').addEventListener('click', saveAgentSettings);
    document.querySelectorAll('[data-agent-edit]').forEach(function (button) {
        button.addEventListener('click', function (event) {
            event.preventDefault();
            showAgentEditor(projectState.agents.find(function (agent) { return agent.id === button.dataset.agentEdit; }));
        });
    });
}

async function saveAgentSettings() {
    var button = document.getElementById('btn-save-agent-settings');
    button.disabled = true;
    try {
        var updates = projectState.agents.map(function (agent) {
            var checkbox = document.querySelector('[data-agent-enabled="' + agent.id + '"]');
            return api.post('/api/agents', { id: agent.id, name: agent.name, userPath: agent.userPath, projectPath: agent.projectPath, enabled: Boolean(checkbox && checkbox.checked) });
        });
        await Promise.all(updates);
        closeModal();
        showToast(t('proj.agentSaved'));
        await renderProjects();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

function showAgentEditor(agent) {
    agent = agent || { id: '', name: '', userPath: '~/.', projectPath: '.', enabled: true, builtIn: false };
    var content = '<div class="form-group"><label class="form-label" for="agent-id-input">' + t('proj.agentID') + '</label><input class="input" id="agent-id-input" value="' + escapeHtml(agent.id) + '"' + (agent.builtIn ? ' readonly' : '') + ' placeholder="cursor-like-id"></div>' +
        '<div class="form-group"><label class="form-label" for="agent-name-input">' + t('proj.agentName') + '</label><input class="input" id="agent-name-input" value="' + escapeHtml(agent.name) + '"></div>' +
        '<div class="form-group"><label class="form-label" for="agent-user-path-input">' + t('proj.userPath') + '</label><input class="input mono" id="agent-user-path-input" value="' + escapeHtml(agent.userPath) + '"></div>' +
        '<div class="form-group"><label class="form-label" for="agent-project-path-input">' + t('proj.projectPath') + '</label><input class="input mono" id="agent-project-path-input" value="' + escapeHtml(agent.projectPath) + '"><p class="form-hint">' + t('proj.agentPathHint') + '</p></div>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>' + (agent.id && !agent.builtIn ? '<button class="btn btn-danger" type="button" id="btn-remove-agent">' + t('lib.remove') + '</button>' : '') + '<button class="btn btn-primary" type="button" id="btn-save-agent">' + t('proj.save') + '</button>';
    showModal(agent.id ? t('proj.editAgent') : t('proj.addAgent'), content, actions);
    document.getElementById('btn-save-agent').addEventListener('click', function () { saveAgent(agent); });
    var removeButton = document.getElementById('btn-remove-agent');
    if (removeButton) removeButton.addEventListener('click', function () { removeAgent(agent); });
}

async function saveAgent(previous) {
    var id = document.getElementById('agent-id-input').value.trim().toLowerCase();
    var name = document.getElementById('agent-name-input').value.trim();
    var userPath = document.getElementById('agent-user-path-input').value.trim();
    var projectPath = document.getElementById('agent-project-path-input').value.trim();
    if (!id || !name || !userPath || !projectPath) { showToast(t('proj.agentRequired'), 'error'); return; }
    var button = document.getElementById('btn-save-agent');
    button.disabled = true;
    try {
        await api.post('/api/agents', { id: id, name: name, userPath: userPath, projectPath: projectPath, enabled: previous ? previous.enabled : true });
        closeModal();
        showToast(t('proj.agentSaved'));
        await renderProjects();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

function removeAgent(agent) {
    showModal(t('lib.remove'), '<p class="confirm-copy">' + t('proj.confirmRemoveAgent') + '</p>', '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-danger" type="button" id="btn-confirm-remove-agent">' + t('lib.remove') + '</button>');
    document.getElementById('btn-confirm-remove-agent').addEventListener('click', async function () {
        this.disabled = true;
        try {
            await api.del('/api/agents/' + encodeURIComponent(agent.id));
            closeModal();
            showToast(t('proj.agentRemoved'));
            await renderProjects();
        } catch (err) { this.disabled = false; showToast(err.message, 'error'); }
    });
}

async function chooseProjectDirectory() {
    var button = document.getElementById('btn-choose-project-path');
    button.disabled = true;
    try {
        var result = await api.post('/api/dialogs/project-directory', {});
        document.getElementById('project-path-input').value = result.path;
    } catch (err) {
        showToast(err.message, 'error');
    } finally {
        button.disabled = false;
    }
}

async function addProject() {
    var path = document.getElementById('project-path-input').value.trim();
    if (!path) { showToast(t('proj.pathRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-project');
    button.disabled = true;
    try {
        var project = await api.post('/api/projects', { path: path });
        closeModal();
        projectState.selectedID = project.id;
        projectState.agentFilter = 'all';
        showToast(t('proj.registered'));
        await renderProjects();
    } catch (err) { button.disabled = false; showToast(err.message, 'error'); }
}

function selectedProjectAgents() {
    return Array.prototype.map.call(document.querySelectorAll('input[name="project-agent"]:checked'), function (input) { return input.value; });
}

async function deployProjectSkill() {
    var skill = document.getElementById('project-skill-select').value;
    if (!skill) { showToast(t('proj.chooseSkill'), 'error'); return; }
    var agents = selectedProjectAgents();
    if (!agents.length) { showToast(t('proj.chooseAgent'), 'error'); return; }
    var activeMode = (document.querySelector('[data-project-mode].active') || {}).dataset.projectMode || 'link';
    var button = document.getElementById('btn-confirm-project-deploy');
    button.disabled = true;
    try {
        await api.post('/api/projects/' + encodeURIComponent(projectState.selectedID) + '/' + activeMode, { skill: skill, agents: agents });
        closeModal();
        showToast(t('proj.deployed'));
        await renderProjects();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function unlinkProjectSkill(skill, force) {
    try {
        await api.post('/api/projects/' + encodeURIComponent(projectState.selectedID) + '/unlink', { skill: skill, force: Boolean(force) });
        showToast(t('proj.unlinked'));
        await renderProjects();
        return true;
    } catch (err) {
        if (!force && err.message.indexOf('use --force') >= 0) {
            showForceUnlinkModal(skill);
            return false;
        }
        showToast(err.message, 'error');
        return false;
    }
}

function showForceUnlinkModal(skill) {
    var content = '<p class="confirm-copy">' + t('proj.forceUnlinkDesc').replace('{0}', escapeHtml(skill)) + '</p>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button>' +
        '<button class="btn btn-danger" type="button" id="btn-confirm-force-unlink">' + t('proj.forceUnlink') + '</button>';
    showModal(t('proj.forceUnlinkTitle'), content, actions);
    document.getElementById('btn-confirm-force-unlink').addEventListener('click', async function () {
        this.disabled = true;
        var removed = await unlinkProjectSkill(skill, true);
        if (removed) {
            closeModal();
        } else {
            this.disabled = false;
        }
    });
}

function unregisterProject() {
    showModal(t('proj.unregister'), '<p class="confirm-copy">' + t('proj.confirmUnregister') + '</p>', '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-danger" type="button" id="btn-confirm-unregister">' + t('proj.unregister') + '</button>');
    document.getElementById('btn-confirm-unregister').addEventListener('click', async function () {
        this.disabled = true;
        try {
            await api.del('/api/projects/' + encodeURIComponent(projectState.selectedID));
            closeModal();
            projectState.selectedID = '';
            projectState.detail = null;
            showToast(t('proj.unregistered'));
            await renderProjects();
        } catch (err) { this.disabled = false; showToast(err.message, 'error'); }
    });
}

window.renderProjects = renderProjects;

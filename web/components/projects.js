/* global api, showToast, showModal, closeModal, escapeHtml, statusBadgeClass, isCurrentPage, t */

var projectState = { projects: [], skills: [], selectedID: '', detail: null };

async function renderProjects() {
    var container = document.getElementById('main-content');
    try {
        var results = await Promise.all([api.get('/api/projects'), api.get('/api/skills')]);
        if (!isCurrentPage('projects')) return;
        projectState.projects = results[0] || [];
        projectState.skills = results[1] || [];
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
    var html = '<div class="page animate-in"><div class="page-header"><div><h1 class="page-title">' + t('proj.title') +
        '</h1><p class="page-subtitle">' + t('proj.list') + '</p></div><div class="header-actions"><button class="btn btn-primary" type="button" id="btn-add-project">+ ' +
        t('proj.add') + '</button></div></div>';
    if (!projectState.projects.length) {
        html += '<div class="empty-state"><div class="empty-state-mark">P</div><div class="empty-state-title">' + t('proj.empty') +
            '</div><div class="empty-state-desc">' + t('proj.emptyDesc') + '</div></div></div>';
        container.innerHTML = html;
        document.getElementById('btn-add-project').addEventListener('click', showAddProjectModal);
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
    var detail = projectState.detail;
    if (!detail) return;
    var deployButton = document.getElementById('btn-project-deploy');
    if (deployButton) deployButton.addEventListener('click', deployProjectSkill);
    var copyButton = document.getElementById('btn-project-copy');
    if (copyButton) copyButton.addEventListener('click', function () { deployProjectSkill('copy'); });
    container.querySelectorAll('[data-project-mode]').forEach(function (button) {
        button.addEventListener('click', function () {
            container.querySelectorAll('[data-project-mode]').forEach(function (item) { item.classList.toggle('active', item === button); });
            document.getElementById('btn-project-deploy').classList.toggle('is-hidden', button.dataset.projectMode !== 'link');
            document.getElementById('btn-project-copy').classList.toggle('is-hidden', button.dataset.projectMode !== 'copy');
        });
    });
    document.getElementById('btn-project-refresh').addEventListener('click', function () { loadProjectDetail(projectState.selectedID); });
    document.getElementById('btn-project-unregister').addEventListener('click', unregisterProject);
    container.querySelectorAll('[data-unlink-skill]').forEach(function (button) {
        button.addEventListener('click', function () { unlinkProjectSkill(button.dataset.unlinkSkill); });
    });
}

function projectListItem(project) {
    var selected = project.id === projectState.selectedID;
    var status = project.exists ? t('proj.ready') : t('proj.missing');
    return '<button class="project-list-item' + (selected ? ' active' : '') + '" type="button" data-project-id="' + escapeHtml(project.id) + '">' +
        '<span class="project-list-name">' + escapeHtml(project.id) + '</span><span class="project-list-path mono">' + escapeHtml(project.path) +
        '</span><span class="project-list-meta"><span class="badge ' + (project.exists ? 'badge-ok' : 'badge-error') + '">' + status +
        '</span><span class="muted">' + project.activationCount + ' ' + t('proj.activations') + '</span></span></button>';
}

function projectDetailMarkup() {
    var detail = projectState.detail;
    if (!detail) return '<div class="inline-empty">' + t('loading') + '</div>';
    var project = detail.project;
    var html = '<div class="project-detail-header"><div><h2 class="section-title project-detail-name">' + escapeHtml(project.id) +
        '</h2><div class="mono project-path">' + escapeHtml(project.path) + '</div></div><div class="header-actions"><button class="btn btn-secondary btn-sm" type="button" id="btn-project-refresh">' +
        t('proj.status') + '</button><button class="btn btn-danger btn-sm" type="button" id="btn-project-unregister">' + t('proj.unregister') + '</button></div></div>';
    html += '<section class="project-section"><div class="section-heading"><h3 class="section-title">' + t('proj.deploy') + '</h3></div>';
    html += '<div class="project-deploy-form"><label class="form-group"><span class="form-label">' + t('proj.skill') + '</span><select class="select" id="project-skill-select">' +
        '<option value="">' + t('proj.selectSkill') + '</option>';
    projectState.skills.forEach(function (skill) {
        html += '<option value="' + escapeHtml(skill.id) + '">' + escapeHtml(skill.id) + '</option>';
    });
    html += '</select></label><div class="form-group"><span class="form-label">' + t('proj.agents') + '</span><div class="choice-grid">' +
        '<label class="check-option"><input type="checkbox" name="project-agent" value="claude" checked><span>Claude</span></label>' +
        '<label class="check-option"><input type="checkbox" name="project-agent" value="codex" checked><span>Codex</span></label></div></div>' +
        '<div class="form-group"><span class="form-label">' + t('proj.mode') + '</span><div class="import-mode project-mode"><button class="import-mode-option active" type="button" data-project-mode="link">' +
        t('proj.link') + '</button><button class="import-mode-option" type="button" data-project-mode="copy">' + t('proj.copy') + '</button></div></div>' +
        '<div class="project-deploy-actions"><button class="btn btn-primary" type="button" id="btn-project-deploy">' + t('proj.link') +
        '</button><button class="btn btn-secondary is-hidden" type="button" id="btn-project-copy">' + t('proj.copy') + '</button></div></div></section>';
    html += '<section class="project-section"><div class="section-heading"><h3 class="section-title">' + t('proj.activations') + '</h3></div>';
    if (!detail.activations || !detail.activations.length) {
        html += '<div class="inline-empty">' + t('proj.noSkills') + '<br><span class="muted">' + t('proj.noSkillsDesc') + '</span></div>';
    } else {
        html += '<div class="table-wrap"><table><thead><tr><th>' + t('proj.skill') + '</th><th>' + t('proj.agents') + '</th><th>' + t('proj.mode') + '</th><th>' + t('proj.actions') + '</th></tr></thead><tbody>';
        detail.activations.forEach(function (activation) {
            html += '<tr><td><strong>' + escapeHtml(activation.skillId) + '</strong></td><td>' + escapeHtml((activation.agents || []).join(', ')) +
                '</td><td><span class="badge badge-source">' + escapeHtml((activation.mode || 'symlink')) + '</span></td><td><button class="btn btn-danger btn-sm" type="button" data-unlink-skill="' +
                escapeHtml(activation.skillId) + '">' + t('proj.unlink') + '</button></td></tr>';
        });
        html += '</tbody></table></div>';
    }
    html += '</section>' + projectPlanMarkup(detail.plan) + '<div class="project-meta"><span class="badge ' + (project.exists ? 'badge-ok' : 'badge-error') + '">' +
        (project.exists ? t('proj.ready') : t('proj.missing')) + '</span></div>';
    return html;
}

function projectPlanMarkup(plan) {
    var operations = (plan && plan.operations) || [];
    var html = '<section class="project-section"><div class="section-heading"><h3 class="section-title">' + t('proj.statusTitle') + '</h3></div>';
    if (!operations.length) return html + '<div class="inline-empty">' + t('proj.statusEmpty') + '</div></section>';
    html += '<div class="table-wrap"><table><thead><tr><th>' + t('proj.skill') + '</th><th>' + t('proj.target') + '</th><th>' + t('proj.mode') + '</th><th>' + t('proj.statusTitle') + '</th></tr></thead><tbody>';
    operations.forEach(function (operation) {
        html += '<tr><td>' + escapeHtml(operation.skillId) + '</td><td class="mono cell-path" title="' + escapeHtml(operation.target) + '">' + escapeHtml(operation.target) +
            '</td><td>' + escapeHtml(operation.mode) + '</td><td><span class="badge ' + statusBadgeClass(operation.status) + '">' + escapeHtml(operation.status) + '</span></td></tr>';
    });
    return html + '</tbody></table></div></section>';
}

function showAddProjectModal() {
    var content = '<div class="form-group"><label class="form-label" for="project-path-input">' + t('proj.path') + '</label><input class="input" id="project-path-input" placeholder="/path/to/project"></div>' +
        '<div class="form-group"><label class="form-label" for="project-name-input">' + t('proj.name') + '</label><input class="input" id="project-name-input"><div class="form-hint">' + t('proj.nameHint') + '</div></div>';
    showModal(t('proj.register'), content, '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-primary" type="button" id="btn-confirm-project">' + t('proj.register') + '</button>');
    document.getElementById('btn-confirm-project').addEventListener('click', addProject);
}

async function addProject() {
    var path = document.getElementById('project-path-input').value.trim();
    var name = document.getElementById('project-name-input').value.trim();
    if (!path) { showToast(t('proj.pathRequired'), 'error'); return; }
    var button = document.getElementById('btn-confirm-project');
    button.disabled = true;
    try {
        var project = await api.post('/api/projects', { path: path, name: name });
        closeModal();
        projectState.selectedID = project.id;
        showToast(t('proj.registered'));
        renderProjects();
    } catch (err) { button.disabled = false; showToast(err.message, 'error'); }
}

function selectedProjectAgents() {
    return Array.prototype.map.call(document.querySelectorAll('input[name="project-agent"]:checked'), function (input) { return input.value; });
}

async function deployProjectSkill(mode) {
    var skill = document.getElementById('project-skill-select').value;
    if (!skill) { showToast(t('proj.chooseSkill'), 'error'); return; }
    var agents = selectedProjectAgents();
    if (!agents.length) { showToast(t('proj.chooseAgent'), 'error'); return; }
    var activeMode = mode || (document.querySelector('[data-project-mode].active') || {}).dataset.projectMode || 'link';
    try {
        await api.post('/api/projects/' + encodeURIComponent(projectState.selectedID) + '/' + activeMode, { skill: skill, agents: agents });
        showToast(t('proj.deployed'));
        await renderProjects();
    } catch (err) { showToast(err.message, 'error'); }
}

async function unlinkProjectSkill(skill) {
    try {
        await api.post('/api/projects/' + encodeURIComponent(projectState.selectedID) + '/unlink', { skill: skill });
        showToast(t('proj.unlinked'));
        await renderProjects();
    } catch (err) { showToast(err.message, 'error'); }
}

function unregisterProject() {
    showModal(t('proj.unregister'), '<p class="confirm-copy">' + t('proj.confirmUnregister') + '</p>', '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-danger" type="button" id="btn-confirm-unregister">' + t('proj.unregister') + '</button>');
    document.getElementById('btn-confirm-unregister').addEventListener('click', async function () {
        this.disabled = true;
        try {
            await api.del('/api/projects/' + encodeURIComponent(projectState.selectedID));
            closeModal();
            projectState.selectedID = '';
            showToast(t('proj.unregistered'));
            renderProjects();
        } catch (err) { this.disabled = false; showToast(err.message, 'error'); }
    });
}

window.renderProjects = renderProjects;

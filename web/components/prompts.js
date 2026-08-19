/* global api, showToast, showModal, closeModal, displayTag, displaySource, formatDate, shortHash, escapeHtml, isCurrentPage, t, uiIcon, confirmationMarkup, tagPickerMarkup, selectedTagValues, showManageTagsModal */

var promptState = { prompts: [], tags: [], query: '', activeTag: '' };
var promptEditorVariables = [];

async function renderPrompts() {
    var container = document.getElementById('main-content');
    try {
        var results = await Promise.all([api.get('/api/prompts'), api.get('/api/tags')]);
        promptState.prompts = results[0] || [];
        promptState.tags = results[1] || [];
        if (!isCurrentPage('prompts')) return;
        paintPrompts();
    } catch (err) {
        if (!isCurrentPage('prompts')) return;
        container.innerHTML = '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('alert') + '</div><div class="empty-state-title">' +
            escapeHtml(t('prompt.loadFailed')) + '</div><div class="empty-state-desc">' + escapeHtml(err.message) + '</div></div>';
        showToast(err.message, 'error');
    }
}

function paintPrompts() {
    if (!isCurrentPage('prompts')) return;
    var container = document.getElementById('main-content');
    var query = promptState.query.toLowerCase();
    var visible = promptState.prompts.filter(function (prompt) {
        var matchesTag = !promptState.activeTag || (prompt.tags || []).includes(promptState.activeTag);
        var matchesQuery = !query || [prompt.name, prompt.id, prompt.description, prompt.source].join(' ').toLowerCase().includes(query);
        return matchesTag && matchesQuery;
    });
    var tags = promptTagCounts();
    var html = '<div class="page prompt-page"><div class="page-header"><div><h1 class="page-title">' + t('prompt.title') + '</h1><p class="page-subtitle">' +
        t('prompt.count').replace('{0}', visible.length) + '</p></div><div class="header-actions"><input class="sr-only" type="file" id="prompt-file-input" accept=".md,text/markdown,text/plain">' +
        '<button class="btn btn-secondary" type="button" id="btn-manage-prompt-tags">' + uiIcon('tags') + t('lib.manageTags') + '</button>' +
        '<button class="btn btn-secondary" type="button" id="btn-import-prompt">' + uiIcon('folder') + t('prompt.import') + '</button>' +
        '<button class="btn btn-primary" type="button" id="btn-new-prompt">' + uiIcon('plus') + t('prompt.new') + '</button></div></div>';
    html += '<div class="library-tools prompt-tools"><label class="search-box"><span class="sr-only">' + t('prompt.search') + '</span><span class="search-mark" aria-hidden="true">' +
        uiIcon('search') + '</span><input class="input" id="prompt-search" value="' + escapeHtml(promptState.query) + '" placeholder="' + t('prompt.searchPlaceholder') + '"></label>' +
        '<div class="filter-bar"><span class="filter-label">' + t('lib.tags') + '</span><div class="tag-filter-list"><button class="tag clickable' +
        (!promptState.activeTag ? ' active' : '') + '" type="button" data-prompt-tag="">' + t('prompt.all') + '</button>';
    tags.forEach(function (tag) {
        html += '<button class="tag clickable' + (promptState.activeTag === tag.name ? ' active' : '') + '" type="button" data-prompt-tag="' +
            escapeHtml(tag.name) + '">' + escapeHtml(displayTag(tag.name)) + ' <small>' + tag.count + '</small></button>';
    });
    html += '</div></div></div>';
    if (!visible.length) {
        html += '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('sparkles') + '</div><div class="empty-state-title">' + t('prompt.noPrompts') +
            '</div><div class="empty-state-desc">' + t('prompt.noPromptsDesc') + '</div></div>';
    } else {
        html += '<div class="card-grid prompt-grid">' + visible.map(promptCard).join('') + '</div>';
    }
    html += '</div>';
    container.innerHTML = html;

    document.getElementById('btn-new-prompt').addEventListener('click', function () { showPromptEditor(null, defaultPromptDocument()); });
    document.getElementById('btn-manage-prompt-tags').addEventListener('click', showManageTagsModal);
    var fileInput = document.getElementById('prompt-file-input');
    document.getElementById('btn-import-prompt').addEventListener('click', function () { fileInput.click(); });
    fileInput.addEventListener('change', importPromptFile);
    document.getElementById('prompt-search').addEventListener('input', function (event) {
        promptState.query = event.target.value;
        paintPrompts();
        var next = document.getElementById('prompt-search');
        next.focus();
        next.setSelectionRange(promptState.query.length, promptState.query.length);
    });
    container.querySelectorAll('[data-prompt-tag]').forEach(function (button) {
        button.addEventListener('click', function () { promptState.activeTag = button.dataset.promptTag; paintPrompts(); });
    });
    container.querySelectorAll('[data-copy-prompt]').forEach(function (button) {
        button.addEventListener('click', function () { copySavedPrompt(button.dataset.copyPrompt, button); });
    });
    container.querySelectorAll('[data-edit-prompt]').forEach(function (button) {
        button.addEventListener('click', function () { openPromptEditor(button.dataset.editPrompt); });
    });
    container.querySelectorAll('[data-export-prompt]').forEach(function (button) {
        button.addEventListener('click', function () { exportPrompt(button.dataset.exportPrompt); });
    });
    container.querySelectorAll('[data-remove-prompt]').forEach(function (button) {
        button.addEventListener('click', function () { confirmRemovePrompt(button.dataset.removePrompt); });
    });
}

function promptTagCounts() {
    var counts = {};
    promptState.prompts.forEach(function (prompt) {
        (prompt.tags || []).forEach(function (tag) { counts[tag] = (counts[tag] || 0) + 1; });
    });
    return Object.keys(counts).sort().map(function (name) { return { name: name, count: counts[name] }; });
}

function promptCard(prompt) {
    var tags = (prompt.tags || []).map(function (tag) { return '<span class="tag">' + escapeHtml(displayTag(tag)) + '</span>'; }).join('');
    return '<article class="card prompt-card"><div class="skill-header"><div><div class="skill-name">' + escapeHtml(prompt.name) + '</div><div class="skill-id mono">' +
        escapeHtml(prompt.id) + '</div></div><span class="badge badge-source">' + escapeHtml(displaySource(prompt.source || 'local')) + '</span></div><p class="skill-desc">' +
        escapeHtml(prompt.description) + '</p><div class="tag-list">' + tags + '</div><div class="prompt-card-stats"><span class="mono">' + shortHash(prompt.hash) + '</span><span>' + formatDate(prompt.updatedAt) +
        '</span></div><div class="prompt-primary-action"><button class="btn btn-success" type="button" data-copy-prompt="' + escapeHtml(prompt.id) + '">' + uiIcon('copy') +
        t('prompt.copy') + '</button></div><div class="prompt-card-actions"><button class="btn btn-ghost btn-sm" type="button" data-edit-prompt="' + escapeHtml(prompt.id) + '">' +
        uiIcon('settings') + t('prompt.edit') + '</button><button class="btn btn-ghost btn-sm" type="button" data-export-prompt="' + escapeHtml(prompt.id) + '">' +
        uiIcon('folder') + t('prompt.export') + '</button><button class="btn btn-danger btn-sm" type="button" data-remove-prompt="' + escapeHtml(prompt.id) + '">' +
        uiIcon('trash') + t('prompt.remove') + '</button></div></article>';
}

function defaultPromptDocument() {
    return { name: '', description: '', tags: [], body: '', variables: [] };
}

function importPromptFile(event) {
    var file = event.target.files[0];
    if (!file) return;
    if (!/\.md$/i.test(file.name)) {
        showToast(t('prompt.invalidFile'), 'error');
        return;
    }
    var reader = new FileReader();
    reader.addEventListener('load', async function () {
        try {
            var documentData = await api.post('/api/prompts/validate', { content: String(reader.result || '') });
            showPromptEditor(null, documentData);
            showToast(t('prompt.imported'), 'info');
        } catch (err) {
            showToast(err.message, 'error');
        } finally {
            event.target.value = '';
        }
    });
    reader.readAsText(file);
}

async function openPromptEditor(id) {
    try {
        var details = await api.get('/api/prompts/' + encodePromptID(id));
        showPromptEditor(details, details);
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function showPromptEditor(existing, documentData) {
    documentData = documentData || defaultPromptDocument();
    promptEditorVariables = documentData.variables || [];
    var nameHint = existing ? t('prompt.nameLocked') : t('prompt.nameHint');
    var editor = '<div class="prompt-editor-shell"><div class="prompt-editor-main"><div class="prompt-editor-toolbar"><label class="form-label" for="prompt-content">' + t('prompt.contentLabel') +
        '</label><div class="prompt-editor-meta"><span id="prompt-editor-stats"></span><span class="prompt-editor-dirty" id="prompt-editor-dirty" aria-live="polite">' + escapeHtml(t('prompt.notValidated')) +
        '</span></div></div><textarea class="input prompt-content-editor" id="prompt-content" required spellcheck="true" placeholder="' + escapeHtml(t('prompt.contentPlaceholder')) +
        '" aria-describedby="prompt-editor-shortcuts" data-prompt-editor-field>' + escapeHtml(documentData.body || '') +
        '</textarea><div class="prompt-editor-shortcuts" id="prompt-editor-shortcuts">' + escapeHtml(t('prompt.shortcuts')) + '</div></div><aside class="prompt-editor-aside"><section class="prompt-editor-fields">' +
        '<div class="prompt-editor-field"><label class="form-label" for="prompt-name">' + t('prompt.nameLabel') + '</label><input class="input mono" id="prompt-name" required autocomplete="off" value="' +
        escapeHtml(documentData.name || '') + '" placeholder="' + escapeHtml(t('prompt.namePlaceholder')) + '" aria-describedby="prompt-name-hint" data-prompt-editor-field' + (existing ? ' readonly' : '') +
        '><small class="form-hint" id="prompt-name-hint">' + escapeHtml(nameHint) + '</small></div><div class="prompt-editor-field"><label class="form-label" for="prompt-description">' +
        t('prompt.descriptionLabel') + '</label><textarea class="input prompt-description-input" id="prompt-description" required placeholder="' + escapeHtml(t('prompt.descriptionPlaceholder')) +
        '" data-prompt-editor-field>' + escapeHtml(documentData.description || '') + '</textarea></div><div class="prompt-editor-field"><label class="form-label" for="prompt-tags">' +
        t('prompt.tagsLabel') + '</label>' + tagPickerMarkup('prompt-tags', documentData.tags || [], promptState.tags, true) + '</div></section><div class="prompt-editor-help">' + uiIcon('sparkles') + '<p>' +
        escapeHtml(t('prompt.editorHint')) + '</p></div><div class="prompt-validation" id="prompt-validation" role="status" aria-live="polite"><span class="muted">' +
        escapeHtml(t('prompt.notValidated')) + '</span></div></aside></div>';
    var actions = '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-secondary" type="button" id="btn-validate-prompt">' +
        uiIcon('check') + t('prompt.validate') + '</button><button class="btn btn-primary" type="button" id="btn-save-prompt">' + uiIcon('sparkles') + t('prompt.save') + '</button>';
    showModal(existing ? t('prompt.editorTitle') : t('prompt.newTitle'), editor, actions);
    document.querySelector('.modal').classList.add('prompt-editor-modal');
    var textarea = document.getElementById('prompt-content');
    document.querySelectorAll('#prompt-tags input').forEach(function (input) { input.setAttribute('data-prompt-editor-field', ''); });
    updatePromptEditorStats(textarea.value, false);
    document.querySelectorAll('[data-prompt-editor-field]').forEach(function (field) {
        field.addEventListener('input', function () {
            field.removeAttribute('aria-invalid');
            document.getElementById('prompt-validation').innerHTML = '<span class="muted">' + escapeHtml(t('prompt.notValidated')) + '</span>';
            updatePromptEditorStats(textarea.value, true);
        });
    });
    textarea.addEventListener('keydown', function (event) {
        var commandKey = event.metaKey || event.ctrlKey;
        if (commandKey && event.key.toLowerCase() === 's') {
            event.preventDefault();
            if (!document.getElementById('btn-save-prompt').disabled) savePromptEditor(existing);
            return;
        }
        if (commandKey && event.key === 'Enter') {
            event.preventDefault();
            validatePromptEditor();
            return;
        }
        if (event.key === 'Tab' && !event.metaKey && !event.ctrlKey && !event.altKey) {
            event.preventDefault();
            textarea.setRangeText('  ', textarea.selectionStart, textarea.selectionEnd, 'end');
            textarea.dispatchEvent(new Event('input', { bubbles: true }));
        }
    });
    document.getElementById('btn-validate-prompt').addEventListener('click', validatePromptEditor);
    document.getElementById('btn-save-prompt').addEventListener('click', function () { savePromptEditor(existing); });
    document.getElementById(existing ? 'prompt-content' : 'prompt-name').focus();
}

function promptEditorPayload() {
    return {
        name: document.getElementById('prompt-name').value,
        description: document.getElementById('prompt-description').value,
        tags: selectedTagValues('prompt-tags'),
        body: document.getElementById('prompt-content').value,
        variables: promptEditorVariables,
    };
}

function promptEditorFieldsReady() {
    var fields = [
        { id: 'prompt-name', label: t('prompt.nameLabel') },
        { id: 'prompt-description', label: t('prompt.descriptionLabel') },
        { id: 'prompt-content', label: t('prompt.contentLabel') },
    ];
    for (var index = 0; index < fields.length; index += 1) {
        var input = document.getElementById(fields[index].id);
        if (input.value.trim()) continue;
        input.setAttribute('aria-invalid', 'true');
        input.focus();
        showToast(t('prompt.required').replace('{0}', fields[index].label), 'error');
        return false;
    }
    return true;
}

function updatePromptEditorStats(content, dirty) {
    var stats = document.getElementById('prompt-editor-stats');
    var indicator = document.getElementById('prompt-editor-dirty');
    var lineCount = content ? content.split('\n').length : 1;
    stats.textContent = t('prompt.lines').replace('{0}', lineCount) + ' · ' + t('prompt.characters').replace('{0}', content.length);
    indicator.textContent = t(dirty ? 'prompt.unsaved' : 'prompt.notValidated');
    indicator.classList.toggle('is-dirty', dirty);
    indicator.classList.remove('is-valid');
}

async function validatePromptEditor() {
    var button = document.getElementById('btn-validate-prompt');
    if (!button || button.disabled) return;
    if (!promptEditorFieldsReady()) return;
    button.disabled = true;
    try {
        var documentData = await api.post('/api/prompts/validate', promptEditorPayload());
        var variables = (documentData.variables || []).map(function (variable) { return '<span class="tag">{{' + escapeHtml(variable.name) + '}}</span>'; }).join('');
        document.getElementById('prompt-validation').innerHTML = '<div class="prompt-validation-ok">' + uiIcon('check') + '<strong>' + escapeHtml(t('prompt.valid')) +
            '</strong></div><div class="prompt-validation-name">' + escapeHtml(documentData.name) + '</div><p>' + escapeHtml(documentData.description) + '</p>' +
            (variables ? '<div class="tag-list">' + variables + '</div>' : '<span class="muted">' + escapeHtml(t('prompt.noVariables')) + '</span>');
        var indicator = document.getElementById('prompt-editor-dirty');
        indicator.textContent = t('prompt.valid');
        indicator.classList.remove('is-dirty');
        indicator.classList.add('is-valid');
    } catch (err) {
        document.getElementById('prompt-validation').innerHTML = '<div class="prompt-validation-error">' + uiIcon('alert') + '<span>' + escapeHtml(err.message) + '</span></div>';
        document.getElementById('prompt-editor-dirty').classList.remove('is-valid');
        document.getElementById('prompt-editor-dirty').classList.add('is-dirty');
    } finally {
        button.disabled = false;
    }
}

async function savePromptEditor(existing) {
    var button = document.getElementById('btn-save-prompt');
    if (!promptEditorFieldsReady()) return;
    button.disabled = true;
    try {
        var data = promptEditorPayload();
        if (existing) {
            data.baseHash = existing.hash;
            await api.put('/api/prompts/' + encodePromptID(existing.id), data);
        } else {
            await api.post('/api/prompts', data);
        }
        closeModal();
        showToast(t(existing ? 'prompt.updated' : 'prompt.created'));
        await renderPrompts();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function exportPrompt(id) {
    try {
        var details = await api.get('/api/prompts/' + encodePromptID(id));
        var blob = new Blob([details.content], { type: 'text/markdown;charset=utf-8' });
        var url = URL.createObjectURL(blob);
        var anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = details.name + '.PROMPT.md';
        document.body.appendChild(anchor);
        anchor.click();
        anchor.remove();
        URL.revokeObjectURL(url);
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function confirmRemovePrompt(id) {
    var prompt = promptState.prompts.find(function (value) { return value.id === id; });
    var name = prompt ? prompt.name : id;
    showModal(t('prompt.confirmRemoveTitle'), confirmationMarkup(t('prompt.confirmRemove').replace('{0}', name), t('prompt.confirmRemoveNote'), 'danger'),
        '<button class="btn btn-ghost" type="button" data-close-modal>' + t('lib.cancel') + '</button><button class="btn btn-danger" type="button" id="btn-confirm-remove-prompt">' +
        t('prompt.remove') + '</button>');
    document.getElementById('btn-confirm-remove-prompt').addEventListener('click', function () { removePrompt(id); });
}

async function removePrompt(id) {
    var button = document.getElementById('btn-confirm-remove-prompt');
    button.disabled = true;
    try {
        await api.del('/api/prompts/' + encodePromptID(id));
        closeModal();
        showToast(t('prompt.removed'));
        await renderPrompts();
    } catch (err) {
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

async function copySavedPrompt(id, button) {
    var original = button.innerHTML;
    button.disabled = true;
    try {
        var detailsPromise = api.get('/api/prompts/' + encodePromptID(id));
        if (navigator.clipboard && navigator.clipboard.write && window.ClipboardItem) {
            await navigator.clipboard.write([new window.ClipboardItem({
                'text/plain': detailsPromise.then(function (details) {
                    return new Blob([details.body || ''], { type: 'text/plain' });
                }),
            })]);
        } else {
            var details = await detailsPromise;
            await copyPromptText(details.body || '');
        }
        button.innerHTML = uiIcon('check') + t('prompt.copiedShort');
        showToast(t('prompt.copied'));
        setTimeout(function () {
            if (!button.isConnected) return;
            button.innerHTML = original;
            button.disabled = false;
        }, 1200);
    } catch (err) {
        button.innerHTML = original;
        button.disabled = false;
        showToast(err.message, 'error');
    }
}

function copyPromptText(content) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
        return navigator.clipboard.writeText(content).catch(function () { return copyPromptTextFallback(content); });
    }
    return copyPromptTextFallback(content);
}

function copyPromptTextFallback(content) {
    var textarea = document.createElement('textarea');
    textarea.value = content;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    var copied = document.execCommand('copy');
    textarea.remove();
    return copied ? Promise.resolve() : Promise.reject(new Error('Clipboard is unavailable'));
}

function encodePromptID(id) {
    return String(id || '').split('/').map(encodeURIComponent).join('/');
}

window.renderPrompts = renderPrompts;

/* global api, showToast, formatDate, shortHash, escapeHtml, displayTag, isCurrentPage, t, uiIcon */

async function renderDashboard() {
    var container = document.getElementById('main-content');
    try {
        var data = await api.get('/api/dashboard');
        if (!isCurrentPage('dashboard')) return;

        var html = '<div class="page">';
        html += '<div class="page-header"><div><h1 class="page-title">' + t('dash.title') + '</h1></div></div>';
        html += '<div class="stat-grid">';
        html += statCard(data.skillCount, t('dash.totalSkills'), 'library', 'accent');
        html += statCard(data.activatedCount, t('dash.enabled'), 'check', 'info');
        html += statCard(data.sourceCount, t('dash.gitSources'), 'link', 'warning');
        html += '</div>';

        html += '<section class="section"><div class="section-heading"><h2 class="section-title">' + t('dash.recentlyAdded') + '</h2></div>';
        if (data.recentSkills && data.recentSkills.length) {
            html += '<div class="table-wrap"><table><thead><tr><th>' + t('dash.id') + '</th><th>' + t('dash.tags') +
                '</th><th>' + t('dash.added') + '</th><th>' + t('dash.hash') + '</th></tr></thead><tbody>';
            data.recentSkills.forEach(function (skill) {
                var tags = (skill.tags || []).map(function (tag) {
                    return '<span class="tag">' + escapeHtml(displayTag(tag)) + '</span>';
                }).join(' ');
                html += '<tr><td><strong>' + escapeHtml(skill.id) + '</strong></td><td><div class="tag-list">' + tags +
                    '</div></td><td>' + formatDate(skill.addedAt) + '</td><td><span class="mono">' + shortHash(skill.hash) + '</span></td></tr>';
            });
            html += '</tbody></table></div>';
        } else {
            html += '<div class="inline-empty">' + t('dash.noRecent') + '</div>';
        }
        html += '</section></div>';
        container.innerHTML = html;
    } catch (err) {
        if (!isCurrentPage('dashboard')) return;
        container.innerHTML = errorState(t('dash.loadFailed'), err.message);
        showToast(err.message, 'error');
    }
}

function statCard(value, label, icon, tone) {
    return '<div class="stat-card stat-card-' + tone + '"><div><div class="stat-value">' + Number(value || 0) + '</div><div class="stat-label">' +
        label + '</div></div><div class="stat-mark" aria-hidden="true">' + uiIcon(icon) + '</div></div>';
}

function errorState(title, message) {
    return '<div class="empty-state"><div class="empty-state-mark">' + uiIcon('sparkles') + '</div><div class="empty-state-title">' + title +
        '</div><div class="empty-state-desc">' + escapeHtml(message) + '</div></div>';
}

window.renderDashboard = renderDashboard;

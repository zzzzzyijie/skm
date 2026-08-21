(function () {
    var saved = localStorage.getItem('skm-theme');
    var theme = saved === 'light' ? 'light' : 'dark';
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    var savedLanguage = localStorage.getItem('skm-lang');
    var language = savedLanguage === 'zh' || savedLanguage === 'en'
        ? savedLanguage
        : ((navigator.language || 'en').toLowerCase().startsWith('zh') ? 'zh' : 'en');
    document.documentElement.lang = language;
    document.documentElement.dataset.language = language;
})();

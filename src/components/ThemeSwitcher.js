const THEME_KEY = "tc_theme";
const DEFAULT_THEME = "qingdai";

const THEMES = {
    qingdai: "青黛古卷",
    danzhu: "丹朱节庆"
};

function getTheme() {
    const saved = localStorage.getItem(THEME_KEY);
    return saved && THEMES[saved] ? saved : DEFAULT_THEME;
}

function setTheme(theme) {
    if (!THEMES[theme]) {
        return;
    }

    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem(THEME_KEY, theme);
}

export function initTheme() {
    setTheme(getTheme());
}

export function ThemeSwitcher() {
    const options = Object.entries(THEMES)
        .map(([value, label]) => `<option value="${value}">${label}</option>`)
        .join("");

    return `
    <label class="theme-switcher">
      <span>主题</span>
      <select class="select-theme" id="theme-select" aria-label="切换主题">
        ${options}
      </select>
    </label>
  `;
}

export function bindThemeSwitcher() {
    const select = document.querySelector("#theme-select");

    if (!select) {
        return;
    }

    select.value = getTheme();
    select.addEventListener("change", (event) => {
        setTheme(event.target.value);
    });
}

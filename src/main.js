import { initTheme } from "./components/ThemeSwitcher.js";
import "./components/TopBar.js";
import { homePageTag } from "./pages/home/index.js";
import { playPageTag } from "./pages/play/index.js";
import { advisorPageTag } from "./pages/advisor/index.js";

const pageRegistry = {
    home: homePageTag,
    play: playPageTag,
    advisor: advisorPageTag
};

function getCurrentPageKey() {
    const rawHash = window.location.hash.replace(/^#\/?/, "");
    return pageRegistry[rawHash] ? rawHash : "home";
}

function mountApp() {
    const app = document.querySelector("#app");

    if (!app) {
        return;
    }

    const pageKey = getCurrentPageKey();
    const pageTag = pageRegistry[pageKey];
    const topBar = document.createElement("tc-top-bar");
    const pageRoot = document.createElement("main");
    const pageElement = document.createElement(pageTag);

    topBar.setAttribute("current-page", pageKey);
    pageRoot.id = "page-root";
    pageRoot.append(pageElement);
    app.replaceChildren(topBar, pageRoot);
}

initTheme();

if (!window.location.hash) {
    window.location.hash = "#/home";
}

mountApp();
window.addEventListener("hashchange", mountApp);

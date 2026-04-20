import { initTheme } from "./components/ThemeSwitcher.js";
import { initHomePage, HomePage } from "./pages/home/home.js";

function mountApp() {
    const app = document.querySelector("#app");

    if (!app) {
        return;
    }

    app.innerHTML = HomePage();
    initHomePage();
}

initTheme();
mountApp();

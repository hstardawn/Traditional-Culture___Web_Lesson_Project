import { SectionTitle } from "../../../components/SectionTitle.js";
import { divinations } from "../constants/divinations.js";

function hashToIndex(text, size) {
    if (!size || size < 1) {
        return 0;
    }

    const normalized = String(text || "").trim();

    if (!normalized) {
        return 0;
    }

    let hash = 0;

    for (let i = 0; i < normalized.length; i += 1) {
        hash = (hash << 5) - hash + normalized.charCodeAt(i);
        hash |= 0;
    }

    return Math.abs(hash) % size;
}

function renderResult(result) {
    if (!result) {
        return "<p class=\"divination-tip\">输入一个问题，例如：我该先做哪件事？</p>";
    }

    return `
    <h3>${result.title}</h3>
    <p>${result.message}</p>
    <p class="divination-tip">提示：相同输入会得到相同签文。</p>
  `;
}

export function DivinationSection() {
    return `
    <section class="section" id="divination">
      ${SectionTitle("今日占卜")}
      <div class="divination-panel">
        <form class="divination-form" id="divination-form">
          <input class="divination-input" id="divination-input" name="question" maxlength="80" placeholder="请输入你的问题" required />
          <button class="divination-submit" type="submit">起一签</button>
        </form>
        <div class="divination-result" id="divination-result">
          ${renderResult(null)}
        </div>
      </div>
    </section>
  `;
}

export function bindDivination() {
    const form = document.querySelector("#divination-form");
    const input = document.querySelector("#divination-input");
    const resultEl = document.querySelector("#divination-result");

    if (!form || !input || !resultEl) {
        return;
    }

    form.addEventListener("submit", (event) => {
        event.preventDefault();
        const index = hashToIndex(input.value, divinations.length);
        const result = divinations[index];
        resultEl.innerHTML = renderResult(result);
    });
}

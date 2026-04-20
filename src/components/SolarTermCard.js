export function SolarTermCard(term) {
    return `
    <article class="term-card" tabindex="0" aria-label="${term.name}介绍">
      <h3>${term.name}</h3>
      <small>${term.season}季节气</small>
      <p>${term.intro}</p>
    </article>
  `;
}

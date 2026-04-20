import { SolarTermCard } from "../../../components/SolarTermCard.js";
import { SectionTitle } from "../../../components/SectionTitle.js";
import { solarTerms } from "../constants/solarTerms.js";

export function SolarTermsSection() {
    return `
    <section class="section" id="solar-terms">
      ${SectionTitle("二十四节气卡片")}
      <div class="card-grid">
        ${solarTerms.map((term) => SolarTermCard(term)).join("")}
      </div>
    </section>
  `;
}

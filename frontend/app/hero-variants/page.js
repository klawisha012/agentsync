import HeroArt, { VARIANT_IDS, VARIANT_TITLES } from "../hero-sticker";

export const metadata = { title: "Варианты героя" };

export default function HeroVariantsPage() {
  return (
    <main style={{ display: "grid", gap: "2rem", padding: "2rem" }}>
      <style>{".hv svg{width:100%;height:auto;display:block}"}</style>
      {VARIANT_IDS.map((id) => (
        <section key={id} className="hv">
          <h2>{VARIANT_TITLES[id]} (?hero={id})</h2>
          <div style={{ display: "flex", gap: "2rem", alignItems: "center" }}>
            <div style={{ width: "13rem" }}><HeroArt variant={id} slot="left" className="" /></div>
            <div style={{ width: "22rem" }}><HeroArt variant={id} slot="right" className="" /></div>
          </div>
        </section>
      ))}
    </main>
  );
}

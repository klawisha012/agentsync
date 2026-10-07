import marks from "./agent-logos.json";

export default function AgentLogo({ name, tone }) {
  const className = `agent-logo tone-${tone || "plain"}`;
  const slug = marks[String(name || "").trim().toLowerCase()];
  if (!slug) {
    return (
      <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
        <rect x="5" y="8" width="14" height="10" rx="2" />
        <path d="M12 8V4.5M9 13h.01M15 13h.01" />
      </svg>
    );
  }
  const url = `/agents/${slug}.svg`;
  return (
    <span
      className={`${className} has-mark`}
      aria-hidden="true"
      style={{ maskImage: `url(${url})`, WebkitMaskImage: `url(${url})` }}
    />
  );
}

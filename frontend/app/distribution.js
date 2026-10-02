"use client";

const palette = ["var(--grok)", "var(--agents)", "var(--claude)", "#e4e4e7", "var(--muted)"];

export function distribution(accounts) {
  const counts = new Map();
  let total = 0;
  for (const account of accounts || []) {
    for (const agent of account.agents || []) {
      if (agent.version == null) {
        continue;
      }
      total += 1;
      counts.set(agent.name, (counts.get(agent.name) || 0) + 1);
    }
  }
  const rows = [...counts.entries()].map(([name, count]) => ({
    name,
    count,
    pct: total > 0 ? Math.round((count / total) * 100) : 0,
  }));
  rows.sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, "ru", { sensitivity: "base" }));
  return {
    total,
    rows: rows.map((row, index) => ({ ...row, color: palette[index % palette.length] })),
  };
}

export function publicationWord(n) {
  const n10 = n % 10;
  const n100 = n % 100;
  if (n10 === 1 && n100 !== 11) {
    return "публикация";
  }
  if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) {
    return "публикации";
  }
  return "публикаций";
}

export function ShareRing({ shares, total, caption = "публикаций" }) {
  const radius = 38;
  const circ = 2 * Math.PI * radius;
  let offset = 0;
  return (
    <div className="ring-wrap">
      <svg className="ring" viewBox="0 0 100 100" aria-hidden="true">
        <circle cx="50" cy="50" r={radius} />
        {shares.map((row) => {
          const length = (row.pct / 100) * circ;
          const dash = `${length} ${circ - length}`;
          const node = (
            <circle
              key={row.name}
              cx="50"
              cy="50"
              r={radius}
              stroke={row.color}
              strokeDasharray={dash}
              strokeDashoffset={-offset}
            />
          );
          offset += length;
          return node;
        })}
      </svg>
      <div className="ring-label">
        <b>{total}</b>
        {caption ? <span>{caption}</span> : null}
      </div>
    </div>
  );
}

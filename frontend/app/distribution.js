"use client";

export const agentExamples = [
  { name: "Grok", color: "var(--grok)" },
  { name: "Agents", color: "var(--agents)" },
  { name: "Claude", color: "var(--claude)" },
];

export function distribution(accounts) {
  const rows = agentExamples.map((item) => ({ ...item, count: 0, pct: 0 }));
  let total = 0;
  for (const account of accounts) {
    for (const agent of account.agents || []) {
      if (agent.version == null) {
        continue;
      }
      total += 1;
      const row = rows.find((item) => item.name === agent.name);
      if (row) {
        row.count += 1;
      }
    }
  }
  if (total > 0) {
    for (const row of rows) {
      row.pct = Math.round((row.count / total) * 100);
    }
  }
  return { total, rows };
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

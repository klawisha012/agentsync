"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "../api";
import AgentLogo from "../agent-logo";
import { publicationWord } from "../distribution";

const PERIODS = [
  { value: "7", label: "За 7 дней" },
  { value: "30", label: "За 30 дней" },
  { value: "90", label: "За 3 месяца" },
  { value: "all", label: "За всё время" },
];

const palette = ["#34d399", "#f472b6", "#facc15", "#a3e635", "#60a5fa"];
const brandColors = { claude: "var(--claude)", grok: "var(--grok)", agents: "var(--agents)" };

export default function StatsPage() {
  const [report, setReport] = useState(null);
  const [failed, setFailed] = useState(false);
  const [period, setPeriod] = useState("all");

  useEffect(() => {
    let gone = false;
    setReport(null);
    setFailed(false);
    api(`/stats?period=${encodeURIComponent(period)}`).then((res) => {
      if (gone) {
        return;
      }
      if (!res.ok) {
        setFailed(true);
        setReport(null);
        return;
      }
      setReport(res.body);
    });
    return () => {
      gone = true;
    };
  }, [period]);

  const selected = PERIODS.find((item) => item.value === period) || PERIODS[PERIODS.length - 1];
  const shares = (report?.shares || []).map((row, index) => ({
    ...row,
    color: brandColors[String(row.name).toLowerCase()] || palette[index % palette.length],
  }));
  const ready = report !== null;
  const sections = [
    {
      title: "Просмотры",
      icon: "eye",
      tiles: [
        { label: "всего просмотров", value: ready ? formatCount(report.viewsTotal) : "—", note: "по всем аккаунтам" },
        { label: "средние просмотры", value: ready ? formatCount(report.viewsAverage) : "—", note: "на аккаунт" },
        {
          label: "лидер по просмотрам",
          value: ready && report.viewsLeader ? report.viewsLeader.name : "—",
          note: ready && report.viewsLeader ? `${formatCount(report.viewsLeader.views)} просмотров` : "нет данных",
        },
      ],
    },
    {
      title: "Лайки",
      icon: "heart",
      tiles: [
        { label: "всего лайков", value: ready ? formatCount(report.likesTotal) : "—", note: "по всем аккаунтам" },
        { label: "средние лайки", value: ready ? formatCount(report.likesAverage) : "—", note: "на аккаунт" },
        { label: "конверсия в лайк", value: ready ? `${Number(report.likeRate).toFixed(1)}%` : "—", note: "лайки / просмотры" },
      ],
    },
    {
      title: "Публикации",
      icon: "files",
      tiles: [
        {
          label: "публикаций",
          value: ready ? String(report.publications) : "—",
          note: ready ? publicationWord(report.publications) : "",
        },
        {
          label: "популярный агент",
          value: ready && report.topAgent ? report.topAgent.name : "—",
          note: ready && report.topAgent ? `${report.topAgent.count} ${publicationWord(report.topAgent.count)}` : "нет данных",
        },
      ],
    },
    {
      title: "Аккаунты",
      icon: "users",
      catalog: true,
      tiles: [
        { label: "аккаунтов", value: ready ? String(report.accounts) : "—", note: "в каталоге" },
        { label: "проверено", value: ready ? String(report.verified) : "—", note: ready ? `из ${report.accounts} аккаунтов` : "" },
        { label: "без публикации", value: ready ? String(report.fresh) : "—", note: "ещё не публиковали" },
        { label: "топ недели", value: ready ? String(report.topWeek) : "—", note: "аккаунтов в топе" },
        { label: "последняя активность", value: ready ? activityLabel(report.lastActivity) : "—", note: "обновление в каталоге" },
      ],
    },
  ];

  return (
    <div className="stats-page">
      <div className="stats-head">
        <h1>Статистика</h1>
        <label className="stats-period">
          <span>Промежуток</span>
          <select value={period} onChange={(event) => setPeriod(event.target.value)} aria-label="Промежуток статистики">
            {PERIODS.map((item) => (
              <option key={item.value} value={item.value}>{item.label}</option>
            ))}
          </select>
        </label>
      </div>
      <div className="stats-layout">
        <div className="stats-sections">
          {sections.map((section) => (
            <section key={section.title} className="stats-section">
              <header className="stats-section-heading">
                <h2 className="stats-section-title"><Glyph name={section.icon} />{section.title}</h2>
                <span>{section.catalog ? "В каталоге" : selected.label}</span>
              </header>
              <div className="stats-grid">
                {section.tiles.map((tile, index) => (
                  <article key={tile.label} className={`stat-tile${index === 0 ? " stat-tile-primary" : ""}`}>
                    <h3>{tile.label}</h3>
                    <b className={`stat-value${String(tile.value).length > 10 ? " stat-value-text" : ""}`}>{tile.value}</b>
                    <p className="hint">{tile.note}</p>
                  </article>
                ))}
              </div>
            </section>
          ))}
        </div>
        <aside className="stats-aside">
          <section className="stats-leaderboard">
            <header className="stats-section-heading">
              <h2 className="stats-section-title"><Glyph name="trophy" />Лидеры</h2>
              <span>По просмотрам · {selected.label.toLowerCase()}</span>
            </header>
            <div className="stats-leaders">
              <div className="stats-leader-labels"><span>Аккаунт</span><span>Просмотры</span></div>
              {(report?.leaders || []).map((account, index) => (
                <Link href={`/${encodeURIComponent(account.name)}`} key={account.name} className="stats-leader-row">
                  <span className="stats-leader-avatar" aria-hidden="true">{account.name.slice(0, 1).toUpperCase()}</span>
                  <span className="stats-leader-name">
                    {account.name}
                    {account.verified ? <Glyph name="verified" size={13} label="Проверен" /> : null}
                  </span>
                  <b>{formatCount(account.views || 0)}</b>
                  <span className="stats-leader-rank">{index + 1}</span>
                </Link>
              ))}
              {report === null && !failed ? <p className="hint">Загрузка…</p> : null}
              {failed ? <p className="hint">Не удалось открыть статистику.</p> : null}
              {ready && !failed && (report.leaders || []).length === 0 ? <p className="hint">Нет аккаунтов.</p> : null}
            </div>
          </section>
          <section className="side-card stats-card" aria-busy={report === null}>
            <header className="stats-section-heading">
              <h2 className="stats-section-title">Распределение агентов</h2>
              <span>{ready ? `${report.sharesTotal} ${publicationWord(report.sharesTotal)}` : ""}</span>
            </header>
            {failed ? <p className="hint">Не удалось открыть сводку.</p> : null}
            {report === null && !failed ? <p className="hint">Открываем сводку…</p> : null}
            {ready && !failed && report.sharesTotal === 0 ? <p className="hint">Нет публикаций.</p> : null}
            {ready && !failed && report.sharesTotal > 0 ? (
              <table className="stats-dist">
                <thead>
                  <tr>
                    <th scope="col">Агент</th>
                    <th scope="col">Публикации</th>
                    <th scope="col">Доля</th>
                  </tr>
                </thead>
                <tbody>
                  {shares.map((row) => (
                    <tr key={row.name}>
                      <th scope="row">
                        <span className="stats-dist-name">
                          <AgentLogo name={row.name} tone={String(row.name).toLowerCase()} />
                          {row.name}
                        </span>
                      </th>
                      <td>{row.count}</td>
                      <td>
                        <span className="stats-dist-share">
                          <span className="stats-dist-bar" aria-hidden="true">
                            <i style={{ width: `${row.pct}%`, background: row.color }} />
                          </span>
                          <b style={{ color: row.color }}>{row.pct}%</b>
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <th scope="row">Всего</th>
                    <td>{report.sharesTotal}</td>
                    <td>100%</td>
                  </tr>
                </tfoot>
              </table>
            ) : null}
          </section>
        </aside>
      </div>
    </div>
  );
}

function formatCount(value) {
  return String(value || 0).replace(/\B(?=(\d{3})+(?!\d))/g, "\u202f");
}

function activityLabel(value) {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "—";
  }
  return parsed.toLocaleDateString("ru-RU", { day: "numeric", month: "short", year: "numeric" });
}

function Glyph({ name, size = 16, label }) {
  const common = {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.8,
    strokeLinecap: "round",
    strokeLinejoin: "round",
    "aria-hidden": label ? undefined : true,
    "aria-label": label,
    role: label ? "img" : undefined,
  };
  if (name === "eye") {
    return <svg {...common}><path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0" /><circle cx="12" cy="12" r="3" /></svg>;
  }
  if (name === "heart") {
    return <svg {...common}><path d="M2 9.5a5.5 5.5 0 0 1 9.591-3.676.56.56 0 0 0 .818 0A5.49 5.49 0 0 1 22 9.5c0 2.29-1.5 4-3 5.5l-5.492 5.313a2 2 0 0 1-3 .019L5 15c-1.5-1.5-3-3.2-3-5.5" /></svg>;
  }
  if (name === "files") {
    return <svg {...common}><path d="M15 2h-4a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V8" /><path d="M16.706 2.706A2.4 2.4 0 0 0 15 2v5a1 1 0 0 0 1 1h5a2.4 2.4 0 0 0-.706-1.706z" /><path d="M5 7a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h8a2 2 0 0 0 1.732-1" /></svg>;
  }
  if (name === "users") {
    return <svg {...common}><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><path d="M16 3.128a4 4 0 0 1 0 7.744" /><path d="M22 21v-2a4 4 0 0 0-3-3.87" /><circle cx="9" cy="7" r="4" /></svg>;
  }
  if (name === "trophy") {
    return (
      <svg {...common}>
        <path d="M10 14.66v1.626a2 2 0 0 1-.976 1.696A5 5 0 0 0 7 21.978" />
        <path d="M14 14.66v1.626a2 2 0 0 0 .976 1.696A5 5 0 0 1 17 21.978" />
        <path d="M18 9h1.5a1 1 0 0 0 0-5H18" />
        <path d="M4 22h16" />
        <path d="M6 9a6 6 0 0 0 12 0V3a1 1 0 0 0-1-1H7a1 1 0 0 0-1 1z" />
        <path d="M6 9H4.5a1 1 0 0 1 0-5H6" />
      </svg>
    );
  }
  return (
    <svg {...common}>
      <path d="M3.85 8.62a4 4 0 0 1 4.78-4.77 4 4 0 0 1 6.74 0 4 4 0 0 1 4.78 4.78 4 4 0 0 1 0 6.74 4 4 0 0 1-4.77 4.78 4 4 0 0 1-6.75 0 4 4 0 0 1-4.78-4.77 4 4 0 0 1 0-6.76Z" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  );
}

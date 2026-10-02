"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "../api";
import Avatar from "../avatar";

const sorts = [
  { id: "likes", label: "По лайкам" },
  { id: "views", label: "По просмотрам" },
  { id: "name", label: "По имени" },
  { id: "time", label: "По времени (0-публ. в\u00a0конце)" },
];

const examples = [
  { name: "Grok", color: "var(--grok)" },
  { name: "Agents", color: "var(--agents)" },
  { name: "Claude", color: "var(--claude)" },
];

const policy = [
  "Токены, ключи API и\u00a0секреты зачищаются локально до отправки в\u00a0сеть.",
  "Абсолютные локальные пути хоста исключаются из манифестов MCP.",
  "Применение среды валидируется контрольной суммой SHA-256.",
];

export default function AccountList() {
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState("likes");
  const [accounts, setAccounts] = useState(null);
  const [explanation, setExplanation] = useState("");

  useEffect(() => {
    let gone = false;
    const params = new URLSearchParams();
    if (query) {
      params.set("q", query);
    }
    params.set("sort", sort);
    api(`/accounts?${params.toString()}`).then((res) => {
      if (gone) {
        return;
      }
      if (!res.ok) {
        setExplanation(res.body?.explanation || "Не удалось открыть список.");
        setAccounts([]);
        return;
      }
      setExplanation("");
      setAccounts(res.body.accounts || []);
    });
    return () => {
      gone = true;
    };
  }, [query, sort]);

  const found = accounts || [];
  const shares = distribution(found);

  return (
    <div className="directory">
      <div className="search-panel">
        <div className="search-row">
          <div className="search-field">
            <label htmlFor="account-search">Поиск</label>
            <span className="search-glass" aria-hidden="true"><SearchIcon /></span>
            <input
              id="account-search"
              value={query}
              placeholder="Поиск аккаунтов по имени"
              autoComplete="off"
              onChange={(event) => setQuery(event.target.value)}
            />
            {query ? (
              <ul className="suggest" role="listbox">
                {found.length === 0 ? <li>Ничего не нашлось</li> : null}
                {found.map((account) => (
                  <li key={account.name}>
                    <Link href={`/${account.name}`}>{account.name}</Link>
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
          <span className="count-pill">{accounts ? ruCount(found.length, "аккаунт", "аккаунта", "аккаунтов") : "…"}</span>
        </div>
        <div className="search-note">
          <p className="hint"><span className="ok-mark" aria-hidden="true">✓</span> Поиск без учёта регистра. Содержимое локальных файлов не индексируется.</p>
          <div className="examples" aria-label="Примеры ИИ-агентов">
            <span>Поддерживаемые агенты:</span>
            {examples.map((item) => (
              <span key={item.name}>
                <i style={{ background: item.color }} />
                {item.name}
              </span>
            ))}
          </div>
        </div>
      </div>

      <div className="sort-bar">
        <span className="sort-label">Сортировка</span>
        <div className="sorts" role="group" aria-label="Сортировка">
          {sorts.map((item) => (
            <button key={item.id} type="button" aria-pressed={sort === item.id} onClick={() => setSort(item.id)}>
              {item.label}
            </button>
          ))}
        </div>
      </div>
      {explanation ? <p className="explanation">{explanation}</p> : null}

      <div className="directory-grid">
        <div className="cards">
          {accounts === null ? <p className="lede">Открываем список…</p> : null}
          {accounts && found.length === 0 ? (
            <p className="lede cards-empty">
              {query ? "Ничего не нашлось по этому имени." : "Пока нет аккаунтов."}
            </p>
          ) : null}
          {found.map((account) => (
            <AccountCard key={account.name} account={account} />
          ))}
        </div>
        <aside className="side">
          <section className="side-card">
            <div className="side-kicker">
              <h2>Распределение агентов</h2>
              <span>сводка</span>
            </div>
            {shares.total === 0 ? (
              <p className="hint">Нет публикаций в{"\u00a0"}этом списке.</p>
            ) : (
              <ShareRing shares={shares.rows} total={shares.total} />
            )}
            <ul className="share-table">
              {shares.rows.map((row) => (
                <li key={row.name}>
                  <span>{row.name}</span>
                  <b>{row.pct}%</b>
                </li>
              ))}
            </ul>
          </section>

          <section className="side-card">
            <h2>Политика строгой изоляции</h2>
            <p className="hint">Учётные данные и машинные пути остаются на компьютере.</p>
            <ul className="policy">
              {policy.map((line) => (
                <li key={line}><span className="ok-mark" aria-hidden="true">✓</span>{line}</li>
              ))}
            </ul>
          </section>
        </aside>
      </div>
    </div>
  );
}

function AccountCard({ account }) {
  const agents = account.agents || [];
  const published = agents.filter((agent) => agent.version != null);
  return (
    <article className={published.length === 0 ? "account-card quiet" : "account-card"}>
      <div className="card-top">
        <div className="card-id">
          <Avatar className="initials" name={account.name} updated={account.avatarUpdated} letter={initials(account.name)} />
          <div>
            <div className="card-title">
              <Link href={`/${account.name}`}>{account.name}</Link>
              {account.verified ? <em className="badge ok">проверен</em> : null}
              {account.fresh ? <em className="badge">новый</em> : null}
              {account.topWeek ? <em className="badge top">топ недели</em> : null}
            </div>
            <p className="card-meta">{metaLine(account, agents, published.length)}</p>
          </div>
        </div>
        <div className="metric-chips">
          <span className="metric-chip"><EyeIcon />{formatCount(account.views)}</span>
          <span className="metric-chip" title="Сумма лайков публикаций"><HeartIcon />{formatCount(account.likes)}</span>
        </div>
      </div>
    </article>
  );
}

function ShareRing({ shares, total }) {
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
        <span>публикаций</span>
      </div>
    </div>
  );
}

function metaLine(account, agents, published) {
  if (published === 0) {
    return "В агентах нет опубликованных настроек";
  }
  const when = account.publishedAt ? `Обновлено: ${ago(account.publishedAt)}` : "Обновлено";
  const active = published === agents.length && agents.length === 3
    ? "Все 3 агента активны"
    : published === 1
      ? `1 активный агент (${agents.find((agent) => agent.version != null).name})`
      : ruCount(published, "активный агент", "активных агента", "активных агентов");
  return `${when} · ${active}`;
}

function ago(value) {
  const minutes = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 60000));
  if (minutes < 1) {
    return "только что";
  }
  if (minutes < 60) {
    return `${ruCount(minutes, "минуту", "минуты", "минут")} назад`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return `${ruCount(hours, "час", "часа", "часов")} назад`;
  }
  const days = Math.round(hours / 24);
  if (days === 1) {
    return "вчера";
  }
  return `${ruCount(days, "день", "дня", "дней")} назад`;
}

function formatCount(value) {
  return String(value).replace(/\B(?=(\d{3})+(?!\d))/g, "\u202f");
}

function SearchIcon() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
      <circle cx="11" cy="11" r="6" fill="none" stroke="currentColor" strokeWidth="1.8" />
      <path d="M16 16l4 4" fill="none" stroke="currentColor" strokeWidth="1.8" />
    </svg>
  );
}

function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
      <path fill="currentColor" d="M12 6c4.5 0 8.2 2.8 9.5 6-1.3 3.2-5 6-9.5 6S3.8 15.2 2.5 12C3.8 8.8 7.5 6 12 6zm0 2a4 4 0 1 0 0 8 4 4 0 0 0 0-8z" />
    </svg>
  );
}

function HeartIcon({ filled }) {
  return (
    <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
      <path fill={filled ? "#fb7185" : "currentColor"} d="M12 19s-6.5-4.1-8.2-8.1C2.6 8.4 4 6 6.6 6c1.6 0 2.6.8 3.4 1.8C10.8 6.8 11.8 6 13.4 6 16 6 17.4 8.4 16.2 10.9 14.5 14.9 12 19 12 19z" />
    </svg>
  );
}

function distribution(accounts) {
  const rows = examples.map((item) => ({ ...item, count: 0, pct: 0 }));
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

function initials(name) {
  const parts = name.split("-").filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
}

function ruCount(n, one, few, many) {
  const n10 = n % 10;
  const n100 = n % 100;
  let word = many;
  if (n10 === 1 && n100 !== 11) {
    word = one;
  } else if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) {
    word = few;
  }
  return `${n}\u00a0${word}`;
}

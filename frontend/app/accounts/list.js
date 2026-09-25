"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "../api";
import { listenerLabel, useAgentProbe } from "../agent";

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
  const [pingNote, setPingNote] = useState("");
  const [agent, refreshAgent] = useAgentProbe();

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

  async function ping() {
    const next = await refreshAgent();
    setPingNote(next.ok ? "" : "Локальный агент не отвечает.");
  }

  const found = accounts || [];
  const shares = distribution(found);

  return (
    <div className="directory">
      <div className="search-row">
        <div className="search-field">
          <label htmlFor="account-search">Поиск</label>
          <input
            id="account-search"
            value={query}
            placeholder="Nova"
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
          <p className="hint">Поиск без учёта регистра. Содержимое файлов публикаций не ищется.</p>
        </div>
        <div className="search-side">
          <b>{accounts ? ruCount(found.length, "аккаунт", "аккаунта", "аккаунтов") : "…"}</b>
          <div className="examples" aria-label="Примеры ИИ-агентов">
            {examples.map((item) => (
              <span key={item.name}>
                <i style={{ background: item.color }} />
                {item.name}
              </span>
            ))}
          </div>
        </div>
      </div>

      <div className="sorts" role="group" aria-label="Сортировка">
        {sorts.map((item) => (
          <button key={item.id} type="button" aria-pressed={sort === item.id} onClick={() => setSort(item.id)}>
            {item.label}
          </button>
        ))}
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
            <div className="side-head">
              <h2>Мост агента</h2>
              <span className={agent.ok ? "live" : "wait"}>{agent.ok ? "Соединён" : "нет связи"}</span>
            </div>
            <dl>
              <div>
                <dt>Слушатель</dt>
                <dd>{agent.ok ? listenerLabel : "нет"}</dd>
              </div>
              <div>
                <dt>Хост</dt>
                <dd>{agent.ok && agent.host ? agent.host : "нет"}</dd>
              </div>
              <div>
                <dt>Канал</dt>
                <dd>{agent.ok ? "IPC Local Socket" : "нет"}</dd>
              </div>
              <div>
                <dt>Задержка</dt>
                <dd>{agent.ok ? `${agent.ms}\u00a0мс` : "нет"}</dd>
              </div>
            </dl>
            <button className="ghost" type="button" onClick={ping}>
              Проверить пинг моста
            </button>
            {pingNote ? <p className="explanation">{pingNote}</p> : null}
          </section>

          <section className="side-card">
            <h2>Распределение ИИ-агентов</h2>
            {shares.total === 0 ? (
              <p className="hint">Нет публикаций в{"\u00a0"}этом списке.</p>
            ) : (
              <ShareRing shares={shares.rows} />
            )}
            <ul className="shares">
              {shares.rows.map((row) => (
                <li key={row.name}>
                  <i style={{ background: row.color }} />
                  <span>{row.name}</span>
                  <b>{row.pct}%</b>
                </li>
              ))}
            </ul>
          </section>

          <section className="side-card">
            <h2>Политика строгой изоляции</h2>
            <ul className="policy">
              {policy.map((line) => (
                <li key={line}>{line}</li>
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
  const published = agents.filter((agent) => agent.version != null).length;
  return (
    <article className="account-card">
      <div className="card-id">
        <span className="initials" aria-hidden="true">{initials(account.name)}</span>
        <div>
          <div className="card-title">
            <Link href={`/${account.name}`}>{account.name}</Link>
            {account.verified ? <em className="badge ok">проверен</em> : null}
            {account.fresh ? <em className="badge">новый</em> : null}
            {account.topWeek ? <em className="badge top">топ недели</em> : null}
          </div>
          <p className="hint">
            {account.publishedAt ? publishedLabel(account.publishedAt) : "публикаций нет"}
            {" · "}
            {published === 0 ? "нет ИИ-агентов с\u00a0публикацией" : ruCount(published, "ИИ-агент", "ИИ-агента", "ИИ-агентов")}
          </p>
        </div>
      </div>
      <div className="card-stats">
        <span><b>{account.views}</b> просмотры</span>
        <span><b>{account.likes}</b> лайки</span>
      </div>
      <ul className="slots">
        {agents.map((agent) => (
          <li key={agent.name}>
            <i style={{ background: colorOf(agent.name) }} />
            <span>{agent.name}</span>
            <em>{agent.version == null ? "Нет публикаций" : `v${agent.version}`}</em>
          </li>
        ))}
      </ul>
    </article>
  );
}

function ShareRing({ shares }) {
  const radius = 15.9155;
  const circ = 2 * Math.PI * radius;
  let offset = 0;
  return (
    <svg className="ring" viewBox="0 0 42 42" aria-hidden="true">
      <circle cx="21" cy="21" r={radius} />
      {shares.map((row) => {
        const length = (row.pct / 100) * circ;
        const dash = `${length} ${circ - length}`;
        const node = (
          <circle
            key={row.name}
            cx="21"
            cy="21"
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

function colorOf(name) {
  return examples.find((item) => item.name === name)?.color || "var(--muted)";
}

function initials(name) {
  const parts = name.split("-").filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
}

function publishedLabel(value) {
  return new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
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

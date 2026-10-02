"use client";

import { useEffect, useState } from "react";
import { api } from "../api";
import { distribution, publicationWord, ShareRing } from "../distribution";

export default function StatsPage() {
  const [accounts, setAccounts] = useState(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let gone = false;
    api("/accounts").then((res) => {
      if (gone) {
        return;
      }
      if (!res.ok) {
        setFailed(true);
        setAccounts([]);
        return;
      }
      setAccounts(res.body.accounts || []);
    });
    return () => {
      gone = true;
    };
  }, []);

  const shares = distribution(accounts || []);

  return (
    <div className="stats-page">
      <h1>Статистика</h1>
      <aside className="side">
        <section className="side-card stats-card" aria-busy={accounts === null}>
          <div className="side-kicker">
            <h2>Распределение агентов</h2>
            <span>сводка</span>
          </div>
          {failed ? <p className="hint">Не удалось открыть сводку.</p> : null}
          {accounts === null && !failed ? <p className="hint">Открываем сводку…</p> : null}
          {accounts !== null && !failed && shares.total === 0 ? <p className="hint">Нет публикаций.</p> : null}
          {accounts !== null && !failed && shares.total > 0 ? (
            <ShareRing shares={shares.rows} total={shares.total} caption={publicationWord(shares.total)} />
          ) : null}
          {accounts !== null && !failed ? (
            <ul className="share-table">
              {shares.rows.map((row) => (
                <li key={row.name}>
                  <span>{row.name}</span>
                  <b>{row.pct}%</b>
                </li>
              ))}
            </ul>
          ) : null}
        </section>
      </aside>
    </div>
  );
}

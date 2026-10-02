"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../../api";

export default function SnapshotListPage() {
  const params = useParams();
  const name = String(params.name || "");
  const agent = String(params.agent || "");
  const [state, setState] = useState(null);

  useEffect(() => {
    let gone = false;
    api(`/accounts/${encodeURIComponent(name)}/versions/${encodeURIComponent(agent)}`).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [name, agent]);

  if (!state) {
    return <p className="lede">Открываем версии…</p>;
  }
  if (!state.ok) {
    return <p className="explanation">{state.body?.explanation || "Не удалось открыть версии."}</p>;
  }
  const shots = [...(state.body.versions || [])].sort((a, b) => a.version - b.version);

  return (
    <section className="directory">
      <nav className="crumbs" aria-label="Путь">
        <Link href={`/${name}`}>{name}</Link>
        <span>/</span>
        <b>{agent}</b>
      </nav>
      <div className="section-title">
        <div className="section-title-main">
          <h2>Версии</h2>
          <span className="count-pill">{shots.length}</span>
        </div>
        <p className="hint section-aside">Публикации этого ИИ-агента</p>
      </div>
      {shots.length === 0 ? (
        <p className="hint">Версий пока нет. Они появляются командой agentsync push.</p>
      ) : (
        <div className="snap-list">
          {shots.map((shot) => (
            <Link key={shot.version} className="snap-row" href={`/${name}/store/${encodeURIComponent(agent)}/${shot.version}`}>
              <span className="mono">v{shot.version}</span>
              <span>{stamp(shot.created)}</span>
              {shot.current ? <em className="badge ok">текущая</em> : <em />}
              <span className="snap-open">Осмотр</span>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

function stamp(value) {
  if (!value) {
    return "";
  }
  return new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

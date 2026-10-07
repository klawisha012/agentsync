"use client";

import Link from "next/link";
import { useState } from "react";
import { api } from "../api";
import AgentLogo from "../agent-logo";

const tones = {
  Grok: "grok",
  Agents: "agents",
  Claude: "claude",
};

export default function AgentBoard({ page, owner, onChange, onExplain }) {
  const [copied, setCopied] = useState("");
  const [notes, setNotes] = useState({});
  const agents = page.agents || [];

  function note(agentName, text) {
    setNotes((current) => ({ ...current, [agentName]: text }));
  }

  async function withdraw(agent) {
    if (!agent.publicationId) {
      note(agent.name, "Публикация не найдена.");
      return;
    }
    const res = await api(`/publications/${agent.publicationId}/withdraw`, { method: "POST", body: "{}" });
    if (!res.ok) {
      note(agent.name, res.body?.explanation || "Не удалось снять публикацию.");
      return;
    }
    note(agent.name, "");
    await onChange();
  }

  async function like(agent) {
    const res = await api(`/accounts/${encodeURIComponent(page.name)}/agents/${encodeURIComponent(agent.name)}/like`, {
      method: "POST",
      body: "{}",
    });
    if (!res.ok) {
      onExplain(res.body?.explanation || "Не удалось поставить лайк.");
      return;
    }
    onExplain("");
    await onChange();
  }

  async function copyText(key, text) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(key);
    } catch {
      onExplain("Не удалось скопировать команду.");
    }
  }

  const published = agents.filter((agent) => agent.version != null).length;

  return (
    <section className="agents">
      <div className="section-title">
        <div className="section-title-main">
          <h2>{owner ? "Управление опубликованными агентами" : "Опубликованные агенты"}</h2>
          <span className="count-pill">{publishedLabel(published)}</span>
        </div>
        {owner ? <p className="hint section-aside">Публикация командой agentsync push</p> : null}
      </div>
      <div className="house-grid">
        {agents.map((agent) => {
          const command = owner ? `agentsync push ${agent.name}` : `agentsync apply ${page.name} ${agent.name}`;
          const tone = tones[agent.name] || "plain";
          const canCopy = owner || (page.shareCopy && agent.version != null);
          const canView = owner || (page.shareView && agent.version != null);
          const canVersions = owner || page.shareVersions;
          return (
            <article className="house-card" key={agent.name}>
              <div>
                <div className="house-head">
                  <span>
                    <AgentLogo name={agent.name} tone={tone} />
                    <strong>{agent.name}</strong>
                  </span>
                  {agent.version != null ? <em className={`version-chip tone-${tone}`}>v{agent.version}</em> : null}
                </div>
                <p className="path-line">
                  <Icon name="folder" />
                  ~/.{agent.name.toLowerCase()}
                </p>
                <p className="house-copy">Переносимая настройка без учётных данных и машинного MCP.</p>
                <dl className="house-stats">
                  <div>
                    <dt>Файлов</dt>
                    <dd>{agent.version == null ? "нет" : `${agent.files} переносимых`}</dd>
                  </div>
                  <div>
                    <dt>Обновлено</dt>
                    <dd className={agent.publishedAt ? "fresh" : ""}>{agent.publishedAt ? ago(agent.publishedAt) : "нет"}</dd>
                  </div>
                  {agent.version != null ? (
                    <div>
                      <dt>Лайки</dt>
                      <dd>
                        {owner ? (
                          <span className="like-count" title="Свою публикацию лайкнуть нельзя">{formatCount(agent.likes || 0)}</span>
                        ) : (
                          <button className={agent.liked ? "like-count on" : "like-count"} type="button" aria-pressed={Boolean(agent.liked)} onClick={() => like(agent)}>
                            {agent.liked ? "Снять лайк" : "Лайк"}
                            <b>{formatCount(agent.likes || 0)}</b>
                          </button>
                        )}
                      </dd>
                    </div>
                  ) : null}
                </dl>
              </div>
              {notes[agent.name] ? <p className="explanation">{notes[agent.name]}</p> : null}
              {owner || canCopy || canView || canVersions ? (
                <div className="house-foot">
                  <div className="house-actions">
                    {canCopy ? (
                      <button type="button" aria-label={`Скопировать команду ${command}`} onClick={() => copyText(`push:${agent.name}`, command)}>
                        <Icon name={copied === `push:${agent.name}` ? "check" : "copy"} />
                        {copied === `push:${agent.name}` ? "Скопировано" : "Скопировать"}
                      </button>
                    ) : null}
                    {canView && agent.version != null ? (
                      <Link href={`/${page.name}/store/${encodeURIComponent(agent.name)}/${agent.version}`}>
                        <Icon name="eye" />
                        Осмотр
                      </Link>
                    ) : null}
                    {owner && agent.version == null ? (
                      <button type="button" onClick={() => note(agent.name, "Нет публикаций.")}>
                        <Icon name="eye" />
                        Осмотр
                      </button>
                    ) : null}
                    {canVersions ? (
                      <Link href={`/${page.name}/store/${encodeURIComponent(agent.name)}`}>
                        <Icon name="history" />
                        Версии
                      </Link>
                    ) : null}
                  </div>
                  {owner ? (
                    <button className="withdraw" type="button" onClick={() => withdraw(agent)}>
                      <Icon name="unpublish" />
                      Снять публикацию
                    </button>
                  ) : null}
                </div>
              ) : null}
            </article>
          );
        })}
      </div>
    </section>
  );
}

function publishedLabel(count) {
  const n10 = count % 10;
  const n100 = count % 100;
  let word = "агентов";
  if (n10 === 1 && n100 !== 11) {
    word = "агент";
  } else if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) {
    word = "агента";
  }
  return `${count}\u00a0${word} с\u00a0публикацией`;
}

function formatCount(value) {
  return String(value || 0).replace(/\B(?=(\d{3})+(?!\d))/g, "\u202f");
}

function ago(value) {
  const minutes = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 60000));
  if (minutes < 1) {
    return "только что";
  }
  if (minutes < 60) {
    return ruUnit(minutes, "минуту", "минуты", "минут");
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return ruUnit(hours, "час", "часа", "часов");
  }
  return ruUnit(Math.round(hours / 24), "день", "дня", "дней");
}

function ruUnit(n, one, few, many) {
  const n10 = n % 10;
  const n100 = n % 100;
  let word = many;
  if (n10 === 1 && n100 !== 11) {
    word = one;
  } else if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) {
    word = few;
  }
  return `${n}\u00a0${word} назад`;
}

function Icon({ name }) {
  const paths = {
    folder: "M3 6h7l2 2h9v10H3V6z",
    upload: "M12 4l5 5h-3v6h-4V9H7l5-5zm-7 14h14v2H5v-2z",
    eye: "M12 6c4.5 0 8.2 2.8 9.5 6-1.3 3.2-5 6-9.5 6S3.8 15.2 2.5 12C3.8 8.8 7.5 6 12 6zm0 2a4 4 0 1 0 0 8 4 4 0 0 0 0-8z",
    history: "M12 5a7 7 0 1 1-6.3 4H8l-3.2-3L1.5 9H4.1A9 9 0 1 0 12 3v2zm-1 3h2v5l3 2-.9 1.4L11 13.2V8z",
    unpublish: "M6 6h12v2H6V6zm2 4h8v8H8v-8z",
    copy: "M8 7h10v12H8V7zm-2 2H4v12h10v-2H6V9z",
    check: "M9 16.2 4.8 12l-1.4 1.4L9 19 21 7l-1.4-1.4z",
  };
  return (
    <svg className="ico" viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
      <path fill="currentColor" d={paths[name]} />
    </svg>
  );
}

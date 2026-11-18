"use client";

import Link from "next/link";
import { useState } from "react";
import { api } from "../api";
import AgentLogo from "../agent-logo";
import Icon from "../profile-icon";
import { shellCommand } from "../shell-arg";

export default function AgentBoard({ page, owner, onChange, onExplain }) {
  const [copied, setCopied] = useState(false);
  const [note, setNote] = useState("");
  const agents = page.agents || [];
  const publishedSlot = agents.find((item) => String(item.name || "").trim().toLowerCase() === "agents" && item.version != null) || null;
  const slot = publishedSlot || (owner ? { name: "agents", version: null, files: 0, publishedAt: null, likes: 0, liked: false } : null);

  async function withdraw(agent) {
    if (!agent?.publicationId) {
      setNote("Публикация не найдена.");
      return;
    }
    const res = await api(`/publications/${agent.publicationId}/withdraw`, { method: "POST", body: "{}" });
    if (!res.ok) {
      setNote(res.body?.explanation || "Не удалось снять публикацию.");
      return;
    }
    setNote("");
    await onChange();
  }

  async function like() {
    const res = await api(`/accounts/${encodeURIComponent(page.name)}/agents/agents/like`, {
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

  async function copyText(text) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      onExplain("Не удалось скопировать команду.");
    }
  }

  const command = owner
    ? shellCommand(["agentsync", "push"])
    : shellCommand(["agentsync", "apply", page.name]);
  const canCopy = owner || (page.shareCopy && slot?.version != null);
  const canView = owner || (page.shareView && slot?.version != null);
  const canVersions = owner || page.shareVersions;

  return (
    <section className="agents">
      <div className="section-title">
        <div className="section-title-main">
          <h2>{owner ? "Управление публикацией agents" : "Публикация agents"}</h2>
          <span className="count-pill">{slot?.version != null ? `v${slot.version}` : "не опубликовано"}</span>
        </div>
        {owner ? <p className="hint section-aside">Публикация командой agentsync push</p> : null}
      </div>
      {!slot ? (
        <p className="hint">Публикация agents пока не открыта.</p>
      ) : (
        <div className="house-grid">
          <article className="house-card">
            <div>
              <div className="house-head">
                <span>
                  <AgentLogo name="agents" tone="agents" />
                  <strong>agents</strong>
                </span>
                {slot.version != null ? <em className="version-chip tone-agents">v{slot.version}</em> : null}
              </div>
              <p className="path-line">
                <Icon name="folder" />
                ~/.agents
              </p>
              <p className="house-copy">
                Общая настройка всех подключённых ИИ-агентов без .env и машинного MCP.
              </p>
              <dl className="house-stats">
                <div>
                  <dt>Файлов</dt>
                  <dd>{slot.version == null ? "нет" : `${slot.files} переносимых`}</dd>
                </div>
                <div>
                  <dt>Обновлено</dt>
                  <dd className={slot.publishedAt ? "fresh" : ""}>{slot.publishedAt ? ago(slot.publishedAt) : "нет"}</dd>
                </div>
                {slot.version != null ? (
                  <div>
                    <dt>Лайки</dt>
                    <dd>
                      {owner ? (
                        <span className="like-count" title="Свою публикацию лайкнуть нельзя">{formatCount(slot.likes || 0)}</span>
                      ) : (
                        <button className={slot.liked ? "like-count on" : "like-count"} type="button" aria-pressed={Boolean(slot.liked)} onClick={like}>
                          {slot.liked ? "Снять лайк" : "Лайк"}
                          <b>{formatCount(slot.likes || 0)}</b>
                        </button>
                      )}
                    </dd>
                  </div>
                ) : null}
              </dl>
            </div>
            {note ? <p className="explanation">{note}</p> : null}
            {owner || canCopy || canView || canVersions ? (
              <div className="house-foot">
                <div className="house-actions">
                  {canCopy ? (
                    <button type="button" aria-label={`Скопировать команду ${command}`} onClick={() => copyText(command)}>
                      <Icon name={copied ? "check" : "copy"} />
                      {copied ? "Скопировано" : "Скопировать"}
                    </button>
                  ) : null}
                  {canView && slot.version != null ? (
                    <Link href={`/${page.name}/store/agents/${slot.version}`}>
                      <Icon name="eye" />
                      Осмотр
                    </Link>
                  ) : null}
                  {owner && slot.version == null ? (
                    <button type="button" onClick={() => setNote("Нет публикаций.")}>
                      <Icon name="eye" />
                      Осмотр
                    </button>
                  ) : null}
                  {canVersions ? (
                    <Link href={`/${page.name}/store/agents`}>
                      <Icon name="history" />
                      Версии
                    </Link>
                  ) : null}
                </div>
                {owner && slot.version != null ? (
                  <button className="withdraw" type="button" onClick={() => withdraw(slot)}>
                    <Icon name="unpublish" />
                    Снять публикацию
                  </button>
                ) : null}
              </div>
            ) : null}
          </article>
        </div>
      )}
    </section>
  );
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



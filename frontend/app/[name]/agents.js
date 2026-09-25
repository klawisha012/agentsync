"use client";

import Link from "next/link";
import { useState } from "react";
import { api } from "../api";
import { probeAgent } from "../agent";

const colors = {
  Grok: "var(--grok)",
  Agents: "var(--agents)",
  Claude: "var(--claude)",
};

export default function AgentBoard({ name, page, owner, onChange, onExplain }) {
  const [copied, setCopied] = useState("");
  const agents = page.agents || [];

  async function updateAgent(agentName) {
    onExplain("");
    if (owner && page.verified === false) {
      onExplain("Подтвердите почту, чтобы загрузить или снять публикацию.");
      return;
    }
    const probe = await probeAgent();
    if (!probe.ok) {
      onExplain("Локальный агент не отвечает.");
      return;
    }
    let exported;
    try {
      const res = await fetch("/agent-bridge/push", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent: agentName }),
      });
      if (!res.ok) {
        onExplain("Локальный агент не отвечает.");
        return;
      }
      exported = await res.json();
    } catch {
      onExplain("Локальный агент не отвечает.");
      return;
    }
    const pushed = await api("/agent/push", {
      method: "POST",
      body: JSON.stringify({
        agent: agentName,
        missing: Boolean(exported.missing),
        files: exported.files || [],
      }),
    });
    if (!pushed.ok) {
      onExplain(pushed.body?.explanation || "Не удалось загрузить ИИ-агента.");
      return;
    }
    await onChange();
  }

  async function copyCommand(agentName) {
    try {
      await navigator.clipboard.writeText(`agentsync push ${agentName}`);
      setCopied(agentName);
    } catch {
      onExplain("Не удалось скопировать команду.");
    }
  }

  return (
    <section className="agents">
      <div className="agent-grid">
        {agents.map((agent) => (
          <article className="account-card" key={agent.name}>
            <div className="card-title">
              <i className="agent-dot" style={{ background: colors[agent.name] || "var(--muted)" }} />
              <strong>{agent.name}</strong>
              {agent.version != null ? <em className="badge">v{agent.version}</em> : null}
            </div>
            <p className="hint">
              {agent.publicationId ? `/publications/${agent.publicationId}` : "Публикации нет"}
            </p>
            <p className="hint">Переносимая настройка без учётных данных и машинного MCP.</p>
            <p className="hint">
              {agent.version == null
                ? "Нет публикаций"
                : `${agent.files}\u00a0файлов · ${publishedLabel(agent.publishedAt)}`}
            </p>
            <div className="agent-actions">
              {agent.version != null && !owner ? (
                <button type="button">Применить v{agent.version}</button>
              ) : null}
              {owner ? (
                <>
                  <button className="solid" type="button" onClick={() => updateAgent(agent.name)}>Обновить</button>
                  {agent.publicationId ? (
                    <Link href={`/publications/${agent.publicationId}`}>Осмотр</Link>
                  ) : null}
                  <button type="button">Снимки</button>
                  <button type="button">Снять публикацию</button>
                </>
              ) : null}
            </div>
          </article>
        ))}
      </div>
      {owner ? (
        <div className="commands">
          <h2>Команды загрузки</h2>
          {agents.map((agent) => (
            <div className="terminal-line" key={agent.name}>
              <code>agentsync push {agent.name}</code>
              <button type="button" onClick={() => copyCommand(agent.name)}>
                {copied === agent.name ? "Скопировано" : "Копировать"}
              </button>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function publishedLabel(value) {
  if (!value) {
    return "";
  }
  return new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

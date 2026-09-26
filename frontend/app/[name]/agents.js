"use client";

import Link from "next/link";
import { useState } from "react";
import { api } from "../api";
import { probeAgent, useAgentProbe } from "../agent";

const colors = {
  Grok: "var(--grok)",
  Agents: "var(--agents)",
  Claude: "var(--claude)",
};

export default function AgentBoard({ name, page, owner, onChange, onExplain }) {
  const [copied, setCopied] = useState("");
  const [agentProbe] = useAgentProbe();
  const [notes, setNotes] = useState({});
  const agents = page.agents || [];

  function note(agentName, text) {
    setNotes((current) => ({ ...current, [agentName]: text }));
  }

  async function updateAgent(agentName) {
    note(agentName, "");
    if (owner && page.verified === false) {
      note(agentName, "Подтвердите почту, чтобы загрузить или снять публикацию.");
      return;
    }
    const probe = await probeAgent();
    if (!probe.ok) {
      note(agentName, "Локальный агент не отвечает.");
      return;
    }
    const same = !probe.account || probe.account.localeCompare(name, "ru", { sensitivity: "accent" }) === 0;
    if (!same) {
      note(agentName, "Аккаунт браузера и\u00a0аккаунт машины различаются.");
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
        note(agentName, "Локальный агент не отвечает.");
        return;
      }
      exported = await res.json();
    } catch {
      note(agentName, "Локальный агент не отвечает.");
      return;
    }
    if (!exported.token) {
      note(agentName, "Аккаунт браузера и\u00a0аккаунт машины различаются.");
      return;
    }
    const pushed = await fetch("/backend/agent/push", {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${exported.token}`,
      },
      body: JSON.stringify({
        agent: agentName,
        missing: Boolean(exported.missing),
        files: exported.files || [],
      }),
    });
    const text = await pushed.text();
    const body = text ? JSON.parse(text) : null;
    if (!pushed.ok) {
      note(agentName, body?.explanation || "Не удалось загрузить ИИ-агента.");
      return;
    }
    note(agentName, "");
    await onChange();
  }

  async function withdraw(agent) {
    if (!agent.publicationId) {
      note(agent.name, "Публикация не найдена.");
      return;
    }
    if (page.verified === false) {
      note(agent.name, "Подтвердите почту, чтобы загрузить или снять публикацию.");
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
            {notes[agent.name] ? <p className="explanation">{notes[agent.name]}</p> : null}
            <div className="agent-actions">
              {agent.version != null && !owner && agentProbe.ok ? (
                <button type="button">Применить v{agent.version}</button>
              ) : null}
              {owner ? (
                <>
                  <button className="solid" type="button" onClick={() => updateAgent(agent.name)}>Обновить</button>
                  {agent.publicationId ? (
                    <Link href={`/publications/${agent.publicationId}`}>Осмотр</Link>
                  ) : null}
                  <button type="button" onClick={() => note(agent.name, agentProbe.ok ? "Цепочка на этом компьютере." : "Локальный агент не отвечает.")}>Снимки</button>
                  <button type="button" onClick={() => withdraw(agent)}>Снять публикацию</button>
                  <button type="button" onClick={() => note(agent.name, agentProbe.ok ? `agentsync revert ${agent.name}` : "Локальный агент не отвечает.")}>Откатить</button>
                </>
              ) : null}
            </div>
          </article>
        ))}
      </div>
      {owner ? (
        <div className="commands">
          <h2>Публикация и обновление конфигураций через терминал</h2>
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

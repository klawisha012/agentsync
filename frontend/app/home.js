"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

const tabs = [
  { id: "ps", label: "PowerShell", text: (origin) => `irm ${origin}/cli/install.ps1 | iex` },
  { id: "bash", label: "WSL / bash", text: (origin) => `curl -fsSL ${origin}/cli/install.sh | bash` },
  { id: "mac", label: "macOS / curl", text: (origin) => `curl -fsSL ${origin}/cli/install.sh | sh` },
];

export default function Home() {
  const [origin, setOrigin] = useState("");
  const [tab, setTab] = useState("ps");
  const [copied, setCopied] = useState(false);
  const [explanation, setExplanation] = useState("");

  useEffect(() => {
    setOrigin(window.location.origin);
  }, []);

  const command = tabs.find((item) => item.id === tab).text(origin);

  async function copy() {
    if (!origin) {
      return;
    }
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      setExplanation("");
    } catch {
      setCopied(false);
      setExplanation("Не удалось скопировать команду.");
    }
  }

  return (
    <div className="home">
      <section className="hero">
        <h1>Синхронизация AI-агентов на вашем компьютере.</h1>
        <p className="lede hero-lede">
          Однократный перенос переносимой настройки ИИ-агентов Grok, Agents и{"\u00a0"}Claude без утечки учётных данных и{"\u00a0"}машинного MCP. Поздняя публикация сама на компьютер не приезжает.
        </p>
        <div className="terminal">
          <div className="terminal-tabs" role="tablist">
            {tabs.map((item) => (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={tab === item.id}
                onClick={() => {
                  setTab(item.id);
                  setCopied(false);
                }}
              >
                {item.label}
              </button>
            ))}
          </div>
          <div className="terminal-line">
            <code>{command}</code>
            <button type="button" onClick={copy} disabled={!origin}>
              {copied ? "Скопировано" : "Копировать"}
            </button>
          </div>
        </div>
        <p className="hint terminal-note">
          Локальный агент сохраняет снимок в{"\u00a0"}цепочку перед применением, а{"\u00a0"}учётные данные и{"\u00a0"}ключи не покидают компьютер.
        </p>
        {explanation ? <p className="explanation">{explanation}</p> : null}
      </section>

      <section className="boundary">
        <h2>Граница публикации</h2>
        <div className="boundary-grid">
          <div>
            <h3>Попадает в{"\u00a0"}публикацию</h3>
            <ul>
              <li>Переносимые инструкции</li>
              <li>Хуки без абсолютного пути</li>
              <li>MCP без машинной привязки</li>
            </ul>
          </div>
          <div>
            <h3>Остаётся на машине</h3>
            <ul>
              <li>Учётные данные</li>
              <li>Абсолютные пути</li>
              <li>Машинный MCP</li>
            </ul>
          </div>
        </div>
      </section>

      <section className="cta">
        <div>
          <h2>Список аккаунтов</h2>
          <p className="hint">Страницы с{"\u00a0"}публикациями ИИ-агентов. Применение разовое: поздняя публикация сама не приезжает.</p>
        </div>
        <Link className="solid" href="/accounts">
          Перейти к{"\u00a0"}списку аккаунтов
        </Link>
      </section>
    </div>
  );
}

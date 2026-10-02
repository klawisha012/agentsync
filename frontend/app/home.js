"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import styles from "./hero-videos.module.css";

const tabs = [
  { id: "ps", label: "PowerShell", text: (origin) => `irm ${origin}/api/cli/install.ps1 | iex` },
  { id: "bash", label: "WSL / bash", text: (origin) => `curl -fsSL ${origin}/api/cli/install.sh | bash` },
  { id: "mac", label: "macOS / curl", text: (origin) => `curl -fsSL ${origin}/api/cli/install.sh | sh` },
];

const portable = [
  "Системные инструкции, архитектурные правила и\u00a0XML-схемы",
  "Относительные хуки жизненного цикла агента (pre-apply, post-apply)",
  "Конфигурации переносимых песочниц MCP без паролей",
];

const localOnly = [
  "API-ключи, токены сессий и\u00a0учётные записи поставщиков",
  "Абсолютные локальные пути (/Users/…, C:\\Users\\…)",
  "Машинно-зависимые MCP с\u00a0доступом к\u00a0файлам хоста",
];

export default function Home() {
  const muse = useRef(null);
  const cast = useRef(null);
  const [origin, setOrigin] = useState("");
  const [tab, setTab] = useState("ps");
  const [copied, setCopied] = useState(false);
  const [explanation, setExplanation] = useState("");

  useEffect(() => {
    setOrigin(window.location.origin);
  }, []);

  useEffect(() => {
    const videos = [muse.current, cast.current].filter(Boolean);
    if (videos.length === 0) {
      return;
    }
    const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const apply = () => {
      for (const video of videos) {
        if (motion.matches) {
          video.pause();
          continue;
        }
        video.play().catch(() => {});
      }
    };
    apply();
    motion.addEventListener("change", apply);
    return () => motion.removeEventListener("change", apply);
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
      <section className="hero" id="install">
        <div className={`hero-intro ${styles.intro}`}>
          <video
            className={`hero-muse ${styles.muse}`}
            ref={muse}
            autoPlay
            muted
            loop
            playsInline
            preload="metadata"
            aria-hidden="true"
          >
            <source src="/hero-muse.webm" type="video/webm" />
          </video>
          <h1 className={styles.title}>
            Синхронизация
            <br className="hero-break" />
            {" "}
            AI-агентов
            <br className="hero-break" />
            {" "}
            на вашем компьютере.
          </h1>
          <video
            className={styles.cast}
            ref={cast}
            autoPlay
            muted
            loop
            playsInline
            preload="metadata"
            aria-hidden="true"
          >
            <source src="/hero-cast.webm" type="video/webm" />
          </video>
        </div>
        <p className="lede hero-lede">
          Однократный перенос переносимой настройки ИИ-агента без риска утечки учётных данных и{"\u00a0"}машинного MCP.
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
            <span className="prompt" aria-hidden="true">&gt;</span>
            <code>{command}</code>
            <button type="button" onClick={copy} disabled={!origin}>
              {copied ? "Скопировано" : "Копировать"}
            </button>
          </div>
        </div>
        <p className="hint terminal-note">
          Локальный агент сохраняет снимок в{"\u00a0"}цепочку перед каждым применением. Учётные данные и{"\u00a0"}ключи не покидают компьютер.
        </p>
        {explanation ? <p className="explanation">{explanation}</p> : null}
      </section>

      <section className="boundary" id="isolation">
        <div className="iso-head">
          <div>
            <p className="iso-kicker">Изоляция первого класса</p>
            <h2>Строгий рубеж изоляции данных</h2>
          </div>
          <p className="iso-lead">
            В{"\u00a0"}манифесты попадают только декларативные инструкции и{"\u00a0"}переносимые песочницы. Локальные ключи и{"\u00a0"}физические пути остаются на машине.
          </p>
        </div>
        <div className="boundary-grid">
          <div>
            <h3 className="iso-title"><i />Переносимо и{"\u00a0"}безопасно публикуется</h3>
            <ul>
              {portable.map((line) => (
                <li key={line}><span className="ok-mark">✓</span><span>{line}</span></li>
              ))}
            </ul>
          </div>
          <div>
            <h3 className="iso-title"><i className="bad" />Строго локально, не покидает устройство</h3>
            <ul>
              {localOnly.map((line) => (
                <li key={line}><span className="bad-mark">✕</span><span>{line}</span></li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      <section className="cta">
        <div>
          <div className="kicker">Каталог синхронизации</div>
          <h2>Готовые публикации от авторов сообщества</h2>
          <p className="hint">
            Манифесты ИИ-агентов. Применение разовое: поздняя публикация сама на компьютер не приезжает.
          </p>
        </div>
        <Link className="solid" href="/accounts">
          Перейти к{"\u00a0"}списку аккаунтов
        </Link>
      </section>
    </div>
  );
}

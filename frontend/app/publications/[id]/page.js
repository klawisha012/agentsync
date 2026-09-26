"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../api";
import { useAgentProbe } from "../../agent";

const commandTabs = [
  { id: "mac", label: "macOS" },
  { id: "linux", label: "Linux" },
  { id: "ps", label: "PowerShell / WSL" },
];

export default function PublicationPage() {
  const params = useParams();
  const [state, setState] = useState(null);
  const [selected, setSelected] = useState("");
  const [tab, setTab] = useState("mac");
  const [copied, setCopied] = useState("");
  const [origin, setOrigin] = useState("");
  const [agent] = useAgentProbe();

  useEffect(() => {
    setOrigin(window.location.origin);
  }, []);

  useEffect(() => {
    let gone = false;
    api(`/publications/${encodeURIComponent(params.id)}`).then((res) => {
      if (gone) {
        return;
      }
      setState(res);
      const first = res.body?.files?.[0]?.path || "";
      setSelected(first);
    });
    return () => {
      gone = true;
    };
  }, [params.id]);

  if (!state) {
    return <p className="lede">Открываем публикацию…</p>;
  }
  if (!state.ok) {
    return <p className="explanation">{state.body?.explanation || "Публикация не найдена."}</p>;
  }

  const page = state.body;
  const files = page.files || [];
  const current = files.find((file) => file.path === selected) || files[0];
  const command = commandText(tab, origin, page.id);

  async function copy(text, key) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(key);
    } catch {
      setCopied("");
    }
  }

  return (
    <article className="preview">
      <nav className="crumbs" aria-label="Путь">
        <Link href="/catalog">каталог</Link>
        <span>/</span>
        <Link href={`/${page.author}`}>@{page.author}</Link>
        <span>/</span>
        <span>{page.agent} (~/…/{page.agent})</span>
        <span>/</span>
        <b>v{page.version}</b>
        <em className="badge">immutable</em>
      </nav>

      <div className="preview-grid">
        <div className="preview-side">
          <section className="side-card">
            <div className="side-head">
              <h2>Файлы манифеста</h2>
              <span className="hint">{ruFiles(files.length)}</span>
            </div>
            <ul className="file-tree">
              {files.map((file) => (
                <li key={file.path}>
                  <button
                    type="button"
                    aria-pressed={current?.path === file.path}
                    onClick={() => setSelected(file.path)}
                  >
                    {file.path}
                  </button>
                </li>
              ))}
            </ul>
            <p className="hint">
              Применение создаст {ruFiles(files.length)}. Перезапишет {current?.path || "файл"}.
            </p>
          </section>
          <section className="side-card">
            <h2>Исключено из публикации</h2>
            <ul className="excluded">
              {(page.excluded || []).map((item) => (
                <li key={item.path}>
                  <s>{item.path}</s>
                  <span>{item.label}</span>
                </li>
              ))}
            </ul>
          </section>
        </div>

        <section className="side-card preview-file">
          <div className="side-head">
            <div>
              <strong className="mono">{current?.path}</strong>
              <em className="badge ok">Переносимый конфиг</em>
            </div>
            <button type="button" onClick={() => copy(current?.body || "", "file")}>
              {copied === "file" ? "Скопировано" : "Копировать"}
            </button>
          </div>
          <p className="hint">{byteLabel(current?.body || "")}</p>
          <pre>{current?.body}</pre>
        </section>
      </div>

      <section className="side-card">
        <div className="side-head">
          <h2>Применение через командную строку</h2>
          <div className="terminal-tabs" role="tablist">
            {commandTabs.map((item) => (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={tab === item.id}
                onClick={() => setTab(item.id)}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
        <div className="terminal-line">
          <code>{command}</code>
          <button type="button" onClick={() => copy(command, "cmd")}>
            {copied === "cmd" ? "Скопировано" : "Скопировать команду"}
          </button>
        </div>
        {agent.ok ? <button type="button">Применить</button> : null}
      </section>
    </article>
  );
}

function commandText(tab, origin, id) {
  const url = `${origin}/publications/${id}`;
  if (tab === "ps") {
    return `irm ${url} | agentsync`;
  }
  return `curl -fsSL ${url} | agentsync`;
}

function byteLabel(text) {
  const size = new TextEncoder().encode(text).length;
  return `${size}\u00a0Б`;
}

function ruFiles(n) {
  const n10 = n % 10;
  const n100 = n % 100;
  let word = "файлов";
  if (n10 === 1 && n100 !== 11) {
    word = "файл";
  } else if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) {
    word = "файла";
  }
  return `${n}\u00a0${word}`;
}

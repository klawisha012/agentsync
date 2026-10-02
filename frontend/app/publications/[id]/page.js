"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../api";
import FileTree from "../../file-tree";
import { buildManifest, ruTokens, skillTokenTotal } from "./manifest";

const commandTabs = [
  { id: "mac", label: "macOS" },
  { id: "linux", label: "Linux" },
  { id: "ps", label: "PowerShell / WSL" },
];

export default function PublicationPage() {
  const params = useParams();
  const [state, setState] = useState(null);
  const [selected, setSelected] = useState("");
  const [open, setOpen] = useState(() => new Set());
  const [tab, setTab] = useState("mac");
  const [copied, setCopied] = useState("");

  useEffect(() => {
    let gone = false;
    setSelected("");
    setOpen(new Set());
    api(`/publications/${encodeURIComponent(params.id)}`).then((res) => {
      if (gone) {
        return;
      }
      setState(res);
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
  const current = files.find((file) => file.path === selected)
    || (selected === "" ? files.find((file) => !String(file.path).includes("/")) : null)
    || null;
  const command = `${commandText(page.author, page.agent)}\nagentsync revert ${page.agent}`;

  async function copy(text, key) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(key);
    } catch {
      setCopied("");
    }
  }

  const tree = buildManifest(files);

  function toggleFolder(path) {
    setOpen((current) => {
      const next = new Set(current);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }

  return (
    <article className="preview">
      <div className="preview-top">
        <nav className="crumbs" aria-label="Путь">
          <Link href="/catalog">каталог</Link>
          <span>/</span>
          <Link href={`/${page.author}`}>@{page.author}</Link>
          <span>/</span>
          <b>{page.agent} (~/.{String(page.agent).toLowerCase()})</b>
          <span>/</span>
          <em className="version-chip">v{page.version}</em>
          <em className="badge ok">immutable</em>
        </nav>
      </div>

      <div className="preview-grid">
        <div className="manifest">
          <div className="side-head">
            <h2>Файлы манифеста</h2>
            <span className="count-pill">{ruTokens(skillTokenTotal(tree))}</span>
          </div>
          <div className="file-tree" role="tree" aria-label="Файлы манифеста">
            <FileTree
              nodes={tree}
              depth={0}
              open={open}
              onToggle={toggleFolder}
              selected={current?.path || ""}
              onSelect={setSelected}
            />
          </div>
          <p className="hint">{"У\u00A0скилла\u00A0— токены описания и\u00A0всех файлов папки."}</p>
          <p className="hint">
            Применение создаст {ruFiles(files.length)}.{current ? ` Перезапишет ${current.path}.` : ""}
          </p>
          <div className="excluded-box">
            <h2>Исключено из публикации</h2>
            <ul className="excluded">
              {(page.excluded || []).map((item) => (
                <li key={item.path}>
                  <s>{item.path}</s>
                  <span>{item.label}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <section className="code-stage">
          <div className="side-head">
            <div>
              <strong className="mono">{current ? current.path : "Файл не выбран"}</strong>
              {current ? <em className="badge ok">Переносимый конфиг</em> : null}
            </div>
            {current ? (
              <button type="button" onClick={() => copy(current.body || "", "file")}>
                {copied === "file" ? "Скопировано" : "Копировать"}
              </button>
            ) : null}
          </div>
          {current ? <pre>{current.body}</pre> : <p className="hint tree-empty">{"Откройте папку и\u00A0выберите файл."}</p>}
        </section>
      </div>

      <section className="term-board">
        <div className="section-title">
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
        <p className="hint">Откат с этой машины: agentsync revert {page.agent}</p>
      </section>
    </article>
  );
}

function commandText(author, agent) {
  return `agentsync apply ${author} ${agent}`;
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

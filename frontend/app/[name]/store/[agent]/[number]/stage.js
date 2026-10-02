"use client";

import { useEffect, useState } from "react";
import { fileDownloadName, isMarkdown, skillBundle, skillZip } from "./pack";
import MarkdownView from "./preview";
import styles from "./stage.module.css";

export default function CodeStage({ files, current, onSelect }) {
  const [mode, setMode] = useState("preview");
  const [copied, setCopied] = useState(false);
  const [note, setNote] = useState("");
  const markdown = Boolean(current && isMarkdown(current.path));
  const skill = current ? skillBundle(files, current.path) : null;
  const showPreview = markdown && mode === "preview";

  useEffect(() => {
    setCopied(false);
    setNote("");
  }, [current?.path]);

  async function copyBody() {
    if (!current) {
      return;
    }
    try {
      await navigator.clipboard.writeText(current.body || "");
      setCopied(true);
      setNote("");
    } catch {
      setCopied(false);
      setNote("Не удалось скопировать.");
    }
  }

  function downloadFile() {
    if (!current) {
      return;
    }
    const type = markdown ? "text/markdown;charset=utf-8" : "text/plain;charset=utf-8";
    saveDownload(new TextEncoder().encode(current.body || ""), fileDownloadName(current.path), type, setNote);
  }

  function downloadSkill() {
    if (!skill) {
      return;
    }
    try {
      saveDownload(skillZip(skill), skill.zipName, "application/zip", setNote);
    } catch {
      setNote("Не удалось скачать.");
    }
  }

  return (
    <div className="code-stage">
      <div className={`side-head ${styles.head}`}>
        <strong className={`mono ${styles.path}`}>{current ? current.path : "Файл не выбран"}</strong>
        {current ? (
          <div className={styles.actions}>
            {markdown ? (
              <div className={styles.mode} role="group" aria-label="Вид файла">
                <button type="button" aria-pressed={mode === "preview"} onClick={() => setMode("preview")}>
                  Просмотр
                </button>
                <button type="button" aria-pressed={mode === "source"} onClick={() => setMode("source")}>
                  Исходник
                </button>
              </div>
            ) : null}
            {markdown ? (
              <button type="button" onClick={copyBody} aria-label={copied ? "Скопировано" : `Скопировать ${current.path}`}>
                {copied ? "Скопировано" : "Копировать"}
              </button>
            ) : null}
            <button type="button" onClick={downloadFile} aria-label={`Скачать ${current.path}`}>
              Скачать
            </button>
            {skill ? (
              <button type="button" onClick={downloadSkill} aria-label={`Скачать навык ${skill.name}`}>
                Скачать навык
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
      {note ? <p className="explanation">{note}</p> : null}
      {current ? (
        showPreview ? (
          <MarkdownView text={current.body || ""} files={files} currentPath={current.path} onSelect={onSelect} styles={styles} />
        ) : (
          <pre>{current.body}</pre>
        )
      ) : (
        <p className="hint tree-empty">{"Откройте папку и\u00A0выберите файл."}</p>
      )}
    </div>
  );
}

function saveDownload(bytes, name, type, setNote) {
  try {
    saveBytes(bytes, name, type);
    setNote("");
  } catch {
    setNote("Не удалось скачать.");
  }
}

function saveBytes(bytes, name, type) {
  const blob = new Blob([bytes], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1500);
}

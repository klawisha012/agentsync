"use client";

import { useState } from "react";

export function Command({ code }) {
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setError(false);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setError(true);
    }
  }

  return (
    <div className="docs-command-wrap">
      <div className="docs-command">
        <pre><code>{code}</code></pre>
        <button
          className="docs-copy"
          type="button"
          onClick={copy}
          aria-label={copied ? "Команда скопирована" : "Копировать команду"}
          title={copied ? "Скопировано" : "Копировать команду"}
        >
          {copied ? <Check /> : <Copy />}
        </button>
      </div>
      <span className="docs-copy-status" role="status">{error ? "Не удалось скопировать. Выделите команду вручную." : copied ? "Скопировано" : ""}</span>
    </div>
  );
}

function Copy() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="9" y="9" width="13" height="13" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  );
}

function Check() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M20 6 9 17l-5-5" />
    </svg>
  );
}

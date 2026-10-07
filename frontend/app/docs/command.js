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
      <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
      <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
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

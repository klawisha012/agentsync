"use client";

import { useEffect, useState } from "react";
import { api } from "../../../../api";

export default function SkillCommand({ author, agent, version, skills, picked }) {
  const [copied, setCopied] = useState(false);
  const [note, setNote] = useState("");
  const [place, setPlace] = useState("global");
  const [query, setQuery] = useState("");
  const [receivers, setReceivers] = useState([]);
  const [selected, setSelected] = useState(() => new Set());
  const clashes = duplicateSkillNames(skills, picked);
  const shown = visibleReceivers(receivers, place, query);
  const chosen = skills.filter((item) => picked.has(item.path));
  const command = clashes.length > 0 ? "" : skillCommand(author, agent, version, chosen, [...selected], place);

  useEffect(() => {
    let gone = false;
    api("/receivers").then((res) => {
      if (!gone && res.ok) {
        setReceivers(res.body.receivers || []);
      }
    });
    return () => {
      gone = true;
    };
  }, []);

  async function copyCommand() {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      setNote("");
    } catch {
      setCopied(false);
      setNote("Не удалось скопировать команду.");
    }
  }

  if (skills.length === 0) {
    return <p className="hint skill-empty">{"В\u00A0этой версии нет навыков"}</p>;
  }

  return (
    <div className="skill-pick">
      <div className="skill-command">
        <div className="place-switch" role="group" aria-label="Место установки">
          <button type="button" aria-pressed={place === "global"} onClick={() => choosePlace("global")}>Во все проекты</button>
          <button type="button" aria-pressed={place === "project"} onClick={() => choosePlace("project")}>В этот проект</button>
        </div>
        <input
          aria-label="Поиск приёмника"
          placeholder="Поиск приёмника"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <ul className="receiver-list">
          {shown.map((item) => (
            <li key={item.slug}>
              <label>
                <input
                  type="checkbox"
                  checked={selected.has(item.slug)}
                  onChange={() => toggleReceiver(item.slug)}
                />
                <span>{item.display}</span>
              </label>
            </li>
          ))}
        </ul>
        {command ? (
          <div className="terminal-line">
            <code>{command}</code>
            <button type="button" onClick={copyCommand}>{copied ? "Скопировано" : "Скопировать"}</button>
          </div>
        ) : clashes.length === 0 ? (
          <p className="hint">{chosen.length === 0 ? "Отметьте навыки в\u00A0списке файлов" : "Отметьте приёмника"}</p>
        ) : null}
      </div>
      {clashes.length > 0 ? <p className="explanation">{clashText(clashes)}</p> : null}
      {note ? <p className="explanation">{note}</p> : null}
    </div>
  );

  function choosePlace(next) {
    setCopied(false);
    setPlace(next);
    setSelected((current) => {
      const allowed = new Set(visibleReceivers(receivers, next, "").map((item) => item.slug));
      return new Set([...current].filter((slug) => allowed.has(slug)));
    });
  }

  function toggleReceiver(slug) {
    setCopied(false);
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(slug)) {
        next.delete(slug);
      } else {
        next.add(slug);
      }
      return next;
    });
  }
}

export function listSkillChoices(nodes) {
  const found = [];
  const walk = (list) => {
    for (const node of list || []) {
      if (node.skill && node.path) {
        found.push({ path: node.path, name: node.name });
      }
      walk(node.children);
    }
  };
  walk(nodes);
  const counts = new Map();
  for (const item of found) {
    const key = item.name.toLocaleLowerCase("ru");
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  return found
    .map((item) => ({
      path: item.path,
      label: (counts.get(item.name.toLocaleLowerCase("ru")) || 0) > 1 ? item.path : item.name,
    }))
    .sort((a, b) => a.label.localeCompare(b.label, "ru", { sensitivity: "base" }));
}

export function skillParentPaths(nodes) {
  const paths = new Set();
  const walk = (list) => {
    for (const node of list || []) {
      if (node.skill && node.path.includes("/")) {
        const parts = node.path.split("/");
        for (let i = 1; i < parts.length; i++) {
          paths.add(parts.slice(0, i).join("/"));
        }
      }
      walk(node.children);
    }
  };
  walk(nodes);
  return paths;
}

export function visibleReceivers(list, place, query) {
  const needle = String(query || "").trim().toLocaleLowerCase("ru");
  return (list || []).filter((item) => {
    if (place === "project" ? !item.project : !item.global) {
      return false;
    }
    if (needle === "") {
      return true;
    }
    return item.display.toLocaleLowerCase("ru").includes(needle) || item.slug.toLocaleLowerCase("ru").includes(needle);
  });
}

export function duplicateSkillNames(skills, picked) {
  const grouped = new Map();
  for (const skill of skills || []) {
    if (!picked.has(skill.path)) {
      continue;
    }
    const base = skill.path.split("/").pop();
    const paths = grouped.get(base) || [];
    paths.push(skill.path);
    grouped.set(base, paths);
  }
  const clashes = [];
  for (const paths of grouped.values()) {
    if (paths.length > 1) {
      clashes.push(...paths);
    }
  }
  return clashes;
}

function clashText(paths) {
  return `Навыки ${paths.map((item) => `«${item}»`).join(" и ")} называются одинаково. Оставьте один.`;
}

function skillCommand(author, source, version, skills, targets, place) {
  if (skills.length === 0 || targets.length === 0) {
    return "";
  }
  const parts = [
    "agentsync",
    "skills",
    shellArg(author),
    shellArg(source),
    "--version",
    String(version),
    place === "project" ? "--project" : "--global",
  ];
  for (const target of targets) {
    parts.push("--into", shellArg(target));
  }
  for (const skill of skills) {
    parts.push(shellArg(skill.label || skill.path));
  }
  return parts.join(" ");
}

function shellArg(value) {
  const text = String(value);
  if (/^[A-Za-z0-9._@+:/-]+$/.test(text)) {
    return text;
  }
  if (!text.includes('"')) {
    return `"${text}"`;
  }
  if (!text.includes("'")) {
    return `'${text}'`;
  }
  return `"${text.replaceAll('"', '\\"')}"`;
}

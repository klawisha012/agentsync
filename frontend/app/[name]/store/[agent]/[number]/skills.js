"use client";

import { useState } from "react";

const knownHomes = [
  { id: "Grok", label: "Grok" },
  { id: "Agents", label: "Agents" },
  { id: "Claude", label: "Claude" },
];

export default function SkillCommand({ author, agent, version, skills, picked, target, onTarget }) {
  const [copied, setCopied] = useState(false);
  const [note, setNote] = useState("");
  const homes = homeOptions(agent);
  const chosen = skills.filter((item) => picked.has(item.path));
  const command = skillCommand(author, agent, version, chosen, target);

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
    return <p className="hint">{"В\u00A0этой версии нет навыков"}</p>;
  }

  return (
    <div className="skill-pick">
      <div className="skill-command">
        <select
          aria-label="ИИ-агент"
          value={target}
          onChange={(event) => {
            setCopied(false);
            onTarget(event.target.value);
          }}
        >
          {homes.map((item) => (
            <option key={item.id} value={item.id}>{item.label}</option>
          ))}
        </select>
        {command ? (
          <div className="terminal-line">
            <code>{command}</code>
            <button type="button" onClick={copyCommand}>{copied ? "Скопировано" : "Скопировать"}</button>
          </div>
        ) : (
          <p className="hint">{"Отметьте навыки в\u00A0списке файлов"}</p>
        )}
      </div>
      {note ? <p className="explanation">{note}</p> : null}
    </div>
  );
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

export function defaultHome(agent) {
  const name = String(agent || "");
  const known = knownHomes.find((item) => item.id.toLowerCase() === name.toLowerCase());
  if (known) {
    return known.id;
  }
  return name || knownHomes[0].id;
}

function homeOptions(agent) {
  const options = knownHomes.map((item) => ({ ...item }));
  const name = String(agent || "");
  const known = options.some((item) => item.id.toLowerCase() === name.toLowerCase());
  if (name !== "" && !known) {
    options.push({ id: name, label: name });
  }
  return options;
}

function skillCommand(author, source, version, skills, home) {
  if (skills.length === 0 || !home) {
    return "";
  }
  const parts = ["agentsync", "skills", shellArg(author), shellArg(source), "--version", String(version)];
  for (const skill of skills) {
    parts.push("--skill", shellArg(skill.path));
  }
  parts.push("--into", shellArg(home));
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

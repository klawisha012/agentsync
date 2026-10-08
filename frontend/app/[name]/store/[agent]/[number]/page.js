"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../../../api";
import FileTree from "../../../../file-tree";
import { buildManifest, loadTokenCounter, ruTokens, skillTokenTotal } from "../../../../publications/[id]/manifest";
import CodeStage from "./stage";
import SkillCommand, { listSkillChoices, skillParentPaths } from "./skills";

export default function VersionPage() {
  const params = useParams();
  const name = String(params.name || "");
  const agent = String(params.agent || "");
  const number = String(params.number || "");
  const [state, setState] = useState(null);
  const [selected, setSelected] = useState("");
  const [open, setOpen] = useState(() => new Set());
  const [picked, setPicked] = useState(() => new Set());
  const [selectionMode, setSelectionMode] = useState(false);
  const [countTokens, setCountTokens] = useState(null);

  useEffect(() => {
    let gone = false;
    loadTokenCounter().then((count) => {
      if (!gone) {
        setCountTokens(() => count);
      }
    });
    return () => {
      gone = true;
    };
  }, []);

  useEffect(() => {
    let gone = false;
    setState(null);
    setSelected("");
    setOpen(new Set());
    setPicked(new Set());
    setSelectionMode(false);
    api(`/accounts/${encodeURIComponent(name)}/versions/${encodeURIComponent(agent)}/${encodeURIComponent(number)}`).then((res) => {
      if (gone) {
        return;
      }
      setState(res);
      if (res.ok) {
        setOpen(skillParentPaths(buildManifest(res.body.files || [])));
      }
    });
    return () => {
      gone = true;
    };
  }, [name, agent, number]);

  const tree = state?.ok ? buildManifest(state.body.files || [], countTokens) : [];
  const skills = listSkillChoices(tree);

  if (!state) {
    return <p className="lede">Открываем версию…</p>;
  }
  if (!state.ok) {
    return <p className="explanation">{state.body?.explanation || "Версия не найдена."}</p>;
  }

  const shot = state.body;
  const files = shot.files || [];
  const current = files.find((file) => file.path === selected)
    || (selected === "" ? files.find((file) => !String(file.path).includes("/")) : null)
    || null;

  function toggleSkill(path) {
    setPicked((currentSet) => {
      const next = new Set(currentSet);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }

  function toggleSkills(paths, checked) {
    setPicked((currentSet) => {
      const next = new Set(currentSet);
      for (const path of paths) {
        if (checked) {
          next.add(path);
        } else {
          next.delete(path);
        }
      }
      return next;
    });
  }

  function toggleFolder(path) {
    setOpen((prev) => {
      const next = new Set(prev);
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
        <div className="crumbs">
          <nav aria-label="Путь">
            <Link href={`/${name}`}>{name}</Link>
            <span>/</span>
            <Link href={`/${name}/store/${encodeURIComponent(agent)}`}>версии</Link>
            <span>/</span>
            <b>v{shot.version}</b>
          </nav>
          {selectionMode || picked.size > 0 ? (
            <SkillCommand
              author={name}
              agent={agent}
              version={shot.version}
              skills={skills}
              picked={picked}
            />
          ) : null}
        </div>
      </div>
      <div className="preview-grid">
        <div className="manifest">
          <div className="side-head side-head-files">
            <div className="side-head-title">
              <h2>Файлы версии</h2>
              {skills.length > 0 ? (
                <label className="skill-toggle" htmlFor="skill-selection-mode">
                  <span>Выделение</span>
                  <SelectionSwitch checked={selectionMode} onChange={setSelectionMode} />
                </label>
              ) : null}
            </div>
            <span className="count-pill">{countTokens ? ruTokens(skillTokenTotal(tree)) : "…"}</span>
          </div>
          <div className="file-tree" role="tree" aria-label="Файлы версии">
            <FileTree
              nodes={tree}
              depth={0}
              open={open}
              onToggle={toggleFolder}
              selected={current?.path || ""}
              onSelect={setSelected}
              picked={picked}
              onToggleSkill={selectionMode ? toggleSkill : undefined}
              onToggleSkills={selectionMode ? toggleSkills : undefined}
            />
          </div>
          <p className="hint">{stamp(shot.created)} · манифест v{shot.version}</p>
        </div>
        <CodeStage files={files} current={current} onSelect={setSelected} />
      </div>
    </article>
  );
}

function SelectionSwitch({ checked, onChange }) {
  return (
    <button
      id="skill-selection-mode"
      type="button"
      role="switch"
      className="skill-switch"
      aria-checked={checked}
      aria-label="Режим выделения навыков"
      onClick={() => onChange(!checked)}
    >
      <span className="skill-switch-thumb" />
    </button>
  );
}

function stamp(value) {
  if (!value) {
    return "";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  return new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(parsed);
}

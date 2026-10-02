"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { api } from "../../../../api";
import FileTree from "../../../../file-tree";
import { buildManifest, ruTokens, skillTokenTotal } from "../../../../publications/[id]/manifest";
import CodeStage from "./stage";
import SkillCommand, { defaultHome, listSkillChoices, skillParentPaths } from "./skills";

export default function VersionPage() {
  const params = useParams();
  const name = String(params.name || "");
  const agent = String(params.agent || "");
  const number = String(params.number || "");
  const [state, setState] = useState(null);
  const [selected, setSelected] = useState("");
  const [open, setOpen] = useState(() => new Set());
  const [picked, setPicked] = useState(() => new Set());
  const [target, setTarget] = useState("");
  const allRef = useRef(null);

  useEffect(() => {
    let gone = false;
    setState(null);
    setSelected("");
    setOpen(new Set());
    setPicked(new Set());
    setTarget(defaultHome(agent));
    api(`/accounts/${encodeURIComponent(name)}/versions/${encodeURIComponent(agent)}/${encodeURIComponent(number)}`).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [name, agent, number]);

  const tree = state?.ok ? buildManifest(state.body.files || []) : [];
  const skills = listSkillChoices(tree);
  const allOn = skills.length > 0 && skills.every((item) => picked.has(item.path));

  useEffect(() => {
    if (!state?.ok) {
      return;
    }
    setOpen(skillParentPaths(buildManifest(state.body.files || [])));
  }, [state]);

  useEffect(() => {
    if (allRef.current) {
      allRef.current.indeterminate = picked.size > 0 && !allOn;
    }
  }, [picked, allOn]);

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
          <SkillCommand
            author={name}
            agent={agent}
            version={shot.version}
            skills={skills}
            picked={picked}
            target={target}
            onTarget={setTarget}
          />
        </div>
      </div>
      <div className="preview-grid">
        <div className="manifest">
          <div className="side-head">
            <h2>Файлы версии</h2>
            <span className="count-pill">{ruTokens(skillTokenTotal(tree))}</span>
          </div>
          <div className="file-tree" role="tree" aria-label="Файлы версии">
            {skills.length > 0 ? (
              <label className="tree-all">
                <input
                  ref={allRef}
                  type="checkbox"
                  checked={allOn}
                  onChange={(event) => {
                    setPicked(event.target.checked ? new Set(skills.map((item) => item.path)) : new Set());
                  }}
                />
                <span>{allOn ? "Убрать выделение" : "Выделить все"}</span>
              </label>
            ) : null}
            <FileTree
              nodes={tree}
              depth={0}
              open={open}
              onToggle={toggleFolder}
              selected={current?.path || ""}
              onSelect={setSelected}
              picked={picked}
              onToggleSkill={toggleSkill}
            />
          </div>
          <p className="hint">{stamp(shot.created)} · манифест v{shot.version}</p>
        </div>
        <CodeStage files={files} current={current} onSelect={setSelected} />
      </div>
    </article>
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

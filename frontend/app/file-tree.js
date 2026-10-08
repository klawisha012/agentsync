"use client";

import FileIcon from "./file-icons";
import { formatCount } from "./publications/[id]/manifest";

function selectablePathsUnder(node) {
  const paths = [];
  const walk = (item) => {
    if (item.skill && item.path) {
      paths.push(item.path);
    }
    if (item.file && item.file.path) {
      paths.push(item.file.path);
    }
    for (const child of item.children || []) {
      walk(child);
    }
  };
  walk(node);
  return paths;
}

export default function FileTree({ nodes, depth, open, onToggle, selected, onSelect, picked, onToggleSkill, onToggleSkills }) {
  return nodes.map((node) => {
    const pad = { paddingLeft: `${0.35 + depth * 0.9}rem` };
    if (node.children.length > 0) {
      const expanded = open.has(node.path);
      const checkable = Boolean(node.skill && onToggleSkill);
      const folderSkills = onToggleSkills ? selectablePathsUnder(node) : [];
      const folderCheckable = !checkable && folderSkills.length > 0;
      const folderAll = folderCheckable && folderSkills.every((path) => picked?.has(path));
      const folderSome = folderCheckable && !folderAll && folderSkills.some((path) => picked?.has(path));
      const lined = checkable || folderCheckable;
      return (
        <div key={node.path} className="tree-branch" role="treeitem" aria-expanded={expanded}>
          <div className={lined ? "tree-line" : undefined} style={lined ? pad : undefined}>
            {checkable ? (
              <input
                className="tree-check"
                type="checkbox"
                checked={picked?.has(node.path) || false}
                aria-label={`Навык ${node.name}`}
                onChange={() => onToggleSkill(node.path)}
              />
            ) : null}
            {folderCheckable ? (
              <input
                className="tree-check"
                type="checkbox"
                checked={folderAll}
                ref={(el) => {
                  if (el) {
                    el.indeterminate = folderSome;
                  }
                }}
                aria-label={`Папка ${node.name} со всеми вложениями`}
                onChange={(event) => onToggleSkills(folderSkills, event.target.checked)}
              />
            ) : null}
            <button
              type="button"
              className="tree-dir"
              style={lined ? undefined : pad}
              aria-expanded={expanded}
              onClick={() => onToggle(node.path)}
            >
              <span className="tree-name">
                <i className="tree-mark" aria-hidden="true">{expanded ? "▾" : "▸"}</i>
                <FileIcon kind="folder" />
                {node.name}
              </span>
              {node.skill ? (
                <em className="skill-tokens">
                  {node.skill.description == null
                    ? "считаем токены…"
                    : `описание ${formatCount(node.skill.description)} · целиком ${formatCount(node.skill.content)}`}
                </em>
              ) : null}
            </button>
          </div>
          {expanded ? (
            <div role="group">
              <FileTree
                nodes={node.children}
                depth={depth + 1}
                open={open}
                onToggle={onToggle}
                selected={selected}
                onSelect={onSelect}
                picked={picked}
                onToggleSkill={onToggleSkill}
                onToggleSkills={onToggleSkills}
              />
            </div>
          ) : null}
        </div>
      );
    }
    const fileCheckable = depth === 0 && Boolean(onToggleSkills);
    if (fileCheckable) {
      return (
        <div key={node.path} className="tree-line" style={pad} role="treeitem">
          <input
            className="tree-check"
            type="checkbox"
            checked={picked?.has(node.file.path) || false}
            aria-label={`Файл ${node.name}`}
            onChange={(event) => onToggleSkills([node.file.path], event.target.checked)}
          />
          <button
            type="button"
            className="tree-file tree-file-lined"
            aria-pressed={selected === node.file.path}
            onClick={() => onSelect(node.file.path)}
          >
            <span className="tree-name">
              <i className="tree-mark" aria-hidden="true" />
              <FileIcon name={node.name} />
              {node.name}
            </span>
          </button>
        </div>
      );
    }
    return (
      <button
        key={node.path}
        type="button"
        role="treeitem"
        className="tree-file"
        style={pad}
        aria-pressed={selected === node.file.path}
        onClick={() => onSelect(node.file.path)}
      >
        <span className="tree-name">
          <i className="tree-mark" aria-hidden="true" />
          <FileIcon name={node.name} />
          {node.name}
        </span>
      </button>
    );
  });
}

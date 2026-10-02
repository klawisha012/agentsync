"use client";

import { formatCount } from "./publications/[id]/manifest";

export default function FileTree({ nodes, depth, open, onToggle, selected, onSelect, picked, onToggleSkill }) {
  return nodes.map((node) => {
    const pad = { paddingLeft: `${0.35 + depth * 0.9}rem` };
    if (node.children.length > 0) {
      const expanded = open.has(node.path);
      const checkable = Boolean(node.skill && onToggleSkill);
      return (
        <div key={node.path} className="tree-branch" role="treeitem" aria-expanded={expanded}>
          <div className={checkable ? "tree-line" : undefined} style={checkable ? pad : undefined}>
            {checkable ? (
              <input
                className="tree-check"
                type="checkbox"
                checked={picked?.has(node.path) || false}
                aria-label={`Навык ${node.name}`}
                onChange={() => onToggleSkill(node.path)}
              />
            ) : null}
            <button
              type="button"
              className="tree-dir"
              style={checkable ? undefined : pad}
              aria-expanded={expanded}
              onClick={() => onToggle(node.path)}
            >
              <span className="tree-name">
                <i className="tree-mark" aria-hidden="true">{expanded ? "▾" : "▸"}</i>
                {node.name}
              </span>
              {node.skill ? (
                <em className="skill-tokens">
                  описание {formatCount(node.skill.description)} · целиком {formatCount(node.skill.content)}
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
              />
            </div>
          ) : null}
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
          {node.name}
        </span>
      </button>
    );
  });
}

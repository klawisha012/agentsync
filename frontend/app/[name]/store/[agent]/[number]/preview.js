"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

export default function MarkdownView({ text, files, currentPath, onSelect, styles }) {
  return (
    <div className={styles.preview}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: (props) => (
            <MdLink {...props} files={files} currentPath={currentPath} onSelect={onSelect} styles={styles} />
          ),
          img: MdImage,
          table: (props) => <MdTable {...props} styles={styles} />,
        }}
      >
        {stripFrontmatter(text)}
      </ReactMarkdown>
    </div>
  );
}

function MdLink({ href, children, files, currentPath, onSelect, styles, node, ...rest }) {
  const external = href && /^(https?:|mailto:)/i.test(href);
  if (external) {
    return (
      <a href={href} target="_blank" rel="noreferrer" {...rest}>
        {children}
      </a>
    );
  }
  const target = resolveDocPath(currentPath, href);
  if (target && (files || []).some((file) => normalize(file.path) === target)) {
    return (
      <button type="button" className={styles.file} onClick={() => onSelect(target)}>
        {children}
      </button>
    );
  }
  if (href && href.startsWith("#")) {
    return <a href={href}>{children}</a>;
  }
  return <span>{children}</span>;
}

function MdImage({ src, alt }) {
  if (!src || !/^https?:/i.test(src)) {
    return null;
  }
  return <img src={src} alt={alt || ""} referrerPolicy="no-referrer" />;
}

function MdTable({ children, styles }) {
  return (
    <div className={styles.table}>
      <table>{children}</table>
    </div>
  );
}

export function stripFrontmatter(text) {
  const raw = String(text || "").replace(/^\uFEFF/, "");
  const lines = raw.split(/\r?\n/);
  if (lines[0].trim() !== "---") {
    return raw;
  }
  for (let i = 1; i < lines.length; i++) {
    if (lines[i].trim() === "---") {
      return lines.slice(i + 1).join("\n").replace(/^\n/, "");
    }
  }
  return raw;
}

export function resolveDocPath(fromFile, href) {
  if (!href || /^(https?:|mailto:|data:)/i.test(href) || href.startsWith("#") || href.startsWith("/")) {
    return "";
  }
  const clean = href.split("#")[0].split("?")[0];
  if (!clean || clean.startsWith("//")) {
    return "";
  }
  const parts = normalize(fromFile).split("/").slice(0, -1);
  for (const seg of clean.split("/")) {
    if (seg === "" || seg === ".") {
      continue;
    }
    if (seg === "..") {
      if (parts.length === 0) {
        return "";
      }
      parts.pop();
      continue;
    }
    parts.push(seg);
  }
  return parts.join("/");
}

function normalize(path) {
  return String(path || "").replaceAll("\\", "/");
}

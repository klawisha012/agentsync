// Иконки типов файлов в духе VS Code Material Icon Theme.
// Плоские SVG 16x16, цвета близки к оригинальному набору.

const ICONS = {
  markdown: {
    color: "#42a5f5",
    path: "M2 2h12a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z",
    extra: "M3 10.5v-5h1.5l1.5 2 1.5-2H9v5H7.5V7.7L6 9.7 4.5 7.7v2.8H3zm8-5h1.5v3h1.2L12 11.5l-1.7-3H11V5.5z",
    extraColor: "#ffffff",
  },
  json: {
    color: "#fbc02d",
    path: "M4.5 3C3.7 3 3 3.7 3 4.5v2C3 7.3 2.3 8 1.5 8H1v.5h.5c.8 0 1.5.7 1.5 1.5v2c0 .8.7 1.5 1.5 1.5H5v-1h-.5c-.3 0-.5-.2-.5-.5v-2c0-.6-.4-1.2-1-1.4.6-.2 1-.8 1-1.4v-2c0-.3.2-.5.5-.5H5V3h-.5zm7 0c.8 0 1.5.7 1.5 1.5v2c0 .8.7 1.5 1.5 1.5h.5v.5h-.5c-.8 0-1.5.7-1.5 1.5v2c0 .8-.7 1.5-1.5 1.5H11v-1h.5c.3 0 .5-.2.5-.5v-2c0-.6.4-1.2 1-1.4-.6-.2-1-.8-1-1.4v-2c0-.3-.2-.5-.5-.5H11V3h.5z",
  },
  javascript: {
    color: "#ffca28",
    path: "M2 2h12v12H2V2zm7.5 9.4c.3.5.7.9 1.4.9.6 0 1-.3 1-.7 0-.5-.4-.7-1-1l-.4-.1c-1-.3-1.7-.7-1.7-1.6 0-.8.6-1.4 1.6-1.4.7 0 1.2.2 1.5.8l-.8.5c-.2-.3-.4-.4-.7-.4-.3 0-.5.2-.5.4 0 .3.2.4.6.6l.4.2c1.2.5 1.9 1 1.9 2.1 0 1.2-.9 1.9-2.2 1.9-1.2 0-2-.6-2.4-1.4l.9-.5zM4 9.5c.2.4.4.7.9.7.4 0 .7-.2.7-.8V7.6h1.1v1.9c0 1.1-.7 1.7-1.7 1.7-.9 0-1.4-.5-1.7-1l.7-.7z",
  },
  typescript: {
    color: "#0288d1",
    path: "M2 2h12v12H2V2zm6.5 9.4c.3.5.7.9 1.4.9.6 0 1-.3 1-.7 0-.5-.4-.7-1-1l-.4-.1c-1-.3-1.7-.7-1.7-1.6 0-.8.6-1.4 1.6-1.4.7 0 1.2.2 1.5.8l-.8.5c-.2-.3-.4-.4-.7-.4-.3 0-.5.2-.5.4 0 .3.2.4.6.6l.4.2c1.2.5 1.9 1 1.9 2.1 0 1.2-.9 1.9-2.2 1.9-1.2 0-2-.6-2.4-1.4l.9-.5zM7.9 8.4V7.5H4.2v.9h1.3v4.7h1.1V8.4h1.3z",
  },
  css: {
    color: "#42a5f5",
    path: "M2 2l1 11 5 1.5L13 13l1-11H2zm8.6 4H5.2l.2 1.6h5l-.3 2.4-2.1.6-2.1-.6-.1-1.1H4.6l.2 2.2 3.2 1 3.2-1 .4-3.6H5.9L5.7 6h5.1l-.2 0z",
  },
  html: {
    color: "#ff7043",
    path: "M2 2l1 11 5 1.5L13 13l1-11H2zm8.4 3.5H5.4l.2 1.6h4.6l-.3 2.9-1.9.5-1.9-.5-.1-1.1h1.1l.1.5 1 .3 1-.3.1-.9H5.9L5.5 4.4h5.1l-.2 1.1z",
  },
  python: {
    color: "#4b8bbe",
    path: "M8 2C5.8 2 5.6 3 5.6 3v1.6h2.6v.5H4s-2 .2-2 3.3 1.7 3.2 1.7 3.2h1.3V10s-.1-1.9 1.8-1.9h3s1.7 0 1.7-1.6V3.6S11.7 2 8 2zM6.7 3.4a.6.6 0 1 1 0 1.2.6.6 0 0 1 0-1.2zM8 14c2.2 0 2.4-1 2.4-1v-1.6H7.8v-.5H12s2-.2 2-3.3-1.7-3.2-1.7-3.2h-1.3V6s.1 1.9-1.8 1.9h-3S4.5 7.9 4.5 9.5v2.9S4.3 14 8 14zm1.3-1.4a.6.6 0 1 1 0-1.2.6.6 0 0 1 0 1.2z",
  },
  image: {
    color: "#26a69a",
    path: "M2 3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V4a1 1 0 0 0-1-1H2zm1 2h10v5.3l-2.3-2.3-2.4 2.4-1.5-1.5L3 12.7V5zm2 .2a1.2 1.2 0 1 0 0 2.4 1.2 1.2 0 0 0 0-2.4z",
  },
  lock: {
    color: "#ffb300",
    path: "M8 2a3 3 0 0 0-3 3v2H4a1 1 0 0 0-1 1v5a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V8a1 1 0 0 0-1-1h-1V5a3 3 0 0 0-3-3zm0 1.5A1.5 1.5 0 0 1 9.5 5v2h-3V5A1.5 1.5 0 0 1 8 3.5z",
  },
  text: {
    color: "#90a4ae",
    path: "M3 2h10a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1zm1 3v1h8V5H4zm0 2.5v1h8v-1H4zm0 2.5v1h5v-1H4z",
  },
  yaml: {
    color: "#ef5350",
    path: "M3 2h10a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1zm1.5 3L6 7.3 7.5 5h-3zM6 8v4.5h1V8H6zm3-2.5h4v1H9v-1zm0 2h4v1H9v-1zm0 2h4v1H9v-1z",
  },
  shell: {
    color: "#8bc34a",
    path: "M2 3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V4a1 1 0 0 0-1-1H2zm2 2.5L7 8l-3 2.5v-1.2L5.7 8 4 6.7V5.5zM8 10.5h4v1H8v-1z",
  },
  git: {
    color: "#f4511e",
    path: "M13.6 7.4L8.6 2.4a.9.9 0 0 0-1.2 0L6.2 3.6l1.2 1.2a1 1 0 0 1 1.3 1.3l1.2 1.2a1 1 0 1 1-.6.6L8.2 6.8v2.9a1 1 0 1 1-.8 0V6.7a1 1 0 0 1-.6-1.4L5.6 4.1 2.4 7.4a.9.9 0 0 0 0 1.2l5 5a.9.9 0 0 0 1.2 0l5-5a.9.9 0 0 0 0-1.2z",
  },
  docker: {
    color: "#ec407a",
    path: "M1 10.5c.1 1.8 1.5 3.3 3.4 3.5 2.9.3 5.7.2 7.9-1.2 1.2-.7 2-1.7 2.5-3 .4 0 1-.1 1.2-.6-.3-.2-.8-.3-1.2-.3-.1-.6-.5-1.3-1.2-1.6l-.3-.1-.1.3c-.2.5-.2 1.2.1 1.7-.2.1-.6.2-1.1.2H1.1l-.1.1zm1.2-1.5h1.6V7.4H2.2V9zm1.9 0h1.6V7.4H4.1V9zm1.9 0h1.6V7.4H6V9zm1.9 0h1.6V7.4H7.9V9zM4.1 7h1.6V5.4H4.1V7zm1.9 0H7.6V5.4H6V7zm1.9 0h1.6V5.4H7.9V7zm0-2.3h1.6V3.2H7.9v1.5z",
  },
  folder: {
    color: "#90a4ae",
    path: "M2 3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V6a1 1 0 0 0-1-1H8L6.7 3H2z",
  },
  file: {
    color: "#78909c",
    path: "M4 2h5l4 4v7a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1zm4.5 1.5v3h3l-3-3z",
  },
};

const BY_EXTENSION = {
  md: "markdown",
  mdx: "markdown",
  markdown: "markdown",
  json: "json",
  jsonc: "json",
  js: "javascript",
  jsx: "javascript",
  mjs: "javascript",
  cjs: "javascript",
  ts: "typescript",
  tsx: "typescript",
  css: "css",
  scss: "css",
  less: "css",
  html: "html",
  htm: "html",
  py: "python",
  png: "image",
  jpg: "image",
  jpeg: "image",
  gif: "image",
  svg: "image",
  webp: "image",
  webm: "image",
  mp4: "image",
  txt: "text",
  yml: "yaml",
  yaml: "yaml",
  toml: "yaml",
  sh: "shell",
  bash: "shell",
  zsh: "shell",
};

const BY_NAME = {
  "agents.md": "markdown",
  "claude.md": "markdown",
  license: "text",
  dockerfile: "docker",
  "docker-compose.yml": "docker",
  "docker-compose.yaml": "docker",
  ".gitignore": "git",
  ".gitattributes": "git",
  ".env": "lock",
};

export function fileIconKind(name) {
  const lower = String(name || "").toLowerCase();
  if (BY_NAME[lower]) {
    return BY_NAME[lower];
  }
  const dot = lower.lastIndexOf(".");
  if (dot > 0) {
    const ext = lower.slice(dot + 1);
    if (BY_EXTENSION[ext]) {
      return BY_EXTENSION[ext];
    }
  }
  return "file";
}

export default function FileIcon({ name, kind, size = 14 }) {
  const icon = ICONS[kind || fileIconKind(name)] || ICONS.file;
  return (
    <svg
      className="file-icon"
      width={size}
      height={size}
      viewBox="0 0 16 16"
      aria-hidden="true"
      focusable="false"
    >
      <path d={icon.path} fill={icon.color} />
      {icon.extra ? <path d={icon.extra} fill={icon.extraColor || "#fff"} /> : null}
    </svg>
  );
}

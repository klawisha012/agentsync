import esbuild from "esbuild";
import { JSDOM } from "jsdom";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const frontend = path.resolve(import.meta.dirname, "..");
const scratch = process.env.VISUAL_SCRATCH
  || path.join(process.env.TEMP || "/tmp", "agentsync-visual");
fs.mkdirSync(scratch, { recursive: true });

const bundlePath = path.join(frontend, "test", "visual-bundle.mjs");
await esbuild.build({
  absWorkingDir: frontend,
  entryPoints: ["test/entry.js"],
  bundle: true,
  format: "esm",
  platform: "browser",
  outfile: bundlePath,
  jsx: "automatic",
  loader: { ".js": "jsx" },
  external: ["react", "react-dom", "react/jsx-runtime", "react/jsx-dev-runtime"],
  alias: {
    "next/link": path.join(frontend, "test/shims/link.js"),
    "next/navigation": path.join(frontend, "test/shims/navigation.js"),
  },
  define: {
    "process.env.NODE_ENV": '"test"',
  },
  plugins: [{
    name: "css-modules",
    setup(build) {
      build.onLoad({ filter: /\.module\.css$/ }, async (args) => {
        const css = await fs.promises.readFile(args.path, "utf8");
        const names = [...css.matchAll(/\.([_a-zA-Z][\w-]*)/g)].map((match) => match[1]);
        const fields = [...new Set(names)].map((name) => `${JSON.stringify(name)}:${JSON.stringify(name)}`).join(",");
        return { contents: `export default {${fields}};`, loader: "js" };
      });
    },
  }],
});
process.on("exit", () => {
  fs.rmSync(bundlePath, { force: true });
});

const dom = new JSDOM("<!doctype html><html><head></head><body><div id=\"root\"></div></body></html>", {
  url: "https://agentsync.local/",
  pretendToBeVisual: true,
});
const { window } = dom;
function expose(name, value) {
  Object.defineProperty(globalThis, name, { configurable: true, writable: true, value });
}
expose("window", window);
expose("document", window.document);
expose("HTMLElement", window.HTMLElement);
expose("SVGElement", window.SVGElement);
expose("Element", window.Element);
expose("Node", window.Node);
expose("DocumentFragment", window.DocumentFragment);
expose("getComputedStyle", window.getComputedStyle.bind(window));
expose("IS_REACT_ACT_ENVIRONMENT", true);
expose("requestAnimationFrame", (callback) => setTimeout(callback, 0));
expose("cancelAnimationFrame", (id) => clearTimeout(id));
expose("Event", window.Event);
expose("CustomEvent", window.CustomEvent);

const style = window.document.createElement("style");
style.textContent = fs.readFileSync(path.join(frontend, "app/globals.css"), "utf8");
window.document.head.appendChild(style);

const calls = [];
let routes = () => ({ status: 404, body: { explanation: "нет" } });
globalThis.fetch = async (url, options = {}) => {
  const method = String(options.method || "GET").toUpperCase();
  const target = String(url);
  calls.push({ url: target, method });
  const spec = routes(target, method);
  const text = spec.body == null ? "" : JSON.stringify(spec.body);
  return new Response(text, { status: spec.status, headers: { "Content-Type": "application/json" } });
};

const mod = await import(pathToFileURL(bundlePath).href);
const React = await import("react");
const { createElement, act } = React;
const { createRoot } = await import("react-dom/client");

const failures = [];
function check(name, ok, detail) {
  if (!ok) {
    failures.push(`${name}: ${detail}`);
  }
}

async function flush() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 30));
  });
}

function mount(node) {
  const host = window.document.createElement("div");
  window.document.body.appendChild(host);
  const root = createRoot(host);
  return { host, root, node };
}

async function render(node) {
  const view = mount(node);
  await act(async () => {
    view.root.render(view.node);
  });
  await flush();
  return view;
}

async function headerDOM(session) {
  calls.length = 0;
  mod.__setPathname("/");
  routes = (url, method) => {
    if (url.endsWith("/api/session") && method === "GET") {
      if (!session) {
        return { status: 401, body: null };
      }
      return { status: 200, body: session };
    }
    return { status: 404, body: { explanation: "нет" } };
  };
  const view = await render(createElement(mod.Shell, null, createElement("p", null, "страница")));
  return view.host.innerHTML;
}

function outlineIcon(svg, label) {
  if (!svg) {
    return `${label} отсутствует`;
  }
  const stroke = svg.getAttribute("stroke");
  const fill = svg.getAttribute("fill");
  const paths = [...svg.querySelectorAll("path")];
  const filledOnly = paths.length === 1 && paths[0].getAttribute("fill") === "currentColor" && !stroke;
  if (stroke !== "currentColor" || fill === "currentColor" || filledOnly) {
    return `${label} не контур: stroke=${stroke} fill=${fill} paths=${paths.length}`;
  }
  return "";
}

const runA = [];
const runB = [];
const runC = [];
for (let pass = 1; pass <= 2; pass += 1) {
  runA.push(await headerDOM({ name: "alice", avatarUpdated: "9" }));
  runB.push(await headerDOM({ name: "alice" }));
  runC.push(await headerDOM(null));
}

const headerLines = [];
function headerReport(title, html) {
  const holder = window.document.createElement("div");
  holder.innerHTML = html;
  const button = holder.querySelector("button.who");
  const photo = holder.querySelector("img.person-photo");
  const name = holder.querySelector(".who-name");
  const caret = holder.querySelector(".who-caret");
  const login = holder.querySelector("a.who");
  const person = holder.querySelector(".person");
  const radius = person ? window.getComputedStyle(person).borderRadius : "";
  headerLines.push(`${title}`);
  headerLines.push(`  button=${Boolean(button)} photo=${photo ? photo.getAttribute("src") : ""} name=${name ? name.textContent : ""} caret=${Boolean(caret)} login=${login ? login.getAttribute("aria-label") : ""} radius=${radius}`);
  return { button, photo, name, caret, login, person, radius, holder };
}

const a1 = headerReport("A1", runA[0]);
const a2 = headerReport("A2", runA[1]);
const b1 = headerReport("B1", runB[0]);
const b2 = headerReport("B2", runB[1]);
const c1 = headerReport("C1", runC[0]);
const c2 = headerReport("C2", runC[1]);

check("A повтор", runA[0] === runA[1], "DOM прогонов A различается");
check("B повтор", runB[0] === runB[1], "DOM прогонов B различается");
check("C повтор", runC[0] === runC[1], "DOM прогонов C различается");
check("A аватар", Boolean(a1.photo) && a1.name?.textContent === "alice" && a1.caret && a1.button, "нет единого контрола с фото, ником и шевроном");
check("A без отдельной ссылки", !a1.holder.querySelector("button.who a, .top-gap > a"), "ник вынесен отдельной ссылкой");
check("B силуэт", !b1.photo && b1.holder.querySelector(".person svg") && b1.name?.textContent === "alice", "нет силуэта или ника");
check("C вход", !c1.button && c1.login?.getAttribute("aria-label") === "Войти" && c1.holder.querySelector(".person svg"), "без сессии должна остаться только ссылка Войти");
check("рамка", a1.radius !== "999px" && a1.radius !== "50%" && a1.radius !== "", `радиус ${a1.radius}`);

calls.length = 0;
mod.__setPathname("/");
routes = (url, method) => {
  if (url.endsWith("/api/session") && method === "GET") {
    return { status: 200, body: { name: "alice", avatarUpdated: "9" } };
  }
  if (url.endsWith("/api/session") && method === "DELETE") {
    return { status: 200, body: {} };
  }
  return { status: 404, body: { explanation: "нет" } };
};
const logoutView = await render(createElement(mod.Shell, null, createElement("p", null, "страница")));
await act(async () => {
  logoutView.host.querySelector("button.who").click();
});
const menuText = logoutView.host.querySelector(".profile-pop")?.textContent || "";
headerLines.push(`menu=${menuText}`);
check("меню", menuText.includes("Открыть мой профиль") && menuText.includes("Настройки") && menuText.includes("Выйти из аккаунта…"), menuText);
check("без Остаться", !menuText.includes("Остаться"), menuText);
const leave = logoutView.host.querySelector(".menu-leave");
const leaveBg = leave ? window.getComputedStyle(leave).backgroundColor : "";
headerLines.push(`leave-background=${leaveBg}`);
check("выход без заливки", leaveBg === "rgba(0, 0, 0, 0)" || leaveBg === "transparent", leaveBg);
await act(async () => {
  leave.click();
});
await flush();
const deleted = calls.some((call) => call.method === "DELETE" && call.url.endsWith("/api/session"));
headerLines.push(`delete=${deleted} pushes=${mod.__takePushes().join(",")}`);
check("выход сразу", deleted && !logoutView.host.textContent.includes("Остаться"), "DELETE /session не вызван");

const account = {
  name: "alice",
  verified: true,
  views: 1280,
  likes: 14,
  avatarUpdated: "",
  shareCopy: true,
  shareView: true,
  shareVersions: true,
  agents: [{ name: "Grok", version: 4, files: 3, publishedAt: "2026-10-01T00:00:00Z", publicationId: "p1", likes: 2 }],
};
function accountRoutes(owner) {
  return (url) => {
    if (url.includes("/api/accounts/alice/comments")) {
      return { status: 200, body: { comments: [], page: 1, pages: 1 } };
    }
    if (url.endsWith("/api/accounts/alice")) {
      const body = owner
        ? { ...account, maskedMail: "a***@example.com", machines: [{ id: "m-1", host: "desk", listener: "127.0.0.1:9" }] }
        : account;
      return { status: 200, body };
    }
    return { status: 404, body: { explanation: "нет" } };
  };
}

routes = accountRoutes(false);
const publicView = await render(createElement(mod.AccountView, { name: "alice" }));
routes = accountRoutes(true);
const ownerView = await render(createElement(mod.AccountView, { name: "alice" }));
const profileHtml = ownerView.host.innerHTML;
fs.writeFileSync(path.join(scratch, "profile-render.html"), profileHtml, "utf8");

const ownerRoot = ownerView.host;
check("имя", ownerRoot.querySelector("h1")?.textContent === "alice", ownerRoot.querySelector("h1")?.textContent);
check("метрики", ownerRoot.textContent.includes("Просмотры") && ownerRoot.textContent.includes("Лайки"), "нет метрик");
check("карточка", Boolean(ownerRoot.querySelector(".house-card")), "нет карточки агента");
check("машина", Boolean(ownerRoot.querySelector(".machine-strip")) && !publicView.host.querySelector(".machine-strip"), "полоса машины");
for (const [selector, label] of [
  [".badge.ok svg", "бейдж"],
  [".metric svg", "глаз или сердце"],
  [".machine-icon svg", "ноутбук"],
  [".identity-actions svg", "отвязка"],
]) {
  const problem = outlineIcon(ownerRoot.querySelector(selector), label);
  check(label, !problem, problem);
}

const css = fs.readFileSync(path.join(frontend, "app/globals.css"), "utf8");
for (const snippet of [
  ".profile .identity {\n  width: 100%; padding: 1.5rem; background: var(--profile-surface);\n  border: 1px solid var(--profile-border); border-radius: 1rem;\n  box-shadow: none; overflow: visible;\n}",
  ".profile .profile-mark {\n  width: 5rem; height: 5rem; border-radius: 1rem;\n  background: var(--profile-secondary); border: 1px solid var(--profile-border-hover);\n  color: var(--profile-accent); font-family: inherit; font-size: 1.5rem;\n}",
  ".profile .house-card {\n  padding: 1.25rem; gap: 1.5rem; border-radius: 1rem;\n  background: var(--profile-surface); border: 1px solid var(--profile-border);\n  transition: border-color 160ms ease;\n}",
]) {
  check("css профиля", css.includes(snippet), snippet.split("\n")[0]);
}

function declaredBackground(el) {
  let found = "";
  const walk = (rules) => {
    for (const rule of rules) {
      if (rule.cssRules) {
        walk(rule.cssRules);
      }
      if (!rule.selectorText || !el.matches(rule.selectorText)) {
        continue;
      }
      const value = rule.style.background || rule.style.backgroundColor;
      if (value) {
        found = value;
      }
    }
  };
  walk(window.document.styleSheets[0].cssRules);
  return found;
}

const docsView = await render(createElement(mod.DocsPage));
const docsRoot = docsView.host;
const pager = docsRoot.querySelector("nav.vdocs-pager, nav[aria-label='Соседние разделы']");
const bottomLinks = [...docsRoot.querySelectorAll("a")].filter((node) => /^(Назад|Далее|Вперёд)$/.test(node.textContent.trim()) || node.querySelector("small")?.textContent?.includes("Назад") || node.querySelector("small")?.textContent?.includes("Далее"));
const beginnerCommands = docsRoot.querySelectorAll("#beginners .docs-command");
const otherCommand = docsRoot.querySelector("#installation .docs-command");
const beginnerBg = beginnerCommands[0] ? declaredBackground(beginnerCommands[0]) : "";
const otherBg = otherCommand ? declaredBackground(otherCommand) : "";
const docsLines = [
  `pager=${Boolean(pager)} bottomLinks=${bottomLinks.length}`,
  `beginnerCommands=${beginnerCommands.length} beginnerBg=${beginnerBg}`,
  `otherBg=${otherBg}`,
  `side=${Boolean(docsRoot.querySelector(".vdocs-side"))} search=${Boolean(docsRoot.querySelector(".vdocs-search"))} crumbs=${Boolean(docsRoot.querySelector(".vdocs-crumbs"))} toc=${Boolean(docsRoot.querySelector(".vdocs-toc"))}`,
];
check("пейджер", !pager && bottomLinks.length === 0, docsLines.join(" | "));
check("шесть команд", beginnerCommands.length === 6, String(beginnerCommands.length));
check("прозрачный фон", beginnerBg === "transparent", beginnerBg);
check("заливка остальных", otherBg === "var(--surface)", otherBg);
check("каркас документации", docsLines[3].includes("side=true") && docsLines[3].includes("search=true") && docsLines[3].includes("crumbs=true") && docsLines[3].includes("toc=true"), docsLines[3]);

fs.writeFileSync(path.join(scratch, "header-render.txt"), headerLines.join("\n") + "\n", "utf8");
fs.writeFileSync(path.join(scratch, "docs-render.txt"), docsLines.join("\n") + "\n", "utf8");

const gaps = [
  "прозрачный фон команд «Для чайников» (.docs-steps .docs-command { background: transparent })",
  "нет нижнего пейджера документации (Назад / Далее)",
];
fs.writeFileSync(path.join(scratch, "visual-gaps.txt"), gaps.map((line) => line + "\n").join(""), "utf8");
fs.writeFileSync(path.join(scratch, "visual-gaps-repeat.txt"), gaps.map((line) => line + "\n").join(""), "utf8");

const iconGroups = {
  markdown: ["agents.md", "claude.md", "sample.md", "sample.mdx", "sample.markdown"],
  text: ["LICENSE", "sample.txt"],
  docker: ["Dockerfile", "docker-compose.yml", "docker-compose.yaml"],
  git: [".gitignore", ".gitattributes"],
  lock: [".env"],
  json: ["sample.json", "sample.jsonc"],
  javascript: ["sample.js", "sample.jsx", "sample.mjs", "sample.cjs"],
  typescript: ["sample.ts", "sample.tsx"],
  css: ["sample.css", "sample.scss", "sample.less"],
  html: ["sample.html", "sample.htm"],
  python: ["sample.py"],
  image: ["sample.png", "sample.jpg", "sample.jpeg", "sample.gif", "sample.svg", "sample.webp", "sample.webm", "sample.mp4"],
  yaml: ["sample.yml", "sample.yaml", "sample.toml", "loose.yml"],
  shell: ["sample.sh", "sample.bash", "sample.zsh"],
  file: ["mystery.bin"],
};

function fileEntry(filePath, body = "x") {
  return { path: filePath, body };
}

const skillBody = "---\ndescription: Пример\n---\nТекст навыка\n";
const versionFiles = [
  fileEntry("agents.md", "# Агенты\n"),
  ...Object.values(iconGroups).flat().filter((name) => name !== "agents.md" && name !== "loose.yml").map((name) => fileEntry(name)),
  fileEntry("pack/loose.yml", "key: 1\n"),
  fileEntry("pack/alpha/SKILL.md", skillBody),
  fileEntry("pack/alpha/body.md", "заметка\n"),
  fileEntry("other/alpha/SKILL.md", skillBody),
  fileEntry("guide/SKILL.md", skillBody),
];

function versionBody(files, version) {
  return { version, created: "2026-10-01T12:00:00Z", files };
}

function publicationRoutes(url) {
  const target = String(url);
  if (target.endsWith("/api/accounts/alice/versions/grok/1")) {
    return { status: 200, body: versionBody(versionFiles, 1) };
  }
  if (target.endsWith("/api/accounts/alice/versions/grok/2")) {
    return { status: 200, body: versionBody([fileEntry("solo.txt", "один\n")], 2) };
  }
  if (target.endsWith("/api/publications/pub-1")) {
    return {
      status: 200,
      body: { author: "alice", agent: "grok", version: 1, files: versionFiles, excluded: [] },
    };
  }
  return { status: 404, body: { explanation: "нет" } };
}

function rowName(button) {
  const name = button.querySelector(".tree-name");
  if (!name) {
    return "";
  }
  const clone = name.cloneNode(true);
  for (const node of clone.querySelectorAll(".tree-mark, svg")) {
    node.remove();
  }
  return clone.textContent.trim();
}

function namedButton(root, name) {
  return [...root.querySelectorAll("button.tree-dir, button.tree-file")].find((button) => rowName(button) === name) || null;
}

function directDir(branch) {
  const wrap = branch.firstElementChild;
  return wrap ? wrap.querySelector(":scope > button.tree-dir") : null;
}

function folderRow(root, name) {
  return [...root.querySelectorAll(".tree-branch")].find((node) => {
    const button = directDir(node);
    return button && rowName(button) === name;
  }) || null;
}

function rowCheck(branch) {
  const wrap = branch?.firstElementChild;
  return wrap ? wrap.querySelector(":scope > input.tree-check") : null;
}

function iconSignature(button) {
  const svg = button?.querySelector("svg.file-icon");
  if (!svg) {
    return "";
  }
  return [...svg.querySelectorAll("path")].map((pathNode) => `${pathNode.getAttribute("fill")}|${pathNode.getAttribute("d")}`).join(";");
}

function isRound(style) {
  const radius = style.borderRadius;
  if (radius === "50%" || radius === "999px") {
    return true;
  }
  const width = parseFloat(style.width);
  const height = parseFloat(style.height);
  const value = parseFloat(radius);
  return width > 0 && Math.abs(width - height) < 0.2 && value >= width / 2 - 0.2;
}

function ruleText(selector) {
  let found = "";
  const walk = (rules) => {
    for (const rule of rules) {
      if (rule.selectorText === selector) {
        found = rule.cssText;
      }
      if (rule.cssRules) {
        walk(rule.cssRules);
      }
    }
  };
  walk(window.document.styleSheets[0].cssRules);
  try {
    return decodeURIComponent(found);
  } catch {
    return found;
  }
}

function flat(text) {
  return String(text || "").replaceAll("\u00A0", " ");
}

function mediaText() {
  const chunks = [];
  const walk = (rules) => {
    for (const rule of rules) {
      if (rule.media && String(rule.conditionText || rule.media.mediaText || "").includes("560px")) {
        chunks.push(rule.cssText);
      }
      if (rule.cssRules) {
        walk(rule.cssRules);
      }
    }
  };
  walk(window.document.styleSheets[0].cssRules);
  return chunks.join("\n");
}

async function userClick(el) {
  await act(async () => {
    el.click();
  });
  await flush();
}

async function waitFor(host, pred) {
  for (let i = 0; i < 50; i += 1) {
    if (pred(host)) {
      return true;
    }
    await flush();
  }
  return pred(host);
}

function commandText(host) {
  return host.querySelector(".skill-command code")?.textContent || "";
}

function assertIcons(root, label) {
  const signatures = new Map();
  for (const [kind, names] of Object.entries(iconGroups)) {
    const signs = names.map((name) => iconSignature(namedButton(root, name)));
    check(`${label} иконка ${kind}`, signs.every((sign) => sign && sign === signs[0]), signs.join(" | "));
    signatures.set(kind, signs[0]);
  }
  const kinds = [...signatures.keys()];
  for (let i = 0; i < kinds.length; i += 1) {
    for (let j = i + 1; j < kinds.length; j += 1) {
      check(
        `${label} разные ${kinds[i]}/${kinds[j]}`,
        signatures.get(kinds[i]) !== signatures.get(kinds[j]),
        "совпали",
      );
    }
  }
  const folders = [...root.querySelectorAll("button.tree-dir")].map(iconSignature);
  check(`${label} папки`, folders.length > 0 && folders.every((sign) => sign && sign === folders[0]), String(folders.length));
  check(`${label} папка не файл`, folders[0] && folders[0] !== signatures.get("file") && folders[0] !== signatures.get("markdown"), folders[0]);
  const bare = [...root.querySelectorAll("button.tree-dir, button.tree-file")].filter((button) => !button.querySelector("svg.file-icon"));
  check(`${label} у каждой строки есть иконка`, bare.length === 0, bare.map(rowName).join(","));
}

function snapshot(host) {
  const sw = host.querySelector("[aria-label='Режим выделения навыков']");
  const checks = [...host.querySelectorAll("input.tree-check")].map((el) => {
    return `${el.getAttribute("aria-label")}:${el.checked ? "1" : "0"}${el.indeterminate ? "?" : ""}`;
  }).sort();
  const stage = host.querySelector(".code-stage");
  const stageText = stage?.textContent || "";
  const hints = [...host.querySelectorAll(".skill-pick .hint")].map((el) => el.textContent).join("|");
  const clash = host.querySelector(".skill-pick .explanation")?.textContent || "";
  return [
    `switch=${sw ? sw.getAttribute("aria-checked") : "none"}`,
    `checks=${checks.join(",")}`,
    `code=${commandText(host)}`,
    `hint=${hints}`,
    `clash=${clash}`,
    `selectAll=${host.textContent.includes("Выделить все")}`,
    `stageControls=${Boolean(stage?.querySelector("[aria-label='Режим выделения навыков'], input"))}`,
    `preview=${stageText.includes("Просмотр")}`,
    `source=${stageText.includes("Исходник")}`,
    `copy=${stageText.includes("Копировать")}`,
    `download=${Boolean(stage?.querySelector("[aria-label^='Скачать ']"))}`,
    `skillZip=${stageText.includes("Скачать навык")}`,
    `crumbs=${host.querySelector("nav[aria-label='Путь']")?.textContent || ""}`,
    `pill=${host.querySelector(".side-head-files .count-pill")?.textContent || ""}`,
    `grouped=${Boolean(host.querySelector(".side-head-title h2")) && host.querySelector(".side-head-title")?.textContent.includes("Выделение") && Boolean(host.querySelector(".side-head-files .count-pill"))}`,
  ].join("\n");
}

async function runVersionPass() {
  const lines = [];
  routes = publicationRoutes;
  mod.__setParams({ id: "pub-1", name: "alice", agent: "grok", number: "1" });
  const view = await render(createElement(mod.VersionPage));
  const host = view.host;
  const ready = await waitFor(host, (node) => Boolean(namedButton(node, "loose.yml")) && node.querySelector(".side-head-files .count-pill")?.textContent !== "…");
  check("версия открылась", ready, host.textContent.slice(0, 180));
  assertIcons(host, "версия");
  const sw = host.querySelector("[aria-label='Режим выделения навыков']");
  const swStyle = sw ? window.getComputedStyle(sw) : null;
  const thumb = sw?.querySelector(".skill-switch-thumb");
  const thumbStyle = thumb ? window.getComputedStyle(thumb) : null;
  check("переключатель выключен", sw?.getAttribute("aria-checked") === "false", sw?.getAttribute("aria-checked"));
  check("нет флажков", host.querySelectorAll("input.tree-check").length === 0, String(host.querySelectorAll("input").length));
  check("нет Выделить все", !host.textContent.includes("Выделить все"), "надпись осталась");
  check("нет команды", !host.querySelector(".skill-pick"), host.querySelector(".skill-pick")?.textContent || "");
  check("пилюля", Boolean(swStyle) && isRound(swStyle) && parseFloat(swStyle.width) > parseFloat(swStyle.height), swStyle ? `${swStyle.width} ${swStyle.height} ${swStyle.borderRadius}` : "");
  check("бегунок", Boolean(thumbStyle) && isRound(thumbStyle) && thumbStyle.width === thumbStyle.height, thumbStyle ? `${thumbStyle.width} ${thumbStyle.borderRadius}` : "");
  const headStyle = window.getComputedStyle(host.querySelector(".side-head-files"));
  check("шапка в одну строку", headStyle.flexWrap === "nowrap", headStyle.flexWrap);
  const narrow = mediaText();
  check("узкая шапка", narrow.includes("flex-wrap: wrap") && narrow.includes("margin-left: auto"), narrow);
  check("группа заголовка", host.querySelector(".side-head-title h2")?.textContent === "Файлы версии" && Boolean(host.querySelector(".side-head-files .count-pill")), "шапка");
  const stage = host.querySelector(".code-stage");
  check("сцена без выделения", stage && !stage.querySelector("input, [role='switch']") && stage.textContent.includes("Просмотр") && stage.textContent.includes("Исходник"), stage?.textContent?.slice(0, 120) || "");
  check("крошки", host.querySelector("nav[aria-label='Путь']")?.textContent.includes("v1"), host.querySelector("nav")?.textContent || "");
  await userClick(namedButton(host, "guide"));
  await userClick(namedButton(host, "SKILL.md"));
  check("скачивание навыка", host.querySelector(".code-stage")?.textContent.includes("Скачать навык"), "нет кнопки");
  lines.push("start");
  lines.push(snapshot(host));

  await userClick(sw);
  check("режим включён", host.querySelector("[aria-label='Режим выделения навыков']")?.getAttribute("aria-checked") === "true", "не включился");
  check("команда пустого выбора", flat(host.textContent).includes("Отметьте навыки в списке файлов"), host.querySelector(".skill-pick")?.textContent || "");
  const pack = folderRow(host, "pack");
  const alpha = folderRow(pack, "alpha");
  const guide = folderRow(host, "guide");
  const other = folderRow(host, "other");
  const otherAlpha = folderRow(other, "alpha");
  check("флажок навыка", rowCheck(guide)?.getAttribute("aria-label") === "Навык guide" && guide.querySelectorAll(":scope > .tree-line > input.tree-check").length === 1, rowCheck(guide)?.getAttribute("aria-label"));
  check("флажок папки", rowCheck(pack)?.getAttribute("aria-label") === "Папка pack со всеми вложениями", rowCheck(pack)?.getAttribute("aria-label"));
  const nestedFile = namedButton(host, "loose.yml");
  check("нет флажка вложенного файла", !nestedFile?.parentElement?.classList.contains("tree-line"), nestedFile?.parentElement?.className || "");
  const rootFile = namedButton(host, "sample.txt");
  const rootCheck = rootFile?.parentElement?.classList.contains("tree-line") ? rootFile.parentElement.querySelector(":scope > input.tree-check") : null;
  check("флажок корневого файла", rootCheck?.getAttribute("aria-label") === "Файл sample.txt", rootCheck?.getAttribute("aria-label") || "нет");
  lines.push("mode-on");
  lines.push(snapshot(host));

  await userClick(rowCheck(alpha));
  check("папка частично", rowCheck(pack)?.indeterminate === true && rowCheck(pack)?.checked === false, `indeterminate=${rowCheck(pack)?.indeterminate}`);
  const partialStyle = window.getComputedStyle(rowCheck(pack));
  const dashRule = ruleText(".tree-check:indeterminate");
  check("черта indeterminate", isRound(partialStyle) && partialStyle.backgroundColor === "oklch(0.62 0.17 250)" && dashRule.includes("M3.5 8h9"), `${partialStyle.backgroundColor} ${dashRule.slice(0, 160)}`);
  lines.push("partial");
  lines.push(snapshot(host));

  await userClick(rowCheck(pack));
  const folderCode = commandText(host);
  check("команда только навык папки", folderCode.includes("agentsync") && folderCode.includes("skills") && folderCode.includes("pack/alpha") && !folderCode.includes("loose.yml") && !folderCode.includes("SKILL.md") && !folderCode.includes("sample.txt") && !folderCode.includes("guide"), folderCode);
  check("папка отмечена целиком", rowCheck(pack)?.checked === true && rowCheck(pack)?.indeterminate === false, `checked=${rowCheck(pack)?.checked}`);
  const checkedStyle = window.getComputedStyle(rowCheck(alpha));
  const markRule = ruleText(".tree-check:checked");
  check("галка флажка", isRound(checkedStyle) && checkedStyle.backgroundColor === "oklch(0.62 0.17 250)" && markRule.includes("M3 8.5") && markRule.includes("L13 5"), `${checkedStyle.backgroundColor} ${markRule.slice(0, 160)}`);
  const fileCheck = namedButton(host, "sample.txt")?.parentElement?.querySelector(":scope > input.tree-check");
  await userClick(fileCheck);
  check("файл отмечен", fileCheck?.checked === true, String(fileCheck?.checked));
  check("файл не входит в команду", commandText(host) === folderCode && !commandText(host).includes("sample.txt"), commandText(host));
  await userClick(rowCheck(guide));
  const both = commandText(host);
  check("два навыка", both.includes("guide") && both.includes("pack/alpha") && !both.includes("sample.txt"), both);
  await userClick(rowCheck(otherAlpha));
  check("конфликт имён", (host.querySelector(".skill-pick .explanation")?.textContent || "").includes("называются одинаково") && commandText(host) === "", host.querySelector(".skill-pick")?.textContent || "");
  lines.push("clash");
  lines.push(snapshot(host));

  const kept = host.querySelector(".skill-pick")?.textContent || "";
  await userClick(host.querySelector("[aria-label='Режим выделения навыков']"));
  check("выключение прячет флажки", host.querySelectorAll("input.tree-check").length === 0, String(host.querySelectorAll("input.tree-check").length));
  check("выбор сохранён", host.querySelector(".skill-pick")?.textContent === kept && kept.includes("называются одинаково"), host.querySelector(".skill-pick")?.textContent || "");
  await userClick(host.querySelector("[aria-label='Режим выделения навыков']"));
  check("отметки на месте", rowCheck(guide)?.checked === true && rowCheck(otherAlpha)?.checked === true, snapshot(host));
  lines.push("kept");
  lines.push(snapshot(host));

  mod.__setParams({ id: "pub-1", name: "alice", agent: "grok", number: "2" });
  await act(async () => {
    view.root.render(createElement(mod.VersionPage));
  });
  const second = await waitFor(host, (node) => node.textContent.includes("v2") && Boolean(namedButton(node, "solo.txt")));
  check("другая версия", second, host.textContent.slice(0, 160));
  check("без навыков нет переключателя", !host.querySelector("[aria-label='Режим выделения навыков']") && !host.querySelector(".skill-pick") && host.querySelectorAll("input.tree-check").length === 0, snapshot(host));
  check("иконка файла без навыков", Boolean(iconSignature(namedButton(host, "solo.txt"))), "нет иконки");
  lines.push("v2");
  lines.push(snapshot(host));

  mod.__setParams({ id: "pub-1", name: "alice", agent: "grok", number: "1" });
  await act(async () => {
    view.root.render(createElement(mod.VersionPage));
  });
  const back = await waitFor(host, (node) => node.textContent.includes("v1") && Boolean(namedButton(node, "loose.yml")));
  check("возврат сбрасывает режим", back && host.querySelector("[aria-label='Режим выделения навыков']")?.getAttribute("aria-checked") === "false" && !host.querySelector(".skill-pick") && host.querySelectorAll("input.tree-check").length === 0, snapshot(host));
  lines.push("reset");
  lines.push(snapshot(host));

  const manifest = await render(createElement(mod.PublicationPage));
  const manifestReady = await waitFor(manifest.host, (node) => Boolean(namedButton(node, "agents.md")));
  check("манифест открылся", manifestReady, manifest.host.textContent.slice(0, 160));
  await userClick(namedButton(manifest.host, "pack"));
  await waitFor(manifest.host, (node) => Boolean(namedButton(node, "loose.yml")));
  assertIcons(manifest.host, "манифест");
  check("манифест без выделения", manifest.host.querySelectorAll("input.tree-check").length === 0 && !manifest.host.textContent.includes("Выделение") && !manifest.host.textContent.includes("Выделить все") && !manifest.host.querySelector("[role='switch']"), manifest.host.textContent.slice(0, 180));
  lines.push("manifest");
  lines.push(`icons=${Boolean(namedButton(manifest.host, "loose.yml")?.querySelector("svg.file-icon"))}`);
  lines.push(`checks=${manifest.host.querySelectorAll("input.tree-check").length}`);
  lines.push(`switch=${Boolean(manifest.host.querySelector("[role='switch']"))}`);

  await act(async () => {
    view.root.unmount();
    manifest.root.unmount();
  });
  view.host.remove();
  manifest.host.remove();
  return lines.join("\n");
}

const versionReports = [];
for (let pass = 1; pass <= 2; pass += 1) {
  versionReports.push(await runVersionPass());
}
function firstDiff(left, right) {
  const a = left.split("\n");
  const b = right.split("\n");
  const count = Math.max(a.length, b.length);
  for (let i = 0; i < count; i += 1) {
    if (a[i] !== b[i]) {
      return `строка ${i + 1}: ${a[i] || ""} <> ${b[i] || ""}`;
    }
  }
  return "";
}
check("прогоны версии совпали", versionReports[0] === versionReports[1], firstDiff(versionReports[0], versionReports[1]));
const publicationLog = ["pass 1", versionReports[0], "pass 2", versionReports[1]].join("\n");
console.log(publicationLog);

if (failures.length) {
  const text = failures.join("\n");
  fs.writeFileSync(path.join(scratch, "visual-fail.txt"), text + "\n", "utf8");
  console.error(text);
  process.exit(1);
}
console.log("visual render ok");
console.log(headerLines.join("\n"));
console.log(docsLines.join("\n"));

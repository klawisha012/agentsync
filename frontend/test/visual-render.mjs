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

if (failures.length) {
  const text = failures.join("\n");
  fs.writeFileSync(path.join(scratch, "visual-fail.txt"), text + "\n", "utf8");
  console.error(text);
  process.exit(1);
}
console.log("visual render ok");
console.log(headerLines.join("\n"));
console.log(docsLines.join("\n"));

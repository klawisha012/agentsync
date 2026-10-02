import { strToU8, zipSync } from "fflate";

export function isMarkdown(path) {
  return String(path || "").toLowerCase().endsWith(".md");
}

export function fileDownloadName(path) {
  const base = normalize(path).split("/").pop() || "file.txt";
  return safeName(base);
}

export function skillBundle(files, filePath) {
  const path = normalize(filePath);
  const root = nearestSkillRoot(files, path);
  if (!root) {
    return null;
  }
  const name = root.split("/").pop();
  const entries = [];
  for (const file of files || []) {
    const rel = relativeUnder(root, normalize(file.path));
    if (!rel) {
      continue;
    }
    entries.push({ path: `${name}/${rel}`, body: file.body || "" });
  }
  entries.sort((a, b) => a.path.localeCompare(b.path, "en"));
  if (entries.length === 0) {
    return null;
  }
  return { name, zipName: `${safeName(name)}.zip`, entries };
}

export function skillZip(bundle) {
  const data = {};
  for (const entry of bundle.entries) {
    data[entry.path] = strToU8(entry.body);
  }
  return zipSync(data);
}

function nearestSkillRoot(files, filePath) {
  const roots = [];
  for (const file of files || []) {
    const path = normalize(file.path);
    const base = path.split("/").pop() || "";
    if (base.toLowerCase() !== "skill.md") {
      continue;
    }
    const dir = path.slice(0, -base.length).replace(/\/$/, "");
    if (dir) {
      roots.push(dir);
    }
  }
  const hits = roots.filter((root) => filePath === root || filePath.startsWith(`${root}/`));
  hits.sort((a, b) => b.length - a.length);
  return hits[0] || "";
}

function relativeUnder(root, path) {
  const prefix = `${root}/`;
  if (!path.startsWith(prefix)) {
    return "";
  }
  const rel = path.slice(prefix.length);
  if (!rel || rel.split("/").some((part) => part === "" || part === "." || part === "..")) {
    return "";
  }
  return rel;
}

function normalize(path) {
  return String(path || "").replaceAll("\\", "/");
}

function safeName(name) {
  const cleaned = String(name).replace(/[<>:"/\\|?*\u0000-\u001f]/g, "-").replace(/^\.+/, "").trim();
  return cleaned || "file";
}

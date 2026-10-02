import { countTokens } from "gpt-tokenizer/encoding/cl100k_base";

export function tokenCount(text) {
  if (!text) {
    return 0;
  }
  return countTokens(text);
}

export function formatCount(value) {
  return String(value).replace(/\B(?=(\d{3})+(?!\d))/g, "\u202f");
}

export function skillTokenTotal(nodes) {
  let total = 0;
  const walk = (list) => {
    for (const node of list) {
      if (node.skill) {
        total += node.skill.content;
        continue;
      }
      walk(node.children || []);
    }
  };
  walk(nodes);
  return total;
}

export function ruTokens(n) {
  const n10 = n % 10;
  const n100 = n % 100;
  let word = "токенов";
  if (n10 === 1 && n100 !== 11) {
    word = "токен";
  } else if (n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14)) {
    word = "токена";
  }
  return `${formatCount(n)}\u00a0${word}`;
}

export function skillDescription(markdown) {
  const text = String(markdown || "").replace(/^\uFEFF/, "");
  if (!text.startsWith("---")) {
    return "";
  }
  const end = text.indexOf("\n---", 3);
  if (end < 0) {
    return "";
  }
  return readDescription(text.slice(text.indexOf("\n") + 1, end).split(/\r?\n/));
}

export function buildManifest(files) {
  const root = { name: "", path: "", file: null, children: [] };
  const index = new Map([["", root]]);
  for (const file of files) {
    const parts = String(file.path || "").replaceAll("\\", "/").split("/").filter(Boolean);
    let parentPath = "";
    for (let i = 0; i < parts.length; i++) {
      const name = parts[i];
      const path = parentPath ? `${parentPath}/${name}` : name;
      const leaf = i === parts.length - 1;
      let node = index.get(path);
      if (!node) {
        node = { name, path, file: leaf ? file : null, children: [] };
        index.set(path, node);
        index.get(parentPath).children.push(node);
      } else if (leaf) {
        node.file = file;
      }
      parentPath = path;
    }
  }
  sortNodes(root);
  annotateSkills(root);
  return root.children;
}

function sortNodes(node) {
  node.children.sort((a, b) => a.name.localeCompare(b.name, "ru", { sensitivity: "base" }));
  for (const child of node.children) {
    sortNodes(child);
  }
}

function annotateSkills(node) {
  const skillMd = node.children.find((child) => child.file && child.name.toLowerCase() === "skill.md");
  if (skillMd && node.path) {
    const bodies = [];
    collectBodies(node, bodies);
    node.skill = {
      description: tokenCount(skillDescription(skillMd.file.body || "")),
      content: bodies.reduce((sum, body) => sum + tokenCount(body), 0),
    };
  }
  for (const child of node.children) {
    annotateSkills(child);
  }
}

function collectBodies(node, bodies) {
  if (node.file) {
    bodies.push(node.file.body || "");
  }
  for (const child of node.children) {
    collectBodies(child, bodies);
  }
}

function readDescription(lines) {
  for (let i = 0; i < lines.length; i++) {
    const match = lines[i].match(/^description:\s*(.*)$/);
    if (!match) {
      continue;
    }
    const rest = match[1].trim();
    if (rest === "" || rest === ">" || rest === "|" || rest === ">-" || rest === "|-" || rest === ">+" || rest === "|+") {
      const block = readBlock(lines, i + 1);
      if (rest.startsWith("|")) {
        return block.join("\n").trim();
      }
      return foldBlock(block);
    }
    return unquote(rest);
  }
  return "";
}

function readBlock(lines, start) {
  const block = [];
  for (let i = start; i < lines.length; i++) {
    const line = lines[i];
    if (line === "") {
      let next = i + 1;
      while (next < lines.length && lines[next] === "") {
        next++;
      }
      if (next >= lines.length || !/^\s/.test(lines[next])) {
        break;
      }
      block.push("");
      continue;
    }
    if (!/^\s/.test(line)) {
      break;
    }
    block.push(line.trim());
  }
  return block;
}

function foldBlock(lines) {
  const parts = [];
  let paragraph = [];
  for (const line of lines) {
    if (line === "") {
      if (paragraph.length) {
        parts.push(paragraph.join(" "));
        paragraph = [];
      }
      continue;
    }
    paragraph.push(line);
  }
  if (paragraph.length) {
    parts.push(paragraph.join(" "));
  }
  return parts.join("\n").trim();
}

function unquote(value) {
  if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
    return value.slice(1, -1);
  }
  return value;
}

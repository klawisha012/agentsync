import assert from "node:assert/strict";
import test from "node:test";
import { kindSummary } from "../app/publications/[id]/manifest.js";

test("kindSummary counts every canon kind a publication carries", () => {
  const files = [
    { path: "skills/note/SKILL.md", body: "" },
    { path: "agents/go.md", body: "" },
    { path: "commands/review.md", body: "" },
    { path: "mcp.json", body: '{"mcpServers":{"a":{},"b":{}}}' },
    { path: "settings.json", body: '{"enabledPlugins":{"x@y":true},"hooks":{"Stop":[{},{}]}}' },
    { path: "rules/own.md", body: "" },
  ];
  assert.deepEqual(kindSummary(files), [
    ["Навыки", 1], ["Субагенты", 1], ["Команды", 1], ["MCP-серверы", 2], ["Плагины", 1], ["Хуки", 2],
  ]);
  assert.deepEqual(kindSummary([{ path: "settings.json", body: "not json" }]), []);
});

test("buildManifest lists files before folders", async () => {
  const { buildManifest } = await import("../app/publications/[id]/manifest.js");
  const tree = buildManifest([
    { path: "skills/a/SKILL.md", body: "" },
    { path: "settings.json", body: "" },
    { path: "plugins/x.json", body: "" },
    { path: "AGENTS.md", body: "" },
  ]);
  assert.deepEqual(tree.map((node) => node.name), ["AGENTS.md", "settings.json", "plugins", "skills"]);
});

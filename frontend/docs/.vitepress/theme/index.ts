import DefaultTheme from "vitepress/theme";
import type { Theme } from "vitepress";
import "./custom.css";

// Old single-page docs used anchors such as /docs#publish; send them to the page that replaced the section.
const oldAnchors: Record<string, string> = {
  beginners: "start/quick-start.html",
  installation: "start/install.html",
  login: "start/login.html",
  publish: "config/publish.html",
  agents: "config/agents.html",
  snapshots: "config/snapshots.html",
  apply: "config/apply.html",
  skills: "config/skills.html",
  help: "reference/cli.html",
  mcp: "reference/mcp-server.html",
};

export default {
  extends: DefaultTheme,
  enhanceApp({ router }) {
    if (typeof window === "undefined") {
      return;
    }
    const base = "/docs/";
    const path = window.location.pathname;
    const target = oldAnchors[window.location.hash.slice(1)];
    if (target && (path === "/docs" || path === base || path === `${base}index.html`)) {
      router.go(base + target);
    }
  },
} satisfies Theme;

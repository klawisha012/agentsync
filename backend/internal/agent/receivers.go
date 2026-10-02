package agent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Receiver is one program from the shipped skill-install catalog.
type Receiver struct {
	Slug     string
	Display  string
	Project  string
	Global   string
	Env      string
	InConfig bool
	OpenClaw bool
}

// ReceiverView is the public list the site shows when a person picks a receiver.
type ReceiverView struct {
	Slug    string `json:"slug"`
	Display string `json:"display"`
	Global  bool   `json:"global"`
	Project bool   `json:"project"`
}

func (r Receiver) HasGlobal() bool {
	return r.Global != "" || r.OpenClaw || r.InConfig
}

func (r Receiver) HasProject() bool {
	return r.Project != ""
}

func (r Receiver) View() ReceiverView {
	return ReceiverView{Slug: r.Slug, Display: r.Display, Global: r.HasGlobal(), Project: r.HasProject()}
}

// ReceiverViews lists receivers by visible name.
func ReceiverViews() []ReceiverView {
	out := make([]ReceiverView, len(receivers))
	for i, item := range receivers {
		out[i] = item.View()
	}
	slices.SortFunc(out, func(a, b ReceiverView) int {
		return strings.Compare(strings.ToLower(a.Display), strings.ToLower(b.Display))
	})
	return out
}

// FindReceiver looks up a command name. Matching ignores letter case.
func FindReceiver(slug string) (Receiver, bool) {
	key := strings.ToLower(strings.TrimSpace(slug))
	for _, item := range receivers {
		if item.Slug == key {
			return item, true
		}
	}
	return Receiver{}, false
}

// GlobalDir is the skills directory for a global install under root, which stands in for the home directory.
func (r Receiver) GlobalDir(root string) (string, bool) {
	if r.OpenClaw {
		return openClawDir(root), true
	}
	if r.Env != "" {
		if value := strings.TrimSpace(os.Getenv(r.Env)); value != "" {
			return filepath.Join(value, "skills"), true
		}
	}
	if r.InConfig {
		if r.Global == "" {
			return "", false
		}
		return filepath.Join(configHome(root), filepath.FromSlash(r.Global)), true
	}
	if r.Global == "" {
		return "", false
	}
	return filepath.Join(root, filepath.FromSlash(r.Global)), true
}

// ProjectDir is the skills directory for an install into cwd.
func (r Receiver) ProjectDir(cwd string) (string, bool) {
	if r.Project == "" {
		return "", false
	}
	return filepath.Join(cwd, filepath.FromSlash(r.Project)), true
}

func openClawDir(root string) string {
	for _, name := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		dir := filepath.Join(root, name)
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			return filepath.Join(dir, "skills")
		}
	}
	return filepath.Join(root, ".openclaw", "skills")
}

func configHome(root string) string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return value
	}
	return filepath.Join(root, ".config")
}

// receivers is the install catalog. Universal is omitted: it is a shared directory, not a program.
// Droid's global directory is the one a new install uses, not the legacy directory kept for removal.
var receivers = []Receiver{
	{Slug: "aider-desk", Display: "AiderDesk", Project: ".aider-desk/skills", Global: ".aider-desk/skills"},
	{Slug: "amp", Display: "Amp", Project: ".agents/skills", Global: "agents/skills", InConfig: true},
	{Slug: "antigravity", Display: "Antigravity", Project: ".agents/skills", Global: ".gemini/antigravity/skills"},
	{Slug: "antigravity-cli", Display: "Antigravity CLI", Project: ".agents/skills", Global: ".gemini/antigravity-cli/skills"},
	{Slug: "astrbot", Display: "AstrBot", Project: "data/skills", Global: ".astrbot/data/skills"},
	{Slug: "autohand-code", Display: "Autohand Code CLI", Project: ".autohand/skills", Global: ".autohand/skills", Env: "AUTOHAND_HOME"},
	{Slug: "augment", Display: "Augment", Project: ".augment/skills", Global: ".augment/skills"},
	{Slug: "bob", Display: "IBM Bob", Project: ".bob/skills", Global: ".bob/skills"},
	{Slug: "claude-code", Display: "Claude Code", Project: ".claude/skills", Global: ".claude/skills", Env: "CLAUDE_CONFIG_DIR"},
	{Slug: "openclaw", Display: "OpenClaw", Project: "skills", OpenClaw: true},
	{Slug: "cline", Display: "Cline", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "codearts-agent", Display: "CodeArts Agent", Project: ".codeartsdoer/skills", Global: ".codeartsdoer/skills"},
	{Slug: "codebuddy", Display: "CodeBuddy", Project: ".codebuddy/skills", Global: ".codebuddy/skills"},
	{Slug: "codemaker", Display: "Codemaker", Project: ".codemaker/skills", Global: ".codemaker/skills"},
	{Slug: "codestudio", Display: "Code Studio", Project: ".codestudio/skills", Global: ".codestudio/skills"},
	{Slug: "codex", Display: "Codex", Project: ".agents/skills", Global: ".codex/skills", Env: "CODEX_HOME"},
	{Slug: "command-code", Display: "Command Code", Project: ".commandcode/skills", Global: ".commandcode/skills"},
	{Slug: "continue", Display: "Continue", Project: ".continue/skills", Global: ".continue/skills"},
	{Slug: "cortex", Display: "Cortex Code", Project: ".cortex/skills", Global: ".snowflake/cortex/skills"},
	{Slug: "crush", Display: "Crush", Project: ".crush/skills", Global: ".config/crush/skills"},
	{Slug: "cursor", Display: "Cursor", Project: ".agents/skills", Global: ".cursor/skills"},
	{Slug: "deepagents", Display: "Deep Agents", Project: ".agents/skills", Global: ".deepagents/agent/skills"},
	{Slug: "devin", Display: "Devin for Terminal", Project: ".devin/skills", Global: "devin/skills", InConfig: true},
	{Slug: "dexto", Display: "Dexto", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "droid", Display: "Droid", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "eve", Display: "Eve", Project: "agent/skills"},
	{Slug: "firebender", Display: "Firebender", Project: ".agents/skills", Global: ".firebender/skills"},
	{Slug: "forgecode", Display: "ForgeCode", Project: ".forge/skills", Global: ".forge/skills"},
	{Slug: "fx", Display: "fx", Project: ".fx/skills", Global: ".fx/skills"},
	{Slug: "gemini-cli", Display: "Gemini CLI", Project: ".agents/skills", Global: ".gemini/skills"},
	{Slug: "github-copilot", Display: "GitHub Copilot", Project: ".agents/skills", Global: ".copilot/skills"},
	{Slug: "goose", Display: "Goose", Project: ".goose/skills", Global: "goose/skills", InConfig: true},
	{Slug: "grok", Display: "Grok Build", Project: ".grok/skills", Global: ".grok/skills", Env: "GROK_HOME"},
	{Slug: "hermes-agent", Display: "Hermes Agent", Project: ".hermes/skills", Global: ".hermes/skills", Env: "HERMES_HOME"},
	{Slug: "inference-sh", Display: "inference.sh", Project: ".inferencesh/skills", Global: ".inferencesh/skills"},
	{Slug: "jazz", Display: "Jazz", Project: ".jazz/skills", Global: ".jazz/skills"},
	{Slug: "junie", Display: "Junie", Project: ".junie/skills", Global: ".junie/skills"},
	{Slug: "iflow-cli", Display: "iFlow CLI", Project: ".iflow/skills", Global: ".iflow/skills"},
	{Slug: "kilo", Display: "Kilo Code", Project: ".agents/skills", Global: ".kilo/skills"},
	{Slug: "kimchi", Display: "Kimchi", Project: ".kimchi/skills", Global: ".config/kimchi/harness/skills"},
	{Slug: "kimi-code-cli", Display: "Kimi Code CLI", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "kiro-cli", Display: "Kiro CLI", Project: ".kiro/skills", Global: ".kiro/skills"},
	{Slug: "kode", Display: "Kode", Project: ".kode/skills", Global: ".kode/skills"},
	{Slug: "lingma", Display: "Lingma", Project: ".lingma/skills", Global: ".lingma/skills"},
	{Slug: "loaf", Display: "Loaf", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "mcpjam", Display: "MCPJam", Project: ".mcpjam/skills", Global: ".mcpjam/skills"},
	{Slug: "minimax-code", Display: "MiniMax Code", Project: ".minimax/skills", Global: ".minimax/skills"},
	{Slug: "mistral-vibe", Display: "Mistral Vibe", Project: ".vibe/skills", Global: ".vibe/skills", Env: "VIBE_HOME"},
	{Slug: "moxby", Display: "Moxby", Project: ".moxby/skills", Global: ".moxby/skills"},
	{Slug: "mux", Display: "Mux", Project: ".mux/skills", Global: ".mux/skills"},
	{Slug: "opencode", Display: "OpenCode", Project: ".agents/skills", Global: "opencode/skills", InConfig: true},
	{Slug: "openhands", Display: "OpenHands", Project: ".openhands/skills", Global: ".openhands/skills"},
	{Slug: "ona", Display: "Ona", Project: ".ona/skills", Global: ".ona/skills"},
	{Slug: "pi", Display: "Pi", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "posit-assistant", Display: "Posit Assistant", Project: ".posit/assistant/skills", Global: ".posit/assistant/skills"},
	{Slug: "qoder", Display: "Qoder", Project: ".qoder/skills", Global: ".qoder/skills"},
	{Slug: "qoder-cn", Display: "Qoder CN", Project: ".qoder/skills", Global: ".qoder-cn/skills"},
	{Slug: "qwen-code", Display: "Qwen Code", Project: ".qwen/skills", Global: ".qwen/skills"},
	{Slug: "replit", Display: "Replit", Project: ".agents/skills", Global: "agents/skills", InConfig: true},
	{Slug: "reasonix", Display: "Reasonix", Project: ".reasonix/skills", Global: ".reasonix/skills"},
	{Slug: "rovodev", Display: "Rovo Dev", Project: ".rovodev/skills", Global: ".rovodev/skills"},
	{Slug: "roo", Display: "Roo Code", Project: ".roo/skills", Global: ".roo/skills"},
	{Slug: "sarvam-code", Display: "Sarvam Code", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "tabnine-cli", Display: "Tabnine CLI", Project: ".tabnine/agent/skills", Global: ".tabnine/agent/skills"},
	{Slug: "terramind", Display: "Terramind", Project: ".terramind/skills", Global: ".terramind/skills"},
	{Slug: "tinycloud", Display: "Tinycloud", Project: ".tinycloud/skills", Global: ".tinycloud/skills"},
	{Slug: "trae", Display: "Trae", Project: ".trae/skills", Global: ".trae/skills"},
	{Slug: "trae-cn", Display: "Trae CN", Project: ".trae/skills", Global: ".trae-cn/skills"},
	{Slug: "warp", Display: "Warp", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "windsurf", Display: "Windsurf", Project: ".windsurf/skills", Global: ".codeium/windsurf/skills"},
	{Slug: "zed", Display: "Zed", Project: ".agents/skills", Global: ".agents/skills"},
	{Slug: "zcode", Display: "ZCode", Project: ".zcode/skills", Global: ".zcode/skills"},
	{Slug: "zencoder", Display: "Zencoder", Project: ".zencoder/skills", Global: ".zencoder/skills"},
	{Slug: "zenflow", Display: "Zenflow", Project: ".zencoder/skills", Global: ".zencoder/skills"},
	{Slug: "neovate", Display: "Neovate", Project: ".neovate/skills", Global: ".neovate/skills"},
	{Slug: "pochi", Display: "Pochi", Project: ".pochi/skills", Global: ".pochi/skills"},
	{Slug: "promptscript", Display: "PromptScript", Project: ".agents/skills"},
	{Slug: "adal", Display: "AdaL", Project: ".adal/skills", Global: ".adal/skills"},
}

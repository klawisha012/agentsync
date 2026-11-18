package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestExportAllAgentsAndConnectRoundTrip(t *testing.T) {
	root := connectRoot(t)

	// Claude: skill, subagent, command, instructions, MCP, plugin, marketplace, portable + machine hook.
	writeRel(t, root, ".claude/skills/lint-go/SKILL.md", "---\nname: lint-go\n---\nLint Go.\n")
	writeRel(t, root, ".claude/skills/lint-go/refs/rule.md", "rule")
	writeRel(t, root, ".claude/agents/rev.md", "---\nname: rev\ndescription: Review\ntools: Read, Grep, NotebookEdit\nmodel: opus\n---\nCheck diff.\n")
	writeRel(t, root, ".claude/commands/ship.md", "---\ndescription: Ship release\nallowed-tools: Bash\n---\nShip $ARGUMENTS\n")
	writeRel(t, root, ".claude/CLAUDE.md", "Global rules.\n")
	writeRel(t, root, ".claude.json", `{"userID":"u1","mcpServers":{"docs":{"type":"http","url":"https://docs.example/mcp"},"local":{"command":"/usr/local/bin/priv"}}}`)
	writeRel(t, root, ".claude/settings.json", `{
		"enabledPlugins":{"ponytail@official":true},
		"extraKnownMarketplaces":{"official":{"source":"https://plugins.example"}},
		"hooks":{
			"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/guard.sh"}]}],
			"Stop":[{"hooks":[{"type":"command","command":"/Users/me/private.sh"}]}]
		}
	}`)

	// Codex: TOML subagent + TOML MCP server.
	writeRel(t, root, ".codex/agents/ro.toml", "name = \"ro\"\ndescription = \"Read only\"\nsandbox_mode = \"read-only\"\nmodel = \"o4\"\ndeveloper_instructions = '''\nInspect only.\n'''\n")
	writeRel(t, root, ".codex/config.toml", "model = \"o4\"\n\n[mcp_servers.gh]\ncommand = \"npx\"\nargs = [\"-y\", \"gh-mcp\"]\nhttp_headers = { Authorization = \"${GH_TOKEN}\" }\n")

	// Gemini: TOML command + httpUrl MCP + BeforeTool hook.
	writeRel(t, root, ".gemini/commands/fix.toml", "description = \"Fix issue\"\nprompt = \"\"\"\nFix {{args}}\n\"\"\"\n")
	writeRel(t, root, ".gemini/settings.json", `{
		"mcpServers":{"search":{"httpUrl":"https://search.example/mcp"}},
		"hooks":{"BeforeTool":[{"matcher":"run_shell_command","hooks":[{"type":"command","command":"check","timeout":2000}]}]}
	}`)

	code, stdout, stderr := runCLI(t, "export")
	if code != 0 {
		t.Fatalf("export failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	for _, want := range []string{
		"claude-code: В\u00a0agents перенесены: навыки: lint-go, субагенты: rev, пользовательские команды: ship, файл инструкций, хуки (1), MCP-серверы: docs, плагины: ponytail@official, маркетплейсы: official.",
		"codex: В\u00a0agents перенесены: субагенты: ro, MCP-серверы: gh.",
		"gemini-cli: В\u00a0agents перенесены: пользовательские команды: fix, хуки (1), MCP-серверы: search.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}

	// Source files in AI agents remain intact.
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "lint-go", "SKILL.md")); err != nil {
		t.Fatalf("source skill removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "agents", "ro.toml")); err != nil {
		t.Fatalf("source codex subagent removed: %v", err)
	}

	// Universal ~/.agents holds all exported items in agents format, excluding machine MCP/hooks.
	canon := filepath.Join(root, ".agents")
	if got := readFile(t, filepath.Join(canon, "skills", "lint-go", "refs", "rule.md")); got != "rule" {
		t.Fatalf("skill nested file: %q", got)
	}
	if got := readFile(t, filepath.Join(canon, "AGENTS.md")); got != "Global rules.\n" {
		t.Fatalf("AGENTS.md: %q", got)
	}
	mcp := readFile(t, filepath.Join(canon, "mcp.json"))
	for _, want := range []string{"docs.example", "gh-mcp", "search.example", `"type": "http"`} {
		if !strings.Contains(mcp, want) {
			t.Fatalf("mcp.json missing %q:\n%s", want, mcp)
		}
	}
	if strings.Contains(mcp, "/usr/local/bin/priv") {
		t.Fatalf("machine MCP leaked into agents:\n%s", mcp)
	}
	plugins := readFile(t, filepath.Join(canon, "plugins.json"))
	if !strings.Contains(plugins, "ponytail@official") || !strings.Contains(plugins, "plugins.example") {
		t.Fatalf("plugins.json:\n%s", plugins)
	}
	hooks := readFile(t, filepath.Join(canon, "hooks.json"))
	if !strings.Contains(hooks, `"version": 1`) || strings.Contains(hooks, "/Users/me/private.sh") {
		t.Fatalf("hooks.json:\n%s", hooks)
	}

	// Re-running export is idempotent and writes no duplicate snapshot.
	snapsBefore := len(agent.Chain(root, "zwarder", "agents"))
	code, stdout, stderr = runCLI(t, "export")
	if code != 0 || strings.TrimSpace(stdout) != "В\u00a0agents нечего переносить." {
		t.Fatalf("second export: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if got := len(agent.Chain(root, "zwarder", "agents")); got != snapsBefore {
		t.Fatalf("snapshots grew from %d to %d on no-op export", snapsBefore, got)
	}

	// Distribute universal ~/.agents into a fresh target agent (cursor) and existing ones via on.
	if code, _, stderr := runCLI(t, "cursor", "on"); code != 0 {
		t.Fatalf("cursor on: %s", stderr)
	}
	if body := readFile(t, filepath.Join(root, ".cursor", "agents", "ro.md")); !strings.Contains(body, "readonly: true") {
		t.Fatalf("cursor translated subagent:\n%s", body)
	}
	if code, _, stderr := runCLI(t, "gemini", "on"); code != 0 {
		t.Fatalf("gemini on: %s", stderr)
	}
	// Gemini receives Claude's ship command translated into TOML and Claude's docs MCP server.
	if body := readFile(t, filepath.Join(root, ".gemini", "commands", "ship.toml")); !strings.Contains(body, "{{args}}") {
		t.Fatalf("gemini ship.toml:\n%s", body)
	}
	if body := readFile(t, filepath.Join(root, ".gemini", "settings.json")); !strings.Contains(body, "docs.example") {
		t.Fatalf("gemini settings.json missing docs MCP:\n%s", body)
	}

	// Revert agents restores empty ~/.agents cleanly.
	if code, _, stderr := runCLI(t, "cursor", "off"); code != 0 {
		t.Fatalf("cursor off: %s", stderr)
	}
	if code, _, stderr := runCLI(t, "gemini", "off"); code != 0 {
		t.Fatalf("gemini off: %s", stderr)
	}
	if code, _, stderr := runCLI(t, "revert", "agents"); code != 0 {
		t.Fatalf("revert agents: %s", stderr)
	}
	if _, err := os.Stat(filepath.Join(canon, "mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("revert agents kept mcp.json")
	}
}

func TestExportNamedAgentsAndConflicts(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, root, ".agents/skills/shared/SKILL.md", "canon-skill")
	writeRel(t, root, ".agents/AGENTS.md", "canon-rules")
	writeRel(t, root, ".agents/mcp.json", `{"mcpServers":{"docs":{"url":"https://canon.example"}}}`)

	writeRel(t, root, ".claude/skills/shared/SKILL.md", "claude-skill")
	writeRel(t, root, ".claude/skills/fresh/SKILL.md", "fresh-skill")
	writeRel(t, root, ".claude/CLAUDE.md", "claude-rules")
	writeRel(t, root, ".claude.json", `{"mcpServers":{"docs":{"url":"https://claude.example"}}}`)
	writeRel(t, root, ".codex/skills/unselected/SKILL.md", "codex-skill")

	code, stdout, stderr := runCLI(t, "export", "claude")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "В\u00a0agents перенесены: навыки: fresh.") {
		t.Fatalf("moved output: %s", stdout)
	}
	for _, want := range []string{
		"навык shared (имя занято в\u00a0agents)",
		"файл инструкций (имя занято в\u00a0agents)",
		"MCP-сервер docs (имя занято в\u00a0agents)",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing conflict %q in:\n%s", want, stdout)
		}
	}
	if got := readFile(t, filepath.Join(root, ".agents", "skills", "shared", "SKILL.md")); got != "canon-skill" {
		t.Fatalf("canon skill overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "unselected")); !os.IsNotExist(err) {
		t.Fatalf("unselected codex skill was exported")
	}
}

func TestExportValidationAndSecrets(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, root string)
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "help export",
			args:       []string{"help", "export"},
			wantCode:   0,
			wantStdout: "agentsync export [ии-агент…]",
		},
		{
			name:       "unknown flag",
			args:       []string{"export", "--unknown"},
			wantCode:   2,
			wantStderr: "Неизвестный аргумент «--unknown».",
		},
		{
			name:       "unknown agent",
			args:       []string{"export", "nonexistent"},
			wantCode:   1,
			wantStderr: "нет в\u00a0каталоге ИИ-агентов",
		},
		{
			name:       "agents target rejected",
			args:       []string{"export", "agents"},
			wantCode:   1,
			wantStderr: "Настройки agents уже лежат",
		},
		{
			name:       "missing agent on disk",
			args:       []string{"export", "cursor"},
			wantCode:   1,
			wantStderr: "на компьютере нет этого ИИ-агента",
		},
		{
			name: "secret blocks single agent export",
			setup: func(t *testing.T, root string) {
				writeRel(t, root, ".gemini/settings.json", `{"mcpServers":{"gh":{"command":"gh","env":{"GITHUB_TOKEN":"ghp_abcdefghijklmnop"}}}}`)
			},
			args:       []string{"export", "gemini"},
			wantCode:   1,
			wantStderr: "Секрет",
		},
		{
			name: "secret in one agent skips it and exports others in auto mode",
			setup: func(t *testing.T, root string) {
				writeRel(t, root, ".claude/skills/ok/SKILL.md", "ok")
				writeRel(t, root, ".gemini/settings.json", `{"mcpServers":{"gh":{"command":"gh","env":{"GITHUB_TOKEN":"ghp_abcdefghijklmnop"}}}}`)
			},
			args:       []string{"export"},
			wantCode:   1,
			wantStdout: "claude-code: В\u00a0agents перенесены: навыки: ok.",
			wantStderr: "Не все ИИ-агенты перенесены в\u00a0agents.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := connectRoot(t)
			if tt.setup != nil {
				tt.setup(t, root)
			}
			code, stdout, stderr := runCLI(t, tt.args...)
			if code != tt.wantCode {
				t.Fatalf("code=%d want=%d stdout=%q stderr=%q", code, tt.wantCode, stdout, stderr)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout, tt.wantStdout) {
				t.Fatalf("stdout=%q want substring %q", stdout, tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Fatalf("stderr=%q want substring %q", stderr, tt.wantStderr)
			}
		})
	}
}

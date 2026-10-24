package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pushServer accepts every push.
func pushServer(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
}

func TestPushMovesOwnHooksAndMCPIntoAgents(t *testing.T) {
	root := connectRoot(t)
	pushServer(t)
	writeRel(t, root, ".claude/settings.json", `{"theme":"dark","hooks":{
		"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/guard.sh"}]}],
		"PermissionRequest":[{"hooks":[{"type":"command","command":"ask"}]}],
		"Stop":[{"hooks":[{"type":"command","command":"/Users/me/run.sh"}]}]}}`)
	writeRel(t, root, ".claude.json", `{"userID":"x","mcpServers":{"docs":{"type":"http","url":"https://docs.example/mcp"}}}`)

	code, stdout, stderr := runCLI(t, "push", "claude")
	if code != 0 || !strings.Contains(stdout, "перенесены: хуки (2), MCP-серверы: docs") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	canon := settingsHooks(t, filepath.Join(root, ".agents", "hooks.json"))
	before := canon["before_tool"]
	if len(before) != 1 || before[0]["input"] != "claude-code" || before[0]["matcher"].([]any)[0] != "shell" ||
		!strings.Contains(before[0]["hooks"].([]any)[0].(map[string]any)["command"].(string), "${AGENTS_PROJECT_DIR}") {
		t.Fatalf("canon hooks %v", canon)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "hooks.json")); !strings.Contains(body, `"claude-code"`) || !strings.Contains(body, "PermissionRequest") || strings.Contains(body, "/Users/me") {
		t.Fatalf("canon hooks file %s", body)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "mcp.json")); !strings.Contains(body, "docs.example") {
		t.Fatalf("canon mcp %s", body)
	}
	claude := settingsHooks(t, filepath.Join(root, ".claude", "settings.json"))
	if len(claude["PreToolUse"]) != 1 || len(claude["PermissionRequest"]) != 1 || len(claude["Stop"]) != 1 {
		t.Fatalf("claude after transfer %v", claude)
	}

	// The moved hook is a record of the connection now: removing it from agents removes it everywhere.
	writeRel(t, root, ".agents/hooks.json", `{"version":1}`)
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	code, stdout, stderr = runCLI(t, "push", "claude")
	if code != 0 || strings.Contains(stdout, "хуки") {
		t.Fatalf("second push: code %d stdout %s stderr %s", code, stdout, stderr)
	}
	if claude := settingsHooks(t, filepath.Join(root, ".claude", "settings.json")); len(claude) != 1 || len(claude["Stop"]) != 1 {
		t.Fatalf("a removed hook came back %v", claude)
	}
}

func TestPushMovesOwnRecordsOfGeminiCodexAndCopilot(t *testing.T) {
	root := connectRoot(t)
	pushServer(t)
	writeRel(t, root, ".gemini/settings.json", `{"hooks":{"BeforeTool":[{"matcher":"run_shell_command|replace","hooks":[{"type":"command","command":"lint","timeout":3000}]}]},
		"mcpServers":{"search":{"httpUrl":"https://search.example/mcp"}}}`)
	writeRel(t, root, ".codex/config.toml", "model = \"o4\"\n\n[mcp_servers.gh]\ncommand = \"npx\"\nargs = [\"-y\", \"gh\"]\nhttp_headers = { X = \"${TOKEN}\" }\n")
	writeRel(t, root, ".copilot/hooks/agentsync.json", `{"version":1,"hooks":{"postToolUse":[{"type":"command","matcher":"edit","bash":"fmt","timeoutSec":5}]}}`)
	writeRel(t, root, ".copilot/skills/x/SKILL.md", "x")
	for _, name := range []string{"gemini", "codex", "copilot"} {
		if code, _, stderr := runCLI(t, "push", name); code != 0 {
			t.Fatalf("%s: %s", name, stderr)
		}
	}
	canon := settingsHooks(t, filepath.Join(root, ".agents", "hooks.json"))
	if before := canon["before_tool"]; len(before) != 1 || before[0]["input"] != "gemini-cli" || len(before[0]["matcher"].([]any)) != 2 ||
		before[0]["hooks"].([]any)[0].(map[string]any)["timeout"] != float64(3) {
		t.Fatalf("gemini hook %v", canon)
	}
	if after := canon["after_tool"]; len(after) != 1 || after[0]["input"] != "github-copilot" || after[0]["hooks"].([]any)[0].(map[string]any)["command"] != "fmt" {
		t.Fatalf("copilot hook %v", canon)
	}
	mcp := readFile(t, filepath.Join(root, ".agents", "mcp.json"))
	for _, want := range []string{`"type": "http"`, "search.example", `"gh"`, `"X": "${TOKEN}"`} {
		if !strings.Contains(mcp, want) {
			t.Fatalf("canon mcp misses %s: %s", want, mcp)
		}
	}
	if config := readFile(t, filepath.Join(root, ".codex", "config.toml")); !strings.Contains(config, "[mcp_servers.gh]") || !strings.Contains(config, `model = "o4"`) {
		t.Fatalf("codex config %s", config)
	}
}

func TestPushKeepsAnOwnServerWhoseNameAgentsHas(t *testing.T) {
	root := connectRoot(t)
	pushServer(t)
	writeRel(t, root, ".agents/mcp.json", `{"mcpServers":{"docs":{"url":"https://agents.example"}}}`)
	writeRel(t, root, ".gemini/settings.json", `{"mcpServers":{"docs":{"url":"https://own.example"}}}`)
	code, stdout, stderr := runCLI(t, "push", "gemini")
	if code != 0 || !strings.Contains(stdout, "Не перенесены в agents: MCP-сервер docs (имя занято в agents)") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "mcp.json")); strings.Contains(body, "own.example") {
		t.Fatalf("agents lost its server %s", body)
	}
}

func TestRevertAgentsUndoesATransfer(t *testing.T) {
	root := connectRoot(t)
	pushServer(t)
	writeRel(t, root, ".agents/skills/note/SKILL.md", "note")
	writeRel(t, root, ".gemini/settings.json", `{"hooks":{"AfterAgent":[{"hooks":[{"type":"command","command":"ping"}]}]}}`)
	if code, _, stderr := runCLI(t, "push", "gemini"); code != 0 {
		t.Fatal(stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "hooks.json")); err != nil {
		t.Fatal("nothing moved")
	}
	if code, _, stderr := runCLI(t, "revert", "agents"); code != 0 {
		t.Fatal(stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "hooks.json")); !os.IsNotExist(err) {
		t.Fatal("revert agents kept the moved hook")
	}
}

func TestPushWithASecretInAnOwnServerChangesNothing(t *testing.T) {
	root := connectRoot(t)
	pushServer(t)
	settings := `{"hooks":{"AfterAgent":[{"hooks":[{"type":"command","command":"ping"}]}]},"mcpServers":{"gh":{"command":"gh","env":{"GITHUB_TOKEN":"ghp_abcdefghijklmnop"}}}}`
	writeRel(t, root, ".gemini/settings.json", settings)
	code, _, stderr := runCLI(t, "push", "gemini")
	if code != 1 || !strings.Contains(stderr, "Секрет") {
		t.Fatalf("code %d stderr %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatal("agents changed before the secret check")
	}
	if readFile(t, filepath.Join(root, ".gemini", "settings.json")) != settings {
		t.Fatal("gemini changed before the secret check")
	}
}

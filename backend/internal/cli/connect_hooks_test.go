package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const canonHooksBody = `{"hooks":{
	"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"lint"}]}],
	"Stop":[{"hooks":[{"type":"command","command":"notify"}]}],
	"PermissionRequest":[{"hooks":[{"type":"command","command":"ask"}]}]}}`

func settingsHooks(t *testing.T, file string) map[string][]map[string]any {
	t.Helper()
	var top struct {
		Hooks map[string][]map[string]any `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(readFile(t, file)), &top); err != nil {
		t.Fatal(err)
	}
	return top.Hooks
}

func TestOnWritesCanonHooksIntoClaudeSettings(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "hooks.json", canonHooksBody)
	settings := filepath.Join(root, ".claude", "settings.json")
	writeRel(t, root, ".claude/settings.json", `{"hooks":{"PreToolUse":[{"matcher":"Write","hooks":[{"type":"command","command":"mine"}]}]}}`)

	code, stdout, stderr := runCLI(t, "claude", "on")
	if code != 0 || !strings.Contains(stdout, "хуки") || strings.Contains(stdout, "Пропущены") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	hooks := settingsHooks(t, settings)
	if len(hooks["PreToolUse"]) != 2 || len(hooks["Stop"]) != 1 || len(hooks["PermissionRequest"]) != 1 {
		t.Fatalf("hooks %v", hooks)
	}

	writeRel(t, canon, "hooks.json", `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"lint"}]}]}}`)
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if hooks := settingsHooks(t, settings); len(hooks["Stop"]) != 0 || len(hooks["PreToolUse"]) != 2 {
		t.Fatalf("second on %v", hooks)
	}

	if code, _, stderr := runCLI(t, "claude", "off"); code != 0 {
		t.Fatal(stderr)
	}
	hooks = settingsHooks(t, settings)
	if len(hooks["PreToolUse"]) != 1 || hooks["PreToolUse"][0]["matcher"] != "Write" || len(hooks) != 1 {
		t.Fatalf("off %v", hooks)
	}
}

func TestOnTranslatesCanonHooksForGemini(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "hooks.json", canonHooksBody)
	code, stdout, stderr := runCLI(t, "gemini", "on")
	if code != 0 || !strings.Contains(stdout, "хуки событий PermissionRequest") {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout, stderr)
	}
	hooks := settingsHooks(t, filepath.Join(root, ".gemini", "settings.json"))
	if len(hooks["BeforeTool"]) != 1 || hooks["BeforeTool"][0]["matcher"] != "run_shell_command" || len(hooks["AfterAgent"]) != 1 {
		t.Fatalf("hooks %v", hooks)
	}
	if hooks["PreToolUse"] != nil || hooks["PermissionRequest"] != nil {
		t.Fatalf("untranslated events %v", hooks)
	}
}

func TestMachineHookStaysOnTheMachine(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	writeRel(t, root, ".claude/settings.json", `{"theme":"dark","hooks":{"Stop":[{"hooks":[{"type":"command","command":"/Users/me/run.sh"}]},{"hooks":[{"type":"command","command":"say done"}]}]}}`)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "claude"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	if strings.Contains(sent, "/Users/me") || !strings.Contains(sent, "say done") {
		t.Fatalf("snapshot %s", sent)
	}

	t.Setenv("AGENTSYNC_TOKEN", "")
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[
		{"path":"settings.json","body":"{\"theme\":\"light\"}"},
		{"path":"agentsync.json","body":"{}"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	body := readFile(t, filepath.Join(root, ".claude", "settings.json"))
	if !strings.Contains(body, "/Users/me/run.sh") || !strings.Contains(body, "light") {
		t.Fatalf("machine hook lost %s", body)
	}
}

func TestPipeApplyOldGeminiCardSendsHooksToTheCanonInClaudeForm(t *testing.T) {
	root := connectRoot(t)
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"gemini","files":[
		{"path":"settings.json","body":"{\"theme\":\"x\",\"hooks\":{\"BeforeTool\":[{\"matcher\":\"run_shell_command\",\"hooks\":[{\"type\":\"command\",\"command\":\"lint\"}]}]}}"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	hooks := settingsHooks(t, filepath.Join(root, ".agents", "hooks.json"))
	if len(hooks["PreToolUse"]) != 1 || hooks["PreToolUse"][0]["matcher"] != "Bash" {
		t.Fatalf("canon hooks %v", hooks)
	}
	if gemini := settingsHooks(t, filepath.Join(root, ".gemini", "settings.json")); len(gemini["BeforeTool"]) != 1 {
		t.Fatalf("gemini not connected %v", gemini)
	}
}

func TestOnTwiceKeepsOneCopyOfAHookWithAmpersands(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "hooks.json", `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"lint && say <done>"}]}]}}`)
	settings := filepath.Join(root, ".claude", "settings.json")
	for range 2 {
		if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
			t.Fatal(stderr)
		}
	}
	if hooks := settingsHooks(t, settings); len(hooks["Stop"]) != 1 {
		t.Fatalf("hooks %v", hooks)
	}
	if code, _, stderr := runCLI(t, "claude", "off"); code != 0 {
		t.Fatal(stderr)
	}
	if hooks := settingsHooks(t, settings); len(hooks) != 0 {
		t.Fatalf("off left %v", hooks)
	}
}

func TestRevertAndCanonApplyKeepMachineHooks(t *testing.T) {
	root := connectRoot(t)
	machine := `{"hooks":[{"type":"command","command":"/Users/me/run.sh"}]}`
	writeRel(t, root, ".claude/settings.json", `{"theme":"dark","hooks":{"Stop":[`+machine+`]}}`)
	writeRel(t, root, ".agents/hooks.json", `{"hooks":{"Stop":[`+machine+`]}}`)
	writeRel(t, root, ".agents/skills/note/SKILL.md", "note")
	if code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[{"path":"settings.json","body":"{\"theme\":\"light\"}"},{"path":"agentsync.json","body":"{}"}]}`); code != 0 {
		t.Fatalf("apply %s", stderr)
	}
	if code, _, stderr := runCLI(t, "revert", "claude"); code != 0 {
		t.Fatalf("revert %s", stderr)
	}
	if body := readFile(t, filepath.Join(root, ".claude", "settings.json")); !strings.Contains(body, "/Users/me/run.sh") || !strings.Contains(body, "dark") {
		t.Fatalf("revert lost the machine hook %s", body)
	}
	if code, stderr := pipeApply(t, `{"author":"zwarder","agent":"agents","files":[{"path":"skills/other/SKILL.md","body":"other"}]}`); code != 0 {
		t.Fatalf("canon apply %s", stderr)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "hooks.json")); !strings.Contains(body, "/Users/me/run.sh") {
		t.Fatalf("canon apply lost the machine hook %s", body)
	}
}

func TestPushKeepsPortableHooksAndDropsMachineHookFiles(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	writeRel(t, root, ".claude/settings.json", `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/check.sh"}]}],"Stop":[{"hooks":[{"type":"command","command":"npm test > /dev/null"}]}]}}`)
	writeRel(t, root, ".claude/hooks/extra.json", `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"say hi"}]}]},"env":{"LOG":"/Users/me/log"}}`)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "claude"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	if !strings.Contains(sent, "check.sh") || !strings.Contains(sent, "npm test") {
		t.Fatalf("portable hooks dropped %s", sent)
	}
	if strings.Contains(sent, "/Users/me/log") {
		t.Fatalf("hook file with a machine path published %s", sent)
	}
}

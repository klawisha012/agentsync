package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestMain lets the test binary play a hook script: with AGENTSYNC_TEST_HOOK set it saves its
// input, prints the given output and exits with the given code.
func TestMain(m *testing.M) {
	if seen := os.Getenv("AGENTSYNC_TEST_HOOK"); seen != "" {
		raw, _ := io.ReadAll(os.Stdin)
		_ = os.WriteFile(seen, raw, 0o644)
		os.Stdout.WriteString(os.Getenv("AGENTSYNC_TEST_HOOK_STDOUT"))
		os.Stderr.WriteString(os.Getenv("AGENTSYNC_TEST_HOOK_STDERR"))
		code, _ := strconv.Atoi(os.Getenv("AGENTSYNC_TEST_HOOK_EXIT"))
		os.Exit(code)
	}
	os.Exit(m.Run())
}

// hookScript is a shell command that runs the test binary as the hook script.
func hookScript(t *testing.T, exit int, stdout, stderr string) (string, string) {
	t.Helper()
	seen := filepath.Join(t.TempDir(), "seen.json")
	t.Setenv("AGENTSYNC_TEST_HOOK", seen)
	t.Setenv("AGENTSYNC_TEST_HOOK_EXIT", strconv.Itoa(exit))
	t.Setenv("AGENTSYNC_TEST_HOOK_STDOUT", stdout)
	t.Setenv("AGENTSYNC_TEST_HOOK_STDERR", stderr)
	return `"` + filepath.ToSlash(os.Args[0]) + `"`, seen
}

func runHookCLI(t *testing.T, input string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(append([]string{"hook"}, args...), strings.NewReader(input), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestHookGivesAClaudeScriptGeminiInputAndBlocksForGemini(t *testing.T) {
	script, seen := hookScript(t, 2, "", "rm is not allowed")
	code, stdout, stderr := runHookCLI(t,
		`{"hook_event_name":"BeforeTool","tool_name":"run_shell_command","tool_input":{"command":"rm -rf /"},"session_id":"s1"}`,
		"--from", "gemini-cli", "--to", "claude-code", "--event", "before_tool", "--", script)
	if code != 2 || !strings.Contains(stderr, "rm is not allowed") || stdout != "" {
		t.Fatalf("code %d stdout %q stderr %q", code, stdout, stderr)
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(readFile(t, seen)), &input); err != nil {
		t.Fatal(err)
	}
	if input["hook_event_name"] != "PreToolUse" || input["tool_name"] != "Bash" || input["session_id"] != "s1" || input["tool_input"].(map[string]any)["command"] != "rm -rf /" {
		t.Fatalf("script input %v", input)
	}
}

func TestHookTranslatesForTheAgentsFormatAndCopilot(t *testing.T) {
	script, seen := hookScript(t, 0, `{"decision":"block","reason":"no"}`, "")
	code, stdout, _ := runHookCLI(t,
		`{"timestamp":1,"cwd":"/w","toolName":"bash","toolArgs":"{\"command\":\"ls\"}"}`,
		"--from", "github-copilot", "--to", "agents", "--event", "before_tool", "--", script)
	if code != 0 || !strings.Contains(stdout, `"permissionDecision":"deny"`) || !strings.Contains(stdout, `"no"`) {
		t.Fatalf("code %d stdout %q", code, stdout)
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(readFile(t, seen)), &input); err != nil {
		t.Fatal(err)
	}
	if input["event"] != "before_tool" || input["tool"] != "shell" || input["tool_input"].(map[string]any)["command"] != "ls" || input["cwd"] != "/w" {
		t.Fatalf("script input %v", input)
	}
	if _, ok := input["session_id"]; ok {
		t.Fatalf("a field Copilot does not give is filled %v", input)
	}
}

func TestHookPassesAnAllowedAnswerThrough(t *testing.T) {
	script, _ := hookScript(t, 0, "context line", "")
	code, stdout, _ := runHookCLI(t, `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{}}`,
		"--from", "grok", "--to", "codex", "--event", "before_tool", "--", script)
	if code != 0 || stdout != "context line" {
		t.Fatalf("code %d stdout %q", code, stdout)
	}
}

func TestHookFailureBlocksOnlyBeforeAnAction(t *testing.T) {
	script, _ := hookScript(t, 0, "", "")
	code, _, stderr := runHookCLI(t, "not json", "--from", "claude-code", "--to", "gemini-cli", "--event", "before_tool", "--", script)
	if code != 2 || !strings.Contains(stderr, "agentsync hook") {
		t.Fatalf("before_tool: code %d stderr %q", code, stderr)
	}
	code, _, stderr = runHookCLI(t, "not json", "--from", "claude-code", "--to", "gemini-cli", "--event", "notification", "--", script)
	if code != 0 || !strings.Contains(stderr, "agentsync hook") {
		t.Fatalf("notification: code %d stderr %q", code, stderr)
	}
}

func TestOnWrapsAHookThatReadsAnotherFormat(t *testing.T) {
	root := connectRoot(t)
	// agentsync itself is not on this PATH, so on warns.
	t.Setenv("PATH", t.TempDir())
	writeRel(t, filepath.Join(root, ".agents"), "hooks.json", `{"version":1,"hooks":{"before_tool":[
		{"matcher":["shell"],"input":"claude","hooks":[{"type":"command","command":"guard.sh 'x'"}]}]}}`)
	code, stdout, stderr := runCLI(t, "gemini", "on")
	if code != 0 {
		t.Fatal(stderr)
	}
	gemini := settingsHooks(t, filepath.Join(root, ".gemini", "settings.json"))
	command := gemini["BeforeTool"][0]["hooks"].([]any)[0].(map[string]any)["command"]
	if command != `agentsync hook --from gemini-cli --to claude-code --event before_tool -- 'guard.sh '\''x'\'''` {
		t.Fatalf("gemini command %v", command)
	}
	if !strings.Contains(stdout, "PATH") {
		t.Fatalf("no warning about agentsync on PATH: %s", stdout)
	}
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	claude := settingsHooks(t, filepath.Join(root, ".claude", "settings.json"))
	if command := claude["PreToolUse"][0]["hooks"].([]any)[0].(map[string]any)["command"]; command != "guard.sh 'x'" {
		t.Fatalf("claude got a wrapper for its own format: %v", command)
	}
}

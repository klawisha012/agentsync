package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const reviewerAgent = "---\nname: reviewer\ndescription: Reads code.\ntools: [read, grep]\nclaude:\n  model: opus\n---\nReview.\n"

func seedSubagents(t *testing.T, root string) {
	t.Helper()
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "agents/reviewer.md", reviewerAgent)
	writeRel(t, canon, "agents/writer.md", "---\nname: writer\ndescription: Writes code.\ntools: [read, shell, web_search]\n---\nWrite.\n")
	writeRel(t, canon, "agents/plain.md", "plain")
}

func TestOnTranslatesSubagentsForEachAgent(t *testing.T) {
	root := connectRoot(t)
	seedSubagents(t, root)
	for _, tc := range []struct {
		agent, file string
		want        []string
		skipped     bool
	}{
		{"claude", ".claude/agents/reviewer.md", []string{"tools: Read, Grep", "model: opus"}, false},
		{"grok", ".grok/agents/reviewer.md", []string{"tools: [read_file, grep]"}, false},
		{"gemini", ".gemini/agents/reviewer.md", []string{"tools: [read_file, search_file_content]"}, false},
		{"copilot", ".copilot/agents/reviewer.agent.md", []string{"tools: [read, search]"}, false},
		{"cursor", ".cursor/agents/reviewer.md", []string{"readonly: true"}, true},
		{"codex", ".codex/agents/reviewer.toml", []string{`sandbox_mode = "read-only"`, "developer_instructions = '''\nReview.\n'''"}, true},
	} {
		code, stdout, stderr := runCLI(t, tc.agent, "on")
		if code != 0 {
			t.Fatalf("%s: code %d stderr %s", tc.agent, code, stderr)
		}
		body := readFile(t, filepath.Join(root, filepath.FromSlash(tc.file)))
		for _, want := range tc.want {
			if !strings.Contains(body, want) {
				t.Fatalf("%s: %s misses %q:\n%s", tc.agent, tc.file, want, body)
			}
		}
		if strings.Contains(body, "claude:") || (tc.agent != "claude" && strings.Contains(body, "opus")) {
			t.Fatalf("%s: foreign extension in %s", tc.agent, body)
		}
		if got := strings.Contains(stdout, "субагенты writer"); got != tc.skipped {
			t.Fatalf("%s: writer skipped %v: %s", tc.agent, got, stdout)
		}
	}
	if body := readFile(t, filepath.Join(root, ".claude", "agents", "writer.md")); !strings.Contains(body, "tools: Read, Bash, WebSearch") {
		t.Fatalf("claude writer %s", body)
	}
	if info, err := os.Lstat(filepath.Join(root, ".claude", "agents", "plain.md")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("a subagent that needs no translation is not a link")
	}
}

func TestOnRefusesASubagentOutsideTheAgentsFormat(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"---\nname: x\ntools: Read, Bash\n---\nx", "«Read» не\u00a0из\u00a0словаря"},
		{"---\nname: x\nmodel: opus\n---\nx", "поле «model» не\u00a0из\u00a0ядра"},
	} {
		root := connectRoot(t)
		writeRel(t, filepath.Join(root, ".agents"), "agents/x.md", tc.body)
		code, _, stderr := runCLI(t, "claude", "on")
		if code != 1 || !strings.Contains(stderr, tc.want) {
			t.Fatalf("%s: code %d stderr %s", tc.want, code, stderr)
		}
		if _, err := os.Lstat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
			t.Fatalf("%s: refusal changed the agent", tc.want)
		}
	}
}

func TestOffRemovesTranslatedSubagentsAndKeepsOwnFiles(t *testing.T) {
	root := connectRoot(t)
	seedSubagents(t, root)
	writeRel(t, root, ".claude/agents/mine.md", "mine")
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if err := os.Remove(filepath.Join(root, ".agents", "agents", "writer.md")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "agents", "writer.md")); !os.IsNotExist(err) {
		t.Fatal("translated file of a removed subagent stays")
	}
	if code, _, stderr := runCLI(t, "claude", "off"); code != 0 {
		t.Fatal(stderr)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".claude", "agents"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "mine.md" {
		t.Fatalf("off left %v %v", namesOf(entries), err)
	}
}

func TestPushCarriesTheAgentsFileOfATranslatedSubagent(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "agents/reviewer.md", reviewerAgent)
	if code, _, stderr := runCLI(t, "codex", "on"); code != 0 {
		t.Fatal(stderr)
	}
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "codex"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	if !strings.Contains(sent, `"path":"agents/reviewer.md"`) || !strings.Contains(sent, "tools: [read, grep]") || strings.Contains(sent, "reviewer.toml") {
		t.Fatalf("card %s", sent)
	}
}

func TestOnTranslatesCommandsForGeminiAndLeavesThemToGrok(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "commands/review.md", "---\ndescription: Review a change\n---\nReview $ARGUMENTS carefully.\n")
	writeRel(t, canon, "commands/pick.md", "Pick $1 of $2.\n")

	code, stdout, stderr := runCLI(t, "gemini", "on")
	if code != 0 || !strings.Contains(stdout, "пользовательские команды pick") {
		t.Fatalf("gemini: code %d stdout %s stderr %s", code, stdout, stderr)
	}
	body := readFile(t, filepath.Join(root, ".gemini", "commands", "review.toml"))
	if !strings.Contains(body, `description = "Review a change"`) || !strings.Contains(body, "{{args}}") || strings.Contains(body, "$ARGUMENTS") {
		t.Fatalf("gemini command %s", body)
	}
	if _, err := os.Lstat(filepath.Join(root, ".gemini", "commands", "pick.toml")); !os.IsNotExist(err) {
		t.Fatal("command with $1 given to Gemini")
	}

	if code, _, stderr := runCLI(t, "claude", "on"); code != 0 {
		t.Fatal(stderr)
	}
	if info, err := os.Lstat(filepath.Join(root, ".claude", "commands", "review.md")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("claude command is not a link")
	}

	code, stdout, stderr = runCLI(t, "grok", "on")
	if code != 0 || !strings.Contains(stdout, "пользовательские команды") || strings.Contains(stdout, "Пропущены") {
		t.Fatalf("grok: code %d stdout %s stderr %s", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(root, ".grok", "commands")); !os.IsNotExist(err) {
		t.Fatal("commands written for Grok, which reads agents itself")
	}
}

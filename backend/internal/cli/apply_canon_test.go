package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func pipeApply(t *testing.T, payload string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(nil, bytes.NewBufferString(payload), &out, &errOut)
	return code, errOut.String()
}

func seedClaudeAndCanon(t *testing.T) string {
	t.Helper()
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "skills/keep/SKILL.md", "keep")
	writeRel(t, canon, "skills/grill/SKILL.md", "old")
	writeRel(t, canon, "skills/grill/notes.md", "old notes")
	writeRel(t, canon, "AGENTS.md", "old rules")
	writeRel(t, canon, "mcp.json", `{"mcpServers":{"local":{"command":"mine"}}}`)
	writeRel(t, canon, ".env", "TOKEN=x")
	home := filepath.Join(root, ".claude")
	writeRel(t, home, "settings.json", "mine")
	writeRel(t, home, "keybindings.json", "keys")
	writeRel(t, home, ".credentials.json", "secret")
	writeRel(t, home, "CLAUDE.md", "my rules")
	writeRel(t, canon, "skills/grill/.env", "KEY=1")
	return root
}

func TestPipeApplyClaudeCardGoesThroughTheCanon(t *testing.T) {
	root := seedClaudeAndCanon(t)
	canon := filepath.Join(root, ".agents")
	home := filepath.Join(root, ".claude")
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[
		{"path":"skills/grill/SKILL.md","body":"new"},
		{"path":"skills/fresh/SKILL.md","body":"fresh"},
		{"path":"CLAUDE.md","body":"new rules"},
		{"path":"agents/go.md","body":"go"},
		{"path":"settings.json","body":"theirs"},
		{"path":"agentsync.json","body":"{\"mcp\":[]}"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	for rel, want := range map[string]string{
		"skills/grill/SKILL.md": "new",
		"skills/fresh/SKILL.md": "fresh",
		"skills/keep/SKILL.md":  "keep",
		"AGENTS.md":             "new rules",
		"agents/go.md":          "go",
		".env":                  "TOKEN=x",
	} {
		if body := readFile(t, filepath.Join(canon, filepath.FromSlash(rel))); body != want {
			t.Fatalf("canon %s %q", rel, body)
		}
	}
	if _, err := os.Stat(filepath.Join(canon, "skills", "grill", "notes.md")); !os.IsNotExist(err) {
		t.Fatal("same-name element not replaced whole")
	}
	if body := readFile(t, filepath.Join(canon, "skills", "grill", ".env")); body != "KEY=1" {
		t.Fatalf("private file of a replaced element lost: %q", body)
	}
	if !strings.Contains(readFile(t, filepath.Join(canon, "mcp.json")), "local") {
		t.Fatal("canon MCP lost")
	}
	for rel, want := range map[string]string{
		"settings.json":         "theirs",
		"keybindings.json":      "keys",
		".credentials.json":     "secret",
		"CLAUDE.md":             "new rules",
		"skills/fresh/SKILL.md": "fresh",
		"agents/go.md":          "go",
	} {
		if body := readFile(t, filepath.Join(home, filepath.FromSlash(rel))); body != want {
			t.Fatalf("home %s %q", rel, body)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "agentsync.json")); !os.IsNotExist(err) {
		t.Fatal("mark written into the home")
	}
	canonSnaps := agent.Chain(root, "zwarder", "agents")
	if len(canonSnaps) != 1 || !hasBody(canonSnaps[0].Files, "skills/grill/SKILL.md", "old") || hasBody(canonSnaps[0].Files, ".env", "TOKEN=x") {
		t.Fatalf("canon snapshot %+v", canonSnaps)
	}
	agentSnaps := agent.Chain(root, "zwarder", "claude")
	if len(agentSnaps) != 1 || !hasBody(agentSnaps[0].Files, "settings.json", "mine") || !hasBody(agentSnaps[0].Files, "CLAUDE.md", "my rules") {
		t.Fatalf("agent snapshot %+v", agentSnaps)
	}

	if code, _, stderr := runCLI(t, "revert", "claude"); code != 0 {
		t.Fatalf("revert claude %s", stderr)
	}
	for rel, want := range map[string]string{"settings.json": "mine", "CLAUDE.md": "my rules", ".credentials.json": "secret"} {
		if body := readFile(t, filepath.Join(home, rel)); body != want {
			t.Fatalf("revert claude %s %q", rel, body)
		}
	}
	if code, _, stderr := runCLI(t, "revert", "agents"); code != 0 {
		t.Fatalf("revert canon %s", stderr)
	}
	for rel, want := range map[string]string{"skills/grill/SKILL.md": "old", "skills/grill/notes.md": "old notes", "AGENTS.md": "old rules", ".env": "TOKEN=x"} {
		if body := readFile(t, filepath.Join(canon, filepath.FromSlash(rel))); body != want {
			t.Fatalf("revert canon %s %q", rel, body)
		}
	}
	if _, err := os.Stat(filepath.Join(canon, "skills", "fresh", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("revert canon kept the applied skill")
	}
}

func TestPipeApplyOldClaudePublicationSendsAllMCPToTheCanon(t *testing.T) {
	root := seedClaudeAndCanon(t)
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[
		{"path":"CLAUDE.md","body":"old pub rules"},
		{"path":"mcp.json","body":"{\"mcpServers\":{\"gh\":{\"command\":\"gh\"},\"local\":{\"command\":\"theirs\"}}}"},
		{"path":"settings.json","body":"theirs"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	canonMCP := readFile(t, filepath.Join(root, ".agents", "mcp.json"))
	if !strings.Contains(canonMCP, `"gh"`) || !strings.Contains(canonMCP, "theirs") {
		t.Fatalf("canon mcp %s", canonMCP)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "mcp.json")); !os.IsNotExist(err) {
		t.Fatal("MCP file written into the home")
	}
	if body := readFile(t, filepath.Join(root, ".agents", "AGENTS.md")); body != "old pub rules" {
		t.Fatalf("canon rules %q", body)
	}
	if !strings.Contains(readFile(t, filepath.Join(root, ".claude.json")), `"gh"`) {
		t.Fatal("Claude not connected to the canon MCP")
	}
}

func TestPipeApplyBreakLeavesCanonAndHomeAsBefore(t *testing.T) {
	root := seedClaudeAndCanon(t)
	writeRel(t, filepath.Join(root, ".claude"), "rules", "a file where a folder must go")
	code, _ := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[
		{"path":"skills/grill/SKILL.md","body":"new"},
		{"path":"settings.json","body":"theirs"},
		{"path":"rules/a.md","body":"a"}]}`)
	if code == 0 {
		t.Fatal("apply succeeded")
	}
	if body := readFile(t, filepath.Join(root, ".agents", "skills", "grill", "SKILL.md")); body != "old" {
		t.Fatalf("canon %q", body)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "skills", "grill", "notes.md")); body != "old notes" {
		t.Fatalf("canon notes %q", body)
	}
	if body := readFile(t, filepath.Join(root, ".claude", "settings.json")); body != "mine" {
		t.Fatalf("home %q", body)
	}
	if len(agent.Chain(root, "zwarder", "agents")) != 0 || len(agent.Chain(root, "zwarder", "claude")) != 0 {
		t.Fatal("snapshots of the attempt stay")
	}
}

func TestPushMarksTheMCPRecordsOnWrote(t *testing.T) {
	var sent struct {
		Files []agent.File `json:"files"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &sent)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	writeRel(t, filepath.Join(root, ".agents"), "mcp.json", `{"mcpServers":{"docs":{"url":"https://docs.example/mcp"}}}`)
	writeRel(t, filepath.Join(root, ".gemini"), "settings.json", `{"mcpServers":{"own":{"url":"https://own.example"}}}`)
	if code, _, stderr := runCLI(t, "gemini", "on"); code != 0 {
		t.Fatal(stderr)
	}
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "gemini"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	if !hasBody(sent.Files, "agentsync.json", `{"mcp":["docs"]}`) {
		t.Fatalf("mark %+v", sent.Files)
	}
}

func TestPipeApplyCanonCardReplacesTheCanonExactly(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "skills/old/SKILL.md", "old")
	writeRel(t, canon, "AGENTS.md", "old rules")
	writeRel(t, canon, ".env", "MINE=1")
	writeRel(t, canon, "mcp.json", `{"mcpServers":{"local":{"command":"/opt/tool/run"}}}`)
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"agents","files":[
		{"path":"skills/note/SKILL.md","body":"note"},
		{"path":"mcp.json","body":"{\"mcpServers\":{\"docs\":{\"url\":\"https://docs.example/mcp\"}}}"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	if body := readFile(t, filepath.Join(canon, "skills", "note", "SKILL.md")); body != "note" {
		t.Fatalf("note %q", body)
	}
	for _, rel := range []string{"skills/old/SKILL.md", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(canon, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("%s stays after an exact replace", rel)
		}
	}
	if body := readFile(t, filepath.Join(canon, ".env")); body != "MINE=1" {
		t.Fatalf(".env %q", body)
	}
	if mcp := readFile(t, filepath.Join(canon, "mcp.json")); !strings.Contains(mcp, "docs") || !strings.Contains(mcp, "/opt/tool/run") {
		t.Fatalf("mcp %s", mcp)
	}
	snaps := agent.Chain(root, "zwarder", "agents")
	if len(snaps) != 1 || !hasBody(snaps[0].Files, "skills/old/SKILL.md", "old") {
		t.Fatalf("canon snapshot %+v", snaps)
	}
	if code, _, stderr := runCLI(t, "revert", "agents"); code != 0 {
		t.Fatalf("revert %s", stderr)
	}
	if body := readFile(t, filepath.Join(canon, "AGENTS.md")); body != "old rules" {
		t.Fatalf("revert %q", body)
	}
	if _, err := os.Stat(filepath.Join(canon, "skills", "note", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("revert kept the applied skill")
	}
}

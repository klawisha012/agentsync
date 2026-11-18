package cli

import (
	"bytes"
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

func TestPipeApplyRejectsNonAgentsCard(t *testing.T) {
	root := seedClaudeAndCanon(t)
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[{"path":"CLAUDE.md","body":"new"}]}`)
	if code == 0 || !strings.Contains(stderr, "только agents") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if body := readFile(t, filepath.Join(root, ".agents", "AGENTS.md")); body != "old rules" {
		t.Fatalf("canon changed: %q", body)
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

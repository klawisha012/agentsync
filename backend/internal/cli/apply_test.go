package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klawisha012/agentsync/internal/agent"
)

func TestParseApplyArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		version int
		paths   string
		err     string
	}{
		{
			name: "whole publication",
			args: []string{"Author", "Grok"},
		},
		{
			name:    "paths after version",
			args:    []string{"Author", "Grok", "--version", "4", "skills/ru-text", "agents/reviewer.md", "AGENTS.md"},
			version: 4,
			paths:   "skills/ru-text,agents/reviewer.md,AGENTS.md",
		},
		{
			name:    "equals form before paths",
			args:    []string{"Author", "Grok", "skills/ru-text", "--version=2"},
			version: 2,
			paths:   "skills/ru-text",
		},
		{
			name: "paths without version",
			args: []string{"Author", "Grok", "skills/ru-text"},
			err:  "номер версии",
		},
		{
			name: "version without paths",
			args: []string{"Author", "Grok", "--version", "4"},
			err:  "элемент",
		},
		{
			name: "bad version",
			args: []string{"Author", "Grok", "--version", "0", "skills/ru-text"},
			err:  "номер версии",
		},
		{
			name: "unknown flag",
			args: []string{"Author", "Grok", "--global"},
			err:  "--global",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseApplyArgs(tt.args)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.author != "Author" || got.source != "Grok" || got.version != tt.version || strings.Join(got.paths, ",") != tt.paths {
				t.Fatalf("call %+v", got)
			}
		})
	}
}

func serveVersion(t *testing.T, want, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != want {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
}

func TestApplyPathsPutsChosenElementsIntoAgents(t *testing.T) {
	root := seedClaudeAndCanon(t)
	canon := filepath.Join(root, ".agents")
	serveVersion(t, "/accounts/Author/versions/claude/4", `{"author":"Author","agent":"claude","version":4,"files":[
		{"path":"skills/grill/SKILL.md","body":"new"},
		{"path":"skills/grill/ref.md","body":"ref"},
		{"path":"skills/other/SKILL.md","body":"other"},
		{"path":"plugins/p/skills/deep/SKILL.md","body":"deep"},
		{"path":"agents/go.md","body":"go"},
		{"path":"CLAUDE.md","body":"new rules"},
		{"path":"settings.json","body":"theirs"},
		{"path":"agentsync.json","body":"{\"mcp\":[]}"}]}`)

	code, stdout, stderr := runCLI(t, "apply", "Author", "claude", "--version", "4",
		"skills/grill", "plugins/p/skills/deep", "agents/go.md", "CLAUDE.md", "settings.json")
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	if !strings.Contains(stdout, "settings.json") {
		t.Fatalf("skipped not named: %q", stdout)
	}
	for rel, want := range map[string]string{
		"skills/grill/SKILL.md": "new",
		"skills/grill/ref.md":   "ref",
		"skills/grill/.env":     "KEY=1",
		"skills/deep/SKILL.md":  "deep",
		"skills/keep/SKILL.md":  "keep",
		"agents/go.md":          "go",
		"AGENTS.md":             "new rules",
	} {
		if body := readFile(t, filepath.Join(canon, filepath.FromSlash(rel))); body != want {
			t.Fatalf("%s = %q, want %q", rel, body, want)
		}
	}
	for _, rel := range []string{"skills/grill/notes.md", "skills/other/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(canon, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Fatalf("%s should be absent: %v", rel, err)
		}
	}
	if body := readFile(t, filepath.Join(root, ".claude", "settings.json")); body != "mine" {
		t.Fatalf("own file changed: %q", body)
	}
	snaps := agent.Chain(root, "zwarder", "agents")
	if len(snaps) != 1 || !hasBody(snaps[0].Files, "skills/grill/SKILL.md", "old") {
		t.Fatalf("agents snapshot %+v", snaps)
	}

	code, _, stderr = runCLI(t, "apply", "Author", "claude", "--version", "4", "nope")
	if code != 1 || !strings.Contains(stderr, "«nope»") {
		t.Fatalf("missing path %d %s", code, stderr)
	}
	code, _, stderr = runCLI(t, "apply", "Author", "claude", "--version", "4", "settings.json")
	if code != 1 || !strings.Contains(stderr, "нет элементов agents") {
		t.Fatalf("own only %d %s", code, stderr)
	}
}

func TestApplyPathsMergesAgentsCardHooks(t *testing.T) {
	root := connectRoot(t)
	canon := filepath.Join(root, ".agents")
	writeRel(t, canon, "hooks.json", `{"version":1,"hooks":{"stop":[{"hooks":[{"type":"command","command":"mine"}]}]}}`)
	serveVersion(t, "/accounts/Author/versions/agents/2", `{"author":"Author","agent":"agents","version":2,"files":[
		{"path":"hooks.json","body":"{\"version\":1,\"hooks\":{\"session_start\":[{\"hooks\":[{\"type\":\"command\",\"command\":\"theirs\"}]}]}}"},
		{"path":"README.md","body":"readme"}]}`)

	code, stdout, stderr := runCLI(t, "apply", "Author", "agents", "--version", "2", "hooks.json", "README.md")
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	body := readFile(t, filepath.Join(canon, "hooks.json"))
	if !strings.Contains(body, "mine") || !strings.Contains(body, "theirs") {
		t.Fatalf("hooks %s", body)
	}
	if !strings.Contains(stdout, "README.md") {
		t.Fatalf("skipped not named: %q", stdout)
	}
}

func TestApplyPathsCleansStoredPaths(t *testing.T) {
	root := seedClaudeAndCanon(t)
	canon := filepath.Join(root, ".agents")
	serveVersion(t, "/accounts/Author/versions/claude/4", `{"author":"Author","agent":"claude","version":4,"files":[
		{"path":"agents/../agents/go.md","body":"go"},
		{"path":"agentsync.json","body":"{\"mcp\":[]}"}]}`)

	code, _, stderr := runCLI(t, "apply", "Author", "claude", "--version", "4", "agents")
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	for rel, want := range map[string]string{"agents/go.md": "go", "skills/keep/SKILL.md": "keep", "AGENTS.md": "old rules"} {
		if body := readFile(t, filepath.Join(canon, filepath.FromSlash(rel))); body != want {
			t.Fatalf("%s = %q, want %q", rel, body, want)
		}
	}
}

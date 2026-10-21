package cli

import (
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

const claudeState = `{"numStartups":7,"oauthAccount":{"emailAddress":"me@example.com"},"mcpServers":{
	"context7":{"command":"npx","args":["-y","@upstash/context7-mcp"]},
	"win":{"type":"stdio","command":"cmd","args":["/c","npx","-y","@upstash/context7-mcp"]},
	"local":{"command":"/Users/me/tools/mcp"}}}`

func TestPushCarriesClaudeUserMCPServersWithoutItsState(t *testing.T) {
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
	writeRel(t, root, ".claude/settings.json", `{"theme":"dark"}`)
	writeRel(t, root, ".claude.json", claudeState)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "claude"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	var user string
	for _, file := range sent.Files {
		if file.Path == "user.mcp.json" {
			user = file.Body
		}
		if strings.Contains(file.Body, "oauthAccount") || strings.Contains(file.Body, "numStartups") {
			t.Fatalf("Claude state published in %s", file.Path)
		}
	}
	if !strings.Contains(user, "context7") || !strings.Contains(user, `"win"`) || strings.Contains(user, "/Users/me") {
		t.Fatalf("user MCP %q", user)
	}
}

func TestPushKeepsClaudeStateInsideAConfigDirPrivate(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)
	root := connectRoot(t)
	config := filepath.Join(root, ".claude")
	t.Setenv("CLAUDE_CONFIG_DIR", config)
	writeRel(t, config, "settings.json", `{"theme":"dark"}`)
	writeRel(t, config, ".claude.json", claudeState)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	if code, _, stderr := runCLI(t, "push", "claude"); code != 0 {
		t.Fatalf("push %s", stderr)
	}
	if strings.Contains(sent, "oauthAccount") || !strings.Contains(sent, "user.mcp.json") || !strings.Contains(sent, "context7") {
		t.Fatalf("sent %s", sent)
	}
}

func TestPipeApplyPutsAuthorsOwnMCPServersIntoClaudeJSON(t *testing.T) {
	root := connectRoot(t)
	writeRel(t, root, ".claude.json", `{"numStartups":3,"mcpServers":{"mine":{"command":"mine"}}}`)
	code, stderr := pipeApply(t, `{"author":"zwarder","agent":"claude","files":[
		{"path":"settings.json","body":"{\"theme\":\"light\"}"},
		{"path":"user.mcp.json","body":"{\"mcpServers\":{\"context7\":{\"command\":\"npx\"}}}"},
		{"path":"agentsync.json","body":"{\"mcp\":[]}"}]}`)
	if code != 0 {
		t.Fatalf("apply %d %s", code, stderr)
	}
	_, servers := claudeServers(t, root)
	if servers["context7"] == nil || servers["mine"] == nil {
		t.Fatalf("servers %v", servers)
	}
	if top, _ := claudeServers(t, root); top["numStartups"] != float64(3) {
		t.Fatalf("Claude state changed %v", top)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "user.mcp.json")); !os.IsNotExist(err) {
		t.Fatal("user.mcp.json written into the home")
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "mcp.json")); !os.IsNotExist(err) {
		t.Fatal("author's own server went to agents")
	}
}

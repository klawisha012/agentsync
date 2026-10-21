package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushCarriesSubagentsAndCommandsWithoutMarketplaceClones(t *testing.T) {
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Files []struct {
				Path string `json:"path"`
			} `json:"files"`
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Error(err)
		}
		for _, file := range body.Files {
			sent = append(sent, file.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	root := t.TempDir()
	writeAgentFile(t, root, ".claude/agents/go-developer.md", "subagent")
	writeAgentFile(t, root, ".claude/commands/review.md", "command")
	writeAgentFile(t, root, ".claude/settings.json", `{"enabledPlugins":{"ponytail@ponytail":true}}`)
	writeAgentFile(t, root, ".claude/plugins/marketplaces/ponytail/README.md", "clone")
	writeAgentFile(t, root, ".claude/plugins/plugin-catalog-cache.json", "{}")
	writeAgentFile(t, root, ".claude/plugins/known_marketplaces.json", "{}")
	t.Setenv("AGENTSYNC_ROOT", root)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	t.Setenv("AGENTSYNC_STATE", t.TempDir())

	var out, errOut bytes.Buffer
	if code := Run([]string{"push", "claude"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("push %d %s %s", code, out.String(), errOut.String())
	}
	has := func(path string) bool {
		for _, item := range sent {
			if item == path {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"agents/go-developer.md", "commands/review.md", "settings.json"} {
		if !has(want) {
			t.Fatalf("snapshot lacks %s: %v", want, sent)
		}
	}
	for _, skip := range []string{"plugins/marketplaces/ponytail/README.md", "plugins/plugin-catalog-cache.json", "plugins/known_marketplaces.json"} {
		if has(skip) {
			t.Fatalf("snapshot carries %s: %v", skip, sent)
		}
	}
}

const machineAndSharedMCP = `{"mcpServers":{` +
	`"shared":{"command":"npx","args":["-y","@upstash/context7-mcp"]},` +
	`"local":{"command":"node","args":["/Users/alex/mcp.js"]},` +
	`"linear":{"url":"https://mcp.linear.app/mcp","headers":{"Authorization":"Bearer ${LINEAR_KEY}"}}}}`

func TestPushSendsMCPWithoutMachineServers(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sent = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	root := t.TempDir()
	writeAgentFile(t, root, ".grok/mcp.json", machineAndSharedMCP)
	t.Setenv("AGENTSYNC_ROOT", root)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	t.Setenv("AGENTSYNC_STATE", t.TempDir())

	var out, errOut bytes.Buffer
	if code := Run([]string{"push", "grok"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("push %d %s %s", code, out.String(), errOut.String())
	}
	if !strings.Contains(sent, "context7") || !strings.Contains(sent, "LINEAR_KEY") {
		t.Fatalf("shared servers lost: %s", sent)
	}
	if strings.Contains(sent, "/Users/alex") {
		t.Fatalf("machine server sent: %s", sent)
	}
}

func TestApplyKeepsMachineMCPServer(t *testing.T) {
	root := t.TempDir()
	writeAgentFile(t, root, ".grok/mcp.json", machineAndSharedMCP)
	t.Setenv("AGENTSYNC_ROOT", root)
	t.Setenv("AGENTSYNC_ACCOUNT", "zwarder")
	t.Setenv("AGENTSYNC_STATE", t.TempDir())

	incoming, _ := json.Marshal(map[string]any{
		"author": "alice", "agent": "grok",
		"files": []map[string]string{{"path": "mcp.json", "body": `{"mcpServers":{"docs":{"command":"npx","args":["docs"]}}}`}},
	})
	var out, errOut bytes.Buffer
	if code := Run(nil, bytes.NewReader(incoming), &out, &errOut); code != 0 {
		t.Fatalf("apply %d %s", code, errOut.String())
	}
	var got struct {
		Servers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, ".grok", "mcp.json"))), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Servers["docs"]; !ok {
		t.Fatalf("published server missing: %v", got.Servers)
	}
	if _, ok := got.Servers["local"]; !ok {
		t.Fatalf("machine server lost: %v", got.Servers)
	}
	if _, ok := got.Servers["shared"]; ok {
		t.Fatalf("replaced portable server stayed: %v", got.Servers)
	}
}

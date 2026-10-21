package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

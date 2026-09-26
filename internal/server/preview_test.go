package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPreviewMatchesPortableFiles(t *testing.T) {
	e := newServer(t)
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": "preview@example.com", "password": "secret", "name": "Previewer",
	}, nil)
	cookie := readSessionCookie(t, created)
	postJSON(t, e, "/email/confirm", map[string]string{
		"token": pathToken(letterPath(t, created.Body.Bytes())),
	}, nil)
	machine := postJSON(t, e, "/machine", map[string]string{
		"id": "pc-preview", "host": "desk", "listener": "127.0.0.1:49152",
	}, cookie)
	token := decodeMachine(t, machine.Body.Bytes()).AgentToken
	secret := "sk-live-PREVIEWSECRET"
	pushed := postAuth(t, e, "/agent/push", token, pushBody("Grok", false,
		pushFile{"rules/ok.md", "hello"},
		pushFile{"credentials.json", `{"token":"` + secret + `"}`},
		pushFile{"hooks/machine.json", "bash /Users/alex/run.sh"},
		pushFile{"mcp/local.json", `{"args":["/Users/alex/mcp.js"]}`},
	), cookie)
	if pushed.Code != http.StatusCreated {
		t.Fatalf("push %d %s", pushed.Code, pushed.Body.String())
	}
	id := publicationID(t, pushed.Body.Bytes())
	if strings.Contains(id, "Previewer") {
		t.Fatalf("address depends on the author name: %s", id)
	}

	opened := getJSON(t, e, "/accounts/Previewer", nil)
	visitor := readNamedCookie(t, opened, "visitor")
	before := viewsOf(t, opened)

	preview := getJSON(t, e, "/publications/"+id, nil)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview %d %s", preview.Code, preview.Body.String())
	}
	if strings.Contains(preview.Body.String(), secret) || strings.Contains(preview.Body.String(), "credentials.json") || strings.Contains(preview.Body.String(), "hooks/machine.json") || strings.Contains(preview.Body.String(), "mcp/local.json") {
		t.Fatalf("preview leaked a private file %s", preview.Body.String())
	}
	if !strings.Contains(preview.Body.String(), "машинный хук") || !strings.Contains(preview.Body.String(), "учётные данные") {
		t.Fatalf("excluded categories missing %s", preview.Body.String())
	}
	got := previewPaths(t, preview.Body.Bytes())
	if len(got) != 1 || got[0] != "rules/ok.md" {
		t.Fatalf("preview files %v", got)
	}
	after := viewsOf(t, getJSON(t, e, "/accounts/Previewer", visitor))
	if after != before {
		t.Fatalf("preview changed the page view from %d to %d", before, after)
	}
}

func previewPaths(t *testing.T, raw []byte) []string {
	t.Helper()
	var body struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(body.Files))
	for _, file := range body.Files {
		paths = append(paths, file.Path)
	}
	return paths
}

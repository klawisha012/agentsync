package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPushKeepsPortableFiles(t *testing.T) {
	e := newServer(t)
	created := postJSON(t, e, "/accounts", map[string]string{
		"email": "author@example.com", "password": "secret", "name": "Author",
	}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	author := readSessionCookie(t, created)
	machine := postJSON(t, e, "/machine", map[string]string{
		"id": "pc-author", "host": "desk", "listener": "127.0.0.1:49152",
	}, author)
	if machine.Code != http.StatusOK {
		t.Fatalf("machine %d %s", machine.Code, machine.Body.String())
	}
	token := decodeMachine(t, machine.Body.Bytes()).AgentToken

	early := postAuth(t, e, "/agent/push", token, pushBody("Grok", false, pushFile{"rules/ok.md", "hello"}), author)
	if early.Code == http.StatusCreated || !strings.Contains(early.Body.String(), "почт") {
		t.Fatalf("upload before mail %d %s", early.Code, early.Body.String())
	}

	confirmed := postJSON(t, e, "/email/confirm", map[string]string{"token": pathToken(letterPath(t, created.Body.Bytes()))}, nil)
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm %d %s", confirmed.Code, confirmed.Body.String())
	}

	other := mustAccount(t, e, "other@example.com", "secret", "Other")
	otherMachine := postJSON(t, e, "/machine", map[string]string{
		"id": "pc-other", "host": "lap", "listener": "127.0.0.1:49152",
	}, other)
	otherToken := decodeMachine(t, otherMachine.Body.Bytes()).AgentToken
	mismatch := postAuth(t, e, "/agent/push", otherToken, pushBody("Grok", false, pushFile{"rules/ok.md", "hello"}), author)
	if mismatch.Code == http.StatusCreated || !strings.Contains(mismatch.Body.String(), "аккаунт") {
		t.Fatalf("mismatch %d %s", mismatch.Code, mismatch.Body.String())
	}

	missing := postAuth(t, e, "/agent/push", token, map[string]any{"agent": "Grok", "missing": true}, author)
	if missing.Code == http.StatusCreated || !strings.Contains(missing.Body.String(), "компьютере") {
		t.Fatalf("missing %d %s", missing.Code, missing.Body.String())
	}
	empty := postAuth(t, e, "/agent/push", token, pushBody("Grok", false, pushFile{"credentials.json", `{"token":"abc"}`}), author)
	if empty.Code == http.StatusCreated || !strings.Contains(empty.Body.String(), "переносим") {
		t.Fatalf("empty %d %s", empty.Code, empty.Body.String())
	}

	bare := postAuth(t, e, "/agent/push", "", pushBody("Grok", false, pushFile{"rules/ok.md", "hello"}), author)
	if bare.Code == http.StatusCreated || !strings.Contains(bare.Body.String(), "аккаунт") {
		t.Fatalf("push without the agent %d %s", bare.Code, bare.Body.String())
	}
	secret := "sk-live-SUPERSECRETVALUE"
	leaked := postAuth(t, e, "/agent/push", token, pushBody("Grok", false, pushFile{"rules/key.md", "token=" + secret}), author)
	if leaked.Code == http.StatusCreated || !strings.Contains(leaked.Body.String(), "rules/key.md") || strings.Contains(leaked.Body.String(), secret) {
		t.Fatalf("secret %d %s", leaked.Code, leaked.Body.String())
	}
	jsonSecret := "JSONSECRETVALUE99"
	leakedJSON := postAuth(t, e, "/agent/push", token, pushBody("Grok", false, pushFile{"rules/key.json", `{"token":"` + jsonSecret + `"}`}), author)
	if leakedJSON.Code == http.StatusCreated || !strings.Contains(leakedJSON.Body.String(), "rules/key.json") || strings.Contains(leakedJSON.Body.String(), jsonSecret) {
		t.Fatalf("json secret %d %s", leakedJSON.Code, leakedJSON.Body.String())
	}

	first := postAuth(t, e, "/agent/push", token, pushBody("Grok", false,
		pushFile{"rules/ok.md", "hello"},
		pushFile{"credentials.json", `{"token":"abc"}`},
		pushFile{"hooks/machine.json", "bash /Users/alex/run.sh"},
		pushFile{"scripts/run.sh", "echo hi"},
		pushFile{"skills/note.md", "see /Users/alex/notes"},
		pushFile{"mcp/local.json", `{"args":["/Users/alex/mcp.js"]}`},
		pushFile{"mcp/shared.json", `{"command":"npx","args":["docs"]}`},
		pushFile{"hooks.json", "bash scripts/ok.sh"},
		pushFile{"hooks/unix.json", "bash /bin/run.sh"},
		pushFile{"bin/run.sh", "echo unix"},
		pushFile{"mcp/env.json", `{"env":{"GITHUB_TOKEN":"ghp_abcdefghijklmnopqrst"}}`},
	), author)
	if first.Code != http.StatusCreated {
		t.Fatalf("push %d %s", first.Code, first.Body.String())
	}
	if strings.Contains(first.Body.String(), "credentials.json") || strings.Contains(first.Body.String(), "hooks/machine.json") || strings.Contains(first.Body.String(), "scripts/run.sh") || strings.Contains(first.Body.String(), "mcp/local.json") || strings.Contains(first.Body.String(), "hooks/unix.json") || strings.Contains(first.Body.String(), "bin/run.sh") || strings.Contains(first.Body.String(), "mcp/env.json") || strings.Contains(first.Body.String(), "ghp_abcdefghijklmnopqrst") {
		t.Fatalf("publication kept a private file %s", first.Body.String())
	}
	if !strings.Contains(first.Body.String(), "rules/ok.md") || !strings.Contains(first.Body.String(), "skills/note.md") || !strings.Contains(first.Body.String(), "mcp/shared.json") || !strings.Contains(first.Body.String(), "hooks.json") {
		t.Fatalf("publication lost a portable file %s", first.Body.String())
	}
	if !strings.Contains(first.Body.String(), `"version":1`) {
		t.Fatalf("version %s", first.Body.String())
	}
	firstID := publicationID(t, first.Body.Bytes())

	second := postAuth(t, e, "/agent/push", token, pushBody("Grok", false, pushFile{"rules/next.md", "again"}), author)
	if second.Code != http.StatusCreated || !strings.Contains(second.Body.String(), `"version":2`) {
		t.Fatalf("second %d %s", second.Code, second.Body.String())
	}
	kept := getJSON(t, e, "/publications/"+firstID, nil)
	if kept.Code != http.StatusOK || !strings.Contains(kept.Body.String(), "rules/ok.md") || !strings.Contains(kept.Body.String(), `"version":1`) {
		t.Fatalf("first publication changed %d %s", kept.Code, kept.Body.String())
	}
	page := getJSON(t, e, "/accounts/Author", nil)
	if !strings.Contains(page.Body.String(), `"version":2`) || strings.Contains(page.Body.String(), "credentials.json") {
		t.Fatalf("page slot %s", page.Body.String())
	}
}

type pushFile struct {
	Path string `json:"path"`
	Body string `json:"body"`
}

func pushBody(agent string, missing bool, files ...pushFile) map[string]any {
	return map[string]any{"agent": agent, "missing": missing, "files": files}
}

func publicationID(t *testing.T, raw []byte) string {
	t.Helper()
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &body); err != nil || body.ID == "" {
		t.Fatalf("id %s", raw)
	}
	return body.ID
}

func postAuth(t *testing.T, h http.Handler, path, token string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

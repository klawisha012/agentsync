package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPushAllPublishesFoundAgents(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Agent string `json:"agent"`
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Error(err)
		}
		got = append(got, body.Agent)
		if body.Agent == "bad" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"explanation":"В файле rules/a.md секрет."}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	root := t.TempDir()
	writeAgentFile(t, root, ".grok/skills/ru-text/SKILL.md", "skill")
	writeAgentFile(t, root, "Claude/AGENTS.md", "rules")
	writeAgentFile(t, root, "bad/rules/a.md", "secret")
	writeAgentFile(t, root, "Notes/readme.txt", "skip")
	t.Setenv("AGENTSYNC_ROOT", root)
	t.Setenv("AGENTSYNC_SERVER", srv.URL)
	t.Setenv("AGENTSYNC_TOKEN", "machine")
	t.Setenv("AGENTSYNC_STATE", t.TempDir())

	var out, err bytes.Buffer
	if code := Run([]string{"all"}, nil, &out, &err); code != 1 {
		t.Fatalf("all %d %s %s", code, out.String(), err.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Опубликован Claude.")) || !bytes.Contains(out.Bytes(), []byte("Опубликован grok.")) {
		t.Fatalf("stdout %q", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("bad: В файле rules/a.md секрет.")) {
		t.Fatalf("stdout %q", out.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("Не все ИИ-агенты опубликованы.")) {
		t.Fatalf("stderr %q", err.String())
	}
	if len(got) != 3 || got[0] != "bad" || got[1] != "Claude" || got[2] != "grok" {
		t.Fatalf("pushed %v", got)
	}
}

func TestPushAllRequiresLoginAndAgents(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AGENTSYNC_ROOT", root)
	t.Setenv("AGENTSYNC_TOKEN", "")
	t.Setenv("AGENTSYNC_COOKIE", "")
	t.Setenv("AGENTSYNC_STATE", t.TempDir())

	var out, err bytes.Buffer
	if code := Run([]string{"all"}, nil, &out, &err); code != 1 {
		t.Fatalf("login %d %s", code, err.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("Войдите в")) {
		t.Fatalf("stderr %q", err.String())
	}

	t.Setenv("AGENTSYNC_TOKEN", "machine")
	out.Reset()
	err.Reset()
	if code := Run([]string{"all"}, nil, &out, &err); code != 1 {
		t.Fatalf("empty %d %s", code, err.String())
	}
	if !bytes.Contains(err.Bytes(), []byte("На компьютере нет ИИ-агентов.")) {
		t.Fatalf("stderr %q", err.String())
	}
	if code := Run([]string{"all", "Grok"}, nil, &out, &err); code != 2 {
		t.Fatalf("arg %d %s", code, err.String())
	}
}

func TestAllHelp(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"help", "all"}, nil, &out, &err); code != 0 {
		t.Fatalf("help %d", code)
	}
	if !bytes.Contains(out.Bytes(), []byte("agentsync all")) {
		t.Fatalf("stdout %q", out.String())
	}
}

func writeAgentFile(t *testing.T, root, rel, body string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

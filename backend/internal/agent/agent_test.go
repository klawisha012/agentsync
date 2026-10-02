package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushReadsDottedHomeWithoutInstall(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		body = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	root := t.TempDir()
	home := filepath.Join(root, ".grok", "rules")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "ok.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(root, ".grok", "credentials.json")
	if err := os.WriteFile(secret, []byte("secret-token"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Push(context.Background(), srv.URL, root, "grok", "machine", "")
	if err != nil {
		t.Fatal(err)
	}
	sentRules := strings.Contains(body, "rules/ok.md")
	sentSecret := strings.Contains(body, "secret-token") || strings.Contains(body, "credentials.json")
	if !sentRules || sentSecret {
		t.Fatalf("body %s", body)
	}
}

func TestPushSkipsProductDelivery(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		body = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	root := t.TempDir()
	home := filepath.Join(root, ".grok")
	for _, rel := range []string{
		filepath.Join("skills", "mine.md"),
		filepath.Join("bundled", "skills", "pdf", "SKILL.md"),
		filepath.Join("marketplace-cache", "plug", "skills", "SKILL.md"),
	} {
		path := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "hello"
		if strings.Contains(rel, "SKILL.md") {
			body = "token=EXAMPLEKEY99"
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Push(context.Background(), srv.URL, root, "grok", "machine", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "skills/mine.md") || strings.Contains(body, "bundled/") || strings.Contains(body, "marketplace-cache/") || strings.Contains(body, "EXAMPLEKEY99") {
		t.Fatalf("body %s", body)
	}
}

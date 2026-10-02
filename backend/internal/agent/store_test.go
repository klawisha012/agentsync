package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordWritesLocalCacheWithoutPrivateFiles(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent/record" || r.Header.Get("Authorization") != "Bearer machine" {
			http.NotFound(w, r)
			return
		}
		got = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"s1","account":"Keeper","agent":"Grok","number":1,"current":true,"files":[{"path":"rules/ok.md","body":"hello"}]}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "Grok", "rules")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "ok.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Grok", "credentials.json"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := Record(context.Background(), srv.URL, root, "Grok", "machine", "session=1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/agent/record" || !strings.Contains(string(raw), `"number":1`) {
		t.Fatalf("record %s %s", got, raw)
	}
	cached := filepath.Join(root, ".agentsync", "store", "keeper", "grok", "0001", "rules", "ok.md")
	body, err := os.ReadFile(cached)
	if err != nil || string(body) != "hello" {
		t.Fatalf("cache %v %s", err, body)
	}
	if _, err := os.Stat(filepath.Join(root, ".agentsync", "store", "keeper", "grok", "0001", "credentials.json")); !os.IsNotExist(err) {
		t.Fatal("cache kept credentials")
	}
	mark, err := os.ReadFile(filepath.Join(root, ".agentsync", "store", "keeper", "grok", "current"))
	if err != nil || strings.TrimSpace(string(mark)) != "1" {
		t.Fatalf("mark %v %s", err, mark)
	}
}

func TestReadStoreCopiesNamedSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent/store/Grok/2" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"id":"s2","account":"Keeper","agent":"Grok","number":2,"current":true,"files":[{"path":"rules/next.md","body":"again"}]}`))
	}))
	defer srv.Close()

	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	raw, err := ReadStore(context.Background(), srv.URL, root, "Grok", "2", "machine", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "rules/next.md") {
		t.Fatalf("body %s", raw)
	}
	body, err := os.ReadFile(filepath.Join(root, ".agentsync", "store", "keeper", "grok", "0002", "rules", "next.md"))
	if err != nil || string(body) != "again" {
		t.Fatalf("cache %v %s", err, body)
	}
}

package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPushReadsCanonWithoutPrivateFiles(t *testing.T) {
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
	home := filepath.Join(root, ".agents", "rules")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "ok.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", ".env"), []byte("TOKEN=secret-token"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "credentials.json"), []byte("secret-token"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Push(context.Background(), srv.URL, root, "machine", ""); err != nil {
		t.Fatal(err)
	}
	sentRules := strings.Contains(body, "rules/ok.md") && strings.Contains(body, `"agent":"agents"`)
	sentSecret := strings.Contains(body, "secret-token") || strings.Contains(body, "credentials.json") || strings.Contains(body, ".env")
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
	home := filepath.Join(root, ".agents")
	for _, rel := range []string{
		filepath.Join("skills", "mine.md"),
		filepath.Join("bundled", "skills", "pdf", "SKILL.md"),
		filepath.Join("marketplace-cache", "plug", "skills", "SKILL.md"),
	} {
		path := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		text := "hello"
		if strings.Contains(rel, "SKILL.md") {
			text = "token=EXAMPLEKEY99"
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Push(context.Background(), srv.URL, root, "machine", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "skills/mine.md") || strings.Contains(body, "bundled/") || strings.Contains(body, "marketplace-cache/") || strings.Contains(body, "EXAMPLEKEY99") {
		t.Fatalf("body %s", body)
	}
}

func TestEveryHookReceiverHasADialect(t *testing.T) {
	for _, receiver := range receivers {
		if _, ok := dialects[receiver.Slug]; receiver.Hooks != "" && !ok {
			t.Errorf("%s takes hooks but has no dialect", receiver.Slug)
		}
	}
}

func TestArchivesStayOnMachine(t *testing.T) {
	kept := portableOnly([]File{
		{Path: "skills/pdf/SKILL.md", Body: "x"},
		{Path: "skills/pdf/assets.zip", Body: "x"},
		{Path: "skills/pdf/Backup.RAR", Body: "x"},
		{Path: "skills/pdf/setup.exe", Body: "x"},
	})
	if len(kept) != 1 || kept[0].Path != "skills/pdf/SKILL.md" {
		t.Fatalf("kept %v", kept)
	}
}

func TestReadTreeSkipsDirLinks(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	for path, body := range map[string]string{
		filepath.Join(root, "skills", "demo", "SKILL.md"): "ok",
		filepath.Join(target, "nested.txt"):               "skip",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "skills", "demo", "linked")
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J: %v: %s", err, out)
		}
	} else if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	files, err := readTree(root)
	if err != nil {
		t.Fatalf("readTree: %v", err)
	}
	if len(files) != 1 || files[0].Path != "skills/demo/SKILL.md" {
		t.Fatalf("files %+v", files)
	}
}

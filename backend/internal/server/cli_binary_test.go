package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCLIBinary(t *testing.T) {
	dir := t.TempDir()
	name := "agentsync-windows-amd64.exe"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("cli-binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTSYNC_CLI_DIR", dir)

	e := echo.New()
	bins := openCLIBinaries()
	e.GET("/cli/:file", bins.get)

	got := getJSON(t, e, "/cli/"+name, nil)
	if got.Code != http.StatusOK || got.Body.String() != "cli-binary" {
		t.Fatalf("binary %d %q", got.Code, got.Body.String())
	}
	if got.Header().Get(echo.HeaderContentType) != "application/octet-stream" {
		t.Fatalf("content type %s", got.Header().Get(echo.HeaderContentType))
	}

	missing := getJSON(t, e, "/cli/agentsync-linux-amd64", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing %d", missing.Code)
	}
	rejected := getJSON(t, e, "/cli/install.ps1", nil)
	if rejected.Code != http.StatusNotFound {
		t.Fatalf("script name %d", rejected.Code)
	}
}

func TestCLIVersion(t *testing.T) {
	dir := t.TempDir()
	name := "agentsync-windows-amd64.exe"
	body := []byte("cli-binary")
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "version"), []byte("1.2.3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTSYNC_CLI_DIR", dir)

	e := echo.New()
	bins := openCLIBinaries()
	e.GET("/cli/version", bins.cliVersion)
	e.GET("/cli/:file", bins.get)

	got := getJSON(t, e, "/cli/version", nil)
	if got.Code != http.StatusOK {
		t.Fatalf("version %d %s", got.Code, got.Body.String())
	}
	var payload struct {
		Version string            `json:"version"`
		SHA256  map[string]string `json:"sha256"`
		Note    string            `json:"note"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if payload.Version != "1.2.3" || payload.SHA256[name] != hex.EncodeToString(sum[:]) {
		t.Fatalf("payload %+v", payload)
	}
	if _, ok := payload.SHA256["notes.txt"]; ok {
		t.Fatal("non-binary entered the checksum map")
	}
	if payload.Note != "" {
		t.Fatal("unexpected note")
	}

	file := getJSON(t, e, "/cli/"+name, nil)
	if file.Code != http.StatusOK || file.Body.String() != "cli-binary" {
		t.Fatalf("binary %d %q", file.Code, file.Body.String())
	}
}

package server

import (
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

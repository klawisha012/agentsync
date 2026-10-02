package server

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestInstallScripts(t *testing.T) {
	scripts, err := readInstallScripts()
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.GET("/cli/install.ps1", scripts.ps)
	e.GET("/cli/install.sh", scripts.sh)
	e.GET("/cli/:file", openCLIBinaries().get)
	dir := repoScripts(t)

	tests := []struct {
		name string
		path string
		file string
		mark string
	}{
		{name: "powershell", path: "/cli/install.ps1", file: "install.ps1", mark: "SetEnvironmentVariable"},
		{name: "shell", path: "/cli/install.sh", file: "install.sh", mark: "export PATH"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := getJSON(t, e, tt.path, nil)
			body := rec.Body.String()
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d", rec.Code)
			}
			if rec.Header().Get(echo.HeaderContentType) != "text/plain; charset=utf-8" {
				t.Fatalf("content type %s", rec.Header().Get(echo.HeaderContentType))
			}
			want := scriptText(t, filepath.Join(dir, tt.file))
			matchesFile := body == want
			unixNewlines := !strings.Contains(body, "\r")
			putsOnPath := strings.Contains(body, "PATH") && !strings.Contains(body, "agentsync push")
			hasMark := strings.Contains(body, tt.mark)
			hasBinary := strings.Contains(body, "agentsync")
			served := matchesFile && unixNewlines && putsOnPath && hasMark && hasBinary
			if !served {
				t.Fatalf("body %q", body)
			}
		})
	}
}

func repoScripts(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "scripts")
}

func scriptText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

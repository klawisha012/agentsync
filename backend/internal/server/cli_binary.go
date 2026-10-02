package server

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/labstack/echo/v4"
)

var cliBinaryName = regexp.MustCompile(`^agentsync-(linux|darwin|windows)-(amd64|arm64)(\.exe)?$`)

type cliBinaries struct {
	dir string
}

func (b cliBinaries) get(c echo.Context) error {
	name := c.Param("file")
	if b.dir == "" || !cliBinaryName.MatchString(name) {
		return writeExplanation(c, http.StatusNotFound, "Файл не найден.")
	}
	path := filepath.Join(b.dir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return writeExplanation(c, http.StatusNotFound, "Файл не найден.")
	}
	f, err := os.Open(path)
	if err != nil {
		return writeExplanation(c, http.StatusNotFound, "Файл не найден.")
	}
	defer f.Close()
	c.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
	http.ServeContent(c.Response(), c.Request(), name, info.ModTime(), f)
	return nil
}

func openCLIBinaries() cliBinaries {
	if dir := os.Getenv("AGENTSYNC_CLI_DIR"); dir != "" {
		return cliBinaries{dir: dir}
	}
	roots := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	for _, root := range roots {
		if dir, ok := walkCLIBinaries(root); ok {
			return cliBinaries{dir: dir}
		}
	}
	return cliBinaries{}
}

func walkCLIBinaries(start string) (string, bool) {
	dir := start
	for {
		candidate := filepath.Join(dir, "cli-bin")
		if scriptExists(filepath.Join(candidate, "agentsync-linux-amd64")) {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

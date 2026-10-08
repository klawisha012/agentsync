package server

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

var cliBinaryName = regexp.MustCompile(`^agentsync-(linux|darwin|windows)-(amd64|arm64)(\.exe)?$`)

type cliBinaries struct {
	dir     string
	release string
	sums    map[string]string
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

func (b cliBinaries) cliVersion(c echo.Context) error {
	if b.dir == "" {
		return writeExplanation(c, http.StatusNotFound, "Файл не найден.")
	}
	sums := b.sums
	if sums == nil {
		sums = map[string]string{}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"version": b.release,
		"sha256":  sums,
	})
}

func openCLIBinaries() cliBinaries {
	if dir := os.Getenv("AGENTSYNC_CLI_DIR"); dir != "" {
		return loadCLIBinaries(dir)
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
			return loadCLIBinaries(dir)
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

func loadCLIBinaries(dir string) cliBinaries {
	return cliBinaries{
		dir:     dir,
		release: readCLIRelease(dir),
		sums:    hashCLIBinaries(dir),
	}
}

func readCLIRelease(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "version"))
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(raw), "\n")
	line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if line == "" || len(line) > 80 || strings.ContainsAny(line, " \t") {
		return ""
	}
	return line
}

func hashCLIBinaries(dir string) map[string]string {
	sums := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return sums
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !cliBinaryName.MatchString(name) {
			continue
		}
		sum, err := hashCLIFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		sums[name] = sum
	}
	return sums
}

func hashCLIFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

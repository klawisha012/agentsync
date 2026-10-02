package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

type installFiles struct {
	powershell string
	shell      string
}

func (f installFiles) ps(c echo.Context) error {
	return writeScript(c, f.powershell)
}

func (f installFiles) sh(c echo.Context) error {
	return writeScript(c, f.shell)
}

func writeScript(c echo.Context, body string) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	return c.String(http.StatusOK, body)
}

// The repository keeps scripts/ at the root. The image copies that directory to /scripts.
func readInstallScripts() (installFiles, error) {
	dir, err := findScriptsDir()
	if err != nil {
		return installFiles{}, err
	}
	powershell, err := readScript(filepath.Join(dir, "install.ps1"))
	if err != nil {
		return installFiles{}, err
	}
	shell, err := readScript(filepath.Join(dir, "install.sh"))
	if err != nil {
		return installFiles{}, err
	}
	return installFiles{powershell: powershell, shell: shell}, nil
}

func readScript(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read install script %s: %w", filepath.Base(path), err)
	}
	body := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("install script %s is empty", filepath.Base(path))
	}
	return body, nil
}

func findScriptsDir() (string, error) {
	roots := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	for _, root := range roots {
		if dir, ok := walkScripts(root); ok {
			return dir, nil
		}
	}
	return "", errors.New("install scripts not found")
}

func walkScripts(start string) (string, bool) {
	dir := start
	for {
		scripts := filepath.Join(dir, "scripts")
		ps := scriptExists(filepath.Join(scripts, "install.ps1"))
		sh := scriptExists(filepath.Join(scripts, "install.sh"))
		if ps && sh {
			return scripts, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func scriptExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

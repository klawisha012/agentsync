package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const installedFile = "installed"

var ErrNeedInstall = errors.New("локальный агент не установлен")

type File struct {
	Path string `json:"path"`
	Body string `json:"body"`
}

type publicationPayload struct {
	Author  string `json:"author"`
	Agent   string `json:"agent"`
	Version int    `json:"version"`
	Files   []File `json:"files"`
}

func Install(dir string) error {
	base := filepath.Join(dir, ".agentsync")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(base, installedFile), []byte("confirm-machine\n"), 0o644)
}

func installed(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".agentsync", installedFile))
	return err == nil
}

func Push(ctx context.Context, server, root, agentName, token, cookie string) error {
	if !installed(root) {
		return ErrNeedInstall
	}
	files, err := readTree(filepath.Join(root, agentName))
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"agent": agentName, "files": files})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(server, "/")+"/agent/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(res.Body)
		return errors.New(strings.TrimSpace(string(msg)))
	}
	return nil
}

func Apply(ctx context.Context, server, root, author, agentName, token, cookie string) error {
	if !installed(root) {
		return ErrNeedInstall
	}
	payload, err := fetchPublication(ctx, server, author, agentName, token, cookie)
	if err != nil {
		return err
	}
	return applyFiles(root, author, agentName, payload.Files, false)
}

func ApplyBroken(ctx context.Context, server, root, author, agentName, token, cookie string) error {
	if !installed(root) {
		return ErrNeedInstall
	}
	payload, err := fetchPublication(ctx, server, author, agentName, token, cookie)
	if err != nil {
		return err
	}
	return applyFiles(root, author, agentName, payload.Files, true)
}

func Revert(root, account, agentName string) error {
	snaps := Chain(root, account, agentName)
	if len(snaps) == 0 {
		return errors.New("цепочка пуста")
	}
	latest := snaps[0]
	home := filepath.Join(root, agentName)
	if err := clearPortable(home); err != nil {
		return err
	}
	for _, file := range latest.Files {
		if err := writeFile(home, file); err != nil {
			return err
		}
	}
	return os.RemoveAll(latest.dir)
}

type Snapshot struct {
	Files []File
	dir   string
}

func Chain(root, account, agentName string) []Snapshot {
	base := chainDir(root, account, agentName)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	out := make([]Snapshot, 0, len(names))
	for _, name := range names {
		dir := filepath.Join(base, name)
		files, err := readTree(dir)
		if err != nil {
			continue
		}
		out = append(out, Snapshot{Files: files, dir: dir})
	}
	return out
}

func fetchPublication(ctx context.Context, server, author, agentName, token, cookie string) (publicationPayload, error) {
	url := strings.TrimRight(server, "/") + "/api/apply/" + author + "/" + agentName
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return publicationPayload{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return publicationPayload{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return publicationPayload{}, errors.New(strings.TrimSpace(string(raw)))
	}
	var payload publicationPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return publicationPayload{}, err
	}
	return payload, nil
}

func applyFiles(root, account, agentName string, next []File, broken bool) error {
	home := filepath.Join(root, agentName)
	current, err := readTree(home)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var portable []File
	for _, file := range current {
		if isPortable(file.Path, file.Body) {
			portable = append(portable, file)
		}
	}
	if broken {
		for _, file := range next[:min(1, len(next))] {
			if err := writeFile(home, file); err != nil {
				return err
			}
		}
		if err := clearPortable(home); err != nil {
			return err
		}
		for _, file := range portable {
			if err := writeFile(home, file); err != nil {
				return err
			}
		}
		return errors.New("применение не выполнено")
	}
	if err := writeSnapshot(root, account, agentName, portable); err != nil {
		return err
	}
	if err := clearPortable(home); err != nil {
		return err
	}
	for _, file := range next {
		if err := writeFile(home, file); err != nil {
			return err
		}
	}
	return nil
}

func writeSnapshot(root, account, agentName string, files []File) error {
	base := chainDir(root, account, agentName)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	entries, _ := os.ReadDir(base)
	name := fmtSeq(len(entries) + 1)
	dir := filepath.Join(base, name)
	for _, file := range files {
		if err := writeFile(dir, file); err != nil {
			return err
		}
	}
	if len(files) == 0 {
		return os.MkdirAll(dir, 0o755)
	}
	return nil
}

func chainDir(root, account, agentName string) string {
	return filepath.Join(root, ".agentsync", "chains", sanitize(account), sanitize(agentName))
}

func sanitize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func fmtSeq(n int) string {
	return strings.Repeat("0", 4-len(itoa(n))) + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func clearPortable(home string) error {
	files, err := readTree(home)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, file := range files {
		if isPortable(file.Path, file.Body) {
			if err := os.Remove(filepath.Join(home, filepath.FromSlash(file.Path))); err != nil {
				return err
			}
		}
	}
	return nil
}

func readTree(root string) ([]File, error) {
	var files []File
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, "..") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, File{Path: rel, Body: string(raw)})
		return nil
	})
	return files, err
}

func writeFile(root string, file File) error {
	clean := filepath.ToSlash(file.Path)
	if clean == "" || strings.HasPrefix(clean, "/") || strings.Contains(clean, "..") {
		return errors.New("путь файла вне ИИ-агента")
	}
	target := filepath.Join(root, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(file.Body), 0o644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

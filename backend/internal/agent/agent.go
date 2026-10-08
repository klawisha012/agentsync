package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const installedFile = "installed"

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

func Push(ctx context.Context, server, root, agentName, token, cookie string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	files, err := readTree(agentHome(root, agentName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("на компьютере нет этого ИИ-агента")
		}
		return err
	}
	_, err = postAgent(ctx, server, "/agent/push", agentName, token, cookie, portableOnly(files), false)
	return err
}

func Apply(ctx context.Context, server, root, author, agentName, account, token, cookie string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	payload, err := fetchPublication(ctx, server, author, agentName, token, cookie, 0)
	if err != nil {
		return err
	}
	if err := remember(ctx, server, root, payload.Agent, token, cookie); err != nil {
		return err
	}
	return applyOwned(root, account, payload)
}

func ApplyBody(ctx context.Context, server, root, account, token, cookie string, raw []byte) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	var payload publicationPayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Author == "" || payload.Agent == "" {
		var explained struct {
			Explanation string `json:"explanation"`
		}
		if json.Unmarshal(raw, &explained) == nil && strings.TrimSpace(explained.Explanation) != "" {
			return errors.New(explained.Explanation)
		}
		return errors.New("команда не содержит публикацию")
	}
	if err := remember(ctx, server, root, payload.Agent, token, cookie); err != nil {
		return err
	}
	return applyOwned(root, account, payload)
}

func applyOwned(root, account string, payload publicationPayload) error {
	owner := strings.TrimSpace(account)
	if owner == "" {
		owner = payload.Author
	}
	return applyFiles(root, owner, payload.Agent, payload.Files, false)
}

func ApplyBroken(ctx context.Context, server, root, author, agentName, token, cookie string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	payload, err := fetchPublication(ctx, server, author, agentName, token, cookie, 0)
	if err != nil {
		return err
	}
	return applyFiles(root, author, agentName, payload.Files, true)
}

func Revert(root, account, agentName string) error {
	unlock, err := lockRoot(root)
	if err != nil {
		return err
	}
	defer unlock()
	snaps := Chain(root, account, agentName)
	if len(snaps) == 0 {
		snaps = soleChain(root, agentName)
	}
	if len(snaps) == 0 {
		return errors.New("цепочка пуста")
	}
	latest := snaps[0]
	home := agentHome(root, agentName)
	if err := pathsFit(latest.Files); err != nil {
		return err
	}
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

func soleChain(root, agentName string) []Snapshot {
	base := filepath.Join(root, ".agentsync", "chains")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var found []Snapshot
	seen := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		snaps := Chain(root, entry.Name(), agentName)
		if len(snaps) == 0 {
			continue
		}
		seen++
		found = snaps
	}
	if seen != 1 {
		return nil
	}
	return found
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

func fetchPublication(ctx context.Context, server, author, agentName, token, cookie string, version int) (publicationPayload, error) {
	target := strings.TrimRight(server, "/") + "/api/apply/" + url.PathEscape(author) + "/" + url.PathEscape(agentName)
	if version > 0 {
		target += "?version=" + strconv.Itoa(version)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
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
		return publicationPayload{}, explainedError(raw)
	}
	var payload publicationPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return publicationPayload{}, err
	}
	return payload, nil
}

func applyFiles(root, account, agentName string, next []File, broken bool) error {
	if err := pathsFit(next); err != nil {
		return err
	}
	home := agentHome(root, agentName)
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
	if _, err := writeSnapshot(root, account, agentName, portable); err != nil {
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

func writeSnapshot(root, account, agentName string, files []File) (string, error) {
	base := chainDir(root, account, agentName)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	entries, _ := os.ReadDir(base)
	name := fmtSeq(len(entries) + 1)
	dir := filepath.Join(base, name)
	for _, file := range files {
		if err := writeFile(dir, file); err != nil {
			return "", err
		}
	}
	if len(files) == 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func explainedError(raw []byte) error {
	var explained struct {
		Explanation string `json:"explanation"`
	}
	if json.Unmarshal(raw, &explained) == nil && strings.TrimSpace(explained.Explanation) != "" {
		return errors.New(explained.Explanation)
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return errors.New("не удалось открыть публикацию")
	}
	return errors.New(text)
}

func chainDir(root, account, agentName string) string {
	return filepath.Join(root, ".agentsync", "chains", sanitize(account), sanitize(agentName))
}

func sanitize(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "_"
	}
	return name
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

func agentHome(root, name string) string {
	name = strings.TrimSpace(name)
	literal := filepath.Join(root, name)
	if info, err := os.Stat(literal); err == nil && info.IsDir() {
		return literal
	}
	if dot := catalogDotHome(name); dot != "" {
		return filepath.Join(root, dot)
	}
	return literal
}

func catalogDotHome(name string) string {
	for _, item := range receivers {
		if !strings.EqualFold(item.Slug, name) && !strings.EqualFold(item.Display, name) {
			continue
		}
		if item.OpenClaw {
			return ".openclaw"
		}
		for _, candidate := range []string{item.Global, item.Project} {
			seg := firstPathSegment(candidate)
			if strings.HasPrefix(seg, ".") && seg != "." && seg != ".." {
				return seg
			}
		}
	}
	return ""
}

func firstPathSegment(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return ""
	}
	seg, _, _ := strings.Cut(value, "/")
	return seg
}

func EnvOrHome(envName, fileName string) string {
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	return HomeFile(fileName)
}

func HomeFile(name string) string {
	if !stateName(name) {
		return ""
	}
	dir, err := stateDir()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func SaveHomeFile(name, body string) error {
	if !stateName(name) {
		return errors.New("неизвестное имя файла")
	}
	dir, err := stateDir()
	if err != nil {
		return err
	}
	unlock, err := lockDir(dir, 0o700)
	if err != nil {
		return err
	}
	defer unlock()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(strings.TrimSpace(body)+"\n"), 0o600)
}

func stateName(name string) bool {
	switch name {
	case "token", "cookie", "account", "session":
		return true
	default:
		return false
	}
}

func stateDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("AGENTSYNC_STATE")); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", errors.New("AGENTSYNC_STATE должен быть абсолютным путём")
		}
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agentsync"), nil
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

func pathsFit(files []File) error {
	for _, file := range files {
		if _, ok := CleanRel(file.Path); !ok {
			return errors.New("путь файла вне ИИ-агента")
		}
	}
	return nil
}

func writeFile(root string, file File) error {
	rel, ok := CleanRel(file.Path)
	if !ok {
		return errors.New("путь файла вне ИИ-агента")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	base, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer base.Close()
	dir := path.Dir(rel)
	if dir != "." {
		if err := base.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	out, err := base.OpenFile(rel, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = out.Write([]byte(file.Body))
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

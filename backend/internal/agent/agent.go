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
	"slices"
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
	files, err := readSnapshot(root, agentHome(root, agentName))
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
	if _, err := clearPortable(root, home); err != nil {
		return err
	}
	for _, file := range latest.Files {
		if err := writeHomeFile(home, file); err != nil {
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
	return getPublication(ctx, target, token, cookie)
}

func fetchSkillPublication(ctx context.Context, server, author, agentName, token, cookie string, version int) (publicationPayload, error) {
	target := strings.TrimRight(server, "/") + "/accounts/" + url.PathEscape(author) + "/versions/" + url.PathEscape(agentName) + "/" + strconv.Itoa(version)
	return getPublication(ctx, target, token, cookie)
}

func getPublication(ctx context.Context, target, token, cookie string) (publicationPayload, error) {
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
	portable := portableOnly(current)
	if broken {
		for _, file := range next[:min(1, len(next))] {
			if err := writeHomeFile(home, file); err != nil {
				return err
			}
		}
		links, err := clearPortable(root, home)
		if err != nil {
			return err
		}
		for _, file := range portable {
			if err := writeHomeFile(home, file); err != nil {
				return err
			}
		}
		for _, link := range links {
			if err := os.Symlink(link.target, link.path); err != nil {
				return err
			}
		}
		return errors.New("применение не выполнено")
	}
	if _, err := writeSnapshot(root, account, agentName, portable); err != nil {
		return err
	}
	if _, err := clearPortable(root, home); err != nil {
		return err
	}
	for _, file := range next {
		if err := writeHomeFile(home, file); err != nil {
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

// clearPortable removes the portable files of a home. It first removes the directory links
// into the canon and returns them, so later writes land in the home itself.
func clearPortable(root, home string) ([]savedLink, error) {
	links, err := unlinkCanonDirs(root, home)
	if err != nil {
		return links, err
	}
	return links, clearPortableFiles(home)
}

func unlinkCanonDirs(root, home string) ([]savedLink, error) {
	canon := realPath(canonDir(root))
	var links []savedLink
	err := filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return filepath.SkipAll
		}
		if err != nil || entry.Type()&os.ModeSymlink == 0 {
			return err
		}
		target, err := filepath.EvalSymlinks(path)
		if err != nil || !within(canon, target) {
			return nil
		}
		if info, err := os.Stat(target); err != nil || !info.IsDir() {
			return nil
		}
		raw, err := os.Readlink(path)
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		links = append(links, savedLink{path: path, target: raw})
		return nil
	})
	return links, err
}

func clearPortableFiles(home string) error {
	files, err := readTree(home)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, file := range files {
		if _, ok := portableFile(file); !ok {
			continue
		}
		target := filepath.Join(home, filepath.FromSlash(file.Path))
		if machine := machineServers(file.Body); len(machine) > 0 {
			top, _, _ := mcpServers(file.Body)
			if err := os.WriteFile(target, []byte(encodeMCP(top, machine)), 0o644); err != nil {
				return err
			}
			continue
		}
		if err := os.Remove(target); err != nil {
			return err
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
		if !strings.EqualFold(item.Slug, name) && !strings.EqualFold(item.Display, name) && !slices.Contains(item.Aliases, strings.ToLower(name)) {
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

// readSnapshot reads an AI agent home for a snapshot. A directory link into the canon or inside
// the home is read as a copy under the link's path; a directory link elsewhere is left out.
func readSnapshot(root, home string) ([]File, error) {
	files, err := readTree(home)
	if err != nil {
		return nil, err
	}
	allowed := []string{realPath(canonDir(root)), realPath(home)}
	err = filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.Type()&os.ModeSymlink == 0 {
			return err
		}
		target, err := filepath.EvalSymlinks(path)
		if err != nil || !slices.ContainsFunc(allowed, func(dir string) bool { return within(dir, target) }) {
			return nil
		}
		rel, err := filepath.Rel(home, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info, err := os.Stat(target); err != nil || !info.IsDir() {
			return err
		}
		inner, err := readTree(target)
		if err != nil {
			return err
		}
		for _, file := range inner {
			files = append(files, File{Path: rel + "/" + file.Path, Body: file.Body})
		}
		return nil
	})
	return files, err
}

func realPath(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return filepath.Clean(dir)
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
		// A link to a directory is skipped: clearing a home must not reach into the canon. readSnapshot follows it.
		if entry.Type()&os.ModeSymlink != 0 {
			if info, err := os.Stat(path); err != nil || info.IsDir() {
				return nil
			}
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

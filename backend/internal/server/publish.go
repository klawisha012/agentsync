package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/klawisha012/agentsync/internal/agent"
	"github.com/labstack/echo/v4"
)

type publication struct {
	id        string
	agent     string
	version   int
	files     []storedFile
	packed    []byte
	created   time.Time
	withdrawn bool
}

type storedFile struct {
	Path string `json:"path"`
	Body string `json:"body"`
}

type excludedNote struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

var previewExcluded = []excludedNote{
	{Path: "~/…/.credentials", Label: "учётные данные"},
	{Path: "~/…/sessions.db", Label: "диалоги"},
	{Path: "~/…/cache/*", Label: "кэш"},
	{Path: "machine-mcp.json", Label: "машинный MCP"},
	{Path: "hooks/….sh", Label: "машинный хук"},
}

type publicationView struct {
	ID       string         `json:"id"`
	Agent    string         `json:"agent"`
	Version  int            `json:"version"`
	Author   string         `json:"author,omitempty"`
	Created  time.Time      `json:"created,omitempty"`
	Files    []storedFile   `json:"files"`
	Excluded []excludedNote `json:"excluded"`
}

var absolutePath = regexp.MustCompile(`(?i)(?:^|[\s"'=])(?:[a-z]:[\\/][^\s"']+|/(?:[a-z0-9._-]+/)*[a-z0-9._-]+)`)

func (s *store) push(sessionID, agentToken, agent string, missing bool, files []storedFile) (publicationView, int, string) {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return publicationView{}, http.StatusBadRequest, "Назовите ИИ-агента."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer := s.accountBySession(sessionID)
	if viewer == nil {
		return publicationView{}, http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if missing {
		return publicationView{}, http.StatusBadRequest, "На компьютере нет этого ИИ-агента."
	}
	kept, explain := classify(files)
	if explain != "" {
		return publicationView{}, http.StatusBadRequest, explain
	}
	packed, err := packFiles(kept)
	if err != nil {
		return publicationView{}, http.StatusInternalServerError, "Не удалось сохранить публикацию."
	}
	version := 1
	for _, old := range viewer.publications {
		if sameAgent(old.agent, agent) && old.version >= version {
			version = old.version + 1
		}
	}
	id, err := newID()
	if err != nil {
		return publicationView{}, http.StatusInternalServerError, "Не удалось сохранить публикацию."
	}
	created := time.Now()
	item := &publication{
		id:      id,
		agent:   agent,
		version: version,
		files:   kept,
		packed:  packed,
		created: created,
	}
	if err := s.insertPublication(context.Background(), viewer, item); err != nil {
		return publicationView{}, http.StatusInternalServerError, "Не удалось сохранить публикацию."
	}
	viewer.publications = append(viewer.publications, item)
	stamp := created
	viewer.publishedAt = &stamp
	return item.view(viewer.name), http.StatusCreated, ""
}

type versionHead struct {
	ID      string    `json:"id"`
	Version int       `json:"version"`
	Created time.Time `json:"created"`
	Current bool      `json:"current"`
}

func (s *store) versions(name, agentName, sessionID string) ([]versionHead, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page := s.byName[foldKey.String(name)]
	if page == nil {
		return nil, http.StatusNotFound, "Страница не найдена."
	}
	if viewer := s.accountBySession(sessionID); viewer != page && !page.shareVersions {
		return nil, http.StatusNotFound, "Версии скрыты."
	}
	var latest *publication
	heads := make([]versionHead, 0)
	for _, item := range page.publications {
		if item.withdrawn || !sameAgent(item.agent, agentName) {
			continue
		}
		if latest == nil || item.version > latest.version {
			latest = item
		}
		heads = append(heads, versionHead{ID: item.id, Version: item.version, Created: item.created})
	}
	for i := range heads {
		heads[i].Current = latest != nil && heads[i].ID == latest.id
	}
	slices.SortFunc(heads, func(a, b versionHead) int { return a.Version - b.Version })
	return heads, http.StatusOK, ""
}

func (a *app) listVersions(c echo.Context) error {
	heads, code, msg := a.accounts.versions(c.Param("name"), c.Param("agent"), sessionID(c))
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]any{"versions": heads})
}

func (s *store) version(name, agentName, sessionID string, number int) (publicationView, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page := s.byName[foldKey.String(name)]
	if page == nil {
		return publicationView{}, http.StatusNotFound, "Страница не найдена."
	}
	var latest *publication
	var found *publication
	for _, item := range page.publications {
		if item.withdrawn || !sameAgent(item.agent, agentName) {
			continue
		}
		if latest == nil || item.version > latest.version {
			latest = item
		}
		if item.version == number {
			found = item
		}
	}
	if found == nil {
		return publicationView{}, http.StatusNotFound, "Версия не найдена."
	}
	viewer := s.accountBySession(sessionID)
	if viewer != page && !page.shareVersions && !(page.shareView && latest != nil && latest.id == found.id) {
		return publicationView{}, http.StatusNotFound, "Эта версия скрыта."
	}
	return found.view(page.name), http.StatusOK, ""
}

func (a *app) showVersion(c echo.Context) error {
	number, err := strconv.Atoi(c.Param("version"))
	if err != nil || number < 1 {
		return writeExplanation(c, http.StatusNotFound, "Версия не найдена.")
	}
	view, code, msg := a.accounts.version(c.Param("name"), c.Param("agent"), sessionID(c), number)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (s *store) publication(id, sessionID string) (publicationView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer := s.accountBySession(sessionID)
	for _, page := range s.byName {
		var found *publication
		for _, item := range page.publications {
			if item.id == id && !item.withdrawn {
				found = item
				break
			}
		}
		if found == nil {
			continue
		}
		var latest *publication
		for _, item := range page.publications {
			if item.withdrawn || !sameAgent(item.agent, found.agent) {
				continue
			}
			if latest == nil || item.version > latest.version {
				latest = item
			}
		}
		if viewer == page || page.shareVersions || (page.shareView && latest != nil && latest.id == found.id) {
			return found.view(page.name), true
		}
		return publicationView{}, false
	}
	return publicationView{}, false
}

func (p publication) view(author string) publicationView {
	files := make([]storedFile, len(p.files))
	copy(files, p.files)
	excluded := append([]excludedNote(nil), previewExcluded...)
	return publicationView{
		ID: p.id, Agent: p.agent, Version: p.version, Author: author, Created: p.created,
		Files: files, Excluded: excluded,
	}
}

func classify(files []storedFile) ([]storedFile, string) {
	candidates := make([]storedFile, 0, len(files))
	for _, file := range files {
		if privatePath(file.Path) {
			continue
		}
		candidates = append(candidates, file)
	}
	hooks := map[string]struct{}{}
	scripts := map[string]struct{}{}
	for _, file := range candidates {
		if !isHook(file.Path) || !absolutePath.MatchString(file.Body) {
			continue
		}
		hooks[file.Path] = struct{}{}
		for _, match := range absolutePath.FindAllString(file.Body, -1) {
			cleaned := strings.Trim(match, " \t\"'=")
			scripts[strings.ToLower(filepath.Base(filepath.ToSlash(cleaned)))] = struct{}{}
		}
	}
	kept := make([]storedFile, 0)
	for _, file := range candidates {
		if _, drop := hooks[file.Path]; drop {
			continue
		}
		if _, drop := scripts[strings.ToLower(filepath.Base(file.Path))]; drop {
			continue
		}
		if isMCP(file.Path) && machineMCP(file.Body) {
			continue
		}
		if !portablePath(file.Path) {
			continue
		}
		if msg := agent.SecretExplanation(file.Path, file.Body); msg != "" {
			return nil, msg
		}
		kept = append(kept, file)
	}
	if len(kept) == 0 {
		return nil, "Нет переносимой настройки."
	}
	return kept, ""
}

func privatePath(path string) bool {
	clean := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(clean)
	switch base {
	case "credentials.json", "credentials", "auth.json", "session.json", ".env":
		return true
	}
	if strings.HasSuffix(base, ".lock") || strings.HasSuffix(base, ".log") {
		return true
	}
	for _, seg := range strings.Split(clean, "/") {
		switch seg {
		case "credentials", "sessions", "session", "cache", "caches", "logs", "log", "memory", "history", "conversations", "node_modules", "vendor", "dist", "locks", "bundled", "marketplace-cache":
			return true
		}
	}
	return false
}

func portablePath(path string) bool {
	clean := filepath.ToSlash(path)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	if isHook(path) || isMCP(path) {
		return true
	}
	base := strings.ToLower(filepath.Base(clean))
	switch base {
	case "config.json", "settings.json", "agents.md", "claude.md":
		return true
	}
	for _, seg := range strings.Split(strings.ToLower(clean), "/") {
		switch seg {
		case "rules", "skills", "plugins", "theme", "hooks", "mcp", "config":
			return true
		}
	}
	return false
}

func isHook(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	if base == "hooks.json" || base == "hook.json" {
		return true
	}
	return pathSegment(path, "hooks")
}

func isMCP(path string) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(path)))
	if base == "mcp.json" || strings.HasSuffix(base, ".mcp.json") {
		return true
	}
	return pathSegment(path, "mcp")
}

func pathSegment(path, want string) bool {
	for _, seg := range strings.Split(strings.ToLower(filepath.ToSlash(path)), "/") {
		if seg == want {
			return true
		}
	}
	return false
}

func machineMCP(body string) bool {
	if absolutePath.MatchString(body) {
		return true
	}
	if strings.Contains(strings.ToLower(body), "authorization") {
		return true
	}
	return agent.HasAssignedSecret(body)
}

func packFiles(files []storedFile) ([]byte, error) {
	raw, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	return encoder.EncodeAll(raw, nil), nil
}

func unpackFiles(packed []byte) ([]storedFile, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	raw, err := decoder.DecodeAll(packed, nil)
	if err != nil {
		return nil, err
	}
	var files []storedFile
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (a *app) pushAgent(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		Agent   string       `json:"agent"`
		Missing bool         `json:"missing"`
		Files   []storedFile `json:"files"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Назовите ИИ-агента.")
	}
	token := strings.TrimSpace(strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer "))
	view, code, msg := a.accounts.push(cookie.Value, token, req.Agent, req.Missing, req.Files)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (a *app) showPublication(c echo.Context) error {
	view, ok := a.accounts.publication(c.Param("id"), sessionID(c))
	if !ok {
		return writeExplanation(c, http.StatusNotFound, "Публикация не найдена.")
	}
	return c.JSON(http.StatusOK, view)
}

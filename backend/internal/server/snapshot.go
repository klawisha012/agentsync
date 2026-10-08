package server

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type snapshot struct {
	id      string
	agent   string
	number  int
	files   []storedFile
	packed  []byte
	created time.Time
}

type snapshotHead struct {
	ID      string    `json:"id"`
	Number  int       `json:"number"`
	Current bool      `json:"current"`
	Created time.Time `json:"created"`
}

type snapshotView struct {
	ID      string       `json:"id"`
	Agent   string       `json:"agent"`
	Account string       `json:"account"`
	Number  int          `json:"number"`
	Current bool         `json:"current"`
	Created time.Time    `json:"created"`
	Files   []storedFile `json:"files"`
}

type snapshotList struct {
	Agent     string         `json:"agent"`
	Snapshots []snapshotHead `json:"snapshots"`
}

func (s *store) record(ctx context.Context, sessionID, agentToken, agent string, missing bool, files []storedFile) (snapshotView, int, string) {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return snapshotView{}, http.StatusBadRequest, "Назовите ИИ-агента."
	}
	if msg := safeAgentName(agent); msg != "" {
		return snapshotView{}, http.StatusBadRequest, msg
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer, code, msg := s.owner(sessionID, agentToken, false)
	if msg != "" {
		return snapshotView{}, code, msg
	}
	if missing {
		return snapshotView{}, http.StatusBadRequest, "На компьютере нет этого ИИ-агента."
	}
	kept, explain := classify(files)
	if explain != "" {
		return snapshotView{}, http.StatusBadRequest, explain
	}
	packed, err := packFiles(kept)
	if err != nil {
		return snapshotView{}, http.StatusInternalServerError, "Не удалось записать снимок."
	}
	number := 1
	for _, old := range viewer.snapshots {
		if sameAgent(old.agent, agent) && old.number >= number {
			number = old.number + 1
		}
	}
	id, err := newID()
	if err != nil {
		return snapshotView{}, http.StatusInternalServerError, "Не удалось записать снимок."
	}
	created := time.Now()
	item := &snapshot{
		id: id, agent: agent, number: number,
		files: kept, packed: packed, created: created,
	}
	if err := s.insertSnapshot(ctx, viewer, item); err != nil {
		return snapshotView{}, http.StatusInternalServerError, "Не удалось записать снимок."
	}
	viewer.snapshots = append(viewer.snapshots, item)
	viewer.marks[agent] = item.id
	return item.view(viewer.name, true), http.StatusCreated, ""
}

func (s *store) listSnapshots(sessionID, agentToken, agent string) (snapshotList, int, string) {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return snapshotList{}, http.StatusBadRequest, "Назовите ИИ-агента."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer, code, msg := s.owner(sessionID, agentToken, false)
	if msg != "" {
		return snapshotList{}, code, msg
	}
	heads := make([]snapshotHead, 0)
	for _, item := range viewer.snapshots {
		if !sameAgent(item.agent, agent) {
			continue
		}
		heads = append(heads, snapshotHead{
			ID: item.id, Number: item.number, Created: item.created,
			Current: viewer.marks[agent] == item.id,
		})
	}
	slices.SortFunc(heads, func(a, b snapshotHead) int { return a.Number - b.Number })
	return snapshotList{Agent: agent, Snapshots: heads}, http.StatusOK, ""
}

func (s *store) snapshot(sessionID, agentToken, agent string, number int) (snapshotView, int, string) {
	agent = strings.TrimSpace(agent)
	if agent == "" || number < 1 {
		return snapshotView{}, http.StatusBadRequest, "Назовите ИИ-агента."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer, code, msg := s.owner(sessionID, agentToken, false)
	if msg != "" {
		return snapshotView{}, code, msg
	}
	for _, item := range viewer.snapshots {
		if sameAgent(item.agent, agent) && item.number == number {
			return item.view(viewer.name, viewer.marks[agent] == item.id), http.StatusOK, ""
		}
	}
	return snapshotView{}, http.StatusNotFound, "Снимок не найден."
}

func (s *store) owner(sessionID, agentToken string, needMachine bool) (*account, int, string) {
	viewer := s.accountBySession(sessionID)
	if viewer != nil {
		return viewer, 0, ""
	}
	bound := s.machineByToken(agentToken)
	if bound != nil && bound.account != nil {
		return bound.account, 0, ""
	}
	if needMachine {
		return nil, http.StatusForbidden, "Аккаунт браузера и аккаунт машины различаются."
	}
	return nil, http.StatusUnauthorized, "Войдите в аккаунт."
}

func (item *snapshot) view(account string, current bool) snapshotView {
	files := make([]storedFile, len(item.files))
	copy(files, item.files)
	return snapshotView{
		ID: item.id, Agent: item.agent, Account: account,
		Number: item.number, Current: current, Created: item.created, Files: files,
	}
}

func (a *app) recordSnapshot(c echo.Context) error {
	sessionID := ""
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		sessionID = cookie.Value
	}
	var req struct {
		Agent   string       `json:"agent"`
		Missing bool         `json:"missing"`
		Files   []storedFile `json:"files"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Назовите ИИ-агента.")
	}
	token := bearer(c)
	view, code, msg := a.accounts.record(c.Request().Context(), sessionID, token, req.Agent, req.Missing, req.Files)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (a *app) listSnapshots(c echo.Context) error {
	sessionID := ""
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		sessionID = cookie.Value
	}
	view, code, msg := a.accounts.listSnapshots(sessionID, bearer(c), c.Param("agent"))
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (a *app) showSnapshot(c echo.Context) error {
	sessionID := ""
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		sessionID = cookie.Value
	}
	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Назовите ИИ-агента.")
	}
	view, code, msg := a.accounts.snapshot(sessionID, bearer(c), c.Param("agent"), number)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func bearer(c echo.Context) string {
	return strings.TrimSpace(strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer "))
}

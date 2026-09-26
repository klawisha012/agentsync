package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *store) applyPublication(author, agentName, sessionID, token string) (publicationView, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page := s.byName[foldKey.String(author)]
	if page == nil {
		return publicationView{}, http.StatusNotFound, "Страница не найдена."
	}
	var pub *publication
	for _, item := range page.publications {
		if item.withdrawn || item.agent != agentName {
			continue
		}
		if pub == nil || item.version > pub.version {
			pub = item
		}
	}
	if pub == nil {
		return publicationView{}, http.StatusNotFound, "Публикация не найдена."
	}
	machine := s.byAgentToken[token]
	if machine == nil || machine.account == nil {
		return publicationView{}, http.StatusUnauthorized, "Локальный агент не вошёл в аккаунт."
	}
	if machine.account == page {
		session := s.accountBySession(sessionID)
		if session == nil || session != page {
			return publicationView{}, http.StatusForbidden, "Свою публикацию можно применить, когда в браузере открыт этот аккаунт."
		}
	}
	return pub.view(page.name), http.StatusOK, ""
}

func (s *store) withdraw(sessionID, id string) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if !owner.verified {
		return http.StatusForbidden, mailClosed
	}
	for _, item := range owner.publications {
		if item.id != id || item.withdrawn {
			continue
		}
		item.withdrawn = true
		if err := s.markWithdrawn(context.Background(), item.id); err != nil {
			item.withdrawn = false
			return http.StatusInternalServerError, "Не удалось снять публикацию."
		}
		return http.StatusOK, ""
	}
	return http.StatusNotFound, "Публикация не найдена."
}

func (s *store) markWithdrawn(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE publications SET withdrawn = true WHERE id = $1`, id)
	return err
}

func (s *store) rename(sessionID, name string) (string, int, string) {
	name = strings.TrimSpace(name)
	if msg, code := checkName(name); msg != "" {
		return "", code, msg
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return "", http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if !owner.verified {
		return "", http.StatusForbidden, "Подтвердите почту, чтобы сменить имя."
	}
	key := foldKey.String(name)
	if other := s.byName[key]; other != nil && other != owner {
		return "", http.StatusConflict, "Это имя уже занято."
	}
	prevName, prevKey := owner.name, owner.nameKey
	owner.name, owner.nameKey = name, key
	if _, err := s.db.ExecContext(context.Background(), `UPDATE accounts SET name = $2, name_key = $3 WHERE id = $1`, owner.id, name, key); err != nil {
		owner.name, owner.nameKey = prevName, prevKey
		if msg, ok := uniqueMessage(err); ok {
			return "", http.StatusConflict, msg
		}
		return "", http.StatusInternalServerError, "Не удалось сменить имя."
	}
	delete(s.byName, prevKey)
	s.byName[key] = owner
	return name, http.StatusOK, ""
}

func (s *store) removeAccount(sessionID, password string) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if strings.TrimSpace(password) == "" || bcrypt.CompareHashAndPassword(owner.password, []byte(password)) != nil {
		return http.StatusForbidden, "Неверный пароль."
	}
	if _, err := s.db.ExecContext(context.Background(), `DELETE FROM accounts WHERE id = $1`, owner.id); err != nil {
		return http.StatusInternalServerError, "Не удалось удалить аккаунт."
	}
	delete(s.byEmail, owner.emailKey)
	delete(s.byName, owner.nameKey)
	for id, item := range s.sessions {
		if item == owner {
			delete(s.sessions, id)
			delete(s.sessionExpiry, id)
		}
	}
	for id, bound := range s.machines {
		if bound.account == owner {
			delete(s.byAgentToken, bound.token)
			delete(s.machines, id)
		}
	}
	for _, page := range s.byName {
		if _, ok := page.likedBy[owner.nameKey]; ok {
			delete(page.likedBy, owner.nameKey)
			page.likes = len(page.likedBy)
		}
	}
	return http.StatusOK, ""
}

func (a *app) applyPublication(c echo.Context) error {
	sessionID := ""
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		sessionID = cookie.Value
	}
	token := strings.TrimSpace(strings.TrimPrefix(c.Request().Header.Get(echo.HeaderAuthorization), "Bearer "))
	view, code, msg := a.accounts.applyPublication(c.Param("name"), c.Param("agent"), sessionID, token)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (a *app) withdrawPublication(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	code, msg := a.accounts.withdraw(cookie.Value, c.Param("id"))
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]bool{"withdrawn": true})
}

func (a *app) renameAccount(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите имя.")
	}
	name, code, msg := a.accounts.rename(cookie.Value, req.Name)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]string{"name": name})
}

func (a *app) deleteAccount(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите пароль.")
	}
	code, msg := a.accounts.removeAccount(cookie.Value, req.Password)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	setSessionCookie(c, "", true)
	return c.JSON(code, map[string]bool{"deleted": true})
}

func installPS(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	return c.String(http.StatusOK, installPowerShell)
}

func installSH(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	return c.String(http.StatusOK, installShell)
}

const installPowerShell = `# AgentSync. Ставит локального агента и останавливается на подтверждении машины на сайте.
# Публикацию не применяет и ИИ-агента не загружает. Пароль в эту инструкцию не входит.
$dir = Join-Path $env:USERPROFILE ".agentsync"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
Set-Content -Path (Join-Path $dir "installed") -Value "confirm-machine"
Write-Output "Локальный агент установлен. Подтвердите машину на сайте."
`

const installShell = `#!/bin/sh
# AgentSync. Ставит локального агента и останавливается на подтверждении машины на сайте.
# Публикацию не применяет и ИИ-агента не загружает. Пароль в эту инструкцию не входит.
dir="${HOME}/.agentsync"
mkdir -p "$dir"
printf '%s\n' confirm-machine > "$dir/installed"
echo "Локальный агент установлен. Подтвердите машину на сайте."
`

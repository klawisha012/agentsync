package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *store) applyPublication(author, agentName, sessionID, token string, version int) (publicationView, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page := s.byName[foldKey.String(author)]
	if page == nil {
		return publicationView{}, http.StatusNotFound, "Страница не найдена."
	}
	var pub *publication
	for _, item := range page.publications {
		if item.withdrawn || !sameAgent(item.agent, agentName) {
			continue
		}
		if version > 0 {
			if item.version == version {
				pub = item
			}
			continue
		}
		if pub == nil || item.version > pub.version {
			pub = item
		}
	}
	if pub == nil {
		if version > 0 {
			return publicationView{}, http.StatusNotFound, "Версия не найдена."
		}
		return publicationView{}, http.StatusNotFound, "Публикация не найдена."
	}
	machine := s.byAgentToken[token]
	session := s.accountBySession(sessionID)
	if (machine == nil || machine.account == nil) && session == nil {
		return publicationView{}, http.StatusUnauthorized, "Локальный агент не вошёл в аккаунт."
	}
	if machine != nil && machine.account == page && (session == nil || session != page) {
		return publicationView{}, http.StatusForbidden, "Свою публикацию можно применить, когда в браузере открыт этот аккаунт."
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

func (s *store) setPrivacy(sessionID string, copy, view, versions bool, comments string) (int, string) {
	switch comments {
	case "", "open", "users", "hidden":
	default:
		return http.StatusBadRequest, "Неизвестный доступ к комментариям."
	}
	if comments == "" {
		comments = "hidden"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	_, err := s.db.ExecContext(context.Background(), `
		UPDATE accounts SET share_copy = $2, share_view = $3, share_versions = $4, comment_access = $5 WHERE id = $1`,
		owner.id, copy, view, versions, comments)
	if err != nil {
		return http.StatusInternalServerError, "Не удалось сохранить приватность."
	}
	owner.shareCopy, owner.shareView, owner.shareVersions, owner.commentAccess = copy, view, versions, comments
	return http.StatusOK, ""
}

func (s *store) changeEmail(sessionID, email, password string) (string, string, int, string) {
	email = strings.TrimSpace(email)
	if !validEmail(email) {
		return "", "", http.StatusBadRequest, "Введите почту в\u00a0виде name@example.com."
	}
	if strings.TrimSpace(password) == "" {
		return "", "", http.StatusBadRequest, "Введите пароль."
	}
	token, err := newID()
	if err != nil {
		return "", "", http.StatusInternalServerError, "Не удалось сменить почту."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return "", "", http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if bcrypt.CompareHashAndPassword(owner.password, []byte(password)) != nil {
		return "", "", http.StatusForbidden, "Неверный пароль."
	}
	key := foldKey.String(email)
	if other := s.byEmail[key]; other != nil && other != owner {
		return "", "", http.StatusConflict, "Аккаунт с\u00a0этой почтой уже есть."
	}
	if key == owner.emailKey {
		letter := ""
		if owner.confirmToken != "" && time.Now().Before(owner.confirmExpires) {
			letter = "/confirm/" + owner.confirmToken
		}
		return maskMail(owner.email), letter, http.StatusOK, ""
	}
	prevEmail, prevKey := owner.email, owner.emailKey
	prevVerified, prevToken, prevExpires := owner.verified, owner.confirmToken, owner.confirmExpires
	owner.email, owner.emailKey = email, key
	owner.verified = false
	owner.confirmToken = token
	owner.confirmExpires = time.Now().Add(letterTTL)
	_, err = s.db.ExecContext(context.Background(), `
		UPDATE accounts SET email = $2, email_key = $3, verified = $4, confirm_token = $5, confirm_expires = $6
		WHERE id = $1`,
		owner.id, owner.email, owner.emailKey, owner.verified,
		nullString(owner.confirmToken), nullTime(owner.confirmExpires),
	)
	if err != nil {
		owner.email, owner.emailKey = prevEmail, prevKey
		owner.verified, owner.confirmToken, owner.confirmExpires = prevVerified, prevToken, prevExpires
		if msg, ok := uniqueMessage(err); ok {
			return "", "", http.StatusConflict, msg
		}
		return "", "", http.StatusInternalServerError, "Не удалось сменить почту."
	}
	delete(s.byEmail, prevKey)
	s.byEmail[key] = owner
	return maskMail(email), "/confirm/" + token, http.StatusOK, ""
}

func (s *store) changePassword(sessionID, current, next string) (int, string) {
	if strings.TrimSpace(next) == "" {
		return http.StatusBadRequest, "Введите новый пароль."
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusInternalServerError, "Не удалось сменить пароль."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	if strings.TrimSpace(current) == "" || bcrypt.CompareHashAndPassword(owner.password, []byte(current)) != nil {
		return http.StatusForbidden, "Неверный пароль."
	}
	prev := append([]byte(nil), owner.password...)
	owner.password = hash
	if err := s.saveAccount(context.Background(), owner); err != nil {
		owner.password = prev
		return http.StatusInternalServerError, "Не удалось сменить пароль."
	}
	return http.StatusOK, ""
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
		for _, likers := range page.agentLikes {
			delete(likers, owner.nameKey)
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
	version, err := requestedVersion(c.QueryParam("version"))
	if err != nil {
		return writeExplanation(c, http.StatusNotFound, "Версия не найдена.")
	}
	view, code, msg := a.accounts.applyPublication(c.Param("name"), c.Param("agent"), sessionID, token, version)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func requestedVersion(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	number, err := strconv.Atoi(raw)
	if err != nil || number < 1 {
		return 0, strconv.ErrSyntax
	}
	return number, nil
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

func (a *app) setPrivacy(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		Copy     bool   `json:"copy"`
		View     bool   `json:"view"`
		Versions bool   `json:"versions"`
		Comments string `json:"comments"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Не удалось сохранить приватность.")
	}
	if req.Comments == "" {
		req.Comments = "hidden"
	}
	code, msg := a.accounts.setPrivacy(cookie.Value, req.Copy, req.View, req.Versions, req.Comments)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]any{"copy": req.Copy, "view": req.View, "versions": req.Versions, "comments": req.Comments})
}

func (a *app) changeEmail(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите почту в\u00a0виде name@example.com.")
	}
	masked, letter, code, msg := a.accounts.changeEmail(cookie.Value, req.Email, req.Password)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	body := map[string]any{"maskedMail": masked, "verified": false}
	if letter != "" {
		body["letterPath"] = letter
	}
	return c.JSON(code, body)
}

func (a *app) changePassword(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	var req struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите новый пароль.")
	}
	code, msg := a.accounts.changePassword(cookie.Value, req.Current, req.Next)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]bool{"ok": true})
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

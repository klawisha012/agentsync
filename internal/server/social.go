package server

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

const visitorCookie = "visitor"

func (s *store) bindVisitor(sessionID, visitorID string) {
	if sessionID == "" || visitorID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer := s.accountBySession(sessionID)
	if viewer == nil {
		return
	}
	from := "v:" + visitorID
	to := "a:" + viewer.nameKey
	for _, page := range s.byName {
		if _, ok := page.viewers[from]; !ok {
			continue
		}
		delete(page.viewers, from)
		_ = s.dropView(context.Background(), page, from)
		if _, ok := page.viewers[to]; !ok {
			if page.viewers == nil {
				page.viewers = map[string]struct{}{}
			}
			page.viewers[to] = struct{}{}
			_ = s.putView(context.Background(), page, to)
		}
		page.views = len(page.viewers)
	}
}

func (s *store) recordView(page, viewer *account, visitorID string) {
	if viewer != nil && viewer == page {
		return
	}
	if page.viewers == nil {
		page.viewers = map[string]struct{}{}
	}
	ctx := context.Background()
	if viewer != nil {
		if visitorID != "" {
			delete(page.viewers, "v:"+visitorID)
			_ = s.dropView(ctx, page, "v:"+visitorID)
		}
		key := "a:" + viewer.nameKey
		page.viewers[key] = struct{}{}
		_ = s.putView(ctx, page, key)
		page.views = len(page.viewers)
		return
	}
	if visitorID == "" {
		return
	}
	key := "v:" + visitorID
	page.viewers[key] = struct{}{}
	_ = s.putView(ctx, page, key)
	page.views = len(page.viewers)
}

func (s *store) toggleLike(name, sessionID string) (int, bool, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer := s.accountBySession(sessionID)
	if viewer == nil {
		return 0, false, http.StatusUnauthorized, "Войдите в аккаунт, чтобы поставить лайк."
	}
	page := s.byName[foldKey.String(name)]
	if page == nil {
		return 0, false, http.StatusNotFound, "Страница не найдена."
	}
	if viewer == page {
		return page.likes, false, http.StatusForbidden, "Свою страницу лайкнуть нельзя."
	}
	if page.likedBy == nil {
		page.likedBy = map[string]struct{}{}
	}
	_, on := page.likedBy[viewer.nameKey]
	ctx := context.Background()
	if on {
		delete(page.likedBy, viewer.nameKey)
		if err := s.dropLike(ctx, page, viewer); err != nil {
			return 0, false, http.StatusInternalServerError, "Не удалось снять лайк."
		}
	} else {
		page.likedBy[viewer.nameKey] = struct{}{}
		if err := s.putLike(ctx, page, viewer); err != nil {
			return 0, false, http.StatusInternalServerError, "Не удалось поставить лайк."
		}
	}
	page.likes = len(page.likedBy)
	return page.likes, !on, http.StatusOK, ""
}

func (a *app) toggleLike(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт, чтобы поставить лайк.")
	}
	likes, liked, code, msg := a.accounts.toggleLike(c.Param("name"), cookie.Value)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]any{"likes": likes, "liked": liked})
}

func sessionID(c echo.Context) string {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func ensureVisitor(c echo.Context) string {
	if cookie, err := c.Cookie(visitorCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	id, err := newID()
	if err != nil {
		return ""
	}
	secure := c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"
	c.SetCookie(&http.Cookie{
		Name:     visitorCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   400 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	return id
}

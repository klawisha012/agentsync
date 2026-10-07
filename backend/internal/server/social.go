package server

import (
	"context"
	"net/http"
	"time"

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
		if page.id == viewer.id {
			s.dropViewerKey(page, from)
			s.dropViewerKey(page, to)
			continue
		}
		seen := time.Now().UTC()
		if page.viewedAt != nil && !page.viewedAt[from].IsZero() {
			seen = page.viewedAt[from]
		}
		if err := s.dropView(context.Background(), page, from); err != nil {
			continue
		}
		if _, ok := page.viewers[to]; !ok {
			if err := s.putView(context.Background(), page, to, seen); err != nil {
				_ = s.putView(context.Background(), page, from, seen)
				continue
			}
		}
		delete(page.viewers, from)
		delete(page.viewedAt, from)
		if page.viewers == nil {
			page.viewers = map[string]struct{}{}
		}
		if page.viewedAt == nil {
			page.viewedAt = map[string]time.Time{}
		}
		page.viewers[to] = struct{}{}
		if _, ok := page.viewedAt[to]; !ok {
			page.viewedAt[to] = seen
		}
		page.views = len(page.viewers)
	}
}

func (s *store) dropViewerKey(page *account, key string) {
	if page.viewers == nil {
		return
	}
	if _, ok := page.viewers[key]; !ok {
		return
	}
	if err := s.dropView(context.Background(), page, key); err != nil {
		return
	}
	delete(page.viewers, key)
	delete(page.viewedAt, key)
	page.views = len(page.viewers)
}

func (s *store) recordView(page, viewer *account, visitorID string) {
	if viewer != nil && viewer.id == page.id {
		if visitorID != "" {
			s.dropViewerKey(page, "v:"+visitorID)
		}
		s.dropViewerKey(page, "a:"+page.nameKey)
		return
	}
	if page.viewers == nil {
		page.viewers = map[string]struct{}{}
	}
	if page.viewedAt == nil {
		page.viewedAt = map[string]time.Time{}
	}
	ctx := context.Background()
	if viewer != nil {
		key := "a:" + viewer.nameKey
		if _, ok := page.viewers[key]; !ok {
			seen := time.Now().UTC()
			if err := s.putView(ctx, page, key, seen); err != nil {
				return
			}
			page.viewers[key] = struct{}{}
			page.viewedAt[key] = seen
		}
		if visitorID != "" {
			if err := s.dropView(ctx, page, "v:"+visitorID); err != nil {
				return
			}
			delete(page.viewers, "v:"+visitorID)
			delete(page.viewedAt, "v:"+visitorID)
		}
		page.views = len(page.viewers)
		return
	}
	if visitorID == "" {
		return
	}
	key := "v:" + visitorID
	if _, ok := page.viewers[key]; ok {
		return
	}
	seen := time.Now().UTC()
	if err := s.putView(ctx, page, key, seen); err != nil {
		return
	}
	page.viewers[key] = struct{}{}
	page.viewedAt[key] = seen
	page.views = len(page.viewers)
}

func (a *account) hasPublication(agent string) bool {
	key := foldKey.String(agent)
	for _, pub := range a.publications {
		if !pub.withdrawn && foldKey.String(pub.agent) == key {
			return true
		}
	}
	return false
}

func (a *account) publicationLikes() int {
	seen := map[string]struct{}{}
	total := 0
	for _, pub := range a.publications {
		if pub.withdrawn {
			continue
		}
		key := foldKey.String(pub.agent)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		total += len(a.agentLikes[key])
	}
	return total
}

func (a *account) agentLikeCount(agent string) int {
	if !a.hasPublication(agent) || a.agentLikes == nil {
		return 0
	}
	return len(a.agentLikes[foldKey.String(agent)])
}

func (a *account) agentLiked(agent string, viewer *account) bool {
	if viewer == nil || a.agentLikes == nil || !a.hasPublication(agent) {
		return false
	}
	_, ok := a.agentLikes[foldKey.String(agent)][viewer.nameKey]
	return ok
}

func (s *store) toggleLike(name, agent, sessionID string) (int, bool, int, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	viewer := s.accountBySession(sessionID)
	if viewer == nil {
		return 0, false, 0, http.StatusUnauthorized, "Войдите в\u00a0аккаунт, чтобы поставить лайк."
	}
	page := s.byName[foldKey.String(name)]
	if page == nil {
		return 0, false, 0, http.StatusNotFound, "Страница не найдена."
	}
	if !page.hasPublication(agent) {
		return 0, false, page.publicationLikes(), http.StatusNotFound, "Публикация не найдена."
	}
	if viewer == page {
		return page.agentLikeCount(agent), false, page.publicationLikes(), http.StatusForbidden, "Свою публикацию лайкнуть нельзя."
	}
	key := foldKey.String(agent)
	if page.agentLikes == nil {
		page.agentLikes = map[string]map[string]struct{}{}
	}
	if page.agentLikes[key] == nil {
		page.agentLikes[key] = map[string]struct{}{}
	}
	_, on := page.agentLikes[key][viewer.nameKey]
	ctx := context.Background()
	if on {
		if err := s.dropAgentLike(ctx, page, key, viewer); err != nil {
			return 0, false, 0, http.StatusInternalServerError, "Не удалось снять лайк."
		}
		delete(page.agentLikes[key], viewer.nameKey)
		if page.likedAt[key] != nil {
			delete(page.likedAt[key], viewer.nameKey)
		}
	} else {
		created := time.Now().UTC()
		if err := s.putAgentLike(ctx, page, key, viewer, created); err != nil {
			return 0, false, 0, http.StatusInternalServerError, "Не удалось поставить лайк."
		}
		page.agentLikes[key][viewer.nameKey] = struct{}{}
		if page.likedAt == nil {
			page.likedAt = map[string]map[string]time.Time{}
		}
		if page.likedAt[key] == nil {
			page.likedAt[key] = map[string]time.Time{}
		}
		page.likedAt[key][viewer.nameKey] = created
	}
	page.likes = page.publicationLikes()
	return page.agentLikeCount(agent), !on, page.likes, http.StatusOK, ""
}

func (a *app) toggleLike(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в\u00a0аккаунт, чтобы поставить лайк.")
	}
	likes, liked, total, code, msg := a.accounts.toggleLike(c.Param("name"), c.Param("agent"), cookie.Value)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]any{"likes": likes, "liked": liked, "total": total})
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

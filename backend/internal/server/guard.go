package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type hitWindow struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func (w *hitWindow) allow(key string, limit int, span time.Duration) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.hits == nil {
		w.hits = map[string][]time.Time{}
	}
	now := time.Now()
	cut := now.Add(-span)
	kept := w.hits[key][:0]
	for _, at := range w.hits[key] {
		if at.After(cut) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= limit {
		w.hits[key] = kept
		return false
	}
	w.hits[key] = append(kept, now)
	return true
}

func (a *app) limitAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !a.authHits.allow(clientIP(c), 120, time.Minute) {
			return writeExplanation(c, http.StatusTooManyRequests, "Слишком много попыток. Подождите минуту.")
		}
		return next(c)
	}
}

func (a *app) guardOrigin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch c.Request().Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !originAllowed(c.Request(), a.webOrigin) {
				return writeExplanation(c, http.StatusForbidden, "Запрос отклонён.")
			}
		}
		return next(c)
	}
}

func originAllowed(r *http.Request, webOrigin string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if webOrigin != "" && origin == strings.TrimRight(webOrigin, "/") {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

func clientIP(c echo.Context) string {
	if ip := strings.TrimSpace(c.Request().Header.Get("X-Real-IP")); ip != "" && !strings.Contains(ip, ",") {
		return ip
	}
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		return c.Request().RemoteAddr
	}
	return host
}

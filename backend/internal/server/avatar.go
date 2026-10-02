package server

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	maxAvatarBytes = 2 << 20
	maxAvatarEdge  = 1024
)

func checkAvatar(raw []byte) (string, string) {
	if len(raw) == 0 {
		return "", "Выберите файл."
	}
	if len(raw) > maxAvatarBytes {
		return "", "Файл больше 2 МБ."
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "gif" && format != "png" && format != "jpeg") {
		return "", "Подойдёт PNG, JPEG или GIF."
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		return "", "Не удалось прочитать картинку."
	}
	if cfg.Width > maxAvatarEdge || cfg.Height > maxAvatarEdge {
		return "", "Сторона картинки больше 1\u202f024 пикселей."
	}
	if _, _, err := image.Decode(bytes.NewReader(raw)); err != nil {
		return "", "Не удалось прочитать картинку."
	}
	switch format {
	case "gif":
		return "image/gif", ""
	case "png":
		return "image/png", ""
	default:
		return "image/jpeg", ""
	}
}

func avatarStamp(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func (s *store) setAvatar(ctx context.Context, sessionID string, raw []byte) (time.Time, int, string) {
	media, msg := checkAvatar(raw)
	if msg != "" {
		return time.Time{}, http.StatusBadRequest, msg
	}
	updated := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return time.Time{}, http.StatusUnauthorized, "Войдите в аккаунт."
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, http.StatusInternalServerError, "Не удалось сохранить аватар."
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO account_avatars (account_id, media_type, body)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET media_type = EXCLUDED.media_type, body = EXCLUDED.body`,
		owner.id, media, raw); err != nil {
		return time.Time{}, http.StatusInternalServerError, "Не удалось сохранить аватар."
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts SET has_avatar = true, avatar_updated = $2 WHERE id = $1`,
		owner.id, updated); err != nil {
		return time.Time{}, http.StatusInternalServerError, "Не удалось сохранить аватар."
	}
	if err := tx.Commit(); err != nil {
		return time.Time{}, http.StatusInternalServerError, "Не удалось сохранить аватар."
	}
	owner.hasAvatar = true
	owner.avatarUpdated = updated
	return updated, http.StatusOK, ""
}

func (s *store) clearAvatar(ctx context.Context, sessionID string) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.accountBySession(sessionID)
	if owner == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return http.StatusInternalServerError, "Не удалось удалить аватар."
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_avatars WHERE account_id = $1`, owner.id); err != nil {
		return http.StatusInternalServerError, "Не удалось удалить аватар."
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE accounts SET has_avatar = false, avatar_updated = NULL WHERE id = $1`, owner.id); err != nil {
		return http.StatusInternalServerError, "Не удалось удалить аватар."
	}
	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, "Не удалось удалить аватар."
	}
	owner.hasAvatar = false
	owner.avatarUpdated = time.Time{}
	return http.StatusOK, ""
}

func (s *store) avatar(ctx context.Context, name string) (string, []byte, time.Time, int, string) {
	s.mu.Lock()
	item := s.byName[foldKey.String(name)]
	if item == nil {
		s.mu.Unlock()
		return "", nil, time.Time{}, http.StatusNotFound, "Страница не найдена."
	}
	if !item.hasAvatar {
		s.mu.Unlock()
		return "", nil, time.Time{}, http.StatusNotFound, "Аватар не найден."
	}
	id := item.id
	s.mu.Unlock()
	var media string
	var body []byte
	var updated time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT media_type, body, avatar_updated
		FROM account_avatars
		JOIN accounts ON accounts.id = account_avatars.account_id
		WHERE account_avatars.account_id = $1`, id).Scan(&media, &body, &updated)
	if err != nil {
		return "", nil, time.Time{}, http.StatusNotFound, "Аватар не найден."
	}
	return media, body, updated, http.StatusOK, ""
}

func (a *app) putAvatar(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxAvatarBytes+8192)
	file, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return writeExplanation(c, http.StatusBadRequest, "Файл больше 2 МБ.")
		}
		return writeExplanation(c, http.StatusBadRequest, "Выберите файл.")
	}
	opened, err := file.Open()
	if err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Выберите файл.")
	}
	defer opened.Close()
	raw, err := io.ReadAll(io.LimitReader(opened, maxAvatarBytes+1))
	if err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Выберите файл.")
	}
	updated, code, msg := a.accounts.setAvatar(c.Request().Context(), cookie.Value, raw)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]any{"hasAvatar": true, "avatarUpdated": updated})
}

func (a *app) deleteAvatar(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	code, msg := a.accounts.clearAvatar(c.Request().Context(), cookie.Value)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]any{"hasAvatar": false})
}

func (a *app) showAvatar(c echo.Context) error {
	media, body, updated, code, msg := a.accounts.avatar(c.Request().Context(), c.Param("name"))
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	tag := `"` + updated.UTC().Format(time.RFC3339Nano) + `"`
	header := c.Response().Header()
	header.Set("ETag", tag)
	header.Set("Cache-Control", "public, max-age=86400")
	header.Set("X-Content-Type-Options", "nosniff")
	if c.Request().Header.Get("If-None-Match") == tag {
		return c.NoContent(http.StatusNotModified)
	}
	return c.Blob(http.StatusOK, media, body)
}

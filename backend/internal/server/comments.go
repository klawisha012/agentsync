package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v4"
)

const (
	commentPageSize = 6
	commentMaxRunes = 1000
)

type profileComment struct {
	id      string
	author  *account
	body    string
	created time.Time
}

type commentView struct {
	ID            string     `json:"id"`
	Author        string     `json:"author"`
	Body          string     `json:"body"`
	Created       time.Time  `json:"created"`
	CanDelete     bool       `json:"canDelete"`
	HasAvatar     bool       `json:"hasAvatar"`
	AvatarUpdated *time.Time `json:"avatarUpdated,omitempty"`
}

type commentList struct {
	Hidden   bool          `json:"hidden"`
	CanPost  bool          `json:"canPost"`
	Page     int           `json:"page"`
	Pages    int           `json:"pages"`
	Total    int           `json:"total"`
	Comments []commentView `json:"comments"`
}

func commentMode(value string) string {
	switch value {
	case "open", "users", "hidden":
		return value
	default:
		return "hidden"
	}
}

func (s *store) loadComments(ctx context.Context, byID map[string]*account) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, author_id, body, created_at FROM profile_comments ORDER BY created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID, authorID string
		item := &profileComment{}
		if err := rows.Scan(&item.id, &accountID, &authorID, &item.body, &item.created); err != nil {
			return err
		}
		page := byID[accountID]
		author := byID[authorID]
		if page == nil || author == nil {
			continue
		}
		item.author = author
		page.comments = append(page.comments, item)
	}
	return rows.Err()
}

func paginateComments(items []commentView, page int) (int, int, []commentView) {
	total := len(items)
	pages := 0
	if total > 0 {
		pages = (total + commentPageSize - 1) / commentPageSize
	}
	if page < 1 {
		page = 1
	}
	if pages > 0 && page > pages {
		page = pages
	}
	if total == 0 {
		return 1, 0, []commentView{}
	}
	start := (page - 1) * commentPageSize
	end := start + commentPageSize
	if end > total {
		end = total
	}
	return page, pages, items[start:end]
}

func (s *store) commentList(name, sessionID string, page int) (commentList, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account := s.byName[foldKey.String(name)]
	if account == nil {
		return commentList{}, http.StatusNotFound, "Страница не найдена."
	}
	viewer := s.accountBySession(sessionID)
	owner := viewer == account
	mode := commentMode(account.commentAccess)
	if !owner && (mode == "hidden" || (mode == "users" && viewer == nil)) {
		return commentList{Hidden: true, Page: 1, Comments: []commentView{}}, http.StatusOK, ""
	}
	views := make([]commentView, 0, len(account.comments))
	for i := len(account.comments) - 1; i >= 0; i-- {
		item := account.comments[i]
		views = append(views, commentView{
			ID: item.id, Author: item.author.name, Body: item.body, Created: item.created,
			CanDelete: owner || viewer == item.author,
			HasAvatar: item.author.hasAvatar, AvatarUpdated: avatarStamp(item.author.avatarUpdated),
		})
	}
	current, pages, slice := paginateComments(views, page)
	return commentList{
		CanPost:  viewer != nil && (owner || mode == "open" || mode == "users"),
		Page:     current,
		Pages:    pages,
		Total:    len(views),
		Comments: slice,
	}, http.StatusOK, ""
}

func (s *store) addComment(name, sessionID, body string) (commentView, int, string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return commentView{}, http.StatusBadRequest, "Введите комментарий."
	}
	if !utf8.ValidString(body) {
		return commentView{}, http.StatusBadRequest, "Комментарий должен быть в кодировке UTF-8."
	}
	if utf8.RuneCountInString(body) > commentMaxRunes {
		return commentView{}, http.StatusBadRequest, "Максимум 1\u202f000 символов."
	}
	id, err := newID()
	if err != nil {
		return commentView{}, http.StatusInternalServerError, "Не удалось сохранить комментарий."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	page := s.byName[foldKey.String(name)]
	if page == nil {
		return commentView{}, http.StatusNotFound, "Страница не найдена."
	}
	viewer := s.accountBySession(sessionID)
	if viewer == nil {
		return commentView{}, http.StatusUnauthorized, "Войдите, чтобы оставить комментарий."
	}
	mode := commentMode(page.commentAccess)
	if viewer != page && mode != "open" && mode != "users" {
		return commentView{}, http.StatusForbidden, "Комментарии на этой странице закрыты."
	}
	item := &profileComment{id: id, author: viewer, body: body, created: time.Now().UTC()}
	_, err = s.db.ExecContext(context.Background(), `
		INSERT INTO profile_comments (id, account_id, author_id, body, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		item.id, page.id, viewer.id, item.body, item.created)
	if err != nil {
		return commentView{}, http.StatusInternalServerError, "Не удалось сохранить комментарий."
	}
	page.comments = append(page.comments, item)
	return commentView{
		ID: item.id, Author: viewer.name, Body: item.body, Created: item.created, CanDelete: true,
		HasAvatar: viewer.hasAvatar, AvatarUpdated: avatarStamp(viewer.avatarUpdated),
	}, http.StatusCreated, ""
}

func (s *store) deleteComment(name, sessionID, commentID string) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page := s.byName[foldKey.String(name)]
	if page == nil {
		return http.StatusNotFound, "Страница не найдена."
	}
	viewer := s.accountBySession(sessionID)
	if viewer == nil {
		return http.StatusUnauthorized, "Войдите в аккаунт."
	}
	for i, item := range page.comments {
		if item.id != commentID {
			continue
		}
		if viewer != page && viewer != item.author {
			return http.StatusForbidden, "Этот комментарий может удалить автор или владелец страницы."
		}
		if _, err := s.db.ExecContext(context.Background(), `DELETE FROM profile_comments WHERE id = $1`, item.id); err != nil {
			return http.StatusInternalServerError, "Не удалось удалить комментарий."
		}
		page.comments = append(page.comments[:i], page.comments[i+1:]...)
		return http.StatusOK, ""
	}
	return http.StatusNotFound, "Комментарий не найден."
}

func (a *app) listComments(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	list, code, msg := a.accounts.commentList(c.Param("name"), sessionID(c), page)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/json; charset=utf-8")
	return c.JSON(code, list)
}

func (a *app) postComment(c echo.Context) error {
	var req struct {
		Body string `json:"body"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите комментарий.")
	}
	view, code, msg := a.accounts.addComment(c.Param("name"), sessionID(c), req.Body)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/json; charset=utf-8")
	return c.JSON(code, view)
}

func (a *app) removeComment(c echo.Context) error {
	code, msg := a.accounts.deleteComment(c.Param("name"), sessionID(c), c.Param("id"))
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.NoContent(http.StatusNoContent)
}

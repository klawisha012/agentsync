package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
)

const sessionCookie = "session"

var (
	foldKey  = cases.Fold()
	reserved = map[string]struct{}{
		"login": {}, "recover": {}, "accounts": {}, "catalog": {},
		"health": {}, "backend": {},
	}
)

type account struct {
	email    string
	emailKey string
	name     string
	nameKey  string
	password []byte
	views    int
	likes    int
}

type accountView struct {
	Name  string `json:"name"`
	Views int    `json:"views"`
	Likes int    `json:"likes"`
}

type explanationBody struct {
	Explanation string `json:"explanation"`
}

type store struct {
	mu       sync.Mutex
	byEmail  map[string]*account
	byName   map[string]*account
	sessions map[string]*account
}

func newStore() *store {
	return &store{
		byEmail:  map[string]*account{},
		byName:   map[string]*account{},
		sessions: map[string]*account{},
	}
}

func (s *store) create(email, password, name string) (accountView, string, int, string) {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	if email == "" || password == "" || name == "" {
		return accountView{}, "", http.StatusBadRequest, "Введите почту, пароль и имя."
	}
	if !validEmail(email) {
		return accountView{}, "", http.StatusBadRequest, "Введите почту в виде name@example.com."
	}
	if msg, code := checkName(name); msg != "" {
		return accountView{}, "", code, msg
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	item := &account{
		email:    email,
		emailKey: foldKey.String(email),
		name:     name,
		nameKey:  foldKey.String(name),
		password: hash,
	}
	id, err := newSessionID()
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.byEmail[item.emailKey]; taken {
		return accountView{}, "", http.StatusConflict, "Аккаунт с этой почтой уже есть."
	}
	if _, taken := s.byName[item.nameKey]; taken {
		return accountView{}, "", http.StatusConflict, "Это имя уже занято."
	}
	s.byEmail[item.emailKey] = item
	s.byName[item.nameKey] = item
	s.sessions[id] = item
	return item.view(), id, http.StatusCreated, ""
}

func (s *store) open(email, password string) (accountView, string, int, string) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return accountView{}, "", http.StatusBadRequest, "Введите почту и пароль."
	}
	s.mu.Lock()
	item := s.byEmail[foldKey.String(email)]
	var hash []byte
	if item != nil {
		hash = append([]byte(nil), item.password...)
	}
	s.mu.Unlock()

	if item == nil || bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
		return accountView{}, "", http.StatusUnauthorized, "Неверная почта или пароль."
	}
	id, err := newSessionID()
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось войти."
	}
	s.mu.Lock()
	s.sessions[id] = item
	view := item.view()
	s.mu.Unlock()
	return view, id, http.StatusOK, ""
}

func (s *store) close(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *store) session(id string) (accountView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.sessions[id]
	if item == nil {
		return accountView{}, false
	}
	return item.view(), true
}

func (s *store) public(name string) (accountView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.byName[foldKey.String(name)]
	if item == nil {
		return accountView{}, false
	}
	return item.view(), true
}

func (a account) view() accountView {
	return accountView{Name: a.name, Views: a.views, Likes: a.likes}
}

func checkName(name string) (string, int) {
	runes := []rune(name)
	shape := "Имя: от 3 до 32 знаков, буквы, цифры и дефис не по краям."
	if len(runes) < 3 || len(runes) > 32 || runes[0] == '-' || runes[len(runes)-1] == '-' {
		return shape, http.StatusBadRequest
	}
	for _, r := range runes {
		if r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return shape, http.StatusBadRequest
	}
	if _, blocked := reserved[foldKey.String(name)]; blocked {
		return "Это имя занято страницей сайта.", http.StatusConflict
	}
	return "", 0
}

func validEmail(email string) bool {
	if strings.ContainsAny(email, " \t") {
		return false
	}
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" || strings.Contains(domain, "@") {
		return false
	}
	return strings.Contains(domain, ".")
}

func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (a *app) createAccount(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите почту, пароль и имя.")
	}
	view, id, code, msg := a.accounts.create(req.Email, req.Password, req.Name)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	setSessionCookie(c, id, false)
	return c.JSON(code, view)
}

func (a *app) createSession(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите почту и пароль.")
	}
	view, id, code, msg := a.accounts.open(req.Email, req.Password)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	setSessionCookie(c, id, false)
	return c.JSON(code, view)
}

func (a *app) deleteSession(c echo.Context) error {
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		a.accounts.close(cookie.Value)
	}
	setSessionCookie(c, "", true)
	return c.NoContent(http.StatusNoContent)
}

func (a *app) currentSession(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	view, ok := a.accounts.session(cookie.Value)
	if !ok {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	return c.JSON(http.StatusOK, view)
}

func (a *app) publicAccount(c echo.Context) error {
	view, ok := a.accounts.public(c.Param("name"))
	if !ok {
		return writeExplanation(c, http.StatusNotFound, "Страница не найдена.")
	}
	return c.JSON(http.StatusOK, view)
}

func writeExplanation(c echo.Context, code int, text string) error {
	return c.JSON(code, explanationBody{Explanation: text})
}

func setSessionCookie(c echo.Context, id string, clear bool) {
	maxAge := 14 * 24 * 60 * 60
	if clear {
		maxAge = -1
		id = ""
	}
	secure := c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"
	c.SetCookie(&http.Cookie{
		Name:     sessionCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/cases"
)

const sessionCookie = "session"

var (
	foldKey  = cases.Fold()
	reserved = map[string]struct{}{
		"login": {}, "recover": {}, "accounts": {}, "catalog": {},
		"health": {}, "backend": {}, "docs": {}, "stats": {},
	}
)

type account struct {
	id             string
	email          string
	emailKey       string
	name           string
	nameKey        string
	password       []byte
	views          int
	likes          int
	verified       bool
	shareCopy      bool
	shareView      bool
	shareVersions  bool
	commentAccess  string
	hasAvatar      bool
	avatarUpdated  time.Time
	publishedAt    *time.Time
	confirmToken   string
	confirmExpires time.Time
	resetToken     string
	resetExpires   time.Time
	chains         map[string]string
	viewers        map[string]struct{}
	viewedAt       map[string]time.Time
	agentLikes     map[string]map[string]struct{}
	likedAt        map[string]map[string]time.Time
	publications   []*publication
	comments       []*profileComment
	snapshots      []*snapshot
	marks          map[string]string
}

type accountView struct {
	Name          string     `json:"name"`
	Views         int        `json:"views"`
	Likes         int        `json:"likes"`
	HasAvatar     bool       `json:"hasAvatar"`
	AvatarUpdated *time.Time `json:"avatarUpdated,omitempty"`
}

type explanationBody struct {
	Explanation string `json:"explanation"`
}

type store struct {
	mu            sync.Mutex
	db            *sql.DB
	byEmail       map[string]*account
	byName        map[string]*account
	sessions      map[string]*account
	sessionExpiry map[string]time.Time
	machines      map[string]*machine
	byAgentToken  map[string]*machine
	byConfirm     map[string]*account
	byReset       map[string]*account
	spentConfirms map[string]time.Time
	letters       map[string]string
}

func newStore(ctx context.Context, db *sql.DB) (*store, error) {
	s := &store{
		db:            db,
		byEmail:       map[string]*account{},
		byName:        map[string]*account{},
		sessions:      map[string]*account{},
		sessionExpiry: map[string]time.Time{},
		machines:      map[string]*machine{},
		byAgentToken:  map[string]*machine{},
		byConfirm:     map[string]*account{},
		byReset:       map[string]*account{},
		spentConfirms: map[string]time.Time{},
		letters:       map[string]string{},
	}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	return s, nil
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
	if msg := passwordPolicy(password); msg != "" {
		return accountView{}, "", http.StatusBadRequest, msg
	}
	hash, err := hashPassword(password)
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	accountID, err := newID()
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	item := &account{
		id:            accountID,
		email:         email,
		emailKey:      foldKey.String(email),
		name:          name,
		nameKey:       foldKey.String(name),
		password:      hash,
		chains:        map[string]string{},
		viewers:       map[string]struct{}{},
		viewedAt:      map[string]time.Time{},
		agentLikes:    map[string]map[string]struct{}{},
		likedAt:       map[string]map[string]time.Time{},
		publications:  []*publication{},
		comments:      []*profileComment{},
		commentAccess: "hidden",
		snapshots:     []*snapshot{},
		marks:         map[string]string{},
	}
	id, err := newID()
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	letterToken, err := newID()
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	item.confirmToken = hashToken(letterToken)
	item.confirmExpires = time.Now().Add(letterTTL)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.byEmail[item.emailKey]; taken {
		return accountView{}, "", http.StatusConflict, "Аккаунт с этой почтой уже есть."
	}
	if _, taken := s.byName[item.nameKey]; taken {
		return accountView{}, "", http.StatusConflict, "Это имя уже занято."
	}
	ctx := context.Background()
	if err := s.insertAccount(ctx, item); err != nil {
		if msg, ok := uniqueMessage(err); ok {
			return accountView{}, "", http.StatusConflict, msg
		}
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	if err := s.insertSession(ctx, id, item); err != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, item.id)
		return accountView{}, "", http.StatusInternalServerError, "Не удалось создать аккаунт."
	}
	s.byEmail[item.emailKey] = item
	s.byName[item.nameKey] = item
	s.byConfirm[item.confirmToken] = item
	s.keepLetter(item.email, "confirm", letterToken)
	return item.view(), id, http.StatusCreated, ""
}

func (s *store) open(email, password string) (accountView, string, int, string) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return accountView{}, "", http.StatusBadRequest, "Введите почту и пароль."
	}
	key := foldKey.String(email)
	s.mu.Lock()
	item := s.byEmail[key]
	hash := append([]byte(nil), dummyPasswordHash...)
	if item != nil {
		hash = append([]byte(nil), item.password...)
	}
	s.mu.Unlock()

	if err := checkPassword(hash, password); err != nil || item == nil {
		return accountView{}, "", http.StatusUnauthorized, "Неверная почта или пароль."
	}
	id, err := newID()
	if err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось войти."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.byEmail[key]
	if current == nil || !sameBytes(current.password, hash) {
		return accountView{}, "", http.StatusUnauthorized, "Неверная почта или пароль."
	}
	if err := s.insertSession(context.Background(), id, current); err != nil {
		return accountView{}, "", http.StatusInternalServerError, "Не удалось войти."
	}
	return current.view(), id, http.StatusOK, ""
}

func (s *store) close(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteSession(context.Background(), id)
}

func (s *store) session(id string) (accountView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.accountBySession(id)
	if item == nil {
		return accountView{}, false
	}
	return item.view(), true
}

func (a account) view() accountView {
	return accountView{
		Name: a.name, Views: a.views, Likes: a.publicationLikes(),
		HasAvatar: a.hasAvatar, AvatarUpdated: avatarStamp(a.avatarUpdated),
	}
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

func newID() (string, error) {
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
	if cookie, err := c.Cookie(visitorCookie); err == nil {
		a.accounts.bindVisitor(id, cookie.Value)
	}
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
	if cookie, err := c.Cookie(visitorCookie); err == nil {
		a.accounts.bindVisitor(id, cookie.Value)
	}
	return c.JSON(code, view)
}

func (a *app) deleteSession(c echo.Context) error {
	if cookie, err := c.Cookie(sessionCookie); err == nil && cookie.Value != "" {
		if err := a.accounts.close(cookie.Value); err != nil {
			return writeExplanation(c, http.StatusInternalServerError, "Не удалось выйти.")
		}
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
	sessionID := ""
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		sessionID = cookie.Value
	}
	visitor := ""
	if !strings.EqualFold(c.Request().Header.Get("Sec-Fetch-Site"), "cross-site") {
		visitor = ensureVisitor(c)
	}
	view, code, msg := a.accounts.page(c.Param("name"), sessionID, visitor)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, view)
}

func (a *app) confirmEmail(c echo.Context) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, letterStale)
	}
	code, msg := a.accounts.confirmEmail(req.Token)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]bool{"verified": true})
}

func (a *app) requestRecovery(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, "Введите почту в виде name@example.com.")
	}
	code, msg := a.accounts.requestReset(req.Email)
	if code != http.StatusOK {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]string{"explanation": msg})
}

func (a *app) resetPassword(c echo.Context) error {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return writeExplanation(c, http.StatusBadRequest, letterStale)
	}
	code, msg := a.accounts.resetPassword(req.Token, req.Password)
	if msg != "" {
		return writeExplanation(c, code, msg)
	}
	return c.JSON(code, map[string]bool{"reset": true})
}

func (a *app) publicationGate(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	if _, ok := a.accounts.session(cookie.Value); !ok {
		return writeExplanation(c, http.StatusUnauthorized, "Войдите в аккаунт.")
	}
	return writeExplanation(c, http.StatusNotImplemented, "Сервер ещё не принимает загрузку и снятие ИИ-агента.")
}

func writeExplanation(c echo.Context, code int, text string) error {
	return c.JSON(code, explanationBody{Explanation: text})
}

func setSessionCookie(c echo.Context, id string, clear bool) {
	maxAge := int(sessionTTL.Seconds())
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

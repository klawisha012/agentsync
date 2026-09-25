package server

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const letterTTL = 24 * time.Hour

const mailClosed = "Подтвердите почту, чтобы загрузить или снять публикацию."
const letterStale = "Ссылка из письма недействительна или устарела."

type publicPage struct {
	Name     string `json:"name"`
	Views    int    `json:"views"`
	Likes    int    `json:"likes"`
	Verified bool   `json:"verified"`
}

type ownerPage struct {
	publicPage
	MaskedMail string        `json:"maskedMail,omitempty"`
	LetterPath string        `json:"letterPath,omitempty"`
	Machines   []machinePage `json:"machines,omitempty"`
}

func (s *store) confirmEmail(token string) (int, string) {
	if token == "" {
		return http.StatusBadRequest, letterStale
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if spent, ok := s.spentConfirms[token]; ok && time.Now().Before(spent.Add(letterTTL)) {
		return http.StatusOK, ""
	}
	item := s.accountByConfirm(token)
	if item == nil || !time.Now().Before(item.confirmExpires) {
		if item != nil {
			item.confirmToken = ""
		}
		return http.StatusBadRequest, letterStale
	}
	item.verified = true
	item.confirmToken = ""
	s.spentConfirms[token] = time.Now()
	return http.StatusOK, ""
}

func (s *store) requestReset(email string) (string, int, string) {
	email = strings.TrimSpace(email)
	ready := "Если эта почта есть, письмо со ссылкой уже готово."
	if email == "" || !validEmail(email) {
		return "", http.StatusBadRequest, "Введите почту в виде name@example.com."
	}
	token, err := newSessionID()
	if err != nil {
		return "", http.StatusInternalServerError, "Не удалось подготовить письмо."
	}
	// Нет почтового сервера: путь письма возвращается вызывающему.
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.byEmail[foldKey.String(email)]
	if item == nil {
		return "", http.StatusOK, ready
	}
	item.resetToken = token
	item.resetExpires = time.Now().Add(letterTTL)
	return "/recover/" + token, http.StatusOK, ready
}

func (s *store) resetPassword(token, password string) (int, string) {
	if strings.TrimSpace(password) == "" {
		return http.StatusBadRequest, "Введите новый пароль."
	}
	if token == "" {
		return http.StatusBadRequest, letterStale
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusInternalServerError, "Не удалось сменить пароль."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.accountByReset(token)
	if item == nil || !time.Now().Before(item.resetExpires) {
		if item != nil {
			item.resetToken = ""
		}
		return http.StatusBadRequest, letterStale
	}
	item.password = hash
	item.resetToken = ""
	return http.StatusOK, ""
}

func (s *store) sessionVerified(id string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.sessions[id]
	if item == nil {
		return false, false
	}
	return item.verified, true
}

func (s *store) page(name, sessionID string) (any, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.byName[foldKey.String(name)]
	if item == nil {
		return nil, http.StatusNotFound, "Страница не найдена."
	}
	public := publicPage{
		Name:     item.name,
		Views:    item.views,
		Likes:    item.likes,
		Verified: item.verified,
	}
	viewer := s.sessions[sessionID]
	if viewer == nil || viewer != item {
		return public, http.StatusOK, ""
	}
	owner := ownerPage{
		publicPage: public,
		MaskedMail: maskMail(item.email),
		Machines:   s.machinesOf(item),
	}
	if item.confirmToken != "" && time.Now().Before(item.confirmExpires) {
		owner.LetterPath = "/confirm/" + item.confirmToken
	}
	return owner, http.StatusOK, ""
}

func (s *store) accountByConfirm(token string) *account {
	for _, item := range s.byEmail {
		if item.confirmToken == token {
			return item
		}
	}
	return nil
}

func (s *store) accountByReset(token string) *account {
	for _, item := range s.byEmail {
		if item.resetToken == token {
			return item
		}
	}
	return nil
}

func maskMail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" {
		return ""
	}
	suffix := domain
	if dot := strings.LastIndex(domain, "."); dot >= 0 && dot < len(domain)-1 {
		suffix = domain[dot+1:]
	}
	return local + "@***." + suffix
}

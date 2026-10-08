package server

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const letterTTL = 24 * time.Hour

const letterStale = "Ссылка из письма недействительна или устарела."

type publicPage struct {
	Name          string      `json:"name"`
	Views         int         `json:"views"`
	Likes         int         `json:"likes"`
	Verified      bool        `json:"verified"`
	Liked         bool        `json:"liked"`
	CanLike       bool        `json:"canLike"`
	ShareCopy     bool        `json:"shareCopy"`
	ShareView     bool        `json:"shareView"`
	ShareVersions bool        `json:"shareVersions"`
	CommentAccess string      `json:"commentAccess"`
	HasAvatar     bool        `json:"hasAvatar"`
	AvatarUpdated *time.Time  `json:"avatarUpdated,omitempty"`
	Agents        []agentSlot `json:"agents"`
}

type ownerPage struct {
	publicPage
	MaskedMail string        `json:"maskedMail,omitempty"`
	Machines   []machinePage `json:"machines,omitempty"`
}

func (s *store) confirmEmail(token string) (int, string) {
	if token == "" {
		return http.StatusBadRequest, letterStale
	}
	key := hashToken(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	if spent, ok := s.spentConfirms[key]; ok && time.Now().Before(spent.Add(letterTTL)) {
		return http.StatusOK, ""
	}
	item := s.accountByToken(token, s.byConfirm, func(item *account) string { return item.confirmToken })
	if item == nil || !time.Now().Before(item.confirmExpires) {
		if item != nil {
			s.clearConfirm(item)
			_ = s.saveAccount(context.Background(), item)
		}
		return http.StatusBadRequest, letterStale
	}
	prevVerified, prevToken, prevExpires := item.verified, item.confirmToken, item.confirmExpires
	item.verified = true
	s.clearConfirm(item)
	used := time.Now()
	if err := s.saveAccount(context.Background(), item); err != nil {
		item.verified, item.confirmToken, item.confirmExpires = prevVerified, prevToken, prevExpires
		if prevToken != "" {
			s.byConfirm[prevToken] = item
		}
		return http.StatusInternalServerError, "Не удалось подтвердить почту."
	}
	if err := s.saveSpent(context.Background(), key, used); err != nil {
		item.verified, item.confirmToken, item.confirmExpires = prevVerified, prevToken, prevExpires
		if prevToken != "" {
			s.byConfirm[prevToken] = item
		}
		_ = s.saveAccount(context.Background(), item)
		return http.StatusInternalServerError, "Не удалось подтвердить почту."
	}
	s.spentConfirms[key] = used
	return http.StatusOK, ""
}

func (s *store) requestReset(email string) (int, string) {
	email = strings.TrimSpace(email)
	ready := "Если эта почта есть, письмо со ссылкой уже готово."
	if email == "" || !validEmail(email) {
		return http.StatusBadRequest, "Введите почту в виде name@example.com."
	}
	token, err := newID()
	if err != nil {
		return http.StatusInternalServerError, "Не удалось подготовить письмо."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.byEmail[foldKey.String(email)]
	if item == nil {
		return http.StatusOK, ready
	}
	prevToken, prevExpires := item.resetToken, item.resetExpires
	if prevToken != "" {
		delete(s.byReset, prevToken)
	}
	item.resetToken = hashToken(token)
	item.resetExpires = time.Now().Add(letterTTL)
	if err := s.saveAccount(context.Background(), item); err != nil {
		item.resetToken, item.resetExpires = prevToken, prevExpires
		if prevToken != "" {
			s.byReset[prevToken] = item
		}
		return http.StatusInternalServerError, "Не удалось подготовить письмо."
	}
	s.byReset[item.resetToken] = item
	s.keepLetter(item.email, "reset", token)
	return http.StatusOK, ready
}

func (s *store) resetPassword(token, password string) (int, string) {
	if strings.TrimSpace(password) == "" {
		return http.StatusBadRequest, "Введите новый пароль."
	}
	if msg := passwordPolicy(password); msg != "" {
		return http.StatusBadRequest, msg
	}
	if token == "" {
		return http.StatusBadRequest, letterStale
	}
	s.mu.Lock()
	item := s.accountByToken(token, s.byReset, func(item *account) string { return item.resetToken })
	fresh := item != nil && time.Now().Before(item.resetExpires)
	if !fresh {
		if item != nil {
			s.clearReset(item)
			_ = s.saveAccount(context.Background(), item)
		}
		s.mu.Unlock()
		return http.StatusBadRequest, letterStale
	}
	s.mu.Unlock()
	hash, err := hashPassword(password)
	if err != nil {
		return http.StatusInternalServerError, "Не удалось сменить пароль."
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item = s.accountByToken(token, s.byReset, func(item *account) string { return item.resetToken })
	if item == nil || !time.Now().Before(item.resetExpires) {
		if item != nil {
			s.clearReset(item)
			_ = s.saveAccount(context.Background(), item)
		}
		return http.StatusBadRequest, letterStale
	}
	prevHash := append([]byte(nil), item.password...)
	prevToken, prevExpires := item.resetToken, item.resetExpires
	item.password = hash
	s.clearReset(item)
	if err := s.saveAccount(context.Background(), item); err != nil {
		item.password, item.resetToken, item.resetExpires = prevHash, prevToken, prevExpires
		if prevToken != "" {
			s.byReset[prevToken] = item
		}
		return http.StatusInternalServerError, "Не удалось сменить пароль."
	}
	if err := s.revokeAccount(context.Background(), item, ""); err != nil {
		item.password, item.resetToken, item.resetExpires = prevHash, prevToken, prevExpires
		if prevToken != "" {
			s.byReset[prevToken] = item
		}
		_ = s.saveAccount(context.Background(), item)
		return http.StatusInternalServerError, "Не удалось сменить пароль."
	}
	return http.StatusOK, ""
}

func (s *store) page(name, sessionID, visitorID string) (any, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.byName[foldKey.String(name)]
	if item == nil {
		return nil, http.StatusNotFound, "Страница не найдена."
	}
	viewer := s.accountBySession(sessionID)
	s.recordView(item, viewer, visitorID)
	public := publicPage{
		Name:          item.name,
		Views:         item.views,
		Likes:         item.publicationLikes(),
		Verified:      item.verified,
		Agents:        item.withLikes(item.agentSlots(), viewer),
		ShareCopy:     item.shareCopy,
		ShareView:     item.shareView,
		ShareVersions: item.shareVersions,
		CommentAccess: commentMode(item.commentAccess),
		HasAvatar:     item.hasAvatar,
		AvatarUpdated: avatarStamp(item.avatarUpdated),
	}
	if viewer != nil && viewer != item {
		public.CanLike = true
	}
	if viewer == nil || viewer != item {
		return public, http.StatusOK, ""
	}
	owner := ownerPage{
		publicPage: public,
		MaskedMail: maskMail(item.email),
		Machines:   s.machinesOf(item),
	}
	return owner, http.StatusOK, ""
}

func (s *store) clearConfirm(item *account) {
	if item.confirmToken != "" {
		delete(s.byConfirm, item.confirmToken)
	}
	item.confirmToken = ""
	item.confirmExpires = time.Time{}
}

func (s *store) clearReset(item *account) {
	if item.resetToken != "" {
		delete(s.byReset, item.resetToken)
	}
	item.resetToken = ""
	item.resetExpires = time.Time{}
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

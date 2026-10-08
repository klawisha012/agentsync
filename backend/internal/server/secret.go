package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordRunes = 8

var (
	bcryptGate        = make(chan struct{}, 8)
	dummyPasswordHash []byte
	testStores        sync.Map
)

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte("agentsync-placeholder-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	dummyPasswordHash = hash
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

func sameToken(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func passwordPolicy(password string) string {
	trimmed := strings.TrimSpace(password)
	if trimmed == "" {
		return "Введите пароль."
	}
	if utf8.RuneCountInString(trimmed) < minPasswordRunes {
		return "Пароль: минимум 8 символов."
	}
	if len(password) > 72 {
		return "Пароль длиннее 72 байт."
	}
	return ""
}

func withBcrypt(fn func() error) error {
	bcryptGate <- struct{}{}
	defer func() { <-bcryptGate }()
	return fn()
}

func hashPassword(password string) ([]byte, error) {
	var hash []byte
	err := withBcrypt(func() error {
		var hashErr error
		hash, hashErr = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		return hashErr
	})
	return hash, err
}

func checkPassword(hash []byte, password string) error {
	return withBcrypt(func() error {
		return bcrypt.CompareHashAndPassword(hash, []byte(password))
	})
}

func (s *store) keepLetter(email, kind, raw string) {
	if !testing.Testing() || raw == "" {
		return
	}
	if s.letters == nil {
		s.letters = map[string]string{}
	}
	s.letters[foldKey.String(email)+"\n"+kind] = raw
}

func (s *store) accountByToken(raw string, index map[string]*account, field func(*account) string) *account {
	if raw == "" {
		return nil
	}
	key := hashToken(raw)
	item := index[key]
	if item == nil || !sameToken(field(item), key) {
		return nil
	}
	return item
}

func (s *store) machineByToken(raw string) *machine {
	if raw == "" {
		return nil
	}
	key := hashToken(raw)
	bound := s.byAgentToken[key]
	if bound == nil || !sameToken(bound.token, key) {
		return nil
	}
	return bound
}
